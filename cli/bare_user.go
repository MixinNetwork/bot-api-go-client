package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
)

var bareUserCmdCli = &cli.Command{
	Name:   "create_bare_user",
	Action: createBareUserCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "session_private_key,sk",
			Usage: "session private key is an ed25519 private key seed",
		},
		&cli.StringFlag{
			Name:  "name,n",
			Usage: "bare user full name",
		},
	},
}

func createBareUserCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	str := c.String("session_private_key")
	name := c.String("name")

	seed, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}

	su := loadKeystore(keystore)
	private := ed25519.NewKeyFromSeed(seed)
	public := private.Public()

	if name == "" {
		name = fmt.Sprintf("%s-%d", su.UserId, time.Now().Unix())
	}

	user, err := bot.CreateUser(context.Background(), base64.RawURLEncoding.EncodeToString(public.(ed25519.PublicKey)), name, su)
	if err != nil {
		return err
	}

	ks := &bot.SafeUser{
		UserId:            user.UserId,
		SessionId:         user.SessionId,
		ServerPublicKey:   user.ServerPublicKey,
		SessionPrivateKey: str,
	}
	data, _ := json.Marshal(ks)
	log.Printf("bare user keystore: %s", string(data))
	return nil
}

// ./cli register_safe_bare_user -keystore=/path/to/keystore-700xxxx006.json -spend=31088c8....40dc0
var registerSafeBareUserCmdCli = &cli.Command{
	Name:   "register_safe_bare_user",
	Action: registerSafeBareUserCmd,
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

func registerSafeBareUserCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	spend := c.String("spend")

	su := loadKeystore(keystore)
	su.SpendPrivateKey = spend

	_, err := bot.RegisterSafeBareUser(context.Background(), su)
	if err != nil {
		return err
	}
	log.Println("register safe bare user success")
	return nil
}
