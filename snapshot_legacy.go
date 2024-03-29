package bot

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

type LegacySnapshot struct {
	Type            string    `json:"type"`
	SnapshotId      string    `json:"snapshot_id"`
	AssetId         string    `json:"asset_id"`
	Amount          string    `json:"amount"`
	OpeningBalance  string    `json:"opening_balance"`
	ClosingBalance  string    `json:"closing_balance"`
	TransactionHash string    `json:"transaction_hash,omitempty"`
	SnapshotHash    string    `json:"snapshot_hash,omitempty"`
	SnapshotAt      time.Time `json:"snapshot_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`

	// deposit &  withdrawal
	OutputIndex int64  `json:"output_index,omitempty"` // deposit
	Sender      string `json:"sender,omitempty"`       // deposit
	OpponentId  string `json:"opponent_id,omitempty"`  // transfer
	TraceId     string `json:"trace_id,omitempty"`     // transfer & raw & withdrawal
	Memo        string `json:"memo,omitempty"`         // transfer & raw & withdrawal

	OpponentKey               string   `json:"opponent_key"`       // raw
	OpponentMultisigReceivers []string `json:"opponent_receivers"` // raw
	OpponentMultisigThreshold int64    `json:"opponent_threshold"` // raw
	State                     string   `json:"state"`              // raw & withdrawal
	// withdrawal
	Receiver      string `json:"receiver,omitempty"`
	Confirmations int64  `json:"confirmations,omitempty"`
	Fee           struct {
		Amount  string `json:"amount"`
		AssetId string `json:"asset_id"`
	} `json:"fee,omitempty"`
}

type LegacySnapshotShort struct {
	Type       string `json:"type"`
	SnapshotId string `json:"snapshot_id"`
	Source     string `json:"source"`
	Amount     string `json:"amount"`
	Asset      struct {
		Type     string `json:"type"`
		AssetId  string `json:"asset_id"`
		ChainId  string `json:"chain_id"`
		MixinId  string `json:"mixin_id"`
		Symbol   string `json:"symbol"`
		Name     string `json:"name"`
		AssetKey string `json:"asset_key"`
		IconUrl  string `json:"icon_url"`
	} `json:"asset"`
	State        string    `json:"state"`
	SnapshotHash string    `json:"snapshot_hash"`
	CreatedAt    time.Time `json:"created_at"`
	TraceId      string    `json:"trace_id"`
	OpponentId   string    `json:"opponent_id"`
	Memo         string    `json:"data"`
}

func Snapshots(ctx context.Context, limit int, offset, assetId, order, uid, sid, sessionKey string) ([]*LegacySnapshot, error) {
	v := url.Values{}
	v.Set("limit", strconv.Itoa(limit))
	if offset != "" {
		v.Set("offset", offset)
	}
	if assetId != "" {
		v.Set("asset", assetId)
	}
	if order != "" {
		v.Set("order", order)
	}

	path := "/snapshots?" + v.Encode()
	su := &SafeUser{
		UserId:            uid,
		SessionId:         sid,
		SessionPrivateKey: sessionKey,
	}
	token, err := SignAuthenticationToken("GET", path, "", su)
	if err != nil {
		return nil, err
	}
	return SnapshotsByToken(ctx, limit, offset, assetId, order, token)
}

func SnapshotsByToken(ctx context.Context, limit int, offset, assetId, order, accessToken string) ([]*LegacySnapshot, error) {
	v := url.Values{}
	v.Set("limit", strconv.Itoa(limit))
	if offset != "" {
		v.Set("offset", offset)
	}
	if assetId != "" {
		v.Set("asset", assetId)
	}
	if order != "" {
		v.Set("order", order)
	}

	path := "/snapshots?" + v.Encode()
	body, err := Request(ctx, "GET", path, nil, accessToken)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  []*LegacySnapshot `json:"data"`
		Error Error             `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func SnapshotById(ctx context.Context, snapshotId string, uid, sid, sessionKey string) (*LegacySnapshot, error) {
	su := &SafeUser{
		UserId:            uid,
		SessionId:         sid,
		SessionPrivateKey: sessionKey,
	}
	path := "/snapshots/" + snapshotId
	token, err := SignAuthenticationToken("GET", path, "", su)
	if err != nil {
		return nil, err
	}
	return SnapshotByToken(ctx, snapshotId, token)
}

func SnapshotByTraceId(ctx context.Context, traceId string, uid, sid, sessionKey string) (*LegacySnapshot, error) {
	su := &SafeUser{
		UserId:            uid,
		SessionId:         sid,
		SessionPrivateKey: sessionKey,
	}
	path := "/snapshots/trace/" + traceId
	token, err := SignAuthenticationToken("GET", path, "", su)
	if err != nil {
		return nil, err
	}

	body, err := Request(ctx, "GET", path, nil, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  *LegacySnapshot `json:"data"`
		Error Error           `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func SnapshotByToken(ctx context.Context, snapshotId string, accessToken string) (*LegacySnapshot, error) {
	path := "/snapshots/" + snapshotId
	body, err := Request(ctx, "GET", path, nil, accessToken)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  *LegacySnapshot `json:"data"`
		Error Error           `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func NetworkSnapshot(ctx context.Context, snapshotId string) (*LegacySnapshot, error) {
	return NetworkSnapshotByToken(ctx, snapshotId, "")
}

func NetworkSnapshotByToken(ctx context.Context, snapshotId, accessToken string) (*LegacySnapshot, error) {
	path := "/network/snapshots/" + snapshotId
	body, err := Request(ctx, "GET", path, nil, accessToken)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  *LegacySnapshot `json:"data"`
		Error Error           `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func NetworkSnapshots(ctx context.Context, limit int, offset, assetId, order string) ([]*LegacySnapshotShort, error) {
	return NetworkSnapshotsByToken(ctx, limit, offset, assetId, order, "", "", "")
}

func NetworkSnapshotsByToken(ctx context.Context, limit int, offset, assetId, order, uid, sid, sessionKey string) ([]*LegacySnapshotShort, error) {
	v := url.Values{}
	v.Set("limit", strconv.Itoa(limit))
	if offset != "" {
		v.Set("offset", offset)
	}
	if assetId != "" {
		v.Set("asset", assetId)
	}
	if order == "ASC" || order == "DESC" {
		v.Set("order", order)
	}

	path := "/network/snapshots?" + v.Encode()
	accessToken := ""
	if sessionKey != "" {
		var err error
		su := &SafeUser{
			UserId:            uid,
			SessionId:         sid,
			SessionPrivateKey: sessionKey,
		}
		accessToken, err = SignAuthenticationToken("GET", path, "", su)
		if err != nil {
			return nil, err
		}
	}
	body, err := Request(ctx, "GET", path, nil, accessToken)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  []*LegacySnapshotShort `json:"data"`
		Error Error                  `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}
