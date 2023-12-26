package main

import (
	"context"
	"log"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
)

// ./cli verify_pin -keystore=/path/to/keystore-700xxxx006.json -spend=31088c8....40dc0
var verifyPINCmdCli = &cli.Command{
	Name:   "verify_pin",
	Action: verifyPINCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "spend,s",
			Usage: "spend",
		},
	},
}

func verifyPINCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	spend := c.String("spend")

	su := loadKeystore(keystore)
	su.SpendPrivateKey = spend

	user, err := bot.VerifyPINTip(context.Background(), su)
	if err != nil {
		panic(err)
	}
	log.Println("user %#v", user)
	return nil
}
