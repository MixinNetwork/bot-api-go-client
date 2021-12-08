package bot

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"time"

	"golang.org/x/crypto/curve25519"
)

func EncryptPIN(ctx context.Context, pin, pinToken, sessionId, privateKey string, iterator uint64) (string, error) {
	_, err := base64.RawURLEncoding.DecodeString(privateKey)
	if err == nil {
		return EncryptEd25519PIN(ctx, pin, pinToken, privateKey, iterator)
	}
	privBlock, _ := pem.Decode([]byte(privateKey))
	if privBlock == nil {
		return "", errors.New("invalid pem private key")
	}
	priv, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		return "", err
	}
	token, _ := base64.StdEncoding.DecodeString(pinToken)
	keyBytes, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, token, []byte(sessionId))
	if err != nil {
		return "", err
	}
	pinByte := []byte(pin)
	timeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timeBytes, uint64(time.Now().Unix()))
	pinByte = append(pinByte, timeBytes...)
	iteratorBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(iteratorBytes, iterator)
	pinByte = append(pinByte, iteratorBytes...)
	padding := aes.BlockSize - len(pinByte)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	pinByte = append(pinByte, padtext...)
	block, err := aes.NewCipher(keyBytes)
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
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func EncryptEd25519PIN(ctx context.Context, pin, pinTokenBase64, privateKey string, iterator uint64) (string, error) {
	privateBytes, err := base64.RawURLEncoding.DecodeString(privateKey)
	if err != nil {
		return "", err
	}

	private := ed25519.PrivateKey(privateBytes)
	public, err := base64.RawURLEncoding.DecodeString(pinTokenBase64)
	if err != nil {
		return "", err
	}
	var keyBytes, curvePriv, pub [32]byte
	PrivateKeyToCurve25519(&curvePriv, private)
	copy(pub[:], public[:])
	curve25519.ScalarMult(&keyBytes, &curvePriv, &pub)

	pinByte := []byte(pin)
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

func VerifyPIN(ctx context.Context, uid, pin, pinToken, sessionId, privateKey string) (*User, error) {
	var err error
	var encryptedPIN string
	pt, err := base64.RawURLEncoding.DecodeString(pinToken)
	if err == nil && len(pt) == 32 {
		encryptedPIN, err = EncryptEd25519PIN(ctx, pin, pinToken, privateKey, uint64(time.Now().UnixNano()))
	} else {
		encryptedPIN, err = EncryptPIN(ctx, pin, pinToken, sessionId, privateKey, uint64(time.Now().UnixNano()))
	}
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(map[string]interface{}{
		"pin": encryptedPIN,
	})
	if err != nil {
		return nil, err
	}
	path := "/pin/verify"
	token, err := SignAuthenticationToken(uid, sessionId, privateKey, "POST", path, string(data))
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
