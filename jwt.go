package bot

import (
	"crypto/ed25519"

	jwt "github.com/dgrijalva/jwt-go"
)

var Ed25519SigningMethod *EdDSASigningMethod

func init() {
	Ed25519SigningMethod = &EdDSASigningMethod{}
	jwt.RegisterSigningMethod("EdDSA", func() jwt.SigningMethod {
		return Ed25519SigningMethod
	})
}

type EdDSASigningMethod struct{}

func (sm *EdDSASigningMethod) Verify(signingString, signature string, key interface{}) error {
	var ed25519Key ed25519.PublicKey
	switch k := key.(type) {
	case ed25519.PublicKey:
		ed25519Key = k
	default:
		return jwt.ErrInvalidKeyType
	}
	sig, err := jwt.DecodeSegment(signature)
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519Key, []byte(signingString), sig) {
		return jwt.ErrECDSAVerification
	}
	return nil
}

func (sm *EdDSASigningMethod) Sign(signingString string, key interface{}) (string, error) {
	var ed25519Key ed25519.PrivateKey
	switch k := key.(type) {
	case ed25519.PrivateKey:
		ed25519Key = k
	default:
		return "", jwt.ErrInvalidKeyType
	}
	sig := ed25519.Sign(ed25519Key, []byte(signingString))
	return jwt.EncodeSegment(sig), nil
}

func (sm *EdDSASigningMethod) Alg() string {
	return "EdDSA"
}
