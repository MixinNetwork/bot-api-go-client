package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/MixinNetwork/bot-api-go-client/v2"
	"github.com/urfave/cli/v2"
)

type Bot struct {
	Pin        string `json:"pin"`
	ClientID   string `json:"client_id"`
	SessionID  string `json:"session_id"`
	PinToken   string `json:"pin_token"`
	PrivateKey string `json:"private_key"`
}

func main() {
	app := &cli.App{
		Name:    "mixin-bot",
		Usage:   "Mixin bot API cli",
		Version: "2.0.1",
		Commands: []*cli.Command{
			{
				Name:   "transfer",
				Action: transferCmd,
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
			},
			{
				Name:   "migrate",
				Action: botMigrateTIPCmd,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "keystore,k",
						Usage: "keystore download from https://developers.mixin.one/dashboard",
					},
				},
			},
			{
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
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}

func transferCmd(c *cli.Context) error {
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
	var user Bot
	err = json.Unmarshal([]byte(dat), &user)
	if err != nil {
		panic(err)
	}

	su := &bot.SafeUser{
		UserId:     user.ClientID,
		SessionId:  user.SessionID,
		SessionKey: user.PrivateKey,
		SpendKey:   user.Pin[:64],
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
	tx, err := bot.SendTransaction(context.Background(), asset, []*bot.TransactionRecipient{tr}, trace, memo, nil, su)
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
		tx, err := bot.SendTransaction(context.Background(), asset, []*bot.TransactionRecipient{tr}, trace, memo, nil, su)
		if err != nil {
			return err
		}
		log.Println("tx:", tx.TransactionHash)
		log.Println("tx raw:", tx.RawTransaction)
	}
	return nil
}

func botMigrateTIPCmd(c *cli.Context) error {
	keystore := c.String("keystore")

	dat, err := os.ReadFile(keystore)
	if err != nil {
		panic(err)
	}
	var app Bot
	err = json.Unmarshal([]byte(dat), &app)
	if err != nil {
		panic(err)
	}

	tipPub, tipPriv, _ := ed25519.GenerateKey(rand.Reader)
	log.Printf("Your tip private seed: %s", hex.EncodeToString(tipPriv.Seed()))

	err = bot.UpdateTipPin(context.Background(), app.Pin, hex.EncodeToString(tipPub), app.PinToken, app.ClientID, app.SessionID, app.PrivateKey)
	if err != nil {
		return fmt.Errorf("bot.UpdateTipPin() => %v", err)
	}

	app.Pin = hex.EncodeToString(tipPriv)
	keystoreRaw, _ := json.Marshal(app)
	log.Printf("your new keystore after migrate: %s", string(keystoreRaw))
	return nil
}

func registerSafeCMD(c *cli.Context) error {
	keystore := c.String("keystore")
	seed := c.String("key")

	dat, err := os.ReadFile(keystore)
	if err != nil {
		panic(err)
	}
	var app Bot
	err = json.Unmarshal([]byte(dat), &app)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	method := "GET"
	path := "/safe/me"
	token, err := bot.SignAuthenticationTokenWithoutBody(app.ClientID, app.SessionID, app.PrivateKey, method, path)
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
	tipPublic := hex.EncodeToString(privateKey[32:])
	sd := hex.EncodeToString(privateKey.Seed())

	me, err = bot.RegisterSafe(ctx, app.ClientID, tipPublic, sd, app.ClientID, app.SessionID, app.PrivateKey, app.Pin, app.PinToken)
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
