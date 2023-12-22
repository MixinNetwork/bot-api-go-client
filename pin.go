package bot

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gofrs/uuid/v5"
	"golang.org/x/crypto/curve25519"
)

func EncryptEd25519PIN(pin string, iterator uint64, current *SafeUser) (string, error) {
	privateBytes, err := hex.DecodeString(current.SessionPrivateKey)
	if err != nil {
		return "", err
	}

	private := ed25519.NewKeyFromSeed(privateBytes)
	public, err := hex.DecodeString(current.ServerPublicKey)
	if err != nil {
		return "", err
	}
	var curvePriv, pub [32]byte
	PrivateKeyToCurve25519(&curvePriv, private)
	public, err = PublicKeyToCurve25519(ed25519.PublicKey(public))
	if err != nil {
		return "", err
	}
	copy(pub[:], public[:])
	keyBytes, err := curve25519.X25519(curvePriv[:], pub[:])
	if err != nil {
		return "", err
	}

	pinByte, err := hex.DecodeString(pin)
	if err != nil {
		return "", err
	}
	timeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timeBytes, uint64(time.Now().Unix()))
	pinByte = append(pinByte, timeBytes...)
	iteratorBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(iteratorBytes, iterator)
	pinByte = append(pinByte, iteratorBytes...)
	padding := aes.BlockSize - len(pinByte)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	pinByte = append(pinByte, padtext...)
	block, err := aes.NewCipher(keyBytes[:])
	if err != nil {
		return "", err
	}
	ciphertext := make([]byte, aes.BlockSize+len(pinByte))
	iv := ciphertext[:aes.BlockSize]
	_, err = io.ReadFull(rand.Reader, iv)
	if err != nil {
		return "", err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], pinByte)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func VerifyPIN(ctx context.Context, pin string, user *SafeUser) (*User, error) {
	encryptedPIN, err := EncryptEd25519PIN(pin, uint64(time.Now().UnixNano()), user)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(map[string]interface{}{
		"pin_base64": encryptedPIN,
	})
	if err != nil {
		return nil, err
	}
	path := "/pin/verify"
	token, err := SignAuthenticationToken("POST", path, string(data), user)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", path, data, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  *User `json:"data"`
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func VerifyPINTip(ctx context.Context, su *SafeUser) (*User, error) {
	TIPVerify := "TIP:VERIFY:"
	timestamp := time.Now().UnixNano()
	tb := []byte(fmt.Sprintf("%s%032d", TIPVerify, timestamp))
	pin, err := signTipBody(tb, su.SpendPrivateKey)
	source, err := EncryptEd25519PIN(pin, uint64(timestamp), su)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(map[string]interface{}{
		"pin_base64": source,
		"timestamp":  timestamp,
	})
	if err != nil {
		return nil, err
	}
	path := "/pin/verify"
	token, err := SignAuthenticationToken("POST", path, string(data), su)
	if err != nil {
		return nil, err
	}
	id := uuid.Must(uuid.NewV4()).String()
	log.Println(id)
	body, err := RequestWithId(ctx, "POST", path, data, token, id)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  *User `json:"data"`
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func signTipBody(body []byte, pin string) (string, error) {
	pinBuf, err := hex.DecodeString(pin)
	if err != nil {
		return "", err
	}
	if len(pinBuf) != 32 {
		return "", errors.New("invalid ed25519 private")
	}
	sigBuf := ed25519.Sign(ed25519.NewKeyFromSeed(pinBuf), body)
	return hex.EncodeToString(sigBuf), nil
}
