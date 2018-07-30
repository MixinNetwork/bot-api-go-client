package bot

import (
	"context"
	"encoding/json"
)

type Asset struct {
	AssetId  string `json:"asset_id"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	IconURL  string `json:"icon_url"`
	PriceBTC string `json:"price_btc"`
	PriceUSD string `json:"price_usd"`
	Balance  string `json:"balance"`
}

func AssetList(ctx context.Context, accessToken string) ([]Asset, error) {
	body, err := Request(ctx, "GET", "/assets", nil, accessToken)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  []Asset `json:"data"`
		Error Error   `json:"error"`
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

func AssetShow(ctx context.Context, assetId string, accessToken string) (*Asset, error) {
	body, err := Request(ctx, "GET", "/assets/"+assetId, nil, accessToken)
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
