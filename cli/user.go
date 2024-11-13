package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
)

var getUsersCmdCli = &cli.Command{
	Name:   "users",
	Action: getUsersCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "users",
			Usage: "user ids",
		},
	},
}

func getUsersCmd(ctx *cli.Context) error {
	keystore := ctx.String("keystore")
	idStr := ctx.String("users")

	log.Println(keystore, idStr)
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

	ids := strings.Split(idStr, ",")
	users, err := bot.GetUsers(context.Background(), ids, &su)
	if err != nil {
		panic(err)
	}
	for _, user := range users {
		log.Printf("%#v", user)
	}
	return nil
}
