package bot

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/crypto/curve25519"
)

type LiveMessagePayload struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	ThumbUrl string `json:"thumb_url"`
	Url      string `json:"url"`
}

type ImageMessagePayload struct {
	AttachmentId string `json:"attachment_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	MimeType     string `json:"mime_type"`
	Thumbnail    string `json:"thumbnail"`
	Size         int64  `json:"size"`
}

type RecallMessagePayload struct {
	MessageId string `json:"message_id"`
}

type MessageRequest struct {
	ConversationId   string `json:"conversation_id"`
	RecipientId      string `json:"recipient_id"`
	MessageId        string `json:"message_id"`
	Category         string `json:"category"`
	DataBase64       string `json:"data_base64"`
	RepresentativeId string `json:"representative_id"`
	QuoteMessageId   string `json:"quote_message_id"`
	Silent           bool   `json:"silent"`
}

const (
	EncryptedMessageStateSuccess = "SUCCESS"
	EncryptedMessageStateFailed  = "FAILED"
)

type EncryptedMessageResponse struct {
	Type        string     `json:"type"`
	MessageId   string     `json:"message_id"`
	RecipientId string     `json:"recipient_id"`
	State       string     `json:"state"`
	Sessions    []*Session `json:"sessions"`
}

// EncryptedMessageError reports messages that still failed after their
// recipients' sessions were refreshed and the messages were retried.
type EncryptedMessageError struct {
	Responses []*EncryptedMessageResponse
}

func (e *EncryptedMessageError) Error() string {
	ids := make([]string, 0, len(e.Responses))
	for _, response := range e.Responses {
		if response != nil {
			ids = append(ids, response.MessageId)
		}
	}
	return fmt.Sprintf("encrypted messages failed after refreshing sessions: %s", strings.Join(ids, ", "))
}

type encryptedMessageRequest struct {
	*MessageRequest
	RecipientSessions []struct {
		SessionId string `json:"session_id"`
	} `json:"recipient_sessions"`
	Checksum string `json:"checksum"`
}

type ReceiptAcknowledgementRequest struct {
	MessageId string `json:"message_id"`
	Status    string `json:"status"`
}

func PostMessageRequest(ctx context.Context, message *MessageRequest, user *SafeUser) error {
	msg, err := json.Marshal(message)
	if err != nil {
		return err
	}
	accessToken, err := SignAuthenticationToken("POST", "/messages", string(msg), user)
	if err != nil {
		return err
	}
	body, err := Request(ctx, "POST", "/messages", msg, accessToken)
	if err != nil {
		return err
	}
	var resp struct {
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}
	if resp.Error.Code > 0 {
		return resp.Error
	}
	return nil
}

func PostMessages(ctx context.Context, messages []*MessageRequest, user *SafeUser) error {
	msg, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	accessToken, err := SignAuthenticationToken("POST", "/messages", string(msg), user)
	if err != nil {
		return err
	}
	body, err := Request(ctx, "POST", "/messages", msg, accessToken)
	if err != nil {
		return err
	}
	var resp struct {
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}
	if resp.Error.Code > 0 {
		return resp.Error
	}
	return nil
}

// PostEncryptedMessages encrypts and sends messages using each recipient's
// current sessions. Sessions are read from store when available and fetched
// from the API on a cache miss. If the API rejects a message because its
// sessions changed, only the failed messages are refreshed and retried once.
func PostEncryptedMessages(ctx context.Context, messages []*MessageRequest, store SessionStore, user *SafeUser) error {
	if len(messages) == 0 {
		return nil
	}
	if user == nil {
		return fmt.Errorf("safe user is nil")
	}

	sessionsByUser := make(map[string][]*Session)
	pending := messages
	for attempt := 0; attempt < 2; attempt++ {
		requests, err := buildEncryptedMessageRequests(ctx, pending, sessionsByUser, store, user)
		if err != nil {
			return err
		}
		responses, err := postEncryptedMessageRequests(ctx, requests, user)
		if err != nil {
			return err
		}

		failedMessages, failedResponses, err := encryptedMessageFailures(pending, responses)
		if err != nil {
			return err
		}
		if len(failedMessages) == 0 {
			return nil
		}

		expired := make(map[string]struct{})
		expiredIDs := make([]string, 0, len(failedMessages))
		for _, message := range failedMessages {
			if _, ok := expired[message.RecipientId]; !ok {
				expiredIDs = append(expiredIDs, message.RecipientId)
			}
			expired[message.RecipientId] = struct{}{}
		}
		for _, recipientID := range expiredIDs {
			delete(sessionsByUser, recipientID)
			if store != nil {
				_ = store.Delete(recipientID)
			}
		}

		if attempt == 1 {
			return &EncryptedMessageError{Responses: failedResponses}
		}
		refreshed, err := fetchRecipientSessions(ctx, expiredIDs, store, user)
		if err != nil {
			return err
		}
		for recipientID, sessions := range refreshed {
			sessionsByUser[recipientID] = sessions
		}
		pending = failedMessages
	}
	return nil
}

func buildEncryptedMessageRequests(ctx context.Context, messages []*MessageRequest, sessionsByUser map[string][]*Session, store SessionStore, user *SafeUser) ([]*encryptedMessageRequest, error) {
	missing := make([]string, 0, len(messages))
	checked := make(map[string]struct{}, len(messages))
	for _, message := range messages {
		if message == nil {
			return nil, fmt.Errorf("message is nil")
		}
		if message.RecipientId == "" {
			return nil, fmt.Errorf("message %s has no recipient_id", message.MessageId)
		}
		if _, ok := sessionsByUser[message.RecipientId]; ok {
			continue
		}
		if _, ok := checked[message.RecipientId]; ok {
			continue
		}
		checked[message.RecipientId] = struct{}{}
		if store != nil {
			if sessions, err := store.Get(message.RecipientId); err == nil {
				sessions = cloneSessions(sessions)
				if len(sessions) > 0 {
					sessionsByUser[message.RecipientId] = sessions
					continue
				}
			}
		}
		missing = append(missing, message.RecipientId)
	}
	if len(missing) > 0 {
		fetched, err := fetchRecipientSessions(ctx, missing, store, user)
		if err != nil {
			return nil, err
		}
		for recipientID, sessions := range fetched {
			sessionsByUser[recipientID] = sessions
		}
	}

	requests := make([]*encryptedMessageRequest, 0, len(messages))
	for _, message := range messages {
		messageSessions := cloneSessions(sessionsByUser[message.RecipientId])
		if len(messageSessions) == 0 {
			return nil, fmt.Errorf("no sessions found for recipient %s", message.RecipientId)
		}
		checksum := GenerateUserChecksum(messageSessions)
		data, err := EncryptMessageData(message.DataBase64, messageSessions, user.SessionPrivateKey)
		if err != nil {
			return nil, err
		}
		plain := *message
		plain.DataBase64 = data
		cipher := &encryptedMessageRequest{
			MessageRequest: &plain,
			Checksum:       checksum,
		}
		for _, s := range messageSessions {
			cipher.RecipientSessions = append(cipher.RecipientSessions, struct {
				SessionId string `json:"session_id"`
			}{SessionId: s.SessionID})
		}
		requests = append(requests, cipher)
	}
	return requests, nil
}

func fetchRecipientSessions(ctx context.Context, recipientIDs []string, store SessionStore, user *SafeUser) (map[string][]*Session, error) {
	if len(recipientIDs) == 0 {
		return map[string][]*Session{}, nil
	}
	userSessions, err := FetchUserSessions(ctx, recipientIDs, user)
	if err != nil {
		return nil, err
	}
	requested := make(map[string]struct{}, len(recipientIDs))
	for _, recipientID := range recipientIDs {
		requested[recipientID] = struct{}{}
	}
	sessionsByUser := make(map[string][]*Session, len(recipientIDs))
	for _, session := range userSessions {
		if session == nil {
			continue
		}
		recipientID := session.UserId
		if recipientID == "" && len(recipientIDs) == 1 {
			recipientID = recipientIDs[0]
		}
		if _, ok := requested[recipientID]; !ok {
			continue
		}
		sessionsByUser[recipientID] = append(sessionsByUser[recipientID], &Session{
			UserID:    recipientID,
			SessionID: session.SessionId,
			PublicKey: session.PublicKey,
		})
	}
	for _, recipientID := range recipientIDs {
		sessions := sessionsByUser[recipientID]
		if len(sessions) == 0 {
			return nil, fmt.Errorf("no sessions found for recipient %s", recipientID)
		}
		if store != nil {
			_ = store.Put(recipientID, sessions)
		}
	}
	return sessionsByUser, nil
}

func postEncryptedMessageRequests(ctx context.Context, requests []*encryptedMessageRequest, user *SafeUser) ([]*EncryptedMessageResponse, error) {
	data, err := json.Marshal(requests)
	if err != nil {
		return nil, err
	}
	const path = "/encrypted_messages"
	accessToken, err := SignAuthenticationToken("POST", path, string(data), user)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", path, data, accessToken)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  []*EncryptedMessageResponse `json:"data"`
		Error Error                       `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func encryptedMessageFailures(messages []*MessageRequest, responses []*EncryptedMessageResponse) ([]*MessageRequest, []*EncryptedMessageResponse, error) {
	responsesByID := make(map[string]*EncryptedMessageResponse, len(responses))
	for _, response := range responses {
		if response != nil {
			responsesByID[response.MessageId] = response
		}
	}

	var failedMessages []*MessageRequest
	var failedResponses []*EncryptedMessageResponse
	for _, message := range messages {
		response, ok := responsesByID[message.MessageId]
		if !ok {
			return nil, nil, fmt.Errorf("encrypted message response missing for %s", message.MessageId)
		}
		switch response.State {
		case EncryptedMessageStateSuccess:
		case EncryptedMessageStateFailed:
			failedMessages = append(failedMessages, message)
			failedResponses = append(failedResponses, response)
		default:
			return nil, nil, fmt.Errorf("encrypted message %s returned unknown state %q", message.MessageId, response.State)
		}
	}
	return failedMessages, failedResponses, nil
}

func PostMessage(ctx context.Context, conversationId, recipientId, messageId, category, dataBase64 string, user *SafeUser) error {
	request := MessageRequest{
		ConversationId: conversationId,
		RecipientId:    recipientId,
		MessageId:      messageId,
		Category:       category,
		DataBase64:     dataBase64,
	}
	return PostMessages(ctx, []*MessageRequest{&request}, user)
}

func PostAcknowledgements(ctx context.Context, requests []*ReceiptAcknowledgementRequest, user *SafeUser) error {
	array, err := json.Marshal(requests)
	if err != nil {
		return err
	}
	path := "/acknowledgements"
	accessToken, err := SignAuthenticationToken("POST", path, string(array), user)
	if err != nil {
		return err
	}
	body, err := Request(ctx, "POST", path, array, accessToken)
	if err != nil {
		return err
	}
	var resp struct {
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}
	if resp.Error.Code > 0 {
		return resp.Error
	}
	return nil
}

func EncryptMessageData(data string, sessions []*Session, sessionPrivateKey string) (string, error) {
	dataBytes, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}

	key := make([]byte, 16)
	_, err = rand.Read(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, 12)
	_, err = rand.Read(nonce)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ciphertext := aesgcm.Seal(nil, nonce, dataBytes, nil)

	var sessionLen [2]byte
	binary.LittleEndian.PutUint16(sessionLen[:], uint16(len(sessions)))

	private := ParseEd25519PrivateKey(sessionPrivateKey)
	pub, _ := PublicKeyToCurve25519(ed25519.PublicKey(private[32:]))

	var sessionsBytes []byte
	for _, s := range sessions {
		clientPublic, err := base64.RawURLEncoding.DecodeString(s.PublicKey)
		if err != nil {
			return "", err
		}
		var priv [32]byte
		PrivateKeyToCurve25519(&priv, private)
		dst, err := curve25519.X25519(priv[:], clientPublic)
		if err != nil {
			return "", err
		}
		block, err := aes.NewCipher(dst)
		if err != nil {
			return "", err
		}
		padding := aes.BlockSize - len(key)%aes.BlockSize
		padtext := bytes.Repeat([]byte{byte(padding)}, padding)
		shared := make([]byte, len(key))
		copy(shared[:], key[:])
		shared = append(shared, padtext...)
		ciphertext := make([]byte, aes.BlockSize+len(shared))
		iv := ciphertext[:aes.BlockSize]
		_, err = rand.Read(iv)
		if err != nil {
			return "", err
		}
		mode := cipher.NewCBCEncrypter(block, iv)
		mode.CryptBlocks(ciphertext[aes.BlockSize:], shared)
		id, err := UuidFromString(s.SessionID)
		if err != nil {
			return "", err
		}
		sessionsBytes = append(sessionsBytes, id.Bytes()...)
		sessionsBytes = append(sessionsBytes, ciphertext...)
	}

	result := []byte{1}
	result = append(result, sessionLen[:]...)
	result = append(result, pub[:]...)
	result = append(result, sessionsBytes...)
	result = append(result, nonce[:]...)
	result = append(result, ciphertext...)
	return base64.RawURLEncoding.EncodeToString(result), nil
}

