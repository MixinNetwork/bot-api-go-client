package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/MixinNetwork/go-number"
)

func TestTransfer(t *testing.T) {
	appId := "8dcf823d-9eb3-4da2-8734-f0aad50c0da6"
	sessionId := "0691bfa5-05ae-4956-a2e4-dce890fd192f"
	serverPublicKey := ""
	sessionPrivateKey := ""
	spend := ""

	in := &TransferInput{
		AssetId:     "965e5c6e-434c-3fa9-b780-c50f43cd955c",
		RecipientId: "e9e5b807-fa8b-455a-8dfa-b189d28310ff",
		Amount:      number.FromString("0.00021"),
		TraceId:     UuidNewV4().String(),
	}
	snapshot, err := CreateTransfer(context.Background(), in, appId, sessionId, sessionPrivateKey, spend, serverPublicKey)
	log.Println(err)
	log.Printf("%#v", snapshot)
}

type TransferInput struct {
	AssetId          string
	RecipientId      string
	Amount           number.Decimal
	TraceId          string
	Memo             string
	OpponentKey      string
	OpponentMultisig struct {
		Receivers []string
		Threshold int64
	}
}

func CreateTransfer(ctx context.Context, in *TransferInput, uid, sid, sessionKey, pin, pinToken string) (*Snapshot, error) {
	if in.Amount.Exhausted() {
		return nil, fmt.Errorf("Amount exhausted")
	}

	su := &SafeUser{
		UserId:            uid,
		SessionId:         sid,
		SessionPrivateKey: sessionKey,
		ServerPublicKey:   pinToken,
		SpendPrivateKey:   pin,
	}

	pin, err := signTipBody(TipBodyForTransfer(in.AssetId, in.RecipientId, in.Amount, in.TraceId, in.Memo), su.SpendPrivateKey)
	if err != nil {
		panic(err)
	}
	encryptedPIN, err := EncryptEd25519PIN(pin, uint64(time.Now().UnixNano()), su)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(map[string]interface{}{
		"asset_id":    in.AssetId,
		"opponent_id": in.RecipientId,
		"amount":      in.Amount.Persist(),
		"trace_id":    in.TraceId,
		"memo":        in.Memo,
		"pin":         encryptedPIN,
	})
	if err != nil {
		return nil, err
	}

	path := "/transfers"
	token, err := SignAuthenticationToken("POST", path, string(data), su)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", path, data, token)
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