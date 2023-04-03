package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MixinNetwork/go-number"
)

type WithdrawalInput struct {
	AddressId string
	Amount    number.Decimal
	Fee       string
	TraceId   string
	Memo      string

	AssetId     string
	Destination string
	Tag         string
}

func CreateWithdrawal(ctx context.Context, in *WithdrawalInput, uid, sid, sessionKey, pin, pinToken string) (*Snapshot, error) {
	if in.Amount.Exhausted() {
		return nil, fmt.Errorf("amount negative")
	}

	encryptedPIN, err := EncryptPIN(pin, pinToken, sid, sessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(map[string]interface{}{
		"address_id": in.AddressId,
		"amount":     in.Amount.Persist(),
		"trace_id":   in.TraceId,
		"memo":       in.Memo,
		"fee":        in.Fee,
		"pin":        encryptedPIN,

		"asset_id":    in.AssetId,
		"destination": in.Destination,
		"tag":         in.Tag,
	})
	if err != nil {
		return nil, err
	}

	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", "/withdrawals", string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", "/withdrawals", data, token)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Error Error    `json:"error"`
		Data  Snapshot `json:"data,omitempty"`
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
