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

func GetTurnServer(ctx context.Context, su *SafeUser) ([]*Turn, error) {
	token, err := SignAuthenticationToken("GET", "/turn", "", su)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", "/turn", nil, token)
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
