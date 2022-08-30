package bot

import (
	"crypto/ed25519"
	"crypto/sha512"
	"sort"

	"filippo.io/edwards25519"
	"github.com/MixinNetwork/mixin/crypto"
	"golang.org/x/crypto/curve25519"
)

func PrivateKeyToCurve25519(curve25519Private *[32]byte, privateKey ed25519.PrivateKey) {
	h := sha512.New()
	h.Write(privateKey.Seed())
	digest := h.Sum(nil)

	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64

	copy(curve25519Private[:], digest)
}

func PublicKeyToCurve25519(publicKey ed25519.PublicKey) ([]byte, error) {
	p, err := (&edwards25519.Point{}).SetBytes(publicKey[:])
	if err != nil {
		return nil, err
	}
	return p.BytesMontgomery(), nil
}

func SharedKey(public ed25519.PublicKey, private ed25519.PrivateKey) ([32]byte, error) {
	var dst, priv, pub [32]byte
	curve25519Public, err := PublicKeyToCurve25519(public)
	if err != nil {
		return dst, err
	}

	PrivateKeyToCurve25519(&priv, private.Seed())
	copy(pub[:], curve25519Public[:])
	d, err := curve25519.X25519(priv[:], pub[:])
	if err != nil {
		return dst, err
	}
	copy(dst[:], d)
	return dst, nil
}

func HashMembers(ids []string) string {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var in string
	for _, id := range ids {
		in = in + id
	}
	return crypto.NewHash([]byte(in)).String()
}
