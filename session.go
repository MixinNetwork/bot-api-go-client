package bot

import (
	"context"
	"encoding/json"
)

type UserSession struct {
	UserId    string `json:"user_id"`
	SessionId string `json:"session_id"`
	PublicKey string `json:"public_key"`
	Platform  string `json:"platform"`
}

func FetchUserSession(ctx context.Context, users []string, su *SafeUser) ([]*UserSession, error) {
	data, err := json.Marshal(users)
	if err != nil {
		return nil, err
	}
	method, path := "POST", "/sessions/fetch"
	token, err := SignAuthenticationToken(method, path, string(data), su)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", path, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*UserSession `json:"data"`
		Error Error          `json:"error"`
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
