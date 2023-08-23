package bot

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAesDecrypt(t *testing.T) {
	assert := assert.New(t)
	key := "19c0e76d8fe245c0fc5530dcd51f9ce98b77b96650f69f2d15aa744b0a51b895"
	v := []byte("Hello")
	k, err := hex.DecodeString(key)
	assert.Nil(err)

	b, err := AesEncrypt(k, v)
	assert.Nil(err)
	d, err := AesDecrypt(k, b)
	assert.Nil(err)
	assert.Equal(v, d)

	d = []byte(`{"uid":"","sid":"","seed":"","pt":"cL""x/="}`)
	d, err = AesDecrypt(k, d)
	assert.NotNil(err)
	assert.Empty(d)
}
