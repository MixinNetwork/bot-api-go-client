package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
)

var requestDepositEntryCmdCli = &cli.Command{
	Name:   "requestdepositentry",
	Action: requestDepositEntry,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "chain",
			Usage: "the chain id",
		},
		&cli.StringFlag{
			Name:  "members",
			Usage: "comma separated UUIDs",
		},
		&cli.Int64Flag{
			Name:  "threshold",
			Usage: "the members threshold",
		},
	},
}

func requestDepositEntry(c *cli.Context) error {
	ctx := context.Background()

	dat, err := os.ReadFile(c.String("keystore"))
	if err != nil {
		panic(err)
	}
	var su bot.SafeUser
	err = json.Unmarshal([]byte(dat), &su)
	if err != nil {
		panic(err)
	}

	members := strings.Split(c.String("members"), ",")
	entries, err := bot.CreateDepositEntry(ctx, c.String("chain"), members, c.Int64("threshold"), &su)
	if err != nil {
		return err
	}
	fmt.Println(entries[0])
	return nil
}
