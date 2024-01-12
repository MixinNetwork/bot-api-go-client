package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
)

var botMigrateTIPCmdCli = &cli.Command{
	Name:   "migrate",
	Action: botMigrateTIPCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
	},
}

func botMigrateTIPCmd(c *cli.Context) error {
	keystore := c.String("keystore")

	dat, err := os.ReadFile(keystore)
	if err != nil {
		panic(err)
	}
	var u SafeUser
	err = json.Unmarshal([]byte(dat), &u)
	if err != nil {
		panic(err)
	}

	tipPub, tipPriv, _ := ed25519.GenerateKey(rand.Reader)
	log.Printf("Your tip private seed: %s", hex.EncodeToString(tipPriv.Seed()))

	su := &bot.SafeUser{
		UserId:            u.AppID,
		SessionId:         u.SessionID,
		ServerPublicKey:   u.ServerPublicKey,
		SessionPrivateKey: u.SessionPrivateKey,
	}

	err = bot.UpdateTipPin(context.Background(), "", hex.EncodeToString(tipPub), su)
	if err != nil {
		return fmt.Errorf("bot.UpdateTipPin() => %v", err)
	}
	return nil
}

var registerSafeCMDCli = &cli.Command{
	Name:   "register",
	Action: registerSafeCMD,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "key,s",
			Usage: "seed for spend private key",
		},
	},
}

func registerSafeCMD(c *cli.Context) error {
	keystore := c.String("keystore")
	seed := c.String("key")

	dat, err := os.ReadFile(keystore)
	if err != nil {
		panic(err)
	}
	var u SafeUser
	err = json.Unmarshal([]byte(dat), &u)
	if err != nil {
		panic(err)
	}

	su := &bot.SafeUser{
		UserId:            u.AppID,
		SessionId:         u.SessionID,
		ServerPublicKey:   u.ServerPublicKey,
		SessionPrivateKey: u.SessionPrivateKey,
	}
	ctx := context.Background()
	method := "GET"
	path := "/safe/me"
	token, err := bot.SignAuthenticationTokenWithoutBody(method, path, su)
	if err != nil {
		return err
	}

	me, err := bot.UserMe(ctx, token)
	if err != nil {
		return err
	}
	if me.HasSafe {
		log.Println("user has registed")
		return nil
	}
	s, err := hex.DecodeString(seed)
	if err != nil {
		panic(err)
	}
	if len(s) != ed25519.SeedSize {
		panic("invalid seed")
	}
	privateKey := ed25519.NewKeyFromSeed(s)
	sd := hex.EncodeToString(privateKey.Seed())

	me, err = bot.RegisterSafe(ctx, su.UserId, sd, su)
	if err != nil {
		return err
	}
	if me.HasSafe {
		log.Println("user registed")
		return nil
	}

	log.Println("user not registed")
	return nil
}
