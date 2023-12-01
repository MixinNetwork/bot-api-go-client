package bot

import (
	"context"
	"encoding/json"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/gofrs/uuid/v5"
)

const (
	BTC        = "c6d0c728-2624-429b-8e0d-d9d19b6592fa"
	ETH        = "43d61dcd-e413-450d-80b8-101d5e903357"
	USDT_ERC20 = "4d8c508b-91c5-375b-92b0-ee702ed2dac5"
	USDC_ERC20 = "9b180ab6-6abe-3dc0-a13f-04169eb34bfa"
	USDT_TRC20 = "b91e18ff-a9ae-3dc7-8679-e935d9a4b34b"
)

type Asset struct {
	Type           string  `json:"type"`
	AssetID        string  `json:"asset_id"`
	ChainID        string  `json:"chain_id"`
	AssetKey       string  `json:"asset_key"`
	Precision      int     `json:"precision"`
	KernelAssetId  string  `json:"kernel_asset_id"`
	Symbol         string  `json:"symbol"`
	Name           string  `json:"name"`
	IconURL        string  `json:"icon_url"`
	Amount         string  `json:"amount"`
	PriceBTC       string  `json:"price_btc"`
	PriceUSD       string  `json:"price_usd"`
	ChangeBTC      string  `json:"change_btc"`
	ChangeUSD      string  `json:"change_usd"`
	Confirmations  int64   `json:"confirmations"`
	Fee            string  `json:"fee"`
	Reserve        string  `json:"reserve"`
	SnapshotsCount int64   `json:"snapshots_count"`
	Capitalization float64 `json:"capitalization"`
	Liquidity      string  `json:"liquidity"`
}

type AssetTicker struct {
	Type     string `json:"type"`
	PriceBTC string `json:"price_btc"`
	PriceUSD string `json:"price_usd"`
}

func ReadAsset(ctx context.Context, name string) (*Asset, error) {
	body, err := Request(ctx, "GET", "/network/assets/"+name, nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  *Asset `json:"data"`
		Error Error  `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func ReadAssetTicker(ctx context.Context, assetId string) (*AssetTicker, error) {
	body, err := Request(ctx, "GET", "/network/ticker?asset="+assetId, nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  *AssetTicker `json:"data"`
		Error Error        `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func AssetSearch(ctx context.Context, name string) ([]*Asset, error) {
	body, err := Request(ctx, "GET", "/network/assets/search/"+name, nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  []*Asset `json:"data"`
		Error Error    `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func AssetBalance(ctx context.Context, assetId, uid, sid, sessionKey string) (common.Integer, error) {
	su := &SafeUser{
		UserId:     uid,
		SessionId:  sid,
		SessionKey: sessionKey,
	}

	if id, _ := uuid.FromString(assetId); assetId == id.String() {
		assetId = crypto.Sha256Hash([]byte(assetId)).String()
	}
	return AssetBalanceWithSafeUser(ctx, assetId, su)
}

func AssetBalanceWithSafeUser(ctx context.Context, kernelAssetId string, su *SafeUser) (common.Integer, error) {
	membersHash := HashMembers([]string{su.UserId})
	outputs, err := ListUnspentOutputs(ctx, membersHash, 1, kernelAssetId, su)
	if err != nil {
		return common.Zero, err
	}
	var total common.Integer
	for _, o := range outputs {
		amt := common.NewIntegerFromString(o.Amount)
		total = total.Add(amt)
	}
	return total, nil
}
