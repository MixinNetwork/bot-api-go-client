package main

import (
	"context"
	"log"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
)

var safeMultisigRequestCmdCli = &cli.Command{
	Name:   "safe_request",
	Action: safeMultisigRequestCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "id",
			Usage: "id",
		},
	},
}

func safeMultisigRequestCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	id := c.String("id")

	su := loadKeystore(keystore)

	r, err := bot.FetchSafeMultisigRequest(context.Background(), id, su)
	if err != nil {
		panic(err)
	}
	log.Printf("request %#v", r)
	return nil
}
