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
	expire := time.Now().UTC().Add(time.Hour * 24 * 30 * 3)
	sum := sha256.Sum256([]byte(method + uri + body))

	claims := jwt.MapClaims{
		"uid": su.UserId,
		"sid": su.SessionId,
		"iat": time.Now().UTC().Unix(),
		"exp": expire.Unix(),
		"jti": UuidNewV4().String(),
		"sig": hex.EncodeToString(sum[:]),
		"scp": "FULL",
	}
	priv, err := hex.DecodeString(su.SessionPrivateKey)
	if err != nil {
		return "", err
	}
	// more validate the private key
	if len(priv) != 32 {
		return "", fmt.Errorf("bad ed25519 private key %s", priv)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(ed25519.NewKeyFromSeed(priv))
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

	priv, err := hex.DecodeString(privateKey)
	if err != nil {
		if err != nil {
			return "", err
		}
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(ed25519.NewKeyFromSeed(priv))
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
