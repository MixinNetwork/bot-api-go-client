package bot

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func SignAuthenticationTokenWithoutBody(method, uri string, user *SafeUser) (string, error) {
	return SignAuthenticationToken(method, uri, "", user)
}

func SignAuthenticationToken(method, uri, body string, su *SafeUser) (string, error) {
	return SignAuthenticationTokenWithRequestID(method, uri, body, UuidNewV4().String(), su)
}

func SignAuthenticationTokenWithRequestID(method, uri, body, requestID string, su *SafeUser) (string, error) {
	if su == nil {
		return "", fmt.Errorf("safe user is nil")
	}
	expire := time.Now().UTC().Add(time.Hour * 24 * 30 * 3)
	sum := sha256.Sum256([]byte(method + uri + body))

	claims := jwt.MapClaims{
		"uid": su.UserId,
		"sid": su.SessionId,
		"iat": time.Now().UTC().Unix(),
		"exp": expire.Unix(),
		"jti": requestID,
		"sig": hex.EncodeToString(sum[:]),
		"scp": "FULL",
	}
	priv, err := parseEd25519PrivateKey(su.SessionPrivateKey)
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(priv)
}

func SignOauthAccessToken(appID, authorizationID, privateKey, method, uri, body, scp string, requestID string) (string, error) {
	expire := time.Now().UTC().Add(time.Hour * 24 * 30 * 3)
	sum := sha256.Sum256([]byte(method + uri + body))
	claims := jwt.MapClaims{
		"iss": appID,
		"aid": authorizationID,
		"iat": time.Now().UTC().Unix(),
		"exp": expire.Unix(),
		"sig": hex.EncodeToString(sum[:]),
		"scp": scp,
		"jti": requestID,
	}

	priv, err := parseEd25519PrivateKey(privateKey)
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(priv)
}

func ParseEd25519PrivateKey(s string) ed25519.PrivateKey {
	priv, err := parseEd25519PrivateKey(s)
	if err != nil {
		panic(err)
	}
	return priv
}

func parseEd25519PrivateKey(s string) (ed25519.PrivateKey, error) {
	seed, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid ed25519 private key encoding: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid ed25519 private key length %d", len(seed))
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

// OAuthGetAccessToken get the access token of a user
// ed25519 is optional, only use it when you want to sign OAuth access token locally
func OAuthGetAccessToken(ctx context.Context, clientID, clientSecret string, authorizationCode string, codeVerifier string, ed25519 string) (string, string, string, error) {
	params, err := json.Marshal(map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          authorizationCode,
		"code_verifier": codeVerifier,
		"ed25519":       ed25519,
	})
	if err != nil {
		return "", "", "", BadDataError(ctx)
	}
	body, err := Request(ctx, "POST", "/oauth/token", params, "")
	if err != nil {
		return "", "", "", ServerError(ctx, err)
	}
	var resp struct {
		Data struct {
			Scope           string `json:"scope"`
			AccessToken     string `json:"access_token"`
			Ed25519         string `json:"ed25519"`
			AuthorizationID string `json:"authorization_id"`
		} `json:"data"`
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", "", "", BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		if resp.Error.Code == 401 {
			return "", "", "", AuthorizationError(ctx)
		}
		if resp.Error.Code == 403 {
			return "", "", "", ForbiddenError(ctx)
		}
		return "", "", "", ServerError(ctx, resp.Error)
	}
	if ed25519 == "" {
		return resp.Data.AccessToken, resp.Data.Scope, "", nil
	}
	return resp.Data.Ed25519, resp.Data.Scope, resp.Data.AuthorizationID, nil
}
