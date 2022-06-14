package bot

import (
	"context"
	"encoding/json"
	"net/url"
)

type Transaction struct {
	Type            string `json:"type"`
	TransactionId   string `json:"transaction_id"`
	TransactionHash string `json:"transaction_hash"`
	Sender          string `json:"sender"`
	ChainId         string `json:"chain_id"`
	AssetId         string `json:"asset_id"`
	Amount          string `json:"amount"`
	Destination     string `json:"destination"`
	Tag             string `json:"tag"`
	Confirmations   int    `json:"confirmations"`
	Threshold       int    `json:"threshold"`
	CreatedAt       string `json:"created_at"`
}

func ExternalTranactions(ctx context.Context, asset, destination, tag string) ([]*Transaction, error) {
	values := url.Values{}
	if destination != "" {
		values.Add("destination", destination)
	}
	if tag != "" {
		values.Add("tag", tag)
	}
	if asset != "" {
		values.Add("asset", asset)
	}

	endpoint := "/external/transactions?" + values.Encode()
	body, err := Request(ctx, "GET", endpoint, nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  []*Transaction `json:"data"`
		Error *Error         `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp.Data, nil
}
