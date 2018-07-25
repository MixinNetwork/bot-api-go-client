package bot

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

func SignAuthenticationToken(uid, sid, privateKey, method, uri, body string) (string, error) {
	expire := time.Now().UTC().Add(time.Hour * 24 * 30 * 3)
	sum := sha256.Sum256([]byte(method + uri + body))
	token := jwt.NewWithClaims(jwt.SigningMethodRS512, jwt.MapClaims{
		"uid": uid,
		"sid": sid,
		"iat": time.Now().UTC().Unix(),
		"exp": expire.Unix(),
		"jti": UuidNewV4().String(),
		"sig": hex.EncodeToString(sum[:]),
	})

	block, _ := pem.Decode([]byte(privateKey))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	return token.SignedString(key)
}

func OAuthGetAccessToken(ctx context.Context, clientId, clientSecret string, authorizationCode string, codeVerifier string) (string, string, error) {
	params, err := json.Marshal(map[string]string{
		"client_id":     clientId,
		"client_secret": clientSecret,
		"code":          authorizationCode,
		"code_verifier": codeVerifier,
	})
	if err != nil {
		return "", "", BadDataError(ctx)
	}
	body, err := Request(ctx, "POST", "/oauth/token", params, "")
	if err != nil {
		return "", "", ServerError(ctx, err)
	}
	var resp struct {
		Data struct {
			AccessToken string `json:"access_token"`
			Scope       string `json:"scope"`
		} `json:"data"`
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", "", BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		if resp.Error.Code == 401 {
			return "", "", AuthorizationError(ctx)
		}
		if resp.Error.Code == 403 {
			return "", "", ForbiddenError(ctx)
		}
		return "", "", ServerError(ctx, resp.Error)
	}
	return resp.Data.AccessToken, resp.Data.Scope, nil
}
