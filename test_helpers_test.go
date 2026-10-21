package bot

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func newTestSafeUser(label string) (*SafeUser, ed25519.PrivateKey) {
	seed := sha256.Sum256([]byte(label))
	privateKey := ed25519.NewKeyFromSeed(seed[:])
	return &SafeUser{
		UserId:            "e95b1d4e-4d49-4ac3-9402-988804458adc",
		SessionId:         "c76310d8-c563-499e-9866-c61ae2cbee11",
		SessionPrivateKey: hex.EncodeToString(seed[:]),
	}, privateKey
}

func useTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	oldClient := httpClient
	oldURI := httpUri
	oldUser := globalUser
	server := httptest.NewServer(handler)
	httpClient = server.Client()
	httpUri = server.URL
	t.Cleanup(func() {
		server.Close()
		httpClient = oldClient
		httpUri = oldURI
		globalUser = oldUser
	})
	return server
}

func requireLiveAPI(t *testing.T) {
	t.Helper()
	if os.Getenv("MIXIN_API_INTEGRATION") == "" {
		t.Skip("set MIXIN_API_INTEGRATION=1 to run live API integration tests")
	}
}

func configureLiveAPIKey(t *testing.T) {
	t.Helper()
	userID := os.Getenv("MIXIN_USER_ID")
	sessionID := os.Getenv("MIXIN_SESSION_ID")
	privateKey := os.Getenv("MIXIN_SESSION_PRIVATE_KEY")
	if userID == "" || sessionID == "" || privateKey == "" {
		t.Skip("MIXIN_USER_ID, MIXIN_SESSION_ID, and MIXIN_SESSION_PRIVATE_KEY are required")
	}
	WithAPIKey(userID, sessionID, privateKey)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
