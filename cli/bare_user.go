package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
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

var createRegisterSafeBareUserCmdCli = &cli.Command{
	Name:   "create_register_safe_bare_user",
	Action: createRegisterSafeBareUserCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "server,sk",
			Usage: "server private key",
		},
		&cli.StringFlag{
			Name:  "spend,s",
			Usage: "spend key",
		},
	},
}

func createRegisterSafeBareUserCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	server := c.String("server")
	spend := c.String("spend")

	su := loadKeystore(keystore) // app user

	// bare user
	if server == "" {
		_, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return err
		}
		server = hex.EncodeToString(privateKey.Seed())
	}
	log.Println("server private key seed: ", server)
	if spend == "" {
		_, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return err
		}
		spend = hex.EncodeToString(privateKey.Seed())
	}
	log.Println("spend private key seed: ", spend)

	name := fmt.Sprintf("%s-%d", su.UserId, time.Now().Unix())

	seedServer, err := hex.DecodeString(server)
	if err != nil {
		panic(err)
	}
	privateKeyServer := ed25519.NewKeyFromSeed(seedServer)
	publicKeyServer := privateKeyServer.Public()

	user, err := bot.CreateUser(context.Background(), base64.RawURLEncoding.EncodeToString(publicKeyServer.(ed25519.PublicKey)), name, su)
	if err != nil {
		return err
	}

	bareUser := &bot.SafeUser{
		UserId:            user.UserId,
		SessionId:         user.SessionId,
		ServerPublicKey:   user.ServerPublicKey,
		SessionPrivateKey: server,
		SpendPrivateKey:   spend,
	}
	data, _ := json.Marshal(bareUser)
	log.Printf("bare user keystore: %s", string(data))

	_, err = bot.RegisterSafeWithSetupPin(context.Background(), bareUser)
	log.Println("register safe bare user success")
	return err
}
