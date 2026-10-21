package bot

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/curve25519"
)

type encryptedMessageTestRequest struct {
	ConversationId string `json:"conversation_id"`
	RecipientId    string `json:"recipient_id"`
	MessageId      string `json:"message_id"`
	Category       string `json:"category"`
	DataBase64     string `json:"data_base64"`
	Checksum       string `json:"checksum"`
}

func TestPostEncryptedMessagesUsesCachedSessions(t *testing.T) {
	recipientID := "11111111-1111-4111-8111-111111111111"
	session := encryptedMessageTestSession(t, recipientID, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", 1)
	store := NewMapSessionStore()
	require.NoError(t, store.Put(recipientID, []*Session{session}))

	message := encryptedMessageTestMessage(recipientID, "10000000-0000-4000-8000-000000000001", "hello")
	var encryptedCalls int
	server := encryptedMessageTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/encrypted_messages" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		encryptedCalls++
		var requests []*encryptedMessageTestRequest
		if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
			t.Errorf("decode encrypted request: %v", err)
		}
		if len(requests) != 1 {
			t.Errorf("got %d requests, want 1", len(requests))
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		request := requests[0]
		assert.Equal(t, GenerateUserChecksum([]*Session{session}), request.Checksum)
		assert.NotEqual(t, message.DataBase64, request.DataBase64)
		assert.Equal(t, []string{session.SessionID}, encryptedMessageSessionIDs(t, request.DataBase64))
		assert.NotEmpty(t, r.Header.Get("Authorization"))
		writeEncryptedMessageTestJSON(t, w, map[string]any{"data": []any{
			map[string]any{"type": "message", "message_id": message.MessageId, "recipient_id": recipientID, "state": EncryptedMessageStateSuccess},
		}})
	}))
	defer server.Close()

	err := PostEncryptedMessages(t.Context(), []*MessageRequest{message}, store, encryptedMessageTestUser())
	require.NoError(t, err)
	assert.Equal(t, 1, encryptedCalls)
}

func TestPostEncryptedMessagesFetchesAndCachesSessions(t *testing.T) {
	recipientID := "12121212-1212-4212-8212-121212121212"
	session := encryptedMessageTestSession(t, recipientID, "abababab-abab-4bab-8bab-abababababab", 9)
	store := NewMapSessionStore()
	var fetchCalls, encryptedCalls int
	server := encryptedMessageTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sessions/fetch":
			fetchCalls++
			writeEncryptedMessageTestJSON(t, w, map[string]any{"data": []any{
				map[string]any{"user_id": recipientID, "session_id": session.SessionID, "public_key": session.PublicKey},
			}})
		case "/encrypted_messages":
			encryptedCalls++
			var requests []*encryptedMessageTestRequest
			if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
				t.Errorf("decode encrypted request: %v", err)
			}
			if len(requests) != 1 {
				t.Errorf("got %d requests, want 1", len(requests))
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			assert.Equal(t, []string{session.SessionID}, encryptedMessageSessionIDs(t, requests[0].DataBase64))
			writeEncryptedMessageTestJSON(t, w, map[string]any{"data": []any{
				map[string]any{"message_id": requests[0].MessageId, "recipient_id": recipientID, "state": EncryptedMessageStateSuccess},
			}})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	first := encryptedMessageTestMessage(recipientID, "12000000-0000-4000-8000-000000000001", "first")
	second := encryptedMessageTestMessage(recipientID, "12000000-0000-4000-8000-000000000002", "second")
	require.NoError(t, PostEncryptedMessages(t.Context(), []*MessageRequest{first}, store, encryptedMessageTestUser()))
	require.NoError(t, PostEncryptedMessages(t.Context(), []*MessageRequest{second}, store, encryptedMessageTestUser()))
	assert.Equal(t, 1, fetchCalls)
	assert.Equal(t, 2, encryptedCalls)
}

