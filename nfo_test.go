package bot

import (
	"crypto/md5"
	"encoding/hex"
	"testing"

	"github.com/MixinNetwork/nfo/mtg"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNFODecode(t *testing.T) {
	assert := assert.New(t)
	// https://v2.viewblock.io/mixin/tx/1e654db8405066f6d45b08961651a711b55a3e0c104077cb118c2e29889f9efd
	b, err := hex.DecodeString("4e464f0001000000000000000143d61dcde413450d80b8101d5e903357143c8c161a18ae2c8b14fda1216fff7da88c419b5d103676a640111b42e4923efc4c68d6de400106204d27df6617015c7da6f606106a7f751bc1175b3fcee7ba3eea2e9fec693cff77")
	assert.Nil(err)
	nfo, err := mtg.DecodeNFOMemo(b)
	assert.Nil(err)
	assert.Equal("NFO", nfo.Prefix)
	assert.Equal("43d61dcd-e413-450d-80b8-101d5e903357", nfo.Chain.String())
	key := nfo.Chain.Bytes()
	key = append(key, nfo.Class...)
	key = append(key, nfo.Collection.Bytes()...)
	key = append(key, nfo.Token...)
	tokenId := uuidBytes(key)
	assert.Equal("8048de2d-8092-3ccc-a47d-e30da9764f05", tokenId)
}

func uuidBytes(b []byte) string {
	h := md5.New()
	h.Write(b)
	sum := h.Sum(nil)
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	return uuid.FromBytesOrNil(sum).String()
}
