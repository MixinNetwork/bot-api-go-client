package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/MixinNetwork/bot-api-go-client/v2"
	"github.com/urfave/cli/v2"
)

type Bot struct {
	Pin        string `json:"pin"`
	ClientID   string `json:"client_id"`
	SessionID  string `json:"session_id"`
	PinToken   string `json:"pin_token"`
	PrivateKey string `json:"private_key"`
}

func main() {
	app := &cli.App{
		Name:    "mixin-bot",
		Usage:   "Mixin bot API cli",
		Version: "2.0.1",
		Commands: []*cli.Command{
			{
				Name:    "transfer",
				Aliases: []string{"t"},
				Action:  transferCmd,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "asset,a",
						Usage: "the asset id",
					},
					&cli.StringFlag{
						Name:  "amount,z",
						Usage: "the asset amount",
					},
					&cli.StringFlag{
						Name:  "receiver,r",
						Usage: "receiver",
					},
					&cli.StringFlag{
						Name:  "keystore,k",
						Usage: "keystore",
					},
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}

func transferCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	asset := c.String("asset")
	amount := c.String("amount")
	receiver := c.String("receiver")

	dat, err := os.ReadFile(keystore)
	if err != nil {
		panic(err)
	}
	var user Bot
	err = json.Unmarshal([]byte(dat), &user)
	if err != nil {
		panic(err)
	}

	su := &bot.SafeUser{
		UserId:     user.ClientID,
		SessionId:  user.SessionID,
		SessionKey: user.PrivateKey,
		SpendKey:   user.Pin[:64],
	}

	ma := bot.NewUUIDMixAddress([]string{receiver}, 1)
	tr := &bot.TransactionRecipient{MixAddress: ma.String(), Amount: amount}
	trace := bot.UuidNewV4().String()
	log.Println("trace:", trace)
	tx, err := bot.SendTransaction(context.Background(), asset, []*bot.TransactionRecipient{tr}, trace, su)
	if err != nil {
		return err
	}
	log.Println("tx:", tx.PayloadHash().String())
	return nil
}
