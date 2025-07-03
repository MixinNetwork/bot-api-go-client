package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
)

/*
{
 "pin": "123321",
 "session_id": "679eaab4-fe31-4ccd-b78b-2a82705b36eb",
 "pin_token": "Sngu+kas/vefj6O2NUEBlCGqZC2E+....+HAeeVOVPK/Wyfz9qVy9wN3k9QBqWzw14SZLddEAJ7pU8E=",
 "private_key": "-----BEGIN RSA PRIVATE KEY-----\r\nMIICXAIBAAKBgQCF1CgF5DOvui/J2t4SEWXB69RfrfHm/uMDfyyTCC2et4DVK+Fk\r\n....YXZEt/MKaxXPUf48RihqbbKVxv11vVW5O2gj+Iu0rXo=\r\n-----END RSA PRIVATE KEY-----\r\n"
}
*/

// ./cli upgrade_legacy_user -keystore=/path/to/keystore.json
var upgradeLegacyUserCmdCli = &cli.Command{
	Name:   "upgrade_legacy_user",
	Action: upgradeLegacyUserCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore format is like above comments",
		},
	},
}

func upgradeLegacyUserCmd(c *cli.Context) error {
	kl := loadKeystoreLegacy(c.String("keystore"))

	user, err := bot.UpgradeLegacyUser(context.Background(), kl)
	if err != nil {
		panic(err)
	}
	data, err := json.Marshal(user)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(data)
	return nil
}

func loadKeystoreLegacy(keystore string) *bot.KeystoreLegacy {
	dat, err := os.ReadFile(keystore)
	if err != nil {
		panic(err)
	}
	var kl bot.KeystoreLegacy
	err = json.Unmarshal([]byte(dat), &kl)
	if err != nil {
		panic(err)
	}
	return &kl
}
