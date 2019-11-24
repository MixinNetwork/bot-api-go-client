package bot

import "time"

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
	CreatedAt       time.Time `json:"created_at"`
}
