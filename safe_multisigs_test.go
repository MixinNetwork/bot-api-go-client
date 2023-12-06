package bot

import (
	"context"
	"encoding/json"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTIPMultisigTransaction(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	var bot BotTest
	err := json.Unmarshal([]byte(botData), &bot)
	assert.Nil(err)
	method := "GET"
	path := "/safe/me"
	token, err := SignAuthenticationTokenWithoutBody(bot.ClientID, bot.SessionID, bot.PrivateKey, method, path)
	assert.Nil(err)

	me, err := UserMe(ctx, token)
	assert.Nil(err)
	assert.NotNil(me)
	assert.True(me.HasSafe)

	members := HashMembers([]string{bot.ClientID, "7766b24c-1a03-4c3a-83a3-b4358266875d"})
	asset := "f3bed3e0f6738938c8988eb8853c5647baa263901deb217ee53586d5de831f3b" // candy
	su := &SafeUser{
		UserId:     bot.ClientID,
		SessionId:  bot.SessionID,
		SessionKey: bot.PrivateKey,
		SpendKey:   bot.Pin[:64],
	}
	outputs, err := ListUnspentOutputs(ctx, members, 1, asset, su)
	assert.Nil(err)
	for _, o := range outputs {
		log.Printf(o.Amount)
	}

	ma := NewUUIDMixAddress([]string{bot.ClientID, "7766b24c-1a03-4c3a-83a3-b4358266875d"}, 1)
	tr := &TransactionRecipient{MixAddress: ma.String(), Amount: "0.00233"}
	trace := UuidNewV4().String()
	log.Println("trace:", trace)
	tx, err := SendMultisigTransaction(ctx, asset, []*TransactionRecipient{tr}, trace, []byte("test-memo"), nil, su)
	assert.Nil(err)
	assert.NotNil(tx)
}
