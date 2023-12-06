package main

import (
	"fmt"
	"os"

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
			// ./cli transfer -keystore=/path/to/keystore-7000105129.tip.json -asset=f3bed3e0f6738938c8988eb8853c5647baa263901deb217ee53586d5de831f3b -receiver=8291d1bb-440c-4557-b69f-91dda17876d1 -amount=0.0003
			transferCmdCli,
			batchTransferCmdCli,
			// ./cli transferMulti -keystore=/path/to/keystore-7000105129.tip.json -asset=f3bed3e0f6738938c8988eb8853c5647baa263901deb217ee53586d5de831f3b -receivers=8291d1bb-440c-4557-b69f-91dda17876d1 -receivers=e9e5b807-fa8b-455a-8dfa-b189d28310ff -threshold=1 -amount=0.0013
			transferMultiCmdCli,
			// ./cli outputs -keystore=/path/to/keystore-7000105129.tip.json -asset=f3bed3e0f6738938c8988eb8853c5647baa263901deb217ee53586d5de831f3b -members=8291d1bb-440c-4557-b69f-91dda17876d1 -threshold=1
			listOutputsCmdCli,
			// ./cli migrate --keystore /path/to/keystore-7000103710.json
			botMigrateTIPCmdCli,
			// ./cli register  --keystore /path/to/keystore-7000103710.tip.json --key 8eaa12f3876edc5...67e6cc60
			registerSafeCMDCli,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
