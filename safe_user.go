package bot

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
)

func RegisterSafe(ctx context.Context, userId, public, seed string, uid, sid, sessionKey, tipPin, pinToken string) (*User, error) {
	s, err := hex.DecodeString(seed)
	if err != nil {
		return nil, err
	}
	private := ed25519.NewKeyFromSeed(s)
	h := crypto.Sha256Hash([]byte(userId))
	signBytes := ed25519.Sign(private, h[:])
	signature := base64.RawURLEncoding.EncodeToString(signBytes[:])

	tipBody := TIPBodyForSequencerRegister(userId, public)
	pinBuf, err := hex.DecodeString(tipPin)
	if err != nil {
		return nil, err
	}
	sigBuf := ed25519.Sign(ed25519.PrivateKey(pinBuf), tipBody)

	encryptedPIN, err := EncryptEd25519PIN(hex.EncodeToString(sigBuf), pinToken, sessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(map[string]string{
		"public_key": public,
		"signature":  signature,
		"pin_base64": encryptedPIN,
	})

	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", "/safe/users", string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", "/safe/users", data, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  *User  `json:"data"`
		Error *Error `json:"error,omitempty"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp.Data, nil
}
