package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/urfave/cli/v2"
)

var transferCmdCli = &cli.Command{
	Name:   "transfer",
	Action: transferCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "spend,s",
			Usage: "spend",
		},
		&cli.StringFlag{
			Name:  "asset,a",
			Usage: "asset",
		},
		&cli.StringFlag{
			Name:  "amount,z",
			Usage: "amount",
		},
		&cli.StringFlag{
			Name:  "receiver,r",
			Usage: "receiver",
		},
		&cli.StringFlag{
			Name:  "trace,t",
			Usage: "trace",
		},
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
	},
}

func transferCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	spend := c.String("spend")
	asset := c.String("asset")
	amount := c.String("amount")
	receiver := c.String("receiver")
	trace := c.String("trace")

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
		SpendPrivateKey:   spend,
	}

	ma := bot.NewUUIDMixAddress([]string{receiver}, 1)
	tr := &bot.TransactionRecipient{MixAddress: ma.String(), Amount: amount}

	memo := c.String("trace")
	if trace == "" {
		trace = bot.UuidNewV4().String()
	}
	traceID, _ := bot.UuidFromString(trace)
	if traceID.String() != trace {
		trace = bot.UniqueObjectId(trace)
	}
	log.Println("asset:", asset)
	log.Println("amount:", amount)
	log.Println("receiver:", receiver)
	log.Println("origin trace is memo:", memo)
	log.Println("trace:", trace)
	tx, err := bot.SendTransaction(context.Background(), asset, []*bot.TransactionRecipient{tr}, trace, []byte(memo), nil, su)
	if err != nil {
		return err
	}
	log.Println("tx:", tx.TransactionHash)
	log.Println("tx raw:", tx.RawTransaction)
	return nil
}

var batchTransferCmdCli = &cli.Command{
	Name:   "batchTransfer",
	Action: batchTransferCmd,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "asset,a",
			Usage: "asset",
		},
		&cli.StringFlag{
			Name:  "amount,z",
			Usage: "amount",
		},
		&cli.StringFlag{
			Name:  "receiver,r",
			Usage: "receiver",
		},
		&cli.StringFlag{
			Name:  "trace,t",
			Usage: "trace",
		},
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.StringFlag{
			Name:  "input,cvs",
			Usage: "read input from csv file and transfer all rows",
		},
	},
}

func batchTransferCmd(c *cli.Context) error {
	keystore := c.String("keystore")
	inputPath := c.String("input")
	asset := c.String("asset")
	amount := c.String("amount")
	receiver := c.String("receiver")
	trace := c.String("trace")

	dat, err := os.ReadFile(keystore)
	if err != nil {
		panic(err)
	}
	var user SafeUser
	err = json.Unmarshal([]byte(dat), &user)
	if err != nil {
		panic(err)
	}

	su := &bot.SafeUser{
		UserId:            user.AppID,
		SessionId:         user.SessionID,
		SessionPrivateKey: user.SessionPrivateKey,
	}
	if inputPath != "" {
		return transferCSV(c, inputPath, asset, su)
	}

	ma := bot.NewUUIDMixAddress([]string{receiver}, 1)
	tr := &bot.TransactionRecipient{MixAddress: ma.String(), Amount: amount}

	memo := c.String("trace")
	traceID, _ := bot.UuidFromString(trace)
	if traceID.String() != trace {
		trace = bot.UniqueObjectId(trace)
	}
	log.Println("asset:", asset)
	log.Println("amount:", amount)
	log.Println("receiver:", receiver)
	log.Println("origin trace is memo:", memo)
	log.Println("trace:", trace)
	fmt.Print("Confirm input Y, otherwise input X: ")
	var input string
	fmt.Scanln(&input)
	if strings.ToUpper(input) != "Y" {
		return nil
	}
	tx, err := bot.SendTransaction(context.Background(), asset, []*bot.TransactionRecipient{tr}, trace, []byte(memo), nil, su)
	if err != nil {
		return err
	}
	log.Println("tx:", tx.TransactionHash)
	log.Println("tx raw:", tx.RawTransaction)
	return nil
}

func transferCSV(c *cli.Context, filePath string, asset string, su *bot.SafeUser) error {
	data, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer data.Close()
	r := csv.NewReader(data)
	records, err := r.ReadAll()
	if err != nil {
		panic(err)
	}
	log.Println("records:", len(records))

	for _, record := range records {
		if record[5] != asset {
			continue
		}
		// user_id, asset_id, amount, hash
		// fmt.Println("record:", record[2], record[5], record[6], record[7])
		receiver := record[2]
		asset := record[5]
		amount := record[6]
		trace := record[7]
		memo := record[7]
		ma := bot.NewUUIDMixAddress([]string{receiver}, 1)
		tr := &bot.TransactionRecipient{MixAddress: ma.String(), Amount: amount}

		traceID, _ := bot.UuidFromString(trace)
		if traceID.String() != trace {
			trace = bot.UniqueObjectId(trace)
		}
		transaction, err := bot.GetTransactionById(c.Context, trace)
		if err != nil {
			if !strings.Contains(err.Error(), "The endpoint is not found") {
				log.Print(err)
				return err
			}
		}
		if transaction != nil {
			log.Println("exist snapshot_id: ", transaction.SnapshotID)
			continue
		}

		log.Println("asset:", asset)
		log.Println("amount:", amount)
		log.Println("receiver:", receiver)
		log.Println("origin trace is memo:", memo)
		log.Println("trace:", trace)
		fmt.Print("Confirm input Y, otherwise input X: ")
		var input string
		fmt.Scanln(&input)
		if strings.ToUpper(input) != "Y" {
			continue
		}
		tx, err := bot.SendTransaction(context.Background(), asset, []*bot.TransactionRecipient{tr}, trace, []byte(memo), nil, su)
		if err != nil {
			return err
		}
		log.Println("tx:", tx.TransactionHash)
		log.Println("tx raw:", tx.RawTransaction)
	}
	return nil
}
