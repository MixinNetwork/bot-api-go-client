package bot

import (
	"context"
	"encoding/hex"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
)

// this function send a raw transaction to mixin api users from a kernel account
// this account should have private view key and private spend key
func SendKernelTransactionFromAccount(ctx context.Context, asset crypto.Hash, receivers []string, threshold byte, amount common.Integer, inputs []*common.UTXO, account *common.Address, traceId, extra string, safeUser *SafeUser) string {
	var total common.Integer
	tx := common.NewTransactionV5(asset)
	for _, in := range inputs {
		tx.AddInput(in.Hash, in.Index)
		total = total.Add(in.Amount)
	}
	if total.Cmp(amount) < 0 {
		panic("total")
	}

	// 1. POST /keys with receivers and threshold, and traceId is the hint
	r := &GhostKeyRequest{
		Receivers: receivers,
		Index:     0,
		Hint:      traceId,
	}
	ghostKeys, err := RequestSafeGhostKeys(ctx, []*GhostKeyRequest{r}, safeUser)
	if err != nil {
		panic(err)
	}
	if len(ghostKeys) != 1 {
		panic("len(ghostKeys)")
	}
	// TODO only support one now.
	keys, _ := crypto.KeyFromString(ghostKeys[0].Keys[0])
	mask, _ := crypto.KeyFromString(ghostKeys[0].Mask)

	// 2. use the inputs and ghost keys to generate a raw transaction
	output := &common.Output{
		Type:   common.OutputTypeScript,
		Amount: amount,
		Keys:   []*crypto.Key{&keys},
		Mask:   mask,
		Script: common.NewThresholdScript(threshold),
	}
	tx.Outputs = append(tx.Outputs, output)

	if total.Cmp(amount) > 0 {
		change := total.Sub(amount)
		script := common.NewThresholdScript(1)
		hash := tx.AsVersioned().PayloadHash()
		seed := append(hash[:], hash[:]...)
		tx.AddScriptOutput([]*common.Address{account}, script, change, seed)
	}
	if extra != "" {
		tx.Extra = []byte(extra)
	}

	// 3. sign all the inputs of the transaction with the account
	ver := tx.AsVersioned()
	for i := range ver.Inputs {
		err := ver.SignUTXO(inputs[i], []*common.Address{account})
		if err != nil {
			panic(err)
		}
	}
	return hex.EncodeToString(ver.Marshal())
}
