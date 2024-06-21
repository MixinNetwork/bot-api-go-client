package bot

import (
	"context"
	"encoding/json"
	"fmt"
)

type WithdrawalFee struct {
	Type    string `json:"type"`
	AssetId string `json:"asset_id"`
	Amount  string `json:"amount"`
}

func RequestWithdrawalFees(ctx context.Context, asset string, user *SafeUser) ([]*WithdrawalFee, error) {
	method, path := "GET", fmt.Sprintf("/safe/assets/%s/fees", asset)
	token, err := SignAuthenticationToken(method, path, "", user)
	if err != nil {
		return nil, err
	}

	body, err := Request(ctx, method, path, nil, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*WithdrawalFee `json:"data"`
		Error Error            `json:"error"`
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