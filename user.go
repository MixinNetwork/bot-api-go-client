package bot

import (
	"context"
	"encoding/json"
)

type User struct {
	UserId         string `json:"user_id"`
	IdentityNumber string `json:"identity_number"`
	FullName       string `json:"full_name"`
	AvatarURL      string `json:"avatar_url"`
	CreatedAt      string `json:"created_at"`
}

func UserMe(ctx context.Context, accessToken string) (User, error) {
	body, err := Request(ctx, "GET", "/me", nil, accessToken)
	if err != nil {
		return User{}, ServerError(ctx, err)
	}
	var resp struct {
		Data  User  `json:"data"`
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return User{}, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		if resp.Error.Code == 401 {
			return User{}, AuthorizationError(ctx)
		}
		return User{}, ServerError(ctx, resp.Error)
	}
	return resp.Data, nil
}
