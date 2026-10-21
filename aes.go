package bot

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
)

func AesDecrypt(secret, b []byte) ([]byte, error) {
	aes, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(aes)
	if err != nil {
		return nil, err
	}
	if len(b) < aead.NonceSize() {
		return nil, fmt.Errorf("ciphertext is too short: got %d bytes, need at least %d", len(b), aead.NonceSize())
	}
	nonce := b[:aead.NonceSize()]
	cipher := b[aead.NonceSize():]
	return aead.Open(nil, nonce, cipher, nil)
}

func AesEncrypt(secret, b []byte) ([]byte, error) {
	aes, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(aes)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}
	cipher := aead.Seal(nil, nonce, b, nil)
	return append(nonce, cipher...), nil
}
