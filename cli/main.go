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
		Version: "3.0.1",
		Commands: []*cli.Command{
			appMeCmdCli,
			userCmdCli,
			searchUserCmdCli,
			getUsersCmdCli,
			transferCmdCli,
			verifyPINCmdCli,
			updatePINCmdCli,
			registerSafeBareUserCmdCli,
			// batchTransferCmdCli,
			botMigrateTIPCmdCli,
			registerSafeCMDCli,
			safeSnapshotsCmdCli,
			safeSnapshotCmdCli,
			safeOutputsCmdCli,
			safeOutputCmdCli,
			safeMultisigRequestCmdCli,
			safeGhostKeysCmdCli,
			withdrawalCmdCli,
			requestDepositEntryCmdCli,
			buildMixAddressCmdCli,
			hashMembersCmdCli,
			spendKernelUTXOsCmdCli,
			claimMintDistributionCmdCli,
			assetBalanceCmdCli,
			assetsBalanceCmdCli,
			notifySnapshotCmdCli,
			bareUserCmdCli,
			createRegisterSafeBareUserCmdCli,
			upgradeLegacyUserCmdCli,
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

type BareUser struct {
	UserId            string `json:"user_id"`
	SessionId         string `json:"session_id"`
	SessionPrivateKey string `json:"session_private_key"`
	ServerPublicKey   string `json:"server_public_key"`
}

func loadKeystoreBareUser(keystore string) *bot.SafeUser {
	dat, err := os.ReadFile(keystore)
	if err != nil {
		panic(err)
	}
	var bu BareUser
	err = json.Unmarshal([]byte(dat), &bu)
	if err != nil {
		panic(err)
	}
	u := bot.SafeUser{
		UserId:            bu.UserId,
		SessionId:         bu.SessionId,
		SessionPrivateKey: bu.SessionPrivateKey,
		ServerPublicKey:   bu.ServerPublicKey,
	}
	return &u
}
