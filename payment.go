package bot

import (
	"context"
	"encoding/json"
	"time"
)

type Payment struct {
	Type      string    `json:"type"`
	TraceId   string    `json:"trace_id"`
	AssetId   string    `json:"asset_id"`
	Amount    string    `json:"amount"`
	Threshold int64     `json:"threshold"`
	Receivers []string  `json:"receivers"`
	Memo      string    `json:"memo"`
	Status    string    `json:"status"`
	CodeId    string    `json:"code_id"`
	CreatedAt time.Time `json:"created_at"`
}

type PaymentRequest struct {
	AssetId          string `json:"asset_id"`
	Amount           string `json:"amount"`
	TraceId          string `json:"trace_id"`
	Memo             string `json:"memo"`
	OpponentMultisig struct {
		Receivers []string `json:"receivers"`
		Threshold int64    `json:"threshold"`
	} `json:"opponent_multisig"`
}

func CreatePaymentRequest(ctx context.Context, payment *PaymentRequest, uid, sid, sessionKey string) (*Payment, error) {
	data, err := json.Marshal(payment)
	if err != nil {
		return nil, err
	}
	method, path := "POST", "/payments"
	token, err := SignAuthenticationToken(uid, sid, sessionKey, method, path, string(data))
	if err != nil {
		return nil, err
	}

	body, err := Request(ctx, method, path, data, token, UuidNewV4().String())
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *Payment `json:"data"`
		Error Error    `json:"error"`
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

func ReadPaymentByCode(ctx context.Context, codeId string) (*Payment, error) {
	body, err := Request(ctx, "GET", "/codes/"+codeId, nil, "", UuidNewV4().String())
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *Payment `json:"data"`
		Error Error    `json:"error"`
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

func CreateRaw(ctx context.Context) {
}
