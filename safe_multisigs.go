package bot

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
)

type OutputReceiverView struct {
	Members     []string `json:"members"`
	MembersHash string   `json:"members_hash"`
	Threshold   int      `json:"threshold"`

	Destination    string `json:"destination"`
	Tag            string `json:"tag"`
	WithdrawalHash string `json:"withdrawal_hash"`
}

type SafeMultisigRequest struct {
	Type             string    `json:"type"`
	RequestID        string    `json:"request_id"`
	TransactionHash  string    `json:"transaction_hash"`
	AssetId          string    `json:"asset_id"`
	KernelAssetID    string    `json:"kernel_asset_id"`
	Amount           string    `json:"amount"`
	SendersHash      string    `json:"senders_hash"`
	SendersThreshold int64     `json:"senders_threshold"`
	Senders          []string  `json:"senders"`
	Signers          []string  `json:"signers"`
	RevokedBy        string    `json:"revoked_by"`
	Extra            string    `json:"extra"`
	RawTransaction   string    `json:"raw_transaction"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	InscriptionHash string                `json:"inscription_hash,omitempty"`
	Receivers       []*OutputReceiverView `json:"receivers,omitempty"`
	Views           []string              `json:"views,omitempty"`
}

func CreateSafeMultisigRequest(ctx context.Context, request []*KernelTransactionRequestCreateRequest, user *SafeUser) ([]*SafeMultisigRequest, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	endpoint := "/safe/multisigs"
	token, err := SignAuthenticationToken("POST", endpoint, string(data), user)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", endpoint, data, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  []*SafeMultisigRequest `json:"data"`
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

func FetchSafeMultisigRequest(ctx context.Context, idOrHash string, user *SafeUser) (*SafeMultisigRequest, error) {
	endpoint := "/safe/multisigs/" + idOrHash
	token, err := SignAuthenticationToken("GET", endpoint, "", user)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", endpoint, nil, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  SafeMultisigRequest `json:"data"`
		Error Error               `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return &resp.Data, nil
}

func CreateMultisigRawTx(ctx context.Context, asset crypto.Hash, senders, receivers []string, threshold byte, inputs []*common.UTXO, amount common.Integer, traceId, extra string, su *SafeUser) (string, error) {
	out := &GhostKeyRequest{
		Receivers: receivers,
		Index:     0,
		Hint:      UniqueObjectId(traceId, "OUTPUT", "0"),
	}
	change := &GhostKeyRequest{
		Receivers: senders,
		Index:     1,
		Hint:      UniqueObjectId(traceId, "OUTPUT", "1"),
	}
	ghostKeys, err := RequestSafeGhostKeys(ctx, []*GhostKeyRequest{out, change}, su)
	if err != nil {
		return "", err
	}

	var receiverKeys []*crypto.Key
	for _, key := range ghostKeys[0].Keys {
		k, _ := crypto.KeyFromString(key)
		receiverKeys = append(receiverKeys, &k)
	}
	receiverMask, _ := crypto.KeyFromString(ghostKeys[0].Mask)
	var changeKeys []*crypto.Key
	for _, key := range ghostKeys[1].Keys {
		k, _ := crypto.KeyFromString(key)
		changeKeys = append(changeKeys, &k)
	}
	changeMask, _ := crypto.KeyFromString(ghostKeys[1].Mask)

	var total common.Integer
	tx := common.NewTransactionV5(asset)
	for _, in := range inputs {
		tx.AddInput(in.Hash, in.Index)
		total = total.Add(in.Amount)
	}
	if total.Cmp(amount) < 0 {
		return "", errors.New("insufficient funds")
	}
	if !receiverMask.CheckKey() {
		return "", errors.New("invalid receiver mask")
	}
	if !changeMask.CheckKey() {
		return "", errors.New("invalid change mask")
	}
	output := &common.Output{
		Type:   common.OutputTypeScript,
		Amount: amount,
		Keys:   receiverKeys,
		Mask:   receiverMask,
		Script: common.NewThresholdScript(1),
	}
	tx.Outputs = append(tx.Outputs, output)

	if total.Cmp(amount) > 0 {
		change := total.Sub(amount)
		out := &common.Output{
			Type:   common.OutputTypeScript,
			Amount: change,
			Script: common.NewThresholdScript(uint8(threshold)),
			Mask:   changeMask,
			Keys:   changeKeys,
		}
		tx.Outputs = append(tx.Outputs, out)
	}

	if extra != "" {
		extraBytes := []byte(extra)
		if len(extraBytes) > 512 {
			return "", errors.New("extra data is too long")
		}
		tx.Extra = extraBytes
	}

	ver := tx.AsVersioned()
	return hex.EncodeToString(ver.Marshal()), nil
}
