package bot

import (
	"context"
	"encoding/json"
)

func ReadAsset(ctx context.Context, name string) (*AssetNetwork, error) {
	body, err := Request(ctx, "GET", "/network/assets/"+name, nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  *AssetNetwork `json:"data"`
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
