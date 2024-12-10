package bot

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
)

const (
	MixinFeeUserId = "674d6776-d600-4346-af46-58e77d8df185"
)

// SendWithdrawal sends a withdrawal request to the Mixin Network.
// preferAssetFeeOverChainFee is used to determine whether to use the asset fee or the chain fee.
func SendWithdrawal(ctx context.Context, assetId, destination, tag, amount, traceId string, preferAssetFeeOverChainFee bool, memo string, u *SafeUser) ([]*SequencerTransactionRequest, error) {
	asset, err := ReadAsset(ctx, assetId)
	if err != nil {
		return nil, err
	}
	var chain *AssetNetwork
	if asset.ChainID == asset.AssetID {
		chain = asset
	} else {
		chain, err = ReadAsset(ctx, asset.ChainID)
		if err != nil {
			return nil, err
		}
	}

	fees, err := ReadAssetFee(ctx, assetId, destination, u)
	if err != nil {
		return nil, err
	}
	var assetFee *AssetFee
	var chainFee *AssetFee
	for _, fee := range fees {
		if fee.AssetID == assetId {
			assetFee = fee
		} else if fee.AssetID == chain.AssetID {
			chainFee = fee
		}
	}
	var fee *AssetFee
	if preferAssetFeeOverChainFee && assetFee != nil {
		fee = assetFee
	} else {
		fee = chainFee
	}

	return withdrawalTransaction(ctx, traceId, MixinFeeUserId, fee.AssetID, fee.Amount, assetId, destination, tag, memo, amount, nil, u)
}

func WithdrawalWithFeeUtxos(ctx context.Context, traceId, feeAssetId, feeAmount, assetId, destination, tag, memo, amount string, feeUtxos []*Output, u *SafeUser) ([]*SequencerTransactionRequest, error) {
	return withdrawalTransaction(ctx, traceId, MixinFeeUserId, feeAssetId, feeAmount, assetId, destination, tag, memo, amount, feeUtxos, u)
}

func withdrawalTransaction(ctx context.Context, traceId, feeReceiverId string, feeAssetId string, feeAmount,
	assetId, destination, tag, memo, amount string, feeUtxos []*Output, u *SafeUser) ([]*SequencerTransactionRequest, error) {
	isFeeDifferentAsset := feeAssetId != assetId
	asset := crypto.Sha256Hash([]byte(assetId))
	if isFeeDifferentAsset {
		feeTraceId := UniqueObjectId(traceId, "FEE")
		feeAsset := crypto.Sha256Hash([]byte(feeAssetId))
		recipients := []*TransactionRecipient{
			{
				Amount:      amount,
				Destination: destination,
				Tag:         tag,
			},
		}
		utxos, change, err := requestUnspentOutputsForRecipients(ctx, assetId, recipients, u)
		if err != nil {
			return nil, err
		}
		if change.Sign() > 0 {
			recipients = append(recipients, &TransactionRecipient{
				Amount:     change.String(),
				MixAddress: NewUUIDMixAddress([]string{u.UserId}, 1),
			})
		}
		feeRecipients := []*TransactionRecipient{
			{
				Amount:     feeAmount,
				MixAddress: NewUUIDMixAddress([]string{feeReceiverId}, 1),
			},
		}

		var feeChange common.Integer
		if len(feeUtxos) > 0 {
			totalFeeOutput := common.NewIntegerFromString(feeAmount)
			var totalFeeInput common.Integer
			for _, o := range feeUtxos {
				amt := common.NewIntegerFromString(o.Amount)
				totalFeeInput = totalFeeInput.Add(amt)
			}
			if totalFeeInput.Cmp(totalFeeOutput) < 0 {
				return nil, &UtxoInsufficientError{
					TotalInput:  totalFeeInput,
					TotalOutput: totalFeeOutput,
					OutputSize:  len(feeUtxos),
				}
			}
			feeChange = totalFeeInput.Sub(totalFeeOutput)
		} else {
			feeUtxos, feeChange, err = requestUnspentOutputsForRecipients(ctx, feeAssetId, feeRecipients, u)
			if err != nil {
				return nil, err
			}
		}
		if feeChange.Sign() > 0 {
			feeRecipients = append(feeRecipients, &TransactionRecipient{
				Amount:     feeChange.String(),
				MixAddress: NewUUIDMixAddress([]string{u.UserId}, 1),
			})
		}

		transaction, err := BuildRawTransaction(ctx, asset, utxos, recipients, []byte(memo), nil, traceId, u)
		if err != nil {
			return nil, fmt.Errorf("BuildRawTransaction(%s): %w", asset, err)
		}
		ver := transaction.AsVersioned()
		feeTransaction, err := BuildRawTransaction(ctx, feeAsset, feeUtxos, feeRecipients, []byte(memo), []string{crypto.Blake3Hash(ver.Marshal()).String()}, feeTraceId, u)
		if err != nil {
			return nil, fmt.Errorf("buildFeeRawTransaction(%s): %w", feeAsset, err)
		}
		feeVer := feeTransaction.AsVersioned()

		requests, err := VerifyRawTransaction(ctx, []*KernelTransactionRequestCreateRequest{
			{
				RequestID: traceId,
				Raw:       hex.EncodeToString(ver.Marshal()),
			},
			{
				RequestID: feeTraceId,
				Raw:       hex.EncodeToString(feeVer.Marshal()),
			},
		}, u)
		if err != nil {
			return nil, err
		} else if len(requests) != 2 {
			return nil, fmt.Errorf("invalid requests count %d", len(requests))
		} else if requests[0].State != "unspent" {
			return nil, fmt.Errorf("invalid transaction state %s", requests[0].State)
		}
		var str *SequencerTransactionRequest
		var feeStr *SequencerTransactionRequest
		for _, r := range requests {
			if r.RequestID == traceId {
				str = r
			} else if r.RequestID == feeTraceId {
				feeStr = r
			}
		}
		if str == nil || feeStr == nil {
			return nil, fmt.Errorf("invalid sequencer transaction requests")
		}
		if len(str.Views) != len(ver.Inputs) {
			return nil, fmt.Errorf("invalid inputs count %d/%d", len(str.Views), len(ver.Inputs))
		}
		if len(feeStr.Views) != len(feeVer.Inputs) {
			return nil, fmt.Errorf("invalid fee inputs count %d/%d", len(feeStr.Views), len(feeVer.Inputs))
		}

		ver, err = signRawTransaction(ver, str.Views, u.SpendPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("signRawTransaction(%s): %w", asset, err)
		}
		feeVer, err = signRawTransaction(feeVer, feeStr.Views, u.SpendPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("signFeeRawTransaction(%s): %w", feeAsset, err)
		}
		results, err := SendRawTransaction(ctx, []*KernelTransactionRequestCreateRequest{
			{
				RequestID: traceId,
				Raw:       hex.EncodeToString(ver.Marshal()),
			},
			{
				RequestID: feeTraceId,
				Raw:       hex.EncodeToString(feeVer.Marshal()),
			},
		}, u)
		if err != nil {
			return nil, fmt.Errorf("SendRawTransaction(%s): %w", traceId, err)
		}
		return results, nil
	} else {
		recipients := []*TransactionRecipient{
			{
				Amount:      amount,
				Destination: destination,
				Tag:         tag,
			},
			{
				Amount:     feeAmount,
				MixAddress: NewUUIDMixAddress([]string{MixinFeeUserId}, 1),
			},
		}
		tx, err := SendTransaction(ctx, assetId, recipients, traceId, []byte(memo), nil, u)
		if err != nil {
			return nil, err
		}
		return []*SequencerTransactionRequest{tx}, nil
	}
}
