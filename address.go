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
	Dust        string `json:"dust"`
	UpdatedAt   string `json:"updated_at"`
}

func CreateAddress(ctx context.Context, in *AddressInput, user *SafeUser) (*Address, error) {
	tipBody := TipBodyForAddressAdd(in.AssetId, in.Destination, in.Tag, in.Label)
	var err error
	pin, err := signTipBody(tipBody, user.SpendPrivateKey)
	if err != nil {
		return nil, err
	}
	encryptedPIN, err := EncryptEd25519PIN(pin, uint64(time.Now().UnixNano()), user)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(map[string]interface{}{
		"asset_id":    in.AssetId,
		"label":       in.Label,
		"destination": in.Destination,
		"tag":         in.Tag,
		"pin_base64":  encryptedPIN,
	})
	if err != nil {
		return nil, err
	}

	token, err := SignAuthenticationToken("POST", "/addresses", string(data), user)
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

func ReadAddress(ctx context.Context, addressId, user *SafeUser) (*Address, error) {
	endpoint := fmt.Sprintf("/addresses/%s", addressId)
	token, err := SignAuthenticationToken("GET", endpoint, "", user)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", endpoint, nil, token)
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

func DeleteAddress(ctx context.Context, addressId string, user *SafeUser) error {
	tipBody := TipBody(TIPAddressRemove + addressId)
	pin, err := signTipBody(tipBody, user.SpendPrivateKey)
	if err != nil {
		return err
	}
	encryptedPIN, err := EncryptEd25519PIN(pin, uint64(time.Now().UnixNano()), user)
	if err != nil {
		return err
	}
	data, err := json.Marshal(map[string]interface{}{
		"pin_base64": encryptedPIN,
	})
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("/addresses/%s/delete", addressId)
	token, err := SignAuthenticationToken("POST", endpoint, string(data), user)
	if err != nil {
		return err
	}
	body, err := Request(ctx, "POST", endpoint, data, token)
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

func GetAddressesByAssetId(ctx context.Context, assetId string, user *SafeUser) ([]*Address, error) {
	endpoint := fmt.Sprintf("/assets/%s/addresses", assetId)
	token, err := SignAuthenticationToken("GET", endpoint, "", user)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", endpoint, nil, token)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  []*Address `json:"data"`
		Error *Error     `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp.Data, nil
}
