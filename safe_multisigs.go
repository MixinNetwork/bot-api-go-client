package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type KernelTransactionRequest struct {
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

	Views []string `json:"views,omitempty"`
}

func CreateMultisigTransactionRequests(ctx context.Context, requests []*KernelTransactionRequestCreateRequest, u *SafeUser) ([]*KernelTransactionRequest, error) {
	data, err := json.Marshal(requests)
	if err != nil {
		return nil, err
	}
	method, path := "POST", "/safe/multisigs"
	token, err := SignAuthenticationToken(u.UserId, u.SessionId, u.SessionKey, method, path, string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*KernelTransactionRequest `json:"data"`
		Error Error                       `json:"error"`
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

func SignMultisigTransactionRequests(ctx context.Context, id string, request *KernelTransactionRequestCreateRequest, u *SafeUser) (*KernelTransactionRequest, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	method, path := "POST", fmt.Sprintf("/safe/multisigs/%s/sign", id)
	token, err := SignAuthenticationToken(u.UserId, u.SessionId, u.SessionKey, method, path, string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *KernelTransactionRequest `json:"data"`
		Error Error                     `json:"error"`
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

func UnlockMultisigTransactionRequests(ctx context.Context, id string, u *SafeUser) (*KernelTransactionRequest, error) {
	method, path := "POST", fmt.Sprintf("/safe/multisigs/%s/unlock", id)
	token, err := SignAuthenticationToken(u.UserId, u.SessionId, u.SessionKey, method, path, "")
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, nil, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *KernelTransactionRequest `json:"data"`
		Error Error                     `json:"error"`
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

func GetMultisigTransactionRequests(ctx context.Context, id string, u *SafeUser) (*KernelTransactionRequest, error) {
	method, path := "GET", fmt.Sprintf("/safe/multisigs/%s", id)
	token, err := SignAuthenticationToken(u.UserId, u.SessionId, u.SessionKey, method, path, "")
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, nil, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *KernelTransactionRequest `json:"data"`
		Error Error                     `json:"error"`
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
