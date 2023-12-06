package bot

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/gofrs/uuid/v5"
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

func BuildMultiTransaction(ctx context.Context, assetId string, recipients []*TransactionRecipient, traceId string, extra []byte, references []string, members []string, threshold byte, u *SafeUser) (*KernelTransactionRequest, error) {
	if uuid.FromStringOrNil(assetId).String() == assetId {
		assetId = crypto.Sha256Hash([]byte(assetId)).String()
	}
	asset, err := crypto.HashFromString(assetId)
	if err != nil {
		return nil, fmt.Errorf("invalid asset id %s", assetId)
	}
	if len(references) > 2 {
		return nil, fmt.Errorf("too many references %d", len(references))
	}

	// get unspent outputs for asset and may return insufficient outputs error
	utxos, changeAmount, err := requestUnspentMultiOutputsForRecipients(ctx, assetId, recipients, members, threshold, u)
	if err != nil {
		return nil, fmt.Errorf("requestUnspentOutputsForRecipients(%s) => %v", assetId, err)
	}
	// change to the sender
	if changeAmount.Sign() > 0 {
		ma := NewUUIDMixAddress(members, threshold)
		recipients = append(recipients, &TransactionRecipient{
			MixAddress: ma.String(),
			Amount:     changeAmount.String(),
		})
	}

	// build the unsigned raw transaction
	tx, err := buildRawTransaction(ctx, asset, utxos, recipients, extra, references, u)
	if err != nil {
		return nil, fmt.Errorf("buildRawTransaction(%s) => %v", asset, err)
	}
	ver := tx.AsVersioned()
	// verify the raw transaction, the same trace id may have been signed already
	requests := []*KernelTransactionRequestCreateRequest{
		{
			RequestID: traceId,
			Raw:       hex.EncodeToString(ver.Marshal()),
		},
	}
	rs, err := CreateMultisigTransactionRequests(ctx, requests, u)
	if err != nil {
		return nil, err
	}
	if len(rs) != 1 {
		return nil, errors.New("invalid response size")
	}
	return rs[0], nil
}

func requestUnspentMultiOutputsForRecipients(ctx context.Context, assetId string, recipients []*TransactionRecipient, members []string, threshold byte, u *SafeUser) ([]*Output, common.Integer, error) {
	membersHash := HashMembers(members)
	outputs, err := ListUnspentOutputs(ctx, membersHash, threshold, assetId, u)
	if err != nil {
		return nil, common.Zero, err
	}
	if len(outputs) == 0 {
		return nil, common.Zero, &UtxoError{
			TotalInput:  common.Zero,
			TotalOutput: common.Zero,
			OutputSize:  0,
		}
	}

	var totalOutput common.Integer
	for _, r := range recipients {
		amt := common.NewIntegerFromString(r.Amount)
		totalOutput = totalOutput.Add(amt)
	}

	var totalInput common.Integer
	for i, o := range outputs {
		amt := common.NewIntegerFromString(o.Amount)
		totalInput = totalInput.Add(amt)
		if totalInput.Cmp(totalOutput) < 0 {
			continue
		}
		return outputs[:i+1], totalInput.Sub(totalOutput), nil
	}
	return nil, common.Zero, &UtxoError{
		TotalInput:  totalInput,
		TotalOutput: totalOutput,
		OutputSize:  len(outputs),
	}
}
