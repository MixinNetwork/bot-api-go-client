package bot

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
)

// If you want to register safe user, you need to call UpdateTipPin upgrade TIP PIN first.
func RegisterSafe(ctx context.Context, userId, spendPrivateKeyOrSeed string, su *SafeUser) (*UserMeView, error) {
	spend, err := hex.DecodeString(spendPrivateKeyOrSeed)
	if err != nil {
		return nil, err
	}
	var private ed25519.PrivateKey
	switch len(spend) {
	case ed25519.SeedSize:
		private = ed25519.NewKeyFromSeed(spend)
	case ed25519.PrivateKeySize:
		private = ed25519.PrivateKey(spend)
	default:
		return nil, fmt.Errorf("invalid seed length")
	}
	h := crypto.Sha256Hash([]byte(userId))
	signBytes := ed25519.Sign(private, h[:])
	signature := base64.RawURLEncoding.EncodeToString(signBytes[:])
	publicKey := hex.EncodeToString(private[32:])
	tipBody := TIPBodyForSequencerRegister(userId, publicKey)
	pinBuf, err := hex.DecodeString(su.SpendPrivateKey)
	if err != nil {
		return nil, err
	}
	if su.SpendPrivateKey != hex.EncodeToString(private) {
		panic("please use the same spend private key with tip private key, spend private key must not be empty")
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
		Data  *UserMeView `json:"data"`
		Error *Error      `json:"error,omitempty"`
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

func RegisterSafeWithSetupPin(ctx context.Context, su *SafeUser) (*UserMeView, error) {
	seed, err := hex.DecodeString(su.SpendPrivateKey)
	if err != nil {
		return nil, err
	}
	private := ed25519.NewKeyFromSeed(seed)
	spendPublicKey := private.Public().(ed25519.PublicKey)

	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, 1)
	pubTipBuf := append(spendPublicKey, counter...)
	encryptedPin, err := EncryptEd25519PIN(hex.EncodeToString(pubTipBuf), uint64(time.Now().UnixNano()), su)
	if err != nil {
		return nil, err
	}
	err = UpdatePin(ctx, "", encryptedPin, su)
	if err != nil {
		return nil, fmt.Errorf("update pin error: %w", err)
	}
	return RegisterSafe(ctx, su.UserId, su.SpendPrivateKey, su)
}

type BareUserKeyStore struct {
	AppId             string `json:"app_id"`
	SessionId         string `json:"session_id"`
	ServerPublicKey   string `json:"server_public_key"`
	SessionPrivateKey string `json:"session_private_key"`
	SpentPrivateKey   string `json:"spent_private_key"`
}

func RegisterSafeBareUser(ctx context.Context, su *SafeUser) (*User, error) {
	s, err := hex.DecodeString(su.SpendPrivateKey)
	if err != nil {
		return nil, err
	}
	private := ed25519.NewKeyFromSeed(s)
	h := crypto.Sha256Hash([]byte(su.UserId))
	signBytes := ed25519.Sign(private, h[:])
	signature := base64.RawURLEncoding.EncodeToString(signBytes[:])
	publicKey := hex.EncodeToString(private[32:])
	tipBody := TIPBodyForSequencerRegister(su.UserId, publicKey)
	sigBuf := ed25519.Sign(private, tipBody)

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
