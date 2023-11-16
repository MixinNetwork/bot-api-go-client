package bot

import (
	"context"
	"encoding/json"
	"fmt"
)

type Output struct {
	TransactionHash string `json:"transaction_hash"`
	OutputIndex     uint   `json:"output_index"`
	Asset           string `json:"asset"`
	Amount          string `json:"amount"`
}

func ListUnspentOutputs(ctx context.Context, membersHash string, threshold byte, assetId string, u *SafeUser) ([]*Output, error) {
	method, path := "GET", fmt.Sprintf("/safe/outputs?members=%s&threshold=%d&asset=%s&state=unspent", membersHash, threshold, assetId)
	token, err := SignAuthenticationToken(u.UserId, u.SessionId, u.SessionKey, method, path, "")
	body, err := Request(ctx, method, path, []byte{}, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*Output `json:"data"`
		Error Error     `json:"error"`
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
