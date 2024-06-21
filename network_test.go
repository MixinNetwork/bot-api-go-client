package bot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetwork(t *testing.T) {
	assert := assert.New(t)

	assets, err := ReadNetworkAssetsTop(context.Background())
	assert.Nil(err)
	assert.Len(assets, 314)
}
