package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "mixin-bot"
	app.Usage = "Mixin bot API cli"
	app.Version = "1.0.0"
	app.Commands = []cli.Command{
		{
			Name:    "transfer",
			Aliases: []string{"t"},
			Usage:   "Transfer asset to Mixin Mainnet",
			Action:  transferCmd,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "asset,a",
					Usage: "the asset id",
				},
				cli.StringFlag{
					Name:  "opponent_key,k",
					Usage: "the opponent key address",
				},
				cli.StringFlag{
					Name:  "amount",
					Usage: "the amount of transfer",
				},
				cli.StringFlag{
					Name:  "uid",
					Usage: "the bot user id",
				},
				cli.StringFlag{
					Name:  "sid",
					Usage: "the bot session id",
				},
				cli.StringFlag{
					Name:  "private",
					Usage: "the bot private key",
				},
				cli.StringFlag{
					Name:  "pin",
					Usage: "the bot PIN",
				},
				cli.StringFlag{
					Name:  "pin_token",
					Usage: "the bot PIN token",
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
	assetId := c.String("asset")
	opponentKey := c.String("opponent_key")
	amount := c.String("amount")
	uid := c.String("uid")
	sid := c.String("sid")
	private := c.String("private")
	pin := c.String("pin")
	pinToken := c.String("pin_token")
	in := &bot.TransferInput{
		AssetId:     assetId,
		OpponentKey: opponentKey,
		Amount:      number.FromString(amount),
	}
	err := bot.CreateTransaction(context.Background(), in, uid, sid, private, pin, pinToken)
	if err != nil {
		return err
	}
	fmt.Println("Mixin transfer success")
	return nil
}
