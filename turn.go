package bot

import (
	"context"
	"encoding/json"
)

type Turn struct {
	Url        string `json:"url"`
	Username   string `json:"username"`
	Credential string `json:"credential"`
}

func GetTurnServer(ctx context.Context, uid, sid, sessionKey string) ([]*Turn, error) {
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "GET", "/turn", "")
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", "/turn", nil, token, UuidNewV4().String())
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*Turn `json:"data"`
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
