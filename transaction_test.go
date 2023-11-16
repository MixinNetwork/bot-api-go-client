package bot

import (
	"context"
	"encoding/json"
	"log"
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
	log.Printf("%#v", me)

	me, err = VerifyPINTip(ctx, bot.ClientID, bot.PinToken, bot.SessionID, bot.PrivateKey, bot.Pin)
	assert.Nil(err)
	assert.NotNil(me)

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
