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
	SnapshotAt      time.Time `json:"snapshot_at"`
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
	} `json:"fee"`
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

func Snapshots(ctx context.Context, limit int, offset, assetId, order string, su *SafeUser) ([]*LegacySnapshot, error) {
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
	body, err := doSignedRequest(ctx, "GET", path, nil, su)
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

func SnapshotById(ctx context.Context, snapshotId string, su *SafeUser) (*LegacySnapshot, error) {
	return readLegacySnapshot(doSignedRequest(ctx, "GET", "/snapshots/"+snapshotId, nil, su))
}

func SnapshotByTraceId(ctx context.Context, traceId string, su *SafeUser) (*LegacySnapshot, error) {
	return readLegacySnapshot(doSignedRequest(ctx, "GET", "/snapshots/trace/"+traceId, nil, su))
}

func NetworkSnapshot(ctx context.Context, snapshotId string, su *SafeUser) (*LegacySnapshot, error) {
	return readLegacySnapshot(RequestWithUser(ctx, "GET", "/network/snapshots/"+snapshotId, nil, su))
}

func readLegacySnapshot(body []byte, err error) (*LegacySnapshot, error) {
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

func NetworkSnapshots(ctx context.Context, limit int, offset, assetId, order string, su *SafeUser) ([]*LegacySnapshotShort, error) {
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
	body, err := RequestWithUser(ctx, "GET", path, nil, su)
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
