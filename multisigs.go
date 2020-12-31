package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

type MultisigUTXO struct {
	Type            string    `json:"type"`
	UserId          string    `json:"user_id"`
	UTXOId          string    `json:"utxo_id"`
	AssetId         string    `json:"asset_id"`
	TransactionHash string    `json:"transaction_hash"`
	OutputIndex     int64     `json:"output_index"`
	Amount          string    `json:"amount"`
	Threshold       int64     `json:"threshold"`
	Members         []string  `json:"members"`
	Memo            string    `json:"memo"`
	State           string    `json:"state"`
	CreatedAt       time.Time `json:"created_at"`
	SignedBy        string    `json:"signed_by"`
	SignedTx        string    `json:"signed_tx"`
}

func ReadMultisigsLegacy(ctx context.Context, limit int, offset, uid, sid, sessionKey string) ([]*MultisigUTXO, error) {
	v := url.Values{}
	v.Set("limit", fmt.Sprint(limit))
	if offset != "" {
		v.Set("offset", offset)
	}
	method, path := "GET", "/multisigs?"+v.Encode()
	token, err := SignAuthenticationToken(uid, sid, sessionKey, method, path, "")
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, nil, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*MultisigUTXO `json:"data"`
		Error Error           `json:"error"`
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

// state: spent, unspent, signed
func ReadMultisigs(ctx context.Context, limit int, offset, state, uid, sid, sessionKey string) ([]*MultisigUTXO, error) {
	v := url.Values{}
	v.Set("limit", fmt.Sprint(limit))
	if offset != "" {
		v.Set("offset", offset)
	}
	v.Set("state", state)
	method, path := "GET", "/multisigs/outputs?"+v.Encode()
	token, err := SignAuthenticationToken(uid, sid, sessionKey, method, path, "")
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, nil, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*MultisigUTXO `json:"data"`
		Error Error           `json:"error"`
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

type MultisigRequest struct {
	Type            string    `json:"type"`
	RequestId       string    `json:"request_id"`
	UserId          string    `json:"user_id"`
	AssetId         string    `json:"asset_id"`
	Amount          string    `json:"amount"`
	Threshold       int64     `json:"threshold"`
	Senders         []string  `json:"senders"`
	Receivers       []string  `json:"receivers"`
	Signers         []string  `json:"signers"`
	Memo            string    `json:"memo"`
	Action          string    `json:"action"`
	State           string    `json:"state"`
	TransactionHash string    `json:"transaction_hash"`
	RawTransaction  string    `json:"raw_transaction"`
	CreatedAt       time.Time `json:"created_at"`
	CodeId          string    `json:"code_id"`
}

// CreateMultisig create a multisigs request which action is `unlock` or `sign`
func CreateMultisig(ctx context.Context, action, raw string, uid, sid, sessionKey string) (*MultisigRequest, error) {
	data, err := json.Marshal(map[string]string{
		"action": action,
		"raw":    raw,
	})
	if err != nil {
		return nil, err
	}
	method, path := "POST", "/multisigs/requests"
	token, err := SignAuthenticationToken(uid, sid, sessionKey, method, path, string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *MultisigRequest `json:"data"`
		Error Error            `json:"error"`
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

func SignMultisig(ctx context.Context, id, pin string, uid, sid, sessionKey string) (*MultisigRequest, error) {
	data, err := json.Marshal(map[string]string{
		"pin": pin,
	})
	if err != nil {
		return nil, err
	}
	method, path := "POST", "/multisigs/requests/"+id+"/sign"
	token, err := SignAuthenticationToken(uid, sid, sessionKey, method, path, string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *MultisigRequest `json:"data"`
		Error Error            `json:"error"`
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

func CancelMultisig(ctx context.Context, id string, uid, sid, sessionKey string) error {
	method, path := "POST", "/multisigs/requests/"+id+"/cancel"
	token, err := SignAuthenticationToken(uid, sid, sessionKey, method, path, "")
	if err != nil {
		return err
	}
	body, err := Request(ctx, method, path, nil, token)
	if err != nil {
		return ServerError(ctx, err)
	}
	var resp struct {
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return resp.Error
	}
	return nil
}

func UnlockMultisig(ctx context.Context, id, pin string, uid, sid, sessionKey string) error {
	data, err := json.Marshal(map[string]string{
		"pin": pin,
	})
	if err != nil {
		return err
	}
	method, path := "POST", "/multisigs/requests/"+id+"/unlock"
	token, err := SignAuthenticationToken(uid, sid, sessionKey, method, path, string(data))
	if err != nil {
		return err
	}
	body, err := Request(ctx, method, path, data, token)
	if err != nil {
		return ServerError(ctx, err)
	}
	var resp struct {
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return resp.Error
	}
	return nil
}

type GhostKeys struct {
	Type string   `json:"type"`
	Mask string   `json:"mask"`
	Keys []string `json:"keys"`
}

type GhostKeyRequest struct {
	Receivers []string `json:"receivers"`
	Index     int      `json:"index"`
	Hint      string   `json:"hint"`
}

func ReadGhostKeys(ctx context.Context, gkr []GhostKeyRequest, uid, sid, sessionKey string) ([]*GhostKeys, error) {
	data, err := json.Marshal(gkr)
	if err != nil {
		return nil, err
	}
	method, path := "POST", "/outputs"
	body, err := Request(ctx, method, path, data, "")
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*GhostKeys `json:"data"`
		Error Error        `json:"error"`
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

func ReadGhostKey(ctx context.Context, receivers []string, index int, hint string, uid, sid, sessionKey string) (*GhostKeys, error) {
	r := GhostKeyRequest{
		Receivers: receivers,
		Index:     index,
		Hint:      hint,
	}
	result, err := ReadGhostKeys(ctx, []GhostKeyRequest{r}, uid, sid, sessionKey)
	if err != nil {
		return nil, err
	}
	return result[0], nil
}
