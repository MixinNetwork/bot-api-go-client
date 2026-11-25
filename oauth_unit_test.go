package bot

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// same as the API oauthVerifyCodeChallenge and oauthValidateCodeChallenge
func testVerifyCodeChallenge(challenge, verifier string) bool {
	cb, err := base64.RawURLEncoding.DecodeString(challenge)
	if err != nil || len(cb) != 32 {
		return false
	}
	vb, err := base64.RawURLEncoding.DecodeString(verifier)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(vb)
	return challenge == base64.RawURLEncoding.EncodeToString(sum[:])
}

func assertTestOAuthRequest(t *testing.T, r *http.Request, body []byte, ou *OAuthUser, publicKey ed25519.PublicKey) {
	t.Helper()
	authorization := r.Header.Get("Authorization")
	require.True(t, strings.HasPrefix(authorization, "Bearer "))
	claims := parseTestJWT(t, strings.TrimPrefix(authorization, "Bearer "), publicKey)
	requestID := r.Header.Get("X-Request-Id")
	assert.NotEmpty(t, requestID)
	assert.Equal(t, requestID, claims["jti"])
	assert.Equal(t, ou.AppID, claims["iss"])
	assert.Equal(t, ou.AuthorizationID, claims["aid"])
	assert.Equal(t, ou.Scope, claims["scp"])
	assert.NotContains(t, claims, "uid")
	sum := sha256.Sum256([]byte(r.Method + r.URL.RequestURI() + string(body)))
	assert.Equal(t, hex.EncodeToString(sum[:]), claims["sig"])
}

func TestOAuthNewCodeVerifier(t *testing.T) {
	verifier, challenge, err := OAuthNewCodeVerifier()
	require.NoError(t, err)
	assert.True(t, testVerifyCodeChallenge(challenge, verifier))
	assert.GreaterOrEqual(t, len(challenge), 16)

	// RFC 7636 hashes the verifier string, which the API rejects
	rfc := sha256.Sum256([]byte(verifier))
	assert.False(t, testVerifyCodeChallenge(base64.RawURLEncoding.EncodeToString(rfc[:]), verifier))

	other, _, err := OAuthNewCodeVerifier()
	require.NoError(t, err)
	assert.NotEqual(t, verifier, other)
}

func TestOAuthExchangeCode(t *testing.T) {
	var registered ed25519.PublicKey
	var ou *OAuthUser
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		switch r.URL.Path {
		case "/oauth/token":
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Empty(t, r.Header.Values("Authorization"))
			var payload map[string]string
			require.NoError(t, json.Unmarshal(body, &payload))
			assert.Equal(t, "client", payload["client_id"])
			assert.Equal(t, "secret", payload["client_secret"])
			assert.Equal(t, "code", payload["code"])
			assert.Equal(t, "verifier", payload["code_verifier"])
			pub, err := base64.RawURLEncoding.DecodeString(payload["ed25519"])
			require.NoError(t, err)
			require.Len(t, pub, ed25519.PublicKeySize)
			registered = pub
			_, _ = w.Write([]byte(`{"data":{"ed25519":"server-public","scope":"PROFILE:READ ASSETS:READ","authorization_id":"auth-id"}}`))
		case "/safe/snapshots":
			assertTestOAuthRequest(t, r, body, ou, registered)
			_, _ = w.Write([]byte(`{"data":[]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))

	ctx := context.Background()
	var err error
	ou, err = OAuthExchangeCode(ctx, "client", "secret", "code", "verifier")
	require.NoError(t, err)
	assert.Equal(t, "client", ou.AppID)
	assert.Equal(t, "auth-id", ou.AuthorizationID)
	assert.Equal(t, "PROFILE:READ ASSETS:READ", ou.Scope)
	priv, err := parseEd25519PrivateKey(ou.PrivateKey)
	require.NoError(t, err)
	assert.Equal(t, registered, priv.Public())

	// the stored OAuthUser keeps working after a JSON round trip
	data, err := json.Marshal(ou)
	require.NoError(t, err)
	var stored OAuthUser
	require.NoError(t, json.Unmarshal(data, &stored))
	assert.Equal(t, *ou, stored)

	body, err := RequestWithOAuthUser(ctx, http.MethodGet, "/safe/snapshots?limit=10", nil, &stored)
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":[]}`, string(body))
}

func TestOAuthExchangeCodeErrors(t *testing.T) {
	tests := []struct {
		name     string
		response string
		code     int
	}{
		{name: "unauthorized", response: `{"error":{"code":401}}`, code: 401},
		{name: "forbidden", response: `{"error":{"code":403}}`, code: 403},
		{name: "no authorization id", response: `{"data":{"access_token":"rsa","scope":"PROFILE:READ"}}`, code: 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.response))
			}))
			ou, err := OAuthExchangeCode(context.Background(), "client", "secret", "code", "")
			require.Error(t, err)
			assert.Nil(t, ou)
			var apiError Error
			require.True(t, errors.As(err, &apiError))
			assert.Equal(t, tt.code, apiError.Code)
		})
	}
}

func TestRequestWithOAuthUser(t *testing.T) {
	user, privateKey := newTestSafeUser(t.Name())
	ou := &OAuthUser{
		AppID:           "e5b3d5c1-0d62-4b41-9a5e-1f1a8c3d2b7a",
		AuthorizationID: "7c0f4e0a-3b9d-4c55-8b7a-2d6c1e9f0a13",
		PrivateKey:      user.SessionPrivateKey,
		Scope:           "PROFILE:READ",
	}
	payload := []byte(`{"full_name":"OAuth"}`)
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, payload, body)
		assertTestOAuthRequest(t, r, body, ou, privateKey.Public().(ed25519.PublicKey))
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))

	_, err := RequestWithOAuthUser(context.Background(), http.MethodPost, "/me?source=oauth", payload, ou)
	require.NoError(t, err)

	_, err = RequestWithOAuthUser(context.Background(), http.MethodGet, "/safe/me", nil, nil)
	require.ErrorContains(t, err, "oauth user is nil")
}
