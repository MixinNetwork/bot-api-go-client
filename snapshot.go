package bot

import (
	"context"
	"encoding/json"
	"net/url"
	"time"
)

type Snapshot struct {
	Type            string    `json:"type"`
	SnapshotId      string    `json:"snapshot_id"`
	Receiver        string    `json:"receiver"`
	TransactionHash string    `json:"transaction_hash"`
	AssetId         string    `json:"asset_id"`
	Amount          string    `json:"amount"`
	OpeningBalance  string    `json:"opening_balance"`
	ClosingBalance  string    `json:"closing_balance"`
	TraceId         string    `json:"trace_id"`
	Memo            string    `json:"memo"`
	Confirmations   int64     `json:"confirmations,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	Fee             struct {
		Amount  string `json:"amount"`
		AssetId string `json:"asset_id"`
	} `json:"fee,omitempty"`
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

func NetworkSnapshots(ctx context.Context, limit int, offset, assetId, order string) ([]*Snapshot, error) {
	return NetworkSnapshotsByToken(ctx, limit, offset, assetId, order, "")
}
func NetworkSnapshotsByToken(ctx context.Context, limit int, offset, assetId, order, accessToken string) ([]*Snapshot, error) {
	v := url.Values{}
	v.Set("limit", string(limit))
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
