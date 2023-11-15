package bot

import (
	"context"
)

type Output struct {
	TransactionHash string `json:"transaction_hash"`
	OutputIndex     uint   `json:"output_index"`
	Amount          string `json:"amount"`
}

func ListUnspentOutputs(ctx context.Context, membersHash string, threshold byte, assetId string, u *SafeUser) ([]*Output, error) {
	panic(0)
}
