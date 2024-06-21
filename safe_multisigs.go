package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type SafeMultisigRequest struct {
	RequestID        string    `json:"request_id"`
	TransactionHash  string    `json:"transaction_hash"`
	AssetID          string    `json:"asset_id"`
	KernelAssetID    string    `json:"kernel_asset_id"`
	Amount           string    `json:"amount"`
	SendersHash      string    `json:"senders_hash"`
	SendersThreshold uint8     `json:"senders_threshold"`
	Senders          []string  `json:"senders"`
	Signers          []string  `json:"signers"`
	Extra            string    `json:"extra"`
	RawTransaction   string    `json:"raw_transaction"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Views            []string  `json:"views"`
}

func FetchSafeMultisigRequest(ctx context.Context, idOrHash string, user *SafeUser) (*SafeMultisigRequest, error) {
	endpoint := "/safe/multisigs/" + idOrHash
	token, err := SignAuthenticationToken("GET", endpoint, "", user)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", endpoint, nil, token)
	if err != nil {
		fmt.Println(err)
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  SafeMultisigRequest `json:"data"`
		Error Error               `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return &resp.Data, nil
}
