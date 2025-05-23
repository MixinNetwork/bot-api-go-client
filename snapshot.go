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
	Destination  string `json:"destination"`
	Tag          string `json:"tag"`
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
	RequestId       string    `json:"request_id"`
	CreatedAt       time.Time `json:"created_at"`

	Deposit    *SafeDepositView    `json:"deposit,omitempty"`
	Withdrawal *SafeWithdrawalView `json:"withdrawal,omitempty"`
}

func SafeSnapshots(ctx context.Context, limit int, app, assetId, opponent, offset string, su *SafeUser) ([]*SafeSnapshot, error) {
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
	token, err := SignAuthenticationToken("GET", path, "", su)
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

func SafeSnapshotById(ctx context.Context, snapshotId string, su *SafeUser) (*SafeSnapshot, error) {
	path := "/safe/snapshots/" + snapshotId
	token, err := SignAuthenticationToken("GET", path, "", su)
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

type MessageWithSession struct {
	Type             string    `json:"type"`
	RepresentativeId string    `json:"representative_id"`
	QuoteMessageId   string    `json:"quote_message_id"`
	ConversationId   string    `json:"conversation_id"`
	UserId           string    `json:"user_id"`
	SessionId        string    `json:"session_id"`
	MessageId        string    `json:"message_id"`
	Category         string    `json:"category"`
	Data             string    `json:"data"`
	DataBase64       string    `json:"data_base64"`
	Status           string    `json:"status"`
	Source           string    `json:"source"`
	Silent           bool      `json:"silent"`
	ExpireIn         int64     `json:"expire_in"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func SafeNotifySnapshot(ctx context.Context, transactionHash string, outputIndex int64, receiverID string, su *SafeUser) (*MessageWithSession, error) {
	data, err := json.Marshal(map[string]any{
		"transaction_hash": transactionHash,
		"output_index":     outputIndex,
		"receiver_id":      receiverID,
	})
	if err != nil {
		return nil, err
	}
	method, path := "POST", "/safe/snapshots/notifications"
	token, err := SignAuthenticationToken(method, path, string(data), su)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, data, token)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  *MessageWithSession `json:"data"`
		Error Error               `json:"error"`
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
