package bot

import (
	"context"
	"encoding/json"
	"time"
)

type AddressInput struct {
	AssetId     string
	Label       string
	PublicKey   string
	AccountName string
	AccountTag  string
}

type Address struct {
	AddressId   string `json:"address_id"`
	AssetId     string `json:"asset_id"`
	Label       string `json:"label"`
	PublicKey   string `json:"public_key,omitempty"`
	AccountName string `json:"account_name,omitempty"`
	AccountTag  string `json:"account_tag,omitempty"`
	Fee         string `json:"fee"`
	Reserve     string `json:"reserve"`
	UpdatedAt   string `json:"updated_at"`
}

func CreateAddress(ctx context.Context, in *AddressInput, uid, sid, sessionKey, pin, pinToken string) (*Address, error) {
	encryptedPIN, err := EncryptPIN(ctx, pin, pinToken, sid, sessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(map[string]interface{}{
		"asset_id":     in.AssetId,
		"label":        in.Label,
		"public_key":   in.PublicKey,
		"account_name": in.AccountName,
		"account_tag":  in.AccountTag,
		"pin":          encryptedPIN,
	})
	if err != nil {
		return nil, err
	}

	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", "/addresses", string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", "/addresses", data, token)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  *Address `json:"data"`
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
