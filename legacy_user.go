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
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"time"
)

type UserUpgrade struct {
	UserId             string `json:"app_id"`
	SessionId          string `json:"session_id"`
	ServerPublicKeyHEX string `json:"server_public_key"`

	SessionPrivateKey string `json:"session_private_key"`
}

type KeystoreLegacy struct {
	Pin        string `json:"pin"`
	SessionId  string `json:"session_id"`
	PinToken   string `json:"pin_token"`
	PrivateKey string `json:"private_key"`
}

func UpgradeLegacyUser(ctx context.Context, kl *KeystoreLegacy) (*UserUpgrade, error) {
	privBlock, _ := pem.Decode([]byte(kl.PrivateKey))
	if privBlock == nil {
		return nil, errors.New("invalid pem private key")
	}
	// encrypt pin
	priv, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		return nil, err
	}
	token, _ := base64.StdEncoding.DecodeString(kl.PinToken)
	keyBytes, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, token, []byte(kl.SessionId))
	if err != nil {
		return nil, err
	}
	pinByte := []byte(kl.Pin)
	timeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timeBytes, uint64(time.Now().Unix()))
	pinByte = append(pinByte, timeBytes...)
	iteratorBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(iteratorBytes, 0)
	pinByte = append(pinByte, iteratorBytes...)
	padding := aes.BlockSize - len(pinByte)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	pinByte = append(pinByte, padtext...)
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize+len(pinByte))
	iv := ciphertext[:aes.BlockSize]
	_, err = io.ReadFull(rand.Reader, iv)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], pinByte)

	// session_secret_legacy
	pub := &priv.PublicKey
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}

	// new session_secret
	hash := sha512.Sum512(privBlock.Bytes)
	seed := hash[:32]
	privEd25519 := ed25519.NewKeyFromSeed(seed)
	pubEd25519 := privEd25519.Public()
	log.Printf("session_private_key: %x", seed)

	data, _ := json.Marshal(map[string]string{
		"session_secret_legacy": base64.RawURLEncoding.EncodeToString(pubBytes),
		"session_secret":        base64.RawURLEncoding.EncodeToString(pubEd25519.(ed25519.PublicKey)),
		"session_id":            kl.SessionId,
		"pin":                   base64.RawURLEncoding.EncodeToString(ciphertext),
	})
	body, err := Request(ctx, "POST", "/legacy/users", data, "")
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *UserUpgrade `json:"data"`
		Error Error        `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}

	uu := resp.Data
	uu.SessionPrivateKey = hex.EncodeToString(seed)
	return uu, nil
}
