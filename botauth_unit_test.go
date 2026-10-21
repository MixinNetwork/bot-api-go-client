package bot

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"

	"golang.org/x/crypto/curve25519"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotAuthSignRequestWithCachedKey(t *testing.T) {
	user, _ := newTestSafeUser(t.Name())
	sharedKey := bytes.Repeat([]byte{7}, 32)
	cache := NewMapCache()
	require.NoError(t, cache.Put("bot-user", sharedKey))
	client := NewBotAuthClient(cache, user, nil)
	req, err := http.NewRequest(http.MethodPost, "https://example.test/resource?x=1", bytes.NewBufferString("payload"))
	require.NoError(t, err)

	signature, err := client.SignRequest(context.Background(), 123, "bot-user", req)
	require.NoError(t, err)
	data := []byte("123" + http.MethodPost + "/resource?x=1" + "payload")
	hash, err := hex.DecodeString(HmacSha256(sharedKey, data))
	require.NoError(t, err)
	want := base64.RawURLEncoding.EncodeToString(append([]byte(user.UserId), hash...))
	assert.Equal(t, want, signature)
	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, "payload", string(body), "signing must leave the request body readable")
}

func TestBotAuthFetchesAndCachesSharedKey(t *testing.T) {
	user, _ := newTestSafeUser(t.Name())
	recipientSeed := sha256.Sum256([]byte("recipient bot auth key"))
	recipientPrivate := ed25519.NewKeyFromSeed(recipientSeed[:])
	recipientPublic, err := PublicKeyToCurve25519(recipientPrivate.Public().(ed25519.PublicKey))
	require.NoError(t, err)
	const botUserID = "489cfe0b-08d8-47f4-a330-fff193cc8086"
	requests := 0
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, "/sessions/fetch", r.URL.Path)
		_, _ = fmt.Fprintf(w, `{"data":[{"user_id":"wrong","public_key":"bad"},{"user_id":%q,"session_id":"session","public_key":%q,"platform":"iOS"}]}`,
			botUserID, base64.RawURLEncoding.EncodeToString(recipientPublic))
	}))
	cache := NewMapCache()
	client := NewBotAuthClient(cache, user, slog.Default())

	sharedKey, err := client.getSharedKey(context.Background(), botUserID)
	require.NoError(t, err)
	senderPrivate, err := parseEd25519PrivateKey(user.SessionPrivateKey)
	require.NoError(t, err)
	var curvePrivate [32]byte
	PrivateKeyToCurve25519(&curvePrivate, senderPrivate)
	want, err := curve25519.X25519(curvePrivate[:], recipientPublic)
	require.NoError(t, err)
	assert.Equal(t, want, sharedKey)

	sharedKey[0] ^= 1
	cached, err := client.getSharedKey(context.Background(), botUserID)
	require.NoError(t, err)
	assert.Equal(t, want, cached, "cache values must not alias caller-owned buffers")
	platform, err := cache.Get(userPlatformPrefix + botUserID)
	require.NoError(t, err)
	assert.Equal(t, "iOS", string(platform))
	assert.Equal(t, 1, requests)
}

func TestBotAuthValidation(t *testing.T) {
	client := NewBotAuthClient(nil, nil, nil)
	_, err := client.SignRequest(context.Background(), 1, "user", &http.Request{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "safe user")

	user, _ := newTestSafeUser(t.Name())
	client = NewDefaultClient(user, nil)
	_, err = client.SignRequest(context.Background(), 1, "user", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request is nil")
}

func TestMapCachePreservesNilAndEmptyValues(t *testing.T) {
	cache := NewMapCache()
	for name, value := range map[string][]byte{"nil": nil, "empty": {}, "populated": []byte("value")} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, cache.Put(name, value))
			cached, err := cache.Get(name)
			require.NoError(t, err)
			assert.Equal(t, value, cached)
		})
	}
	missing, err := cache.Get("missing")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestMapCacheConcurrentAccessAndCopies(t *testing.T) {
	cache := NewMapCache()
	value := []byte("value")
	require.NoError(t, cache.Put("key", value))
	value[0] = 'X'
	got, err := cache.Get("key")
	require.NoError(t, err)
	assert.Equal(t, "value", string(got))
	got[0] = 'Y'
	again, err := cache.Get("key")
	require.NoError(t, err)
	assert.Equal(t, "value", string(again))

	var wait sync.WaitGroup
	for i := range 20 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			key := fmt.Sprintf("key-%d", i)
			_ = cache.Put(key, []byte(key))
			_, _ = cache.Get(key)
		}()
	}
	wait.Wait()
	require.NoError(t, cache.Delete("key"))
	deleted, err := cache.Get("key")
	require.NoError(t, err)
	assert.Nil(t, deleted)
}
