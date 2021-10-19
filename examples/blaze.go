package main

import (
	"context"
	"encoding/base64"
	"github.com/MixinNetwork/bot-api-go-client"
	"log"
)

type mixinBlazeHandler func(ctx context.Context, msg bot.MessageView, clientID string) error

func (f mixinBlazeHandler) OnTransfer(ctx context.Context, msg bot.MessageView, clientID string) error {
	bytes, _ := base64.StdEncoding.DecodeString(msg.Data)
	log.Println("onTransfer----------------", string(bytes))
	return nil
}

func (f mixinBlazeHandler) OnMessage(ctx context.Context, msg bot.MessageView, clientID string) error {
	return f(ctx, msg, clientID)
}

func (f mixinBlazeHandler) OnAckReceipt(ctx context.Context, msg bot.MessageView, clientID string) error {
	log.Println("ack Message...", msg)
	return nil
}

func (f mixinBlazeHandler) SyncAck() bool {
	return false
}

func main() {
	var ctx context.Context
	h := func(ctx context.Context, botMsg bot.MessageView, clientID string) error {
		log.Println(botMsg)
		return nil
	}
	for {
		client := bot.NewBlazeClient("", "", "")
		if err := client.Loop(ctx, mixinBlazeHandler(h)); err != nil {
			log.Println("test...", err)
		}
	}
}
