package bot

import (
	"context"
	"encoding/json"
)

type User struct {
	UserId         string `json:"user_id"`
	SessionId      string `json:"session_id"`
	PinToken       string `json:"pin_token"`
	IdentityNumber string `json:"identity_number"`
	Phone          string `json:"phone"`
	FullName       string `json:"full_name"`
	AvatarURL      string `json:"avatar_url"`
	DeviceStatus   string `json:"device_status"`
	CreatedAt      string `json:"created_at"`
}

func CreateUser(ctx context.Context, sessionSecret, fullName, uid, sid, sessionKey string) (*User, error) {
	data, err := json.Marshal(map[string]string{
		"session_secret": sessionSecret,
		"full_name":      fullName,
	})
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", "/users", string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", "/users", data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *User `json:"data"`
		Error Error `json:"error"`
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

func UpdatePin(ctx context.Context, oldEncryptedPin, encryptedPin, uid, sid, sessionKey string) error {
	data, err := json.Marshal(map[string]string{
		"old_pin": oldEncryptedPin,
		"pin":     encryptedPin,
	})

	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", "/pin/update", string(data))
	if err != nil {
		return err
	}
	body, err := Request(ctx, "POST", "/pin/update", data, token)
	if err != nil {
		return ServerError(ctx, err)
	}
	var resp struct {
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return ServerError(ctx, err)
	}
	if resp.Error.Code > 0 {
		return resp.Error
	}
	return nil
}

func UserMe(ctx context.Context, accessToken string) (*User, error) {
	body, err := Request(ctx, "GET", "/me", nil, accessToken)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *User `json:"data"`
		Error Error `json:"error"`
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
