package bot

import (
	"context"
	"encoding/json"
)

type SafeDepositPending struct {
	Amount          string `json:"amount"`
	AssetID         string `json:"asset_id"`
	AssetKey        string `json:"asset_key"`
	BlockHash       string `json:"block_hash"`
	BlockNumber     int    `json:"block_number"`
	ChainID         string `json:"chain_id"`
	Confirmations   int    `json:"confirmations"`
	CreatedAt       string `json:"created_at"`
	DepositID       string `json:"deposit_id"`
	Destination     string `json:"destination"`
	Extra           string `json:"extra"`
	KernelAssetID   string `json:"kernel_asset_id"`
	OutputIndex     int    `json:"output_index"`
	Sender          string `json:"sender"`
	State           string `json:"state"`
	Tag             string `json:"tag"`
	Threshold       int    `json:"threshold"`
	TransactionHash string `json:"transaction_hash"`
	UpdatedAt       string `json:"updated_at"`
}

func FetchSafeDeposit(ctx context.Context) ([]*SafeDepositPending, error) {
	endpoint := "/safe/deposits"
	body, err := Request(ctx, "GET", endpoint, nil, "")
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*SafeDepositPending `json:"data"`
		Error Error                 `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}
