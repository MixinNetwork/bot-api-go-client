package bot

import (
	"context"
	"fmt"

	"github.com/MixinNetwork/mixin/common"
)

func CreateObjectStorageTransaction(ctx context.Context, extra []byte, traceId string, u *SafeUser) (*SequencerTransactionRequest, error) {
	if len(extra) > common.ExtraSizeStorageCapacity {
		return nil, fmt.Errorf("too large extra %d > %d", len(extra), common.ExtraSizeStorageCapacity)
	}
	step := common.NewIntegerFromString(common.ExtraStoragePriceStep)
	amount := step.Mul(len(extra)/common.ExtraSizeStorageStep + 1)
	addr := common.NewAddressFromSeed(make([]byte, 64))
	mix := NewMainnetMixAddress([]string{addr.String()}, 1)
	mix.Threshold = 64
	recipients := []*TransactionRecipient{{
		MixAddress: mix,
		Amount:     amount.String(),
	}}
	return SendTransaction(ctx, common.XINAssetId.String(), recipients, traceId, extra, nil, u)
}