func DecryptMessageData(data string, sessionId, sessionPrivateKey string) (string, error) {
	privateKey := ParseEd25519PrivateKey(sessionPrivateKey)
	bytes, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	size := 16 + 48 // session id bytes + encrypted key bytes size
	total := len(bytes)
	if total < 1+2+32+size+12 {
		return "", nil
	}
	sessionLen := int(binary.LittleEndian.Uint16(bytes[1:3]))
	prefixSize := 35 + sessionLen*size
	var key []byte
	for i := 35; i < prefixSize; i += size {
		if uid, _ := UuidFromBytes(bytes[i : i+16]); uid.String() == sessionId {
			var priv [32]byte
			var pub [32]byte
			copy(pub[:], bytes[3:35])
			PrivateKeyToCurve25519(&priv, privateKey)
			dst, err := curve25519.X25519(priv[:], pub[:])
			if err != nil {
				return "", err
			}

			block, err := aes.NewCipher(dst[:])
			if err != nil {
				return "", err
			}
			iv := bytes[i+16 : i+16+aes.BlockSize]
			key = bytes[i+16+aes.BlockSize : i+size]
			mode := cipher.NewCBCDecrypter(block, iv)
			mode.CryptBlocks(key, key)
			key = key[:16]
			break
		}
	}
	if len(key) != 16 {
		return "", nil
	}
	nonce := bytes[prefixSize : prefixSize+12]
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", nil // TODO
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", nil // TODO
	}
	plaintext, err := aesgcm.Open(nil, nonce, bytes[prefixSize+12:], nil)
	if err != nil {
		return "", nil // TODO
	}
	return base64.RawURLEncoding.EncodeToString(plaintext), nil
}
