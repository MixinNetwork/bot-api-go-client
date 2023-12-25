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

// If you want to register safe user, you need to call UpdateTipPin upgrade TIP PIN first.
func RegisterSafe(ctx context.Context, userId, publicKey, seed string, su *SafeUser) (*User, error) {
	s, err := hex.DecodeString(seed)
	if err != nil {
		return nil, err
	}
	private := ed25519.NewKeyFromSeed(s)
	h := crypto.Sha256Hash([]byte(userId))
	signBytes := ed25519.Sign(private, h[:])
	signature := base64.RawURLEncoding.EncodeToString(signBytes[:])

	tipBody := TIPBodyForSequencerRegister(userId, publicKey)
	pinBuf, err := hex.DecodeString(su.SpendPrivateKey)
	if err != nil {
		return nil, err
	}
	sigBuf := ed25519.Sign(ed25519.PrivateKey(pinBuf), tipBody)

	encryptedPIN, err := EncryptEd25519PIN(hex.EncodeToString(sigBuf), uint64(time.Now().UnixNano()), su)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(map[string]string{
		"public_key": publicKey,
		"signature":  signature,
		"pin_base64": encryptedPIN,
	})

	token, err := SignAuthenticationToken("POST", "/safe/users", string(data), su)
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
