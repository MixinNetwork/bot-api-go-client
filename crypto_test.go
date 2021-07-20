package bot

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"testing"

	"filippo.io/edwards25519"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/curve25519"
)

func TestHashMember(t *testing.T) {
	assert := assert.New(t)
	mems := []string{"0355e4c0-ba3a-4ba6-82c0-6311e332f57a", "506694b3-0c16-4bd2-8ea0-ba3de6840a5a", "834c17e1-1427-434a-a280-1b3cfee05111"}
	hash := HashMembers(mems)
	assert.Equal(hash, "36972be0d8ef46deb3974334aa2242bfcdab044cbb99864ecc7f6ddbb9ee8ed9")
}

func TestCurve25519Conversion(t *testing.T) {
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	fmt.Println(public)
	fmt.Println(private.Seed())
	var curve25519Private [32]byte
	PrivateKeyToCurve25519(&curve25519Private, private)
	curve25519Public, err := curve25519.X25519(curve25519Private[:], curve25519.Basepoint)
	if err != nil {
		t.Fatalf("PublicKeyToCurve25519 failed")
	}

	p, err := (&edwards25519.Point{}).SetBytes(public)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(p.BytesMontgomery())

	curve25519Public2, err := PublicKeyToCurve25519(public)
	if err != nil {
		t.Fatalf("PublicKeyToCurve25519 failed")
	}

	fmt.Println(curve25519Public)
	if !bytes.Equal(curve25519Public, curve25519Public2[:]) {
		t.Errorf("Values didn't match: curve25519 produced %x, conversion produced %x", curve25519Public[:], curve25519Public2[:])
	}
}
