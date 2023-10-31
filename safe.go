package bot

import (
	"context"
	"encoding/json"
	"net/url"
)

type GhostKeys struct {
	Type string   `json:"type"`
	Mask string   `json:"mask"`
	Keys []string `json:"keys"`
}

type GhostKeyRequest struct {
	Receivers []string `json:"receivers"`
	Index     int      `json:"index"`
	Hint      string   `json:"hint"`
}

func RequestSafeGhostKeys(ctx context.Context, gkr []*GhostKeyRequest, uid, sid, sessionKey string) ([]*GhostKeys, error) {
	data, err := json.Marshal(gkr)
	if err != nil {
		return nil, err
	}
	method, path := "POST", "/safe/keys"
	token, err := SignAuthenticationToken(uid, sid, sessionKey, method, path, string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*GhostKeys `json:"data"`
		Error Error        `json:"error"`
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

type SafeExternalAddress struct {
	ChainId     string `json:"chain_id"`
	Destination string `json:"destination"`
}

func (a *SafeExternalAddress) IsExternalAddress() bool {
	return a.ChainId == ""
}

func SafeExternalAdddressCheck(ctx context.Context, asset, destination, tag string) (*SafeExternalAddress, error) {
	values := url.Values{}
	if destination != "" {
		values.Add("destination", destination)
	}
	if tag != "" {
		values.Add("tag", tag)
	}
	if asset != "" {
		values.Add("asset", asset)
	}

	endpoint := "/safe/external/addresses/check?" + values.Encode()
	body, err := Request(ctx, "GET", endpoint, nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  *SafeExternalAddress `json:"data"`
		Error *Error               `json:"error"`
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
