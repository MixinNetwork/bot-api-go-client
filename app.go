package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type App struct {
	Type             string    `json:"type"`
	AppId            string    `json:"app_id"`
	AppNumber        string    `json:"app_number"`
	RedirectURI      string    `json:"redirect_uri"`
	HomeURI          string    `json:"home_uri"`
	Name             string    `json:"name"`
	IconURL          string    `json:"icon_url"`
	Description      string    `json:"description"`
	Capabilities     []string  `json:"capabilities"`
	ResourcePatterns []string  `json:"resource_patterns"`
	Category         string    `json:"category"`
	CreatorId        string    `json:"creator_id"`
	UpdatedAt        time.Time `json:"updated_at"`
	IsVerified       bool      `json:"is_verified"`
}

func Migrate(ctx context.Context, receiver string, user *SafeUser) (*App, error) {
	tipBody := TipBodyForOwnershipTransfer(receiver)
	pin, err := signTipBody(tipBody, user.SpendPrivateKey)
	if err != nil {
		return nil, err
	}
	encryptedPIN, err := EncryptEd25519PIN(pin, uint64(time.Now().UnixNano()), user)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(map[string]string{
		"user_id":    receiver,
		"pin_base64": encryptedPIN,
	})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/apps/%s/transfer", uid)
	token, err := SignAuthenticationToken("POST", path, string(data), user)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", path, data, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  *App  `json:"data"`
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
