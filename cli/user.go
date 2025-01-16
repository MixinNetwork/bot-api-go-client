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

var userCmdCli = &cli.Command{
	Name:   "user",
	Action: userCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "id",
			Usage: "user id",
		},
	},
}

func userCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	id := c.String("id")

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

	user, err := bot.GetUser(context.Background(), id, &su)
	if err != nil {
		panic(err)
	}
	log.Printf("%#v", user)
	return nil
}

var searchUserCmdCli = &cli.Command{
	Name:   "search",
	Action: searchUserCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "query",
			Usage: "query param",
		},
	},
}

func searchUserCmd(ctx *cli.Context) error {
	keystore := ctx.String("keystore")
	q := ctx.String("query")

	dat, err := os.ReadFile(keystore)
	if err != nil {
		panic(err)
	}
	var su bot.SafeUser
	err = json.Unmarshal([]byte(dat), &su)
	if err != nil {
		panic(err)
	}

	user, err := bot.SearchUser(context.Background(), q, &su)
	if err != nil {
		panic(err)
	}
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		panic(err)
	}
	logger := log.New(os.Stdout, "", 0)
	logger.Println(string(data))
	return nil
}
