package bot

import (
	"context"
	"encoding/json"
)

const (
	BTC        = "c6d0c728-2624-429b-8e0d-d9d19b6592fa"
	ETH        = "43d61dcd-e413-450d-80b8-101d5e903357"
	USDT_ERC20 = "4d8c508b-91c5-375b-92b0-ee702ed2dac5"
	USDC_ERC20 = "9b180ab6-6abe-3dc0-a13f-04169eb34bfa"
)

type DepositEntry struct {
	Destination string   `json:"destination"`
	Tag         string   `json:"tag"`
	Properties  []string `json:"properties"`
}

type Asset struct {
	AssetId        string         `json:"asset_id"`
	ChainId        string         `json:"chain_id"`
	FeeAssetId     string         `json:"fee_asset_id"`
	Symbol         string         `json:"symbol"`
	Name           string         `json:"name"`
	IconURL        string         `json:"icon_url"`
	PriceBTC       string         `json:"price_btc"`
	PriceUSD       string         `json:"price_usd"`
	Balance        string         `json:"balance"`
	Destination    string         `json:"destination"`
	Tag            string         `json:"tag"`
	Confirmations  int            `json:"confirmations"`
	DepositEntries []DepositEntry `json:"deposit_entries"`
}

func AssetListWithRequestID(ctx context.Context, accessToken, requestID string) ([]*Asset, error) {
	body, err := RequestWithId(ctx, "GET", "/assets", nil, accessToken, requestID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  []*Asset `json:"data"`
		Error Error    `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		if resp.Error.Code == 401 {
			return nil, AuthorizationError(ctx)
		} else if resp.Error.Code == 403 {
			return nil, ForbiddenError(ctx)
		}
		return nil, ServerError(ctx, resp.Error)
	}
	return resp.Data, nil
}

func AssetList(ctx context.Context, accessToken string) ([]*Asset, error) {
	return AssetListWithRequestID(ctx, accessToken, UuidNewV4().String())
}

func AssetShowWithRequestID(ctx context.Context, assetId string, accessToken, requestID string) (*Asset, error) {
	body, err := RequestWithId(ctx, "GET", "/assets/"+assetId, nil, accessToken, requestID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  Asset `json:"data"`
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		if resp.Error.Code == 401 {
			return nil, AuthorizationError(ctx)
		} else if resp.Error.Code == 403 {
			return nil, ForbiddenError(ctx)
		}
		return nil, resp.Error
	}
	return &resp.Data, nil
}
func AssetShow(ctx context.Context, assetId string, accessToken string) (*Asset, error) {
	return AssetShowWithRequestID(ctx, assetId, accessToken, UuidNewV4().String())
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
