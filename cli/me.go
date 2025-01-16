package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
)

var appMeCmdCli = &cli.Command{
	Name:   "me",
	Action: appMeCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
	},
}

func appMeCmd(c *cli.Context) error {
	keystore := c.String("keystore")

	dat, err := os.ReadFile(keystore)
	if err != nil {
		panic(err)
	}
	log.Println(string(dat))
	var su bot.SafeUser
	err = json.Unmarshal([]byte(dat), &su)
	if err != nil {
		panic(err)
	}
	me, err := bot.RequestUserMe(context.Background(), &su)
	if err != nil {
		panic(err)
	}
	log.Printf("%#v", me)
	return nil
}
