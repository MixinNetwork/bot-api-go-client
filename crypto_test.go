package bot

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"testing"

	"filippo.io/edwards25519"
	"golang.org/x/crypto/curve25519"
)

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
