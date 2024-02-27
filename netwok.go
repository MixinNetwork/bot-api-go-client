package bot

import (
	"context"
	"encoding/json"
)

type AssetNetwork struct {
	Amount  string `json:"amount"`
	AssetID string `json:"asset_id"`
	IconURL string `json:"icon_url"`
	Symbol  string `json:"symbol"`
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
