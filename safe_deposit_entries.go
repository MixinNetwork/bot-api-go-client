package bot

import (
	"context"
	"encoding/json"
)

type DepositEntryView struct {
	Type          string   `json:"type"`
	EntryID       string   `json:"entry_id"`
	Threshold     int64    `json:"threshold"`
	Members       []string `json:"members"`
	Destination   string   `json:"destination"`
	Tag           string   `json:"tag"`
	SafeSignature string   `json:"signature"`
	ChainID       string   `json:"chain_id"`
	IsPrimary     bool     `json:"is_primary"`
}

func CreateDepositEntry(ctx context.Context, chainID string, members []string, threshold int64, user *SafeUser) ([]*DepositEntryView, error) {
	data, _ := json.Marshal(map[string]any{
		"chain_id":  chainID,
		"members":   members,
		"threshold": threshold,
	})
	endpoint := "/safe/deposit/entries"

	token, err := SignAuthenticationToken("POST", endpoint, string(data), user)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", endpoint, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*DepositEntryView `json:"data"`
		Error Error               `json:"error"`
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
