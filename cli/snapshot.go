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
	},
}

func safeSnapshotsCmd(c *cli.Context) error {
	keystore := c.String("keystore")

	su := loadKeystore(keystore)

	snapshots, err := bot.SafeSnapshots(context.Background(), 100, "", "", "", "", su)
	if err != nil {
		panic(err)
	}
	for _, s := range snapshots {
		log.Printf("snapshot %#v", s)
	}
	return nil
}

var safeSnapshotCmdCli = &cli.Command{
	Name:   "safe_snapshot",
	Action: safeSnapshotCmd,
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

func safeSnapshotCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	id := c.String("id")

	su := loadKeystore(keystore)

	snapshot, err := bot.SafeSnapshotById(context.Background(), id, su)
	if err != nil {
		panic(err)
	}
	log.Printf("snapshot %#v", snapshot)
	return nil
}
