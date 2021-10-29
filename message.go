package bot

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"strings"

	"golang.org/x/crypto/curve25519"
)

type LiveMessagePayload struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	ThumbUrl string `json:"thumb_url"`
	Url      string `json:"url"`
}

type RecallMessagePayload struct {
	MessageId string `json:"message_id"`
}

type MessageRequest struct {
	ConversationId   string `json:"conversation_id"`
	RecipientId      string `json:"recipient_id"`
	MessageId        string `json:"message_id"`
	Category         string `json:"category"`
	Data             string `json:"data"`
	RepresentativeId string `json:"representative_id"`
	QuoteMessageId   string `json:"quote_message_id"`
}

type ReceiptAcknowledgementRequest struct {
	MessageId string `json:"message_id"`
	Status    string `json:"status"`
}

func PostMessages(ctx context.Context, messages []*MessageRequest, clientId, sessionId, secret string) error {
	msg, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	accessToken, err := SignAuthenticationToken(clientId, sessionId, secret, "POST", "/messages", string(msg))
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

func PostMessage(ctx context.Context, conversationId, recipientId, messageId, category, data string, clientId, sessionId, secret string) error {
	request := MessageRequest{
		ConversationId: conversationId,
		RecipientId:    recipientId,
		MessageId:      messageId,
		Category:       category,
		Data:           data,
	}
	return PostMessages(ctx, []*MessageRequest{&request}, clientId, sessionId, secret)
}

func PostAcknowledgements(ctx context.Context, requests []*ReceiptAcknowledgementRequest, clientId, sessionId, secret string) error {
	array, err := json.Marshal(requests)
	if err != nil {
		return err
	}
	path := "/acknowledgements"
	accessToken, err := SignAuthenticationToken(clientId, sessionId, secret, "POST", path, string(array))
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

func UniqueConversationId(userId, recipientId string) string {
	minId, maxId := userId, recipientId
	if strings.Compare(userId, recipientId) > 0 {
		maxId, minId = userId, recipientId
	}
	h := md5.New()
	io.WriteString(h, minId)
	io.WriteString(h, maxId)
	sum := h.Sum(nil)
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	id, _ := UuidFromBytes(sum)
	return id.String()
}

func Chunked(source []interface{}, size int) [][]interface{} {
	var result [][]interface{}
	index := 0
	for index < len(source) {
		end := index + size
		if end >= len(source) {
			end = len(source)
		}
		result = append(result, source[index:end])
		index += size
	}
	return result
}

func EncryptMessageData(data string, sessions []*Session, privateKey string) (string, error) {
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

	privateBytes, err := base64.RawURLEncoding.DecodeString(privateKey)
	if err != nil {
		return "", err
	}

	private := ed25519.PrivateKey(privateBytes)
	pub, _ := PublicKeyToCurve25519(ed25519.PublicKey(private[32:]))

	var sessionsBytes []byte
	for _, s := range sessions {
		clientPublic, err := base64.RawURLEncoding.DecodeString(s.PublicKey)
		if err != nil {
			return "", err
		}
		var dst, priv, clientPub [32]byte
		copy(clientPub[:], clientPublic[:])
		PrivateKeyToCurve25519(&priv, private)
		curve25519.ScalarMult(&dst, &priv, &clientPub)

		block, err := aes.NewCipher(dst[:])
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

func DecryptMessageData(data string, sessionId, private string) (string, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	size := 16 + 48 // session id bytes + encypted key bytes size
	total := len(bytes)
	if total < 1+2+32+size+12 {
		return "", nil
	}
	sessionLen := int(binary.LittleEndian.Uint16(bytes[1:3]))
	prefixSize := 35 + sessionLen*size
	var key []byte
	for i := 35; i < prefixSize; i += size {
		if uid, _ := UuidFromBytes(bytes[i : i+16]); uid.String() == sessionId {
			private, err := base64.RawURLEncoding.DecodeString(private)
			if err != nil {
				return "", err
			}
			var dst, priv, pub [32]byte
			copy(pub[:], bytes[3:35])
			PrivateKeyToCurve25519(&priv, ed25519.PrivateKey(private))
			curve25519.ScalarMult(&dst, &priv, &pub)

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
