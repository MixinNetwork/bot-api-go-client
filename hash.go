package bot

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
)

func HmacSha256(key, data []byte) string {
	hash := hmac.New(sha256.New, key)
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

func HmacSha1(hmacKey string, data []byte) string {
	hash := hmac.New(sha1.New, []byte(hmacKey))
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}
