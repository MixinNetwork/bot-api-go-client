package bot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFiats(t *testing.T) {
	assert := assert.New(t)
	a, err := Fiats(context.Background())
	assert.Nil(err)
	println(len(a))
}
