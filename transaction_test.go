package bot

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type BotTest struct {
	Pin        string `json:"pin"`
	ClientID   string `json:"client_id"`
	SessionID  string `json:"session_id"`
	PinToken   string `json:"pin_token"`
	PrivateKey string `json:"private_key"`
}

func TestTIPTransaction(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	assert := assert.New(t)
	ctx := context.Background()

	var bot BotTest
	err := json.Unmarshal([]byte(botData), &bot)
	assert.Nil(err)
	su := &SafeUser{
		UserId:            bot.ClientID,
		SessionId:         bot.SessionID,
		SessionPrivateKey: bot.PrivateKey,
		SpendPrivateKey:   bot.Pin[:64],
	}

	method := "GET"
	path := "/safe/me"
	token, err := SignAuthenticationTokenWithoutBody(method, path, su)
	assert.Nil(err)

	me, err := UserMe(ctx, token)
	assert.Nil(err)
	assert.NotNil(me)
	log.Printf("%#v", me)

	me, err = VerifyPINTip(ctx, su)
	assert.Nil(err)
	assert.NotNil(me)

	members := HashMembers([]string{bot.ClientID})
	asset := "f3bed3e0f6738938c8988eb8853c5647baa263901deb217ee53586d5de831f3b" // candy
	outputs, err := ListUnspentOutputs(ctx, members, 1, asset, su)
	assert.Nil(err)
	assert.Len(outputs, 1)

	ma := NewUUIDMixAddress([]string{"e9e5b807-fa8b-455a-8dfa-b189d28310ff"}, 1)
	tr := &TransactionRecipient{MixAddress: ma, Amount: "0.013"}
	trace := UuidNewV4().String()
	log.Println("trace:", trace)
	tx, err := SendTransaction(ctx, asset, []*TransactionRecipient{tr}, trace, []byte("test-memo"), nil, su)
	assert.Nil(err)
	assert.NotNil(tx)

	/*
	 user, err := RegisterSafe(ctx, bot.ClientID, bot.Pin[64:], bot.Pin[:64], bot.ClientID, bot.SessionID, bot.PrivateKey, bot.Pin, bot.PinToken)
	 assert.Nil(err)
	 assert.NotNil(user)

	 me, err = UserMe(ctx, token)
	 assert.Nil(err)
	 assert.NotNil(me)
	 log.Printf("%#v", me)
	*/
}

const botData = `{
  "pin": "",
  "client_id": "8291d1bb-440c-4557-b69f-91dda17876d1",
  "session_id": "",
  "pin_token": "",
  "private_key": ""
}`