func TestPostEncryptedMessagesRefreshesExpiredSessions(t *testing.T) {
	recipientID := "22222222-2222-4222-8222-222222222222"
	oldSession := encryptedMessageTestSession(t, recipientID, "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", 2)
	newSession := encryptedMessageTestSession(t, recipientID, "cccccccc-cccc-4ccc-8ccc-cccccccccccc", 3)
	store := NewMapSessionStore()
	require.NoError(t, store.Put(recipientID, []*Session{oldSession}))
	message := encryptedMessageTestMessage(recipientID, "20000000-0000-4000-8000-000000000002", "refresh me")

	var fetchCalls, encryptedCalls int
	server := encryptedMessageTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sessions/fetch":
			fetchCalls++
			var users []string
			if err := json.NewDecoder(r.Body).Decode(&users); err != nil {
				t.Errorf("decode session request: %v", err)
			}
			assert.Equal(t, []string{recipientID}, users)
			writeEncryptedMessageTestJSON(t, w, map[string]any{"data": []any{
				map[string]any{"user_id": recipientID, "session_id": newSession.SessionID, "public_key": newSession.PublicKey, "platform": "iOS"},
			}})
		case "/encrypted_messages":
			encryptedCalls++
			var requests []*encryptedMessageTestRequest
			if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
				t.Errorf("decode encrypted request: %v", err)
			}
			if len(requests) != 1 {
				t.Errorf("got %d requests, want 1", len(requests))
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if encryptedCalls == 1 {
				assert.Equal(t, []string{oldSession.SessionID}, encryptedMessageSessionIDs(t, requests[0].DataBase64))
				writeEncryptedMessageTestJSON(t, w, map[string]any{"data": []any{
					map[string]any{"type": "message", "message_id": message.MessageId, "recipient_id": recipientID, "state": EncryptedMessageStateFailed},
				}})
				return
			}
			assert.Equal(t, []string{newSession.SessionID}, encryptedMessageSessionIDs(t, requests[0].DataBase64))
			writeEncryptedMessageTestJSON(t, w, map[string]any{"data": []any{
				map[string]any{"type": "message", "message_id": message.MessageId, "recipient_id": recipientID, "state": EncryptedMessageStateSuccess},
			}})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	err := PostEncryptedMessages(t.Context(), []*MessageRequest{message}, store, encryptedMessageTestUser())
	require.NoError(t, err)
	assert.Equal(t, 1, fetchCalls)
	assert.Equal(t, 2, encryptedCalls)
	cached, err := store.Get(recipientID)
	require.NoError(t, err)
	require.Len(t, cached, 1)
	assert.Equal(t, newSession.SessionID, cached[0].SessionID)
}

func TestPostEncryptedMessagesRetriesOnlyFailedMessages(t *testing.T) {
	firstRecipient := "33333333-3333-4333-8333-333333333333"
	secondRecipient := "44444444-4444-4444-8444-444444444444"
	firstSession := encryptedMessageTestSession(t, firstRecipient, "dddddddd-dddd-4ddd-8ddd-dddddddddddd", 4)
	secondOldSession := encryptedMessageTestSession(t, secondRecipient, "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee", 5)
	secondNewSession := encryptedMessageTestSession(t, secondRecipient, "ffffffff-ffff-4fff-8fff-ffffffffffff", 6)
	store := NewMapSessionStore()
	require.NoError(t, store.Put(firstRecipient, []*Session{firstSession}))
	require.NoError(t, store.Put(secondRecipient, []*Session{secondOldSession}))

	firstMessage := encryptedMessageTestMessage(firstRecipient, "30000000-0000-4000-8000-000000000003", "already sent")
	secondMessage := encryptedMessageTestMessage(secondRecipient, "40000000-0000-4000-8000-000000000004", "retry me")
	messageCalls := make(map[string]int)
	var encryptedCalls int
	server := encryptedMessageTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sessions/fetch":
			var users []string
			if err := json.NewDecoder(r.Body).Decode(&users); err != nil {
				t.Errorf("decode session request: %v", err)
			}
			assert.Equal(t, []string{secondRecipient}, users)
			writeEncryptedMessageTestJSON(t, w, map[string]any{"data": []any{
				map[string]any{"user_id": secondRecipient, "session_id": secondNewSession.SessionID, "public_key": secondNewSession.PublicKey},
			}})
		case "/encrypted_messages":
			encryptedCalls++
			var requests []*encryptedMessageTestRequest
			if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
				t.Errorf("decode encrypted request: %v", err)
			}
			for _, request := range requests {
				messageCalls[request.MessageId]++
			}
			if encryptedCalls == 1 {
				assert.Len(t, requests, 2)
				writeEncryptedMessageTestJSON(t, w, map[string]any{"data": []any{
					map[string]any{"message_id": firstMessage.MessageId, "recipient_id": firstRecipient, "state": EncryptedMessageStateSuccess},
					map[string]any{"message_id": secondMessage.MessageId, "recipient_id": secondRecipient, "state": EncryptedMessageStateFailed},
				}})
				return
			}
			if len(requests) != 1 || requests[0].MessageId != secondMessage.MessageId {
				t.Errorf("retry contains unexpected messages: %#v", requests)
			}
			writeEncryptedMessageTestJSON(t, w, map[string]any{"data": []any{
				map[string]any{"message_id": secondMessage.MessageId, "recipient_id": secondRecipient, "state": EncryptedMessageStateSuccess},
			}})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	err := PostEncryptedMessages(t.Context(), []*MessageRequest{firstMessage, secondMessage}, store, encryptedMessageTestUser())
	require.NoError(t, err)
	assert.Equal(t, 1, messageCalls[firstMessage.MessageId])
	assert.Equal(t, 2, messageCalls[secondMessage.MessageId])
}

