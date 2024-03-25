package main

import (
	"context"
	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
	"log"
)

// ./cli withdrawal -keystore=/path/to/keystore-700xxxx006.json -spend=31088c8....40dc0 -asset=43d61dcd-e413-450d-80b8-101d5e903357 -amount=0.01 -destination=0x....
var withdrawalCmdCli = &cli.Command{
	Name:   "withdrawal",
	Action: withdrawalCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "asset,a",
			Usage: "asset",
		},
		&cli.StringFlag{
			Name:  "amount,z",
			Usage: "amount",
		},
		&cli.StringFlag{
			Name:  "destination,d",
			Usage: "destination",
		},
		&cli.StringFlag{
			Name:  "tag",
			Usage: "tag",
		},
		&cli.StringFlag{
			Name:  "trace,t",
			Usage: "trace",
		},
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "spend,s",
			Usage: "spend",
		},
		&cli.BoolFlag{
			Name:  "prefer-asset-fee",
			Usage: "prefer-asset-fee",
		},
	},
}

func withdrawalCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	spend := c.String("spend")
	asset := c.String("asset")
	amount := c.String("amount")
	destination := c.String("destination")
	tag := c.String("tag")
	trace := c.String("trace")
	preferAssetFee := c.Bool("prefer-asset-fee")

	su := loadKeystore(keystore)
	su.SpendPrivateKey = spend

	traceId := bot.UuidNewV4().String()
	if trace != "" {
		traceId = trace
	}

	log.Printf("withdrawal %s %s %s %s %s", asset, amount, destination, tag, traceId)
	log.Printf("prefer-asset-fee %v", preferAssetFee)

	_, err := bot.SendWithdrawal(context.Background(), asset, destination, tag, amount, traceId, preferAssetFee, "", su)
	if err != nil {
		panic(err)
	}
	return nil
}
