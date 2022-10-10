package bot

import (
	"context"
	"encoding/json"
)

type NetworkAsset struct {
	Type           string  `json:"type"`
	AssetID        string  `json:"asset_id"`
	FeeAssetId     string  `json:"fee_asset_id"`
	ChainID        string  `json:"chain_id"`
	AssetKey       string  `json:"asset_key"`
	MixinID        string  `json:"mixin_id"`
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

func ReadNetworkAsset(ctx context.Context, name string) (*NetworkAsset, error) {
	body, err := Request(ctx, "GET", "/network/assets/"+name, nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  *NetworkAsset `json:"data"`
		Error Error         `json:"error"`
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
