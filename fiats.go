package bot

import (
	"context"
	"encoding/json"
)

type Fiat struct {
	Code string  `json:"code"`
	Rate float64 `json:"rate"`
}

func GetFiats(ctx context.Context) ([]*Fiat, error) {
	body, err := SimpleRequest(ctx, "GET", "/external/fiats", nil)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*Fiat `json:"data"`
		Error Error   `json:"error"`
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

func Fiats(ctx context.Context) ([]*Fiat, error) {
	body, err := Request(ctx, "GET", "/external/fiats", nil, "")
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*Fiat `json:"data"`
		Error Error   `json:"error"`
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
