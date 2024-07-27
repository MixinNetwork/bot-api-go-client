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
			Name:  "extra",
			Usage: "hex encoded extra data",
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

var claimMintDistributionCmdCli = &cli.Command{
	Name:   "claimmintdistribution",
	Action: ClaimMintDistribution,
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
			Name:  "outputs",
			Usage: "comma seperated outputs of receiver:amount",
		},
		&cli.StringFlag{
			Name:  "keystore,k",
			Usage: "keystore download from https://developers.mixin.one/dashboard",
		},
		&cli.Uint64Flag{
			Name:  "batch",
			Value: 1707,
			Usage: "the mint batch since 1707",
		},
	},
}

func SpendKernelUTXOs(c *cli.Context) error {
	ctx := context.Background()

	traceId := uuid.Must(uuid.NewV4()).String()

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
	account := &common.Address{
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
		recipients = append(recipients, &bot.TransactionRecipient{
			MixAddress: parseMixinAddress(parts[0]),
			Amount:     amount.String(),
		})
	}
	mks, err := bot.RequestGhostRecipientsWithTraceId(ctx, recipients, traceId, &su)
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

	return signAndSendRawTransaction(tx, account)
}

func ClaimMintDistribution(c *cli.Context) error {
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
	account := &common.Address{
		PrivateViewKey:  viewKey,
		PrivateSpendKey: spendKey,
		PublicViewKey:   viewKey.Public(),
		PublicSpendKey:  spendKey.Public(),
	}

	traceId := bot.UniqueObjectId(viewKey.String(), spendKey.String())
	traceId = bot.UniqueObjectId(traceId, "MINTDIST", fmt.Sprint(c.Uint64("batch")))

	mints, err := rpc.ListMintDistributions(KernelRPC, c.Uint64("batch"), 1)
	if err != nil {
		return err
	}
	if b := mints[0].Inputs[0].Mint.Batch; b != c.Uint64("batch") {
		fmt.Printf("MINT %s %d %d", mints[0].PayloadHash(), b, c.Uint64("batch"))
		return nil
	}

	safe, err := rpc.GetUTXO(KernelRPC, mints[0].PayloadHash().String(), uint64(len(mints[0].Outputs)-2))
	if err != nil {
		return err
	}
	if !safe.LockHash.HasValue() {
		return fmt.Errorf("no safe distribution yet %s", mints[0].PayloadHash())
	}
	stx, _, err := rpc.GetTransaction(KernelRPC, safe.LockHash.String())
	if err != nil {
		return err
	}

	var total common.Integer
	tx := common.NewTransactionV5(common.XINAssetId)
	tx.Extra = []byte(fmt.Sprintf("MINT %d", c.Uint64("batch")))
	for i, out := range mints[0].Outputs {
		if !checkMyOutput(out, uint64(i), account) {
			continue
		}
		tx.AddInput(mints[0].PayloadHash(), uint(i))
		total = total.Add(out.Amount)
	}
	for i, out := range stx.Outputs {
		if !checkMyOutput(out, uint64(i), account) {
			continue
		}
		tx.AddInput(stx.PayloadHash(), uint(i))
		total = total.Add(out.Amount)
	}

	var outputTotal common.Integer
	var recipients []*bot.TransactionRecipient
	for _, out := range strings.Split(c.String("outputs"), ",") {
		parts := strings.Split(out, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid output %s", out)
		}
		share, _ := strconv.ParseInt(parts[1], 10, 64)
		if share <= 0 || share > 10000 {
			return fmt.Errorf("invalid output %s", out)
		}
		amount := total.Mul(int(share)).Div(10000)
		outputTotal = outputTotal.Add(amount)
		recipients = append(recipients, &bot.TransactionRecipient{
			MixAddress: parseMixinAddress(parts[0]),
			Amount:     amount.String(),
		})
	}
	change := total.Sub(outputTotal)
	threshold := common.NewIntegerFromString("0.00000005")
	if change.Sign() > 0 && change.Cmp(threshold) < 0 {
		r := recipients[len(recipients)-1]
		r.Amount = common.NewIntegerFromString(r.Amount).Add(change).String()
	}

	mks, err := bot.RequestGhostRecipientsWithTraceId(ctx, recipients, traceId, &su)
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

	return signAndSendRawTransaction(tx, account)
}

func parseMixinAddress(str string) *bot.MixAddress {
	if strings.HasPrefix(str, "XIN") {
		return bot.NewMainnetMixAddress([]string{str}, 1)
	} else if strings.HasPrefix(str, "MIX") {
		mix, err := bot.NewMixAddressFromString(str)
		if err != nil {
			panic(err)
		}
		return mix
	} else if len(str) == 36 {
		uid := uuid.Must(uuid.FromString(str)).String()
		return bot.NewUUIDMixAddress([]string{uid}, 1)
	} else {
		panic(str)
	}
}

func signAndSendRawTransaction(tx *common.Transaction, account *common.Address) error {
	signed := tx.AsVersioned()
	ur := rpc.NewUTXOKeysRPCReader(KernelRPC)
	for i := range tx.Inputs {
		err := signed.SignInput(ur, i, []*common.Address{account})
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

func checkMyOutput(out *common.Output, index uint64, a *common.Address) bool {
	p := crypto.DeriveGhostPrivateKey(&out.Mask, &a.PrivateViewKey, &a.PrivateSpendKey, index)
	return p.Public() == *out.Keys[0]
}
