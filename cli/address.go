package main

import (
	"fmt"
	"strings"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
)

var buildMixAddressCmdCli = &cli.Command{
	Name:   "buildmixaddress",
	Action: buildMixAddress,
	Flags: []cli.Flag{
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

func buildMixAddress(c *cli.Context) error {
	members := strings.Split(c.String("members"), ",")
	mix := bot.NewUUIDMixAddress(members, byte(c.Int("threshold")))
	fmt.Println(mix.String())
	return nil
}
