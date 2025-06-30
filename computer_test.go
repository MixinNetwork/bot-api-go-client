package bot

import (
	"context"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestComptuer(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	info, err := GetComputerInfo(ctx)
	assert.Nil(err)
	assert.Equal("c1931517-4a0e-41c2-a1eb-eaae7c6242ee", info.ObserverId)
	assert.Equal("BaEPep2hKPNLyafnv51foxRbymh5BbZ2H6DKRiKww4ka", info.Payer)
	assert.True(info.Height > 348843998)
	assert.Equal("0acfe278-714f-4cfc-ae52-70ce34e3eb11", info.Members.AppId)
	assert.Len(info.Members.Members, 7)
	assert.Equal(5, info.Members.Threshold)
	assert.Equal("c94ac88f-4671-3976-b60a-09064f1811e8", info.Params.Operation.Asset)
	assert.Equal("0.001", info.Params.Operation.Price)

	err = ComputerDeployExternalAsset(ctx, []string{XINAssetId})
	assert.Nil(err)
	as, err := GetComputerDeployedAssets(ctx)
	assert.Nil(err)
	assert.Equal(XINAssetId, as[0].AssetID)
	assert.Equal("4s4H5v4TXpmS4Ss66nxcCLgxrU5nunuwtkQceinZfGuw", as[0].Address)

	user, err := GetComputerUser(ctx, "MIX3QEetjLB1hKcPGEbBKF8PvMaxSuttJg")
	assert.Nil(err)
	assert.Equal("281474976710657", user.ID)
	assert.Equal("MIX3QEetjLB1hKcPGEbBKF8PvMaxSuttJg", user.MixAddress)
	assert.Equal("6LeUogC869ABqSCQM9ysRjH6eWdTEJRYxYJnoNq5g2tf", user.ChainAddress)

	nonce, err := LockComputerNonceAccount(ctx, user.MixAddress)
	assert.Nil(err)
	assert.Equal("MIX3QEetjLB1hKcPGEbBKF8PvMaxSuttJg", nonce.Mix)

	fee, err := GetFeeOnXINBasedOnSOL(ctx, "0.001")
	assert.Nil(err)
	_, err = uuid.FromString(fee.FeeID)
	assert.Nil(err)
	assert.True(decimal.RequireFromString(fee.XINAmount).GreaterThan(decimal.RequireFromString("0.001")))

	call, err := GetComputerSystemCall(ctx, "c4080ecf-044a-3ef5-8da6-de3e9beb1030")
	assert.Nil(err)
	assert.Equal("c4080ecf-044a-3ef5-8da6-de3e9beb1030", call.ID)
	assert.Equal("main", call.Type)
	assert.Len(call.SubCalls, 2)
	assert.Equal("prepare", call.SubCalls[0].Type)
	assert.Equal("3gwRRFbE4R9F1zx6EJQArht8GZW9cZ2YSHMKxDsah8H8sa5stPuYN8Q3KnX2wYhMNBc8VYBhmRGtKqDxAtXEnZpH", call.SubCalls[0].Hash)
	assert.Equal("post_process", call.SubCalls[1].Type)
	assert.Equal("4QNaXPsXttmD4pt9d2VydT56opyjrRGLqTu4iGc7fkZfCFJXKf1CUy8VeFTuTMEYkRv4RhXpMCni6urikXMBbr42", call.SubCalls[1].Hash)
}
