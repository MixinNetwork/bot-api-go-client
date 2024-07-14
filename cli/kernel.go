package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/rpc"
	"github.com/gofrs/uuid/v5"
	"github.com/urfave/cli/v2"
)

const KernelRPC = "https://kernel.mixin.dev"

var spendKernelUTXOsCmdCli = &cli.Command{
	Name:   "spendkernelutxos",
	Action: SpendKernelUTXOs,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "view",
			Usage: "private view key",
		},
		&cli.StringFlag{
			Name:  "spend",
			Usage: "private spend key",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset id",
		},
		&cli.StringFlag{
			Name:  "inputs",
			Usage: "comma sperated inputs of hash:index",
		},
		&cli.StringFlag{
			Name:  "outputs",
			Usage: "comma seperated outputs of receiver:amount",
		},
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
	},
}

func SpendKernelUTXOs(c *cli.Context) error {
	ctx := context.Background()

	dat, err := os.ReadFile(c.String("keystore"))
	if err != nil {
		panic(err)
	}
	var su bot.SafeUser
	err = json.Unmarshal([]byte(dat), &su)
	if err != nil {
		panic(err)
	}

	viewKey, err := crypto.KeyFromString(c.String("view"))
	if err != nil {
		return err
	}
	spendKey, err := crypto.KeyFromString(c.String("spend"))
	if err != nil {
		return err
	}
	account := common.Address{
		PrivateViewKey:  viewKey,
		PrivateSpendKey: spendKey,
		PublicViewKey:   viewKey.Public(),
		PublicSpendKey:  spendKey.Public(),
	}

	asset, err := crypto.HashFromString(c.String("asset"))
	if err != nil {
		return err
	}

	extra, err := hex.DecodeString(c.String("extra"))
	if err != nil {
		return err
	}

	tx := common.NewTransactionV5(asset)
	tx.Extra = extra
	for _, in := range strings.Split(c.String("inputs"), ",") {
		parts := strings.Split(in, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid input %s", in)
		}
		hash, err := crypto.HashFromString(parts[0])
		if err != nil {
			return err
		}
		index, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return err
		}
		tx.AddInput(hash, uint(index))
	}

	var recipients []*bot.TransactionRecipient
	for _, out := range strings.Split(c.String("outputs"), ",") {
		parts := strings.Split(out, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid output %s", out)
		}
		amount := common.NewIntegerFromString(parts[1])
		if amount.Sign() == 0 {
			return fmt.Errorf("invalid output %s", out)
		}
		var mix *bot.MixAddress
		if strings.HasPrefix(parts[0], "XIN") {
			mix = bot.NewMainnetMixAddress([]string{parts[0]}, 1)
		} else if strings.HasPrefix(parts[0], "MIX") {
			sm, err := bot.NewMixAddressFromString(parts[0])
			if err != nil {
				return err
			}
			mix = sm
		} else if len(parts[0]) == 36 {
			uid := uuid.Must(uuid.FromString(parts[0])).String()
			mix = bot.NewUUIDMixAddress([]string{uid}, 1)
		} else {
			panic(parts[0])
		}
		recipients = append(recipients, &bot.TransactionRecipient{
			MixAddress: mix,
			Amount:     amount.String(),
		})
	}
	mks, err := bot.RequestGhostRecipients(ctx, recipients, &su)
	if err != nil || len(mks) != len(recipients) {
		return err
	}
	for i, r := range recipients {
		g := mks[i]
		mask, err := crypto.KeyFromString(g.Mask)
		if err != nil {
			panic(g.Mask)
		}
		tx.Outputs = append(tx.Outputs, &common.Output{
			Type:   common.OutputTypeScript,
			Amount: common.NewIntegerFromString(r.Amount),
			Script: common.NewThresholdScript(r.MixAddress.Threshold),
			Keys:   g.KeysSlice(),
			Mask:   mask,
		})
	}

	signed := tx.AsVersioned()
	ur := &utxoKeysReader{}
	for i := range tx.Inputs {
		err = signed.SignInput(ur, i, []*common.Address{&account})
		if err != nil {
			return err
		}
	}
	raw := hex.EncodeToString(signed.Marshal())
	fmt.Println(raw)
	hash, err := rpc.SendRawTransaction(KernelRPC, raw)
	if err != nil {
		return err
	}
	fmt.Println(hash.String())
	if hash != signed.PayloadHash() {
		panic(signed.PayloadHash().String())
	}
	return nil
}

func ClaimMintDistribution() {
}

type utxoKeysReader struct{}

func (ur *utxoKeysReader) ReadUTXOKeys(hash crypto.Hash, index uint) (*common.UTXOKeys, error) {
	utxo := &common.UTXOKeys{}
	out, err := rpc.GetUTXO(KernelRPC, hash.String(), uint64(index))
	if err != nil || out == nil {
		return nil, err
	}
	utxo.Keys = out.Keys
	utxo.Mask = out.Mask
	return utxo, nil
}

func (ur *utxoKeysReader) ReadDepositLock(deposit *common.DepositData) (crypto.Hash, error) {
	return crypto.Hash{}, nil
}
