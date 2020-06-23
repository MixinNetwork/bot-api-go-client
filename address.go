package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type AddressInput struct {
	AssetId     string
	Label       string
	Destination string
	Tag         string
}

type Address struct {
	AddressId   string `json:"address_id"`
	AssetId     string `json:"asset_id"`
	Label       string `json:"label"`
	Destination string `json:"destination"`
	Tag         string `json:"tag"`
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
		"asset_id":    in.AssetId,
		"label":       in.Label,
		"destination": in.Destination,
		"tag":         in.Tag,
		"pin":         encryptedPIN,
	})
	if err != nil {
		return nil, err
	}

	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", "/addresses", string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", "/addresses", data, token, UuidNewV4().String())
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

func ReadAddress(ctx context.Context, addressId, uid, sid, sessionKey string) (*Address, error) {
	endpoint := fmt.Sprintf("/addresses/%s", addressId)
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "GET", endpoint, "")
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", endpoint, nil, token, UuidNewV4().String())
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

func DeleteAddress(ctx context.Context, addressId, uid, sid, sessionKey, pin, pinToken string) error {
	encryptedPIN, err := EncryptPIN(ctx, pin, pinToken, sid, sessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return err
	}
	data, err := json.Marshal(map[string]interface{}{
		"pin": encryptedPIN,
	})
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("/addresses/%s/delete", addressId)
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", endpoint, string(data))
	if err != nil {
		return err
	}
	body, err := Request(ctx, "POST", endpoint, data, token, UuidNewV4().String())
	if err != nil {
		return err
	}

	var resp struct {
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return resp.Error
	}
	return nil
}

func GetAddressesByAssetId(ctx context.Context, assetId, uid, sid, sessionKey string) ([]*Address, error) {
	endpoint := fmt.Sprintf("/assets/%s/addresses", assetId)
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "GET", endpoint, "")
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", endpoint, nil, token, UuidNewV4().String())
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  []*Address `json:"data"`
		Error Error      `json:"error"`
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
