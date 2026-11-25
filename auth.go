package bot

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const jwtValidDuration = time.Minute * 5

func SignAuthenticationTokenWithoutBody(method, uri, requestID string, user *SafeUser) (string, error) {
	return SignAuthenticationTokenWithRequestID(method, uri, "", requestID, user)
}

// SignAuthenticationTokenWithRequestID includes requestID in the signed jti claim.
func SignAuthenticationTokenWithRequestID(method, uri, body, requestID string, su *SafeUser) (string, error) {
	if su == nil {
		return "", fmt.Errorf("safe user is nil")
	}
	return signRequestJWT(method, uri, body, requestID, su.SessionPrivateKey, jwt.MapClaims{
		"uid": su.UserId,
		"sid": su.SessionId,
		"scp": "FULL",
	})
}

func signRequestHeaders(header http.Header, method, uri string, body []byte, su *SafeUser) error {
	token, err := SignAuthenticationTokenWithRequestID(method, uri, string(body), header.Get(requestIDHeader), su)
	if err != nil {
		return err
	}
	header.Set("Authorization", "Bearer "+token)
	return nil
}

func SignOauthAccessToken(appID, authorizationID, privateKey, method, uri, body, scp, requestID string) (string, error) {
	return signRequestJWT(method, uri, body, requestID, privateKey, jwt.MapClaims{
		"iss": appID,
		"aid": authorizationID,
		"scp": scp,
	})
}

func oauthAccessTokenRequestID(accessToken string) (string, error) {
	claims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(accessToken, claims); err != nil {
		return "", fmt.Errorf("parse OAuth access token: %w", err)
	}
	jti, exists := claims["jti"]
	if !exists {
		return UuidNewV4().String(), nil
	}
	requestID, ok := jti.(string)
	if !ok {
		return "", fmt.Errorf("OAuth access token requires a string jti")
	}
	if err := validateRequestID(requestID); err != nil {
		return "", fmt.Errorf("invalid OAuth token jti: %w", err)
	}
	return requestID, nil
}

func signRequestJWT(method, uri, body, requestID, privateKey string, claims jwt.MapClaims) (string, error) {
	if err := validateRequestID(requestID); err != nil {
		return "", err
	}
	iat := time.Now().UTC()
	sum := sha256.Sum256([]byte(method + uri + body))
	claims["jti"] = requestID
	claims["iat"] = iat.Unix()
	claims["exp"] = iat.Add(jwtValidDuration).Unix()
	claims["sig"] = hex.EncodeToString(sum[:])

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
func OAuthGetAccessToken(ctx context.Context, clientID, clientSecret string, authorizationCode string, codeVerifier string, ed25519 string, su *SafeUser) (string, string, string, error) {
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
	body, err := RequestWithUser(ctx, "POST", "/oauth/token", params, su)
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
