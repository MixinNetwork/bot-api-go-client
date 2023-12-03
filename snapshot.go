package bot

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

type SafeDepositView struct {
	DepositHash  string `json:"deposit_hash"`
	DepositIndex int64  `json:"deposit_index"`
	Sender       string `json:"sender"`
}

type SafeWithdrawalView struct {
	WithdrawalHash string `json:"withdrawal_hash"`
	Receiver       string `json:"receiver"`
}

type SafeSnapshot struct {
	Type            string    `json:"type"`
	SnapshotID      string    `json:"snapshot_id"`
	UserID          string    `json:"user_id"`
	OpponentID      string    `json:"opponent_id"`
	TransactionHash string    `json:"transaction_hash"`
	AssetID         string    `json:"asset_id"`
	KernelAssetID   string    `json:"kernel_asset_id"`
	Amount          string    `json:"amount"`
	Memo            string    `json:"memo"`
	CreatedAt       time.Time `json:"created_at"`

	Deposit    *SafeDepositView    `json:"deposit,omitempty"`
	Withdrawal *SafeWithdrawalView `json:"withdrawal,omitempty"`
}

func SafeSnapshots(ctx context.Context, limit int, app, assetId, opponent, offset, uid, sid, sessionKey string) ([]*SafeSnapshot, error) {
	v := url.Values{}
	v.Set("limit", strconv.Itoa(limit))
	if app != "" {
		v.Set("app", app)
	}
	if assetId != "" {
		v.Set("asset", assetId)
	}
	if offset != "" {
		v.Set("offset", offset)
	}
	if opponent != "" {
		v.Set("opponent", opponent)
	}
	path := "/safe/snapshots?" + v.Encode()
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "GET", path, "")
	if err != nil {
		return nil, err
	}
	return SafeSnapshotsByToken(ctx, limit, app, assetId, opponent, offset, token)
}

func SafeSnapshotsByToken(ctx context.Context, limit int, app, assetId, opponent, offset, accessToken string) ([]*SafeSnapshot, error) {
	v := url.Values{}
	v.Set("limit", strconv.Itoa(limit))
	if app != "" {
		v.Set("app", app)
	}
	if assetId != "" {
		v.Set("asset", assetId)
	}
	if offset != "" {
		v.Set("offset", offset)
	}
	if opponent != "" {
		v.Set("opponent", opponent)
	}
	path := "/safe/snapshots?" + v.Encode()
	body, err := Request(ctx, "GET", path, nil, accessToken)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  []*SafeSnapshot `json:"data"`
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

func SafeSnapshotById(ctx context.Context, snapshotId string, uid, sid, sessionKey string) (*SafeSnapshot, error) {
	path := "/safe/snapshots/" + snapshotId
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "GET", path, "")
	if err != nil {
		return nil, err
	}
	return SafeSnapshotByToken(ctx, snapshotId, token)
}

func SafeSnapshotByToken(ctx context.Context, snapshotId string, accessToken string) (*SafeSnapshot, error) {
	path := "/safe/snapshots/" + snapshotId
	body, err := Request(ctx, "GET", path, nil, accessToken)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  *SafeSnapshot `json:"data"`
		Error Error         `json:"error"`
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
