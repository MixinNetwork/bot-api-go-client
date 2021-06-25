package main

import (
	"context"
	"log"

	"github.com/MixinNetwork/bot-api-go-client"
)

type mixinBlazeHandler func(ctx context.Context, msg bot.MessageView, clientID string) error

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
