package bot

import (
	"context"
	"encoding/json"
	"net/url"
)

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
