package bot

import (
	"context"

	"github.com/MixinNetwork/mixin/common"
)

func SendWithdrawal(ctx context.Context, assetId, destination, tag, amount, traceId string, u *SafeUser) (*common.VersionedTransaction, error) {
	recipients := []*TransactionRecipient{{
		Destination: destination,
		Tag:         tag,
		Amount:      amount,
	}}
	return SendTransaction(ctx, assetId, recipients, traceId, "", u)
}

func PayWithdrawalFee(ctx context.Context, traceId, feeId, amount string, u *SafeUser) (*common.VersionedTransaction, error) {
	// 1. get withdrawal transaction request with the trace id
	// 2. get unspent outputs for fee id
	// 3. build the transaction to cashier, with withdrawal transaction hash as reference
	panic(0)
}
