package bot

import (
	"context"
	"encoding/json"
	"net/url"
)

func CallMixinRPC(ctx context.Context, uid, sid, sessionKey, method string, params ...interface{}) ([]byte, error) {
	p := map[string]interface{}{
		"method": method,
		"params": params,
	}
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", "/external/proxy", string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", "/external/proxy", data, token)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func ExternalAdddressCheck(ctx context.Context, asset, destination, tag string) (*Address, error) {
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

	endpoint := "/external/addresses/check?" + values.Encode()
	body, err := Request(ctx, "GET", endpoint, nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  *Address `json:"data"`
		Error *Error   `json:"error"`
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
