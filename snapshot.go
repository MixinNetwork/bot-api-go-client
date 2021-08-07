package bot

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

type Snapshot struct {
	Type           string    `json:"type"`
	SnapshotId     string    `json:"snapshot_id"`
	AssetId        string    `json:"asset_id"`
	Amount         string    `json:"amount"`
	OpeningBalance string    `json:"opening_balance"`
	ClosingBalance string    `json:"closing_balance"`
	CreatedAt      time.Time `json:"created_at"`
	// deposit &  withdrawal
	TransactionHash string `json:"transaction_hash,omitempty"`
	OutputIndex     int64  `json:"output_index,omitempty"`
	Sender          string `json:"sender,omitempty"`
	Receiver        string `json:"receiver,omitempty"`
	// transfer
	SnapshotHash  string    `json:"snapshot_hash,omitempty"`
	SnapshotAt    time.Time `json:"snapshot_at,omitempty"`
	OpponentId    string    `json:"opponent_id,omitempty"`
	TraceId       string    `json:"trace_id,omitempty"`
	Memo          string    `json:"memo,omitempty"`
	Confirmations int64     `json:"confirmations,omitempty"`
	State         string    `json:"state,omitempty"`
	Fee           struct {
		Amount  string `json:"amount"`
		AssetId string `json:"asset_id"`
	} `json:"fee,omitempty"`
}

type SnapshotShort struct {
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
	CreatedAt  time.Time `json:"created_at"`
	TraceId    string    `json:"trace_id"`
	OpponentId string    `json:"opponent_id"`
	Memo       string    `json:"data"`
}

func Snapshots(ctx context.Context, limit int, offset, assetId, order, uid, sid, sessionKey string) ([]*Snapshot, error) {
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
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "GET", path, "")
	if err != nil {
		return nil, err
	}
	return SnapshotsByToken(ctx, limit, offset, assetId, order, token)
}

func SnapshotsByToken(ctx context.Context, limit int, offset, assetId, order, accessToken string) ([]*Snapshot, error) {
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
		Data  []*Snapshot `json:"data"`
		Error Error       `json:"error"`
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

func SnapshotById(ctx context.Context, snapshotId string, uid, sid, sessionKey string) (*Snapshot, error) {
	path := "/snapshots/" + snapshotId
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "GET", path, "")
	if err != nil {
		return nil, err
	}
	return SnapshotByToken(ctx, snapshotId, token)
}

func SnapshotByToken(ctx context.Context, snapshotId string, accessToken string) (*Snapshot, error) {
	path := "/snapshots/" + snapshotId
	body, err := Request(ctx, "GET", path, nil, accessToken)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  *Snapshot `json:"data"`
		Error Error     `json:"error"`
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

func NetworkSnapshot(ctx context.Context, snapshotId string) (*Snapshot, error) {
	return NetworkSnapshotByToken(ctx, snapshotId, "")
}

func NetworkSnapshotByToken(ctx context.Context, snapshotId, accessToken string) (*Snapshot, error) {
	path := "/network/snapshots/" + snapshotId
	body, err := Request(ctx, "GET", path, nil, accessToken)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  *Snapshot `json:"data"`
		Error Error     `json:"error"`
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

func NetworkSnapshots(ctx context.Context, limit int, offset, assetId, order string) ([]*SnapshotShort, error) {
	return NetworkSnapshotsByToken(ctx, limit, offset, assetId, order, "", "", "")
}

func NetworkSnapshotsByToken(ctx context.Context, limit int, offset, assetId, order, uid, sid, sessionKey string) ([]*SnapshotShort, error) {
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
	accessToken, err := SignAuthenticationToken(uid, sid, sessionKey, "GET", path, "")
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", path, nil, accessToken)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  []*SnapshotShort `json:"data"`
		Error Error            `json:"error"`
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
