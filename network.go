package bot

import (
	"context"
	"encoding/json"
)

type AssetNetwork struct {
	AssetID                   string  `json:"asset_id"`
	ChainID                   string  `json:"chain_id"`
	FeeAssetID                string  `json:"fee_asset_id"`
	Symbol                    string  `json:"symbol"`
	Name                      string  `json:"name"`
	IconURL                   string  `json:"icon_url"`
	Balance                   string  `json:"balance"`
	Destination               string  `json:"destination"`
	Tag                       string  `json:"tag"`
	PriceBTC                  string  `json:"price_btc"`
	PriceUSD                  string  `json:"price_usd"`
	ChangeBTC                 string  `json:"change_btc"`
	ChangeUSD                 string  `json:"change_usd"`
	AssetKey                  string  `json:"asset_key"`
	Precision                 int     `json:"precision"`
	MixinID                   string  `json:"mixin_id"`
	KernelAssetID             string  `json:"kernel_asset_id"`
	Reserve                   string  `json:"reserve"`
	Dust                      string  `json:"dust"`
	Confirmations             int     `json:"confirmations"`
	Capitalization            float64 `json:"capitalization"`
	Liquidity                 string  `json:"liquidity"`
	PriceUpdatedAt            string  `json:"price_updated_at"`
	WithdrawalMemoPossibility string  `json:"withdrawal_memo_possibility"`
}

func ReadNetworkAssets(ctx context.Context) ([]*AssetNetwork, error) {
	body, err := Request(ctx, "GET", "/network", nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data struct {
			Assets []*AssetNetwork `json:"assets"`
		} `json:"data"`
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data.Assets, nil
}

func ReadNetworkAssetsTop(ctx context.Context) ([]*AssetNetwork, error) {
	body, err := Request(ctx, "GET", "/network/assets/top", nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  []*AssetNetwork `json:"data"`
		Error Error           `json:"error"`
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
