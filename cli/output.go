package main

import (
	"context"
	"log"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
)

var safeOutputsCmdCli = &cli.Command{
	Name:   "safe_outputs",
	Action: safeOutputsCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "spend,s",
			Usage: "spend",
		},
		&cli.StringFlag{
			Name:  "asset,a",
			Usage: "asset",
		},
	},
}

func safeOutputsCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	spend := c.String("spend")
	asset := c.String("asset")

	su := loadKeystore(keystore)
	su.SpendPrivateKey = spend

	hash := bot.HashMembers([]string{su.UserId})
	outputs, err := bot.ListUnspentOutputs(context.Background(), hash, 1, asset, su)
	if err != nil {
		panic(err)
	}
	for _, o := range outputs {
		o.Extra = ""
		log.Printf("output %#v", o)
	}
	return nil
}

var safeOutputCmdCli = &cli.Command{
	Name:   "safe_output",
	Action: safeOutputCmd,
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

func safeOutputCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	id := c.String("id")

	su := loadKeystore(keystore)

	output, err := bot.GetOutput(context.Background(), id, su)
	if err != nil {
		panic(err)
	}
	output.Extra = ""
	log.Printf("output %#v", output)
	return nil
}

var assetBalanceCmdCli = &cli.Command{
	Name:   "asset_balance",
	Action: assetBalanceCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "spend,s",
			Usage: "spend",
		},
		&cli.StringFlag{
			Name:  "asset,a",
			Usage: "asset",
		},
	},
}

func assetBalanceCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	spend := c.String("spend")
	asset := c.String("asset")

	su := loadKeystore(keystore)
	su.SpendPrivateKey = spend

	assets, err := bot.ListAssetWithBalance(context.Background(), su)
	if err != nil {
		panic(err)
	}
	for _, a := range assets {
		if a.KernelAssetId == asset || a.AssetID == asset {
			log.Printf("%s balance %s", a.Symbol, a.Amount)
		}
	}
	return nil
}

func assetsBalanceCmd(c *cli.Context) error {
	keystore := c.String("keystore")

	su := loadKeystore(keystore)

	assets, err := bot.ListAssetWithBalance(context.Background(), su)
	if err != nil {
		panic(err)
	}
	for _, a := range assets {
		log.Printf("%s(%s) balance %s", a.Symbol, a.AssetID, a.Amount)
	}
	return nil
}

var assetsBalanceCmdCli = &cli.Command{
	Name:   "assets_balance",
	Action: assetsBalanceCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
	},
}