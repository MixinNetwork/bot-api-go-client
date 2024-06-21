package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type OutputReceiverView struct {
	Members     []string `json:"members"`
	MembersHash string   `json:"members_hash"`
	Threshold   int      `json:"threshold"`

	Destination    string `json:"destination"`
	Tag            string `json:"tag"`
	WithdrawalHash string `json:"withdrawal_hash"`
}

type SafeMultisigRequest struct {
	Type             string    `json:"type"`
	RequestID        string    `json:"request_id"`
	TransactionHash  string    `json:"transaction_hash"`
	AssetId          string    `json:"asset_id"`
	KernelAssetID    string    `json:"kernel_asset_id"`
	Amount           string    `json:"amount"`
	SendersHash      string    `json:"senders_hash"`
	SendersThreshold int64     `json:"senders_threshold"`
	Senders          []string  `json:"senders"`
	Signers          []string  `json:"signers"`
	Extra            string    `json:"extra"`
	RawTransaction   string    `json:"raw_transaction"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	InscriptionHash string                `json:"inscription_hash,omitempty"`
	Receivers       []*OutputReceiverView `json:"receivers,omitempty"`
	Views           []string              `json:"views,omitempty"`
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
