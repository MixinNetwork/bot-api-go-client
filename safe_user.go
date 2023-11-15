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

func RegisterSafe(ctx context.Context, userId, public, seed, saltBase64 string, uid, sid, sessionKey, pin, pinToken string) (*User, error) {
	s, err := hex.DecodeString(seed)
	if err != nil {
		panic(err)
	}
	encryptedPIN, err := EncryptPIN(pin, pinToken, sid, sessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}
	private := ed25519.NewKeyFromSeed(s)
	h := crypto.Sha256Hash([]byte(userId))
	signBytes := ed25519.Sign(private, h[:])
	sign := base64.RawURLEncoding.EncodeToString(signBytes[:])

	data, _ := json.Marshal(map[string]string{
		"public_key":  public,
		"signature":   sign,
		"user_id":     userId,
		"pin_base64":  encryptedPIN,
		"salt_base64": saltBase64,
	})

	token, err := SignAuthenticationToken(uid, sid, privateKey, "POST", "/safe/users", string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", "/safe/users", data, token)
	if err != nil {
		panic(err)
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
