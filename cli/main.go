package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "mixin-bot",
		Usage:   "Mixin bot API cli",
		Version: "2.0.1",
		Commands: []*cli.Command{
			appMeCmdCli,
			userCmdCli,
			transferCmdCli,
			// batchTransferCmdCli,
			// botMigrateTIPCmdCli,
			// registerSafeCMDCli,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
