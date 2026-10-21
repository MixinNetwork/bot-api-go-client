package bot

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateUserSimple(t *testing.T) {
	requireLiveAPI(t)
	configureLiveAPIKey(t)
	assert := assert.New(t)
	pub, private, err := ed25519.GenerateKey(rand.Reader)
	assert.Nil(err)
	sessionPrivateKey := hex.EncodeToString(private.Seed())
	fmt.Println(sessionPrivateKey)
	sessionSecret := base64.RawURLEncoding.EncodeToString(pub[:])
	u, err := CreateUserSimple(context.Background(), sessionSecret, "abccc")
	assert.Nil(err)
	fmt.Println(u)
}
