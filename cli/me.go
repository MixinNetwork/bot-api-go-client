package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/MixinNetwork/bot-api-go-client/v2"
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
	var u SafeUser
	err = json.Unmarshal([]byte(dat), &u)
	if err != nil {
		panic(err)
	}

	su := &bot.SafeUser{
		UserId:            u.AppID,
		SessionId:         u.SessionID,
		SessionPrivateKey: u.SessionPrivateKey,
	}
	me, err := bot.RequestUserMe(context.Background(), su)
	if err != nil {
		panic(err)
	}
	log.Printf("%#v", me)
	return nil
}
