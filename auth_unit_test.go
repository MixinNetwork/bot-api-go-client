package bot

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignAuthenticationTokenClaims(t *testing.T) {
	user, privateKey := newTestSafeUser(t.Name())
	before := time.Now().Unix()
	tokenString, err := SignAuthenticationTokenWithRequestID(
		http.MethodPost,
		"/items?limit=1",
		`{"name":"item"}`,
		"fixed-request-id",
		user,
	)
	require.NoError(t, err)

	claims := parseTestJWT(t, tokenString, privateKey.Public().(ed25519.PublicKey))
	sum := sha256.Sum256([]byte(http.MethodPost + "/items?limit=1" + `{"name":"item"}`))
	assert.Equal(t, user.UserId, claims["uid"])
	assert.Equal(t, user.SessionId, claims["sid"])
	assert.Equal(t, "fixed-request-id", claims["jti"])
	assert.Equal(t, "FULL", claims["scp"])
	assert.Equal(t, hex.EncodeToString(sum[:]), claims["sig"])
	issuedAt := int64(claims["iat"].(float64))
	expiresAt := int64(claims["exp"].(float64))
	assert.GreaterOrEqual(t, issuedAt, before)
	assert.InDelta(t, int64(90*24*time.Hour/time.Second), expiresAt-issuedAt, 1)
}

func TestSignAuthenticationTokenWithoutBody(t *testing.T) {
	user, privateKey := newTestSafeUser(t.Name())
	tokenString, err := SignAuthenticationTokenWithoutBody(http.MethodGet, "/me", user)
	require.NoError(t, err)
	claims := parseTestJWT(t, tokenString, privateKey.Public().(ed25519.PublicKey))
	sum := sha256.Sum256([]byte(http.MethodGet + "/me"))
	assert.Equal(t, hex.EncodeToString(sum[:]), claims["sig"])
	_, err = uuid.FromString(claims["jti"].(string))
	assert.NoError(t, err)
}

func TestSignOAuthAccessTokenClaims(t *testing.T) {
	user, privateKey := newTestSafeUser(t.Name())
	tokenString, err := SignOauthAccessToken(
		user.UserId,
		"authorization-id",
		user.SessionPrivateKey,
		http.MethodGet,
		"/assets/asset-id",
		"",
		"ASSETS:READ",
		"request-id",
	)
	require.NoError(t, err)
	claims := parseTestJWT(t, tokenString, privateKey.Public().(ed25519.PublicKey))
	assert.Equal(t, user.UserId, claims["iss"])
	assert.Equal(t, "authorization-id", claims["aid"])
	assert.Equal(t, "ASSETS:READ", claims["scp"])
	assert.Equal(t, "request-id", claims["jti"])
}

func TestSigningRejectsMalformedPrivateKeys(t *testing.T) {
	for _, key := range []string{"not-hex", "00"} {
		t.Run(key, func(t *testing.T) {
			user := &SafeUser{SessionPrivateKey: key}
			_, err := SignAuthenticationToken(http.MethodGet, "/me", "", user)
			require.Error(t, err)
			_, err = SignOauthAccessToken("app", "authorization", key, http.MethodGet, "/me", "", "FULL", "id")
			require.Error(t, err)
		})
	}
	assert.Panics(t, func() { ParseEd25519PrivateKey("00") })
}

func TestAuthenticatorBuildJWT(t *testing.T) {
	user, privateKey := newTestSafeUser(t.Name())
	authenticator := NewAuthenticator(user.UserId, user.SessionId, user.SessionPrivateKey)
	tokenString, err := authenticator.BuildJWT(http.MethodDelete, "/items/1", "")
	require.NoError(t, err)
	claims := parseTestJWT(t, tokenString, privateKey.Public().(ed25519.PublicKey))
	assert.Equal(t, user.UserId, claims["uid"])
	assert.Equal(t, user.SessionId, claims["sid"])
}

func TestOAuthGetAccessToken(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		ed25519        string
		wantToken      string
		wantScope      string
		wantAuthID     string
		wantErrorCode  int
		responseStatus int
	}{
		{
			name:      "server issued token",
			response:  `{"data":{"access_token":"access","scope":"PROFILE:READ","ed25519":"local","authorization_id":"auth"}}`,
			wantToken: "access",
			wantScope: "PROFILE:READ",
		},
		{
			name:       "locally signed token data",
			response:   `{"data":{"access_token":"access","scope":"ASSETS:READ","ed25519":"generated-key","authorization_id":"auth-id"}}`,
			ed25519:    "requested-key",
			wantToken:  "generated-key",
			wantScope:  "ASSETS:READ",
			wantAuthID: "auth-id",
		},
		{name: "unauthorized", response: `{"error":{"code":401}}`, wantErrorCode: 401},
		{name: "forbidden", response: `{"error":{"code":403}}`, wantErrorCode: 403},
		{name: "API error", response: `{"error":{"status":202,"code":1000,"description":"bad"}}`, wantErrorCode: 500},
		{name: "invalid JSON", response: `{`, wantErrorCode: 10002},
		{name: "HTTP failure", response: `unavailable`, responseStatus: http.StatusServiceUnavailable, wantErrorCode: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/oauth/token", r.URL.Path)
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var payload map[string]string
				require.NoError(t, json.Unmarshal(body, &payload))
				assert.Equal(t, "client", payload["client_id"])
				assert.Equal(t, tt.ed25519, payload["ed25519"])
				status := tt.responseStatus
				if status == 0 {
					status = http.StatusOK
				}
				w.WriteHeader(status)
				_, _ = w.Write([]byte(tt.response))
			}))

			token, scope, authorizationID, err := OAuthGetAccessToken(
				context.Background(), "client", "secret", "code", "verifier", tt.ed25519,
			)
			if tt.wantErrorCode != 0 {
				require.Error(t, err)
				var apiError Error
				require.True(t, errors.As(err, &apiError))
				assert.Equal(t, tt.wantErrorCode, apiError.Code)
				assert.Empty(t, token)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantToken, token)
			assert.Equal(t, tt.wantScope, scope)
			assert.Equal(t, tt.wantAuthID, authorizationID)
		})
	}
}

func TestErrorConstructors(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		err         Error
		status      int
		code        int
		description string
	}{
		{BlazeServerError(ctx, nil), 500, 7000, "Blaze server error."},
		{ServerError(ctx, nil), 500, 500, "Internal Server Error"},
		{BadDataError(ctx), 202, 10002, "The request data has invalid field."},
		{AuthorizationError(ctx), 202, 401, "Unauthorized, maybe invalid token."},
		{ForbiddenError(ctx), 202, 403, "Forbidden"},
		{NotFoundError(ctx), 202, 404, "The endpoint is not found."},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.status, tt.err.Status)
		assert.Equal(t, tt.code, tt.err.Code)
		assert.Equal(t, tt.description, tt.err.Description)
		assert.Contains(t, tt.err.trace, tt.description)
		assert.JSONEq(t, string(mustJSON(t, map[string]any{
			"status": tt.status, "code": tt.code, "description": tt.description,
		})), tt.err.Error())
	}

	nested := ServerError(ctx, errors.New("root cause"))
	assert.Contains(t, nested.trace, "root cause")
	wrapper := ServerError(ctx, nested)
	assert.Contains(t, wrapper.trace, nested.trace)
}

func parseTestJWT(t *testing.T, tokenString string, publicKey ed25519.PublicKey) jwt.MapClaims {
	t.Helper()
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return publicKey, nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)
	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	return claims
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return data
}
