package bot

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
)

// OAuthUser is a user's OAuth authorization of an app, with the ed25519 key
// pair whose public key was registered at /oauth/token. Every request is
// signed locally with PrivateKey, the token is bound to that request's method,
// URI and body, and its jti is sent as X-Request-Id.
//
// It is a long lived credential of the user, store it encrypted. It stops
// working when the user revokes or authorizes the app again, or the
// authorization expires, then the API returns 401 and the user must authorize
// again. The API returns 403 when Scope lacks what an endpoint requires.
type OAuthUser struct {
	AppID           string `json:"app_id"`
	AuthorizationID string `json:"authorization_id"`
	PrivateKey      string `json:"private_key"` // hex ed25519 seed
	Scope           string `json:"scope"`
}

// OAuthNewCodeVerifier returns a PKCE code verifier and its code challenge.
// Send the challenge when requesting authorization and the verifier to
// OAuthExchangeCode. Mixin hashes the base64url decoded verifier bytes, not
// the verifier string as RFC 7636 does, so use this instead of a generic PKCE
// library.
func OAuthNewCodeVerifier() (verifier, challenge string, err error) {
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(seed)
	return base64.RawURLEncoding.EncodeToString(seed), base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

// OAuthExchangeCode exchanges an OAuth authorization code for an OAuthUser. It
// generates the ed25519 key pair and registers its public key, no SafeUser is
// needed. codeVerifier is required when the authorization was requested with a
// code challenge, otherwise clientSecret must be the app secret.
func OAuthExchangeCode(ctx context.Context, clientID, clientSecret, code, codeVerifier string) (*OAuthUser, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	_, scope, authorizationID, err := OAuthGetAccessToken(ctx, clientID, clientSecret, code, codeVerifier, base64.RawURLEncoding.EncodeToString(pub), nil)
	if err != nil {
		return nil, err
	}
	if authorizationID == "" {
		return nil, fmt.Errorf("oauth token response without authorization id")
	}
	return &OAuthUser{
		AppID:           clientID,
		AuthorizationID: authorizationID,
		PrivateKey:      hex.EncodeToString(priv.Seed()),
		Scope:           scope,
	}, nil
}

// RequestWithOAuthUser sends a request signed for ou and returns the raw
// response body. Any endpoint allowed by ou.Scope can be called.
func RequestWithOAuthUser(ctx context.Context, method, path string, body []byte, ou *OAuthUser) ([]byte, error) {
	req, err := newOAuthSignedRequest(ctx, method, path, body, ou)
	if err != nil {
		return nil, err
	}
	return doRequest(req)
}

func newOAuthSignedRequest(ctx context.Context, method, path string, body []byte, ou *OAuthUser) (*http.Request, error) {
	req, err := newAPIRequest(ctx, method, path, body, UuidNewV4().String())
	if err != nil {
		return nil, err
	}
	if err := signOAuthRequestHeaders(req.Header, req.Method, req.URL.RequestURI(), body, ou); err != nil {
		return nil, err
	}
	return req, nil
}

func signOAuthRequestHeaders(header http.Header, method, uri string, body []byte, ou *OAuthUser) error {
	if ou == nil {
		return fmt.Errorf("oauth user is nil")
	}
	token, err := SignOauthAccessToken(ou.AppID, ou.AuthorizationID, ou.PrivateKey, method, uri, string(body), ou.Scope, header.Get(requestIDHeader))
	if err != nil {
		return err
	}
	header.Set("Authorization", "Bearer "+token)
	return nil
}
