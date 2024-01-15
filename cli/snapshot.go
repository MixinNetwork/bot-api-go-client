package main

import (
	"context"
	"log"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
)

var safeSnapshotsCmdCli = &cli.Command{
	Name:   "safe_snapshots",
	Action: safeSnapshotsCmd,
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

func safeSnapshotsCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	spend := c.String("spend")

	su := loadKeystore(keystore)
	su.SpendPrivateKey = spend

	snapshots, err := bot.SafeSnapshots(context.Background(), 100, "", "", "", "", su)
	if err != nil {
		panic(err)
	}
	for _, s := range snapshots {
		log.Printf("snapshot %#v", s)
	}
	return nil
}
