package bot

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateUserSimple(t *testing.T) {
	assert := assert.New(t)
	WithAPIKey("", "", "")
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	assert.Nil(err)
	sessionSecret := base64.RawURLEncoding.EncodeToString(pub[:])
	u, err := CreateUserSimple(context.Background(), sessionSecret, "abccc")
	assert.Nil(err)
	fmt.Println(u)
}
