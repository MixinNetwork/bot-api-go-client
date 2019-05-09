package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MixinNetwork/go-number"
)

type TransferInput struct {
	AssetId     string
	RecipientId string
	Amount      number.Decimal
	TraceId     string
	Memo        string
	OpponentKey string
}

func CreateTransaction(ctx context.Context, in *TransferInput, uid, sid, sessionKey, pin, pinToken string) error {
	if in.Amount.Exhausted() {
		return fmt.Errorf("Amount exhausted")
	}

	encryptedPIN, err := EncryptPIN(ctx, pin, pinToken, sid, sessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return err
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
		return err
	}

	path := "/transactions"
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", path, string(data))
	if err != nil {
		return err
	}
	body, err := Request(ctx, "POST", path, data, token)
	if err != nil {
		return err
	}

	var resp struct {
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}
	if resp.Error.Code > 0 {
		return resp.Error
	}
	return nil
}

func CreateTransfer(ctx context.Context, in *TransferInput, uid, sid, sessionKey, pin, pinToken string) error {
	if in.Amount.Exhausted() {
		return fmt.Errorf("Amount exhausted")
	}

	encryptedPIN, err := EncryptPIN(ctx, pin, pinToken, sid, sessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return err
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
		return err
	}

	path := "/transfers"
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", path, string(data))
	if err != nil {
		return err
	}
	body, err := Request(ctx, "POST", path, data, token)
	if err != nil {
		return err
	}

	var resp struct {
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}
	if resp.Error.Code > 0 {
		return resp.Error
	}
	return nil
}
