package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MixinNetwork/go-number"
)

type TransferInput struct {
	AssetId          string
	RecipientId      string
	Amount           number.Decimal
	TraceId          string
	Memo             string
	OpponentKey      string
	OpponentMultisig struct {
		Receivers []string
		Threshold int64
	}
}

type RawTransaction struct {
	Type            string    `json:"type"`
	SnapshotId      string    `json:"snapshot_id"`
	OpponentKey     string    `json:"opponent_key"`
	AssetId         string    `json:"asset_id"`
	Amount          string    `json:"amount"`
	TraceId         string    `json:"trace_id"`
	Memo            string    `json:"memo"`
	State           string    `json:"state"`
	CreatedAt       time.Time `json:"created_at"`
	TransactionHash string    `json:"transaction_hash"`
	SnapshotHash    string    `json:"snapshot_hash"`
	SnapshotAt      time.Time `json:"snapshot_at"`
}

func CreateMultisigTransaction(ctx context.Context, in *TransferInput, uid, sid, sessionKey, pin, pinToken string) (*RawTransaction, error) {
	if in.Amount.Exhausted() {
		return nil, fmt.Errorf("amount exhausted")
	}

	encryptedPIN, err := EncryptPIN(pin, pinToken, sid, sessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(map[string]interface{}{
		"asset_id":          in.AssetId,
		"opponent_multisig": map[string]interface{}{"receivers": in.OpponentMultisig.Receivers, "threshold": in.OpponentMultisig.Threshold},
		"amount":            in.Amount.Persist(),
		"trace_id":          in.TraceId,
		"memo":              in.Memo,
		"pin":               encryptedPIN,
	})
	if err != nil {
		return nil, err
	}

	path := "/transactions"
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", path, string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", path, data, token)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Error Error          `json:"error"`
		Data  RawTransaction `json:"data"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return &resp.Data, nil
}

func CreateTransaction(ctx context.Context, in *TransferInput, uid, sid, sessionKey, pin, pinToken string) (*RawTransaction, error) {
	if in.Amount.Exhausted() {
		return nil, fmt.Errorf("amount exhausted")
	}

	encryptedPIN, err := EncryptPIN(pin, pinToken, sid, sessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(map[string]interface{}{
		"asset_id":     in.AssetId,
		"opponent_key": in.OpponentKey,
		"amount":       in.Amount.Persist(),
		"trace_id":     in.TraceId,
		"memo":         in.Memo,
		"pin":          encryptedPIN,
	})
	if err != nil {
		return nil, err
	}

	path := "/transactions"
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", path, string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", path, data, token)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Error Error          `json:"error"`
		Data  RawTransaction `json:"data"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return &resp.Data, nil
}

func CreateTransfer(ctx context.Context, in *TransferInput, uid, sid, sessionKey, pin, pinToken string) (*Snapshot, error) {
	if in.Amount.Exhausted() {
		return nil, fmt.Errorf("amount exhausted")
	}

	encryptedPIN, err := EncryptPIN(pin, pinToken, sid, sessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(map[string]interface{}{
		"asset_id":    in.AssetId,
		"opponent_id": in.RecipientId,
		"amount":      in.Amount.Persist(),
		"trace_id":    in.TraceId,
		"memo":        in.Memo,
		"pin":         encryptedPIN,
	})
	if err != nil {
		return nil, err
	}

	path := "/transfers"
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", path, string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", path, data, token)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  *Snapshot `json:"data"`
		Error Error     `json:"error"`
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

func ReadTransferByTrace(ctx context.Context, traceId, uid, sid, sessionKey string) (*Snapshot, error) {
	path := "/transfers/trace/" + traceId
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "GET", path, "")
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", path, nil, token)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  *Snapshot `json:"data"`
		Error Error     `json:"error"`
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
