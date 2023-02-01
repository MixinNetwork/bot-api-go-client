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
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gofrs/uuid"
	"golang.org/x/crypto/curve25519"
)

func EncryptPIN(pin, pinToken, sessionId, privateKey string, iterator uint64) (string, error) {
	_, err := base64.RawURLEncoding.DecodeString(privateKey)
	if err == nil {
		return EncryptEd25519PIN(pin, pinToken, privateKey, iterator)
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

func EncryptEd25519PIN(pin, pinTokenBase64, privateKey string, iterator uint64) (string, error) {
	privateBytes, err := base64.RawURLEncoding.DecodeString(privateKey)
	if err != nil {
		return "", err
	}

	private := ed25519.PrivateKey(privateBytes)
	public, err := base64.RawURLEncoding.DecodeString(pinTokenBase64)
	if err != nil {
		return "", err
	}
	var curvePriv, pub [32]byte
	PrivateKeyToCurve25519(&curvePriv, private)
	copy(pub[:], public[:])
	keyBytes, err := curve25519.X25519(curvePriv[:], pub[:])
	if err != nil {
		return "", err
	}

	pinByte := []byte(pin)
	if len(pin) > 6 {
		pinByte, err = hex.DecodeString(pin)
		if err != nil {
			return "", err
		}
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

func VerifyPIN(ctx context.Context, uid, pin, pinToken, sessionId, privateKey string) (*User, error) {
	var err error
	var encryptedPIN string
	pt, err := base64.RawURLEncoding.DecodeString(pinToken)
	if err == nil && len(pt) == 32 {
		encryptedPIN, err = EncryptEd25519PIN(pin, pinToken, privateKey, uint64(time.Now().UnixNano()))
	} else {
		encryptedPIN, err = EncryptPIN(pin, pinToken, sessionId, privateKey, uint64(time.Now().UnixNano()))
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

func VerifyPINTip(ctx context.Context, uid, pinToken, sessionId, privateKey, privateTip string) (*User, error) {
	TIPVerify := "TIP:VERIFY:"
	timestamp := time.Now().UnixNano()
	tb := []byte(fmt.Sprintf("%s%032d", TIPVerify, timestamp))
	privateTipBuf, err := hex.DecodeString(privateTip)
	if err != nil {
		return nil, err
	}
	sig := ed25519.Sign(ed25519.PrivateKey(privateTipBuf), tb)
	source, err := EncryptEd25519PIN(hex.EncodeToString(sig), pinToken, privateKey, uint64(timestamp))
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
	token, err := SignAuthenticationToken(uid, sessionId, privateKey, "POST", path, string(data))
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
