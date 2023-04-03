package bot

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadCode(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	c, err := ReadCode[*MultisigRequest](ctx, "c76310d8-c563-499e-9866-c61ae2cbee11")
	assert.Nil(err)
	fmt.Println(c)
}
