package bot

import (
	"context"
	"encoding/json"
)

type Fiat struct {
	Code string  `json:"code"`
	Rate float64 `json:"rate"`
}

func Fiats(ctx context.Context, uid, sid, sessionKey string) ([]*Fiat, error) {
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "GET", "/fiats", "")
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", "/fiats", nil, token)
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
