package bot

import (
	"context"
	"fmt"

	"github.com/MixinNetwork/mixin/common"
)

func CreateObjectStorageTransaction(ctx context.Context, recipients []*TransactionRecipient, utxos []*Output, extra []byte, traceId string, references []string, limit string, u *SafeUser) (*SequencerTransactionRequest, error) {
	if len(extra) > common.ExtraSizeStorageCapacity {
		return nil, fmt.Errorf("too large extra %d > %d", len(extra), common.ExtraSizeStorageCapacity)
	}
	amount := EstimateStorageCost(extra)
	if limit != "" {
		strl := common.NewIntegerFromString(limit)
		if strl.Cmp(amount) > 0 {
			amount = strl
		}
	}
	mix := StorageRecipient()
	rec := []*TransactionRecipient{{
		MixAddress: mix,
		Amount:     amount.String(),
	}}
	if len(recipients) > 0 {
		rec = append(rec, recipients...)
	}
	if len(utxos) > 0 {
		return SendTransactionWithOutputs(ctx, common.XINAssetId.String(), rec, utxos, traceId, extra, references, u)
	}
	return SendTransaction(ctx, common.XINAssetId.String(), rec, traceId, extra, references, u)
}

func EstimateStorageCost(extra []byte) common.Integer {
	if len(extra) > common.ExtraSizeStorageCapacity {
		panic(len(extra))
	}
	step := common.NewIntegerFromString(common.ExtraStoragePriceStep)
	return step.Mul(len(extra)/common.ExtraSizeStorageStep + 1)
}

func StorageRecipient() *MixAddress {
	addr := common.NewAddressFromSeed(make([]byte, 64))
	mix := NewMainnetMixAddress([]string{addr.String()}, 1)
	mix.Threshold = 64
	return mix
}
