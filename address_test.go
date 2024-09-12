package bot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddress(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	address, err := CheckAddress(ctx, "c94ac88f-4671-3976-b60a-09064f1811e8", "0x1616b057f8a89955d4a4f9fd9eb10289ac0e44a1", "")
	assert.Nil(err)
	assert.NotNil(address)
	assert.Equal("0x1616b057F8a89955d4A4f9fd9Eb10289ac0e44A1", address.Destination)
	assert.Equal("", address.Tag)
}
