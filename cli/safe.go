package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/gofrs/uuid/v5"
	"github.com/urfave/cli/v2"
)

var safeGhostKeysCmdCli = &cli.Command{
	Name:   "safe_ghost_keys",
	Action: safeGhostKeysCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "receivers,r",
			Usage: "comma separated list of receiver UUIDs",
		},
		&cli.UintFlag{
			Name:  "index,i",
			Usage: "index value",
			Value: 0,
		},
		&cli.StringFlag{
			Name:  "hint,h",
			Usage: "hint text",
			Value: "",
		},
	},
}

func safeGhostKeysCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	receiversStr := c.String("receivers")
	index := c.Uint("index")
	hint := c.String("hint")

	// Parse receivers from comma-separated string
	receivers := strings.Split(receiversStr, ",")
	if len(receivers) == 1 && receivers[0] == "" {
		panic("invalid receivers")
	}
	if hint == "" {
		hint = uuid.Must(uuid.NewV4()).String()
	}

	su := loadKeystore(keystore)

	// Create ghost key request
	gkr := &bot.GhostKeyRequest{
		Receivers: receivers,
		Index:     index,
		Hint:      hint,
	}

	// Request ghost keys
	ghostKeys, err := bot.RequestSafeGhostKeys(context.Background(), []*bot.GhostKeyRequest{gkr}, su)
	if err != nil {
		panic(err)
	}

	// Output results
	for i, gk := range ghostKeys {
		log.Printf("Ghost Key %d:\n", i+1)
		jsonData, err := json.MarshalIndent(gk, "", "  ")
		if err != nil {
			log.Printf("Error marshaling ghost key: %v", err)
			continue
		}
		log.Printf("%s\n", string(jsonData))
	}

	return nil
}
