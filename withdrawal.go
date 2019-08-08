package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MixinNetwork/go-number"
)

type Withdrawal struct {
	Type            string    `json:"type"`
	SnapshotId      string    `json:"snapshot_id"`
	Receiver        string    `json:"receiver"`
	TransactionHash string    `json:"transaction_hash"`
	AssetId         string    `json:"asset_id"`
	Amount          string    `json:"amount"`
	TraceId         string    `json:"trace_id"`
	Memo            string    `json:"memo"`
	CreatedAt       time.Time `json:"created_at"`
}

type WithdrawalInput struct {
	AddressId string
	Amount    number.Decimal
	TraceId   string
	Memo      string
}

func CreateWithdrawal(ctx context.Context, in *WithdrawalInput, uid, sid, sessionKey, pin, pinToken string) (*Withdrawal, error) {
	if in.Amount.Exhausted() {
		return nil, fmt.Errorf("Acmount negative")
	}

	encryptedPIN, err := EncryptPIN(ctx, pin, pinToken, sid, sessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(map[string]interface{}{
		"address_id": in.AddressId,
		"amount":     in.Amount.Persist(),
		"trace_id":   in.TraceId,
		"memo":       in.Memo,
		"pin":        encryptedPIN,
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
		Error Error      `json:"error"`
		Data  Withdrawal `json:"data,omitempty"`
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
