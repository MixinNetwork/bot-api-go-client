package bot

import (
	"context"
	"encoding/hex"
	"fmt"

	"filippo.io/edwards25519"
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/gofrs/uuid/v5"
)

type TransactionRecipient struct {
	MixAddress string
	Amount     string

	Destination string
	Tag         string
}

type SequencerTransactionRequest struct {
	RawTransaction string
	State          string
	Views          []string
}

func SendTransaction(ctx context.Context, assetId string, recipients []*TransactionRecipient, traceId string, u *SafeUser) (*common.VersionedTransaction, error) {
	if uuid.FromStringOrNil(assetId).String() == assetId {
		assetId = crypto.Sha256Hash([]byte(assetId)).String()
	}
	asset, err := crypto.HashFromString(assetId)
	if err != nil {
		return nil, fmt.Errorf("invalid asset id %s", assetId)
	}

	// get unspent outputs for asset and may return insufficient outputs error
	utxos, changeAmount, err := requestUnspentOutputsForRecipients(ctx, assetId, recipients, u)
	if err != nil {
		return nil, err
	}
	// change to the sender
	if changeAmount.Sign() > 0 {
		ma := NewUUIDMixAddress([]string{u.UserId}, 1)
		recipients = append(recipients, &TransactionRecipient{
			MixAddress: ma.String(),
			Amount:     changeAmount.String(),
		})
	}

	// build the unsigned raw transaction
	tx, err := buildRawTransaction(ctx, asset, utxos, recipients, u)
	if err != nil {
		return nil, err
	}
	ver := tx.AsVersioned()
	// verify the raw transaction, the same trace id may have been signed already
	str, err := verifyRawTransactionBySequencer(ctx, traceId, ver, u)
	if err != nil || str.State != "unspent" {
		return ver, err
	}

	// sign the raw transaction with user private spend key
	if len(str.Views) != len(ver.Inputs) {
		return nil, fmt.Errorf("invalid view keys count %d %d", len(str.Views), len(ver.Inputs))
	}
	ver, err = signRawTransaction(ctx, ver, str.Views, u.SpendKey)
	if err != nil {
		return nil, err
	}

	// send the raw transaction to the sequencer api
	str, err = sendRawTransactionToSequencer(ctx, traceId, ver, u)
	if err != nil {
		return nil, err
	}
	if hex.EncodeToString(ver.Marshal()) != str.RawTransaction {
		panic(str.RawTransaction)
	}
	return ver, nil
}

func buildRawTransaction(ctx context.Context, asset crypto.Hash, utxos []*Output, recipients []*TransactionRecipient, u *SafeUser) (*common.Transaction, error) {
	tx := common.NewTransactionV5(asset)
	for _, in := range utxos {
		h, err := crypto.HashFromString(in.TransactionHash)
		if err != nil {
			panic(in.TransactionHash)
		}
		tx.AddInput(h, in.OutputIndex)
	}

	for i, r := range recipients {
		if r.Destination != "" {
			tx.Outputs = append(tx.Outputs, &common.Output{
				Type:   common.OutputTypeWithdrawalSubmit,
				Amount: common.NewIntegerFromString(r.Amount),
				Withdrawal: &common.WithdrawalData{
					Address: r.Destination,
					Tag:     r.Tag,
				},
			})
			continue
		}
		ma, err := NewMixAddressFromString(r.MixAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid mix address %s", r.MixAddress)
		}
		ghost, err := ma.RequestOrGenerateGhostKeys(ctx, uint(i), u)
		if err != nil {
			return nil, err
		}
		mask, err := crypto.KeyFromString(ghost.Mask)
		if err != nil {
			panic(ghost.Mask)
		}
		tx.Outputs = append(tx.Outputs, &common.Output{
			Type:   common.OutputTypeScript,
			Amount: common.NewIntegerFromString(r.Amount),
			Script: common.NewThresholdScript(ma.Threshold),
			Keys:   ghost.KeysSlice(),
			Mask:   mask,
		})
	}
	return tx, nil
}

func verifyRawTransactionBySequencer(ctx context.Context, traceId string, ver *common.VersionedTransaction, u *SafeUser) (*SequencerTransactionRequest, error) {
	panic(0)
}

func signRawTransaction(ctx context.Context, ver *common.VersionedTransaction, views []string, spendKey string) (*common.VersionedTransaction, error) {
	msg := ver.PayloadHash()
	spent, err := crypto.KeyFromString(spendKey)
	if err != nil {
		return nil, err
	}
	y, err := edwards25519.NewScalar().SetCanonicalBytes(spent[:])
	if err != nil {
		return nil, err
	}
	signaturesMap := make([]map[uint16]*crypto.Signature, len(ver.Inputs))
	for i := range ver.Inputs {
		viewBytes, err := crypto.KeyFromString(views[i])
		if err != nil {
			return nil, err
		}
		x, err := edwards25519.NewScalar().SetCanonicalBytes(viewBytes[:])
		t := edwards25519.NewScalar().Add(x, y)
		var key crypto.Key
		copy(key[:], t.Bytes())
		sig := key.Sign(msg)
		sigs := make(map[uint16]*crypto.Signature)
		sigs[0] = &sig // for 1/1 bot transaction
		signaturesMap[i] = sigs
	}
	ver.SignaturesMap = signaturesMap
	return ver, nil
}

func sendRawTransactionToSequencer(ctx context.Context, traceId string, ver *common.VersionedTransaction, u *SafeUser) (*SequencerTransactionRequest, error) {
	panic(0)
}

func requestUnspentOutputsForRecipients(ctx context.Context, assetId string, recipients []*TransactionRecipient, u *SafeUser) ([]*Output, common.Integer, error) {
	var totalOutput common.Integer
	for _, r := range recipients {
		amt := common.NewIntegerFromString(r.Amount)
		totalOutput = totalOutput.Add(amt)
	}

	membersHash := HashMembers([]string{u.UserId})
	outputs, err := ListUnspentOutputs(ctx, membersHash, 1, assetId, u)
	if err != nil {
		return nil, common.Zero, err
	}

	var totalInput common.Integer
	for i, o := range outputs {
		amt := common.NewIntegerFromString(o.Amount)
		totalInput = totalInput.Add(amt)
		if totalInput.Cmp(totalOutput) < 0 {
			continue
		}
		return outputs[:i], totalInput.Sub(totalOutput), nil
	}
	return nil, common.Zero, fmt.Errorf("insufficient outputs %s@%d %s", totalInput, len(outputs), totalOutput)
}
