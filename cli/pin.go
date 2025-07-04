package main

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"log"
	"time"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
)

// ./cli verify_pin -keystore=/path/to/keystore-700xxxx006.json -spend=31088c8....40dc0
var verifyPINCmdCli = &cli.Command{
	Name:   "verify_pin",
	Action: verifyPINCmd,
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

func verifyPINCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	spend := c.String("spend")

	su := loadKeystore(keystore)
	su.SpendPrivateKey = spend

	user, err := bot.VerifyPINTip(context.Background(), su)
	if err != nil {
		panic(err)
	}
	log.Printf("user %#v", user)
	return nil
}

// ./cli update_pin -keystore=/path/to/keystore-700xxxx006.json -spend=31088c8....40dc0
var updatePINCmdCli = &cli.Command{
	Name:   "update_pin",
	Action: updatePINCmd,
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

func updatePINCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	spend := c.String("spend")

	su := loadKeystore(keystore)
	su.SpendPrivateKey = spend

	seed, err := hex.DecodeString(su.SpendPrivateKey)
	if err != nil {
		return err
	}
	private := ed25519.NewKeyFromSeed(seed)
	spendPublicKey := private.Public().(ed25519.PublicKey)

	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, 1)
	pubTipBuf := append(spendPublicKey, counter...)
	encryptedPin, err := bot.EncryptEd25519PIN(hex.EncodeToString(pubTipBuf), uint64(time.Now().UnixNano()), su)
	if err != nil {
		return err
	}

	err = bot.UpdatePin(context.Background(), "", encryptedPin, su)
	if err != nil {
		return err
	}
	log.Println("update pin success")
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