func TestPostEncryptedMessagesReturnsFailuresAfterRefresh(t *testing.T) {
	recipientID := "55555555-5555-4555-8555-555555555555"
	oldSession := encryptedMessageTestSession(t, recipientID, "10101010-1010-4010-8010-101010101010", 10)
	newSession := encryptedMessageTestSession(t, recipientID, "20202020-2020-4020-8020-202020202020", 11)
	store := NewMapSessionStore()
	require.NoError(t, store.Put(recipientID, []*Session{oldSession}))
	message := encryptedMessageTestMessage(recipientID, "50000000-0000-4000-8000-000000000005", "still failing")

	server := encryptedMessageTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sessions/fetch":
			writeEncryptedMessageTestJSON(t, w, map[string]any{"data": []any{
				map[string]any{"user_id": recipientID, "session_id": newSession.SessionID, "public_key": newSession.PublicKey},
			}})
		case "/encrypted_messages":
			writeEncryptedMessageTestJSON(t, w, map[string]any{"data": []any{
				map[string]any{"message_id": message.MessageId, "recipient_id": recipientID, "state": EncryptedMessageStateFailed},
			}})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	err := PostEncryptedMessages(t.Context(), []*MessageRequest{message}, store, encryptedMessageTestUser())
	var failure *EncryptedMessageError
	require.ErrorAs(t, err, &failure)
	require.Len(t, failure.Responses, 1)
	assert.Equal(t, message.MessageId, failure.Responses[0].MessageId)
	cached, cacheErr := store.Get(recipientID)
	require.NoError(t, cacheErr)
	assert.Empty(t, cached)
}

func encryptedMessageTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	previousClient, previousURI := httpClient, httpUri
	httpClient, httpUri = server.Client(), server.URL
	t.Cleanup(func() {
		httpClient, httpUri = previousClient, previousURI
	})
	return server
}

func encryptedMessageTestUser() *SafeUser {
	return &SafeUser{
		UserId:            "99999999-9999-4999-8999-999999999999",
		SessionId:         "88888888-8888-4888-8888-888888888888",
		SessionPrivateKey: hex.EncodeToString(bytes.Repeat([]byte{7}, 32)),
	}
}

func encryptedMessageTestMessage(recipientID, messageID, data string) *MessageRequest {
	return &MessageRequest{
		ConversationId: "77777777-7777-4777-8777-777777777777",
		RecipientId:    recipientID,
		MessageId:      messageID,
		Category:       MessageCategoryPlainText,
		DataBase64:     base64.RawURLEncoding.EncodeToString([]byte(data)),
	}
}

func encryptedMessageTestSession(t *testing.T, userID, sessionID string, scalar byte) *Session {
	t.Helper()
	publicKey, err := curve25519.X25519(bytes.Repeat([]byte{scalar}, 32), curve25519.Basepoint)
	require.NoError(t, err)
	return &Session{
		UserID:    userID,
		SessionID: sessionID,
		PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
	}
}

func encryptedMessageSessionIDs(t *testing.T, data string) []string {
	t.Helper()
	decoded, err := base64.RawURLEncoding.DecodeString(data)
	if !assert.NoError(t, err) || !assert.GreaterOrEqual(t, len(decoded), 35) {
		return nil
	}
	count := int(binary.LittleEndian.Uint16(decoded[1:3]))
	ids := make([]string, 0, count)
	for offset := 35; len(ids) < count; offset += 64 {
		if !assert.GreaterOrEqual(t, len(decoded), offset+16) {
			return nil
		}
		id, err := UuidFromBytes(decoded[offset : offset+16])
		if !assert.NoError(t, err) {
			return nil
		}
		ids = append(ids, id.String())
	}
	return ids
}

func writeEncryptedMessageTestJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode response: %v", err)
	}
}
