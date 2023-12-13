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
			transferCmdCli,
			batchTransferCmdCli,
			botMigrateTIPCmdCli,
			registerSafeCMDCli,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
