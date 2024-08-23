package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/MixinNetwork/bot-api-go-client/v3"
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
			verifyPINCmdCli,
			// batchTransferCmdCli,
			botMigrateTIPCmdCli,
			registerSafeCMDCli,
			safeSnapshotsCmdCli,
			safeOutputsCmdCli,
			safeOutputCmdCli,
			safeMultisigRequestCmdCli,
			withdrawalCmdCli,
			requestDepositEntryCmdCli,
			buildMixAddressCmdCli,
			hashMembersCmdCli,
			spendKernelUTXOsCmdCli,
			claimMintDistributionCmdCli,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}

func loadKeystore(keystore string) *bot.SafeUser {
	dat, err := os.ReadFile(keystore)
	if err != nil {
		panic(err)
	}
	var u bot.SafeUser
	err = json.Unmarshal([]byte(dat), &u)
	if err != nil {
		panic(err)
	}
	return &u
}
