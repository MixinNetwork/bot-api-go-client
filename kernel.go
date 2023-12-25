package bot

import (
	"context"
	"encoding/json"
)

func CallKernelRPC(ctx context.Context, user *SafeUser, method string, params ...interface{}) ([]byte, error) {
	p := map[string]interface{}{
		"method": method,
		"params": params,
	}
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	token, err := SignAuthenticationToken("POST", "/external/kernel", string(data), user)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", "/external/kernel", data, token)
	if err != nil {
		return nil, err
	}
	return body, nil
}
