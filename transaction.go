package bot

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"filippo.io/edwards25519"
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/gofrs/uuid/v5"
)

type TransactionRecipient struct {
	MixAddress string
	Amount     string

	Destination string
	Tag         string
}

type SequencerTransactionRequest struct {
	RequestID       string    `json:"request_id"`
	TransactionHash string    `json:"transaction_hash"`
	Asset           string    `json:"asset"`
	Amount          string    `json:"amount"`
	Extra           string    `json:"extra"`
	State           string    `json:"state"`
	RawTransaction  string    `json:"raw_transaction"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	SnapshotID      string    `json:"snapshot_id"`
	SnapshotHash    string    `json:"snapshot_hash"`
	SnapshotAt      time.Time `json:"snapshot_at"`

	Views []string `json:"views"`
}

type UtxoError struct {
	TotalInput  common.Integer
	TotalOutput common.Integer
	OutputSize  int
}

func (ue *UtxoError) Error() string {
	return fmt.Sprintf("insufficient outputs %s@%d %s", ue.TotalInput, ue.OutputSize, ue.TotalOutput)
}

func SendTransferTransaction(ctx context.Context, assetId, receiver, amount, traceId string, extra []byte, u *SafeUser) (*SequencerTransactionRequest, error) {
	ma := NewUUIDMixAddress([]string{receiver}, 1)
	tr := &TransactionRecipient{
		MixAddress: ma.String(),
		Amount:     amount,
	}
	return SendTransaction(ctx, assetId, []*TransactionRecipient{tr}, traceId, extra, nil, u)
}

func SendTransaction(ctx context.Context, assetId string, recipients []*TransactionRecipient, traceId string, extra []byte, references []string, u *SafeUser) (*SequencerTransactionRequest, error) {
	if uuid.FromStringOrNil(assetId).String() == assetId {
		assetId = crypto.Sha256Hash([]byte(assetId)).String()
	}
	asset, err := crypto.HashFromString(assetId)
	if err != nil {
		return nil, fmt.Errorf("invalid asset id %s", assetId)
	}
	if len(references) > 2 {
		return nil, fmt.Errorf("too many references %d", len(references))
	}

	// get unspent outputs for asset and may return insufficient outputs error
	utxos, changeAmount, err := requestUnspentOutputsForRecipients(ctx, assetId, recipients, u)
	if err != nil {
		return nil, fmt.Errorf("requestUnspentOutputsForRecipients(%s) => %v", assetId, err)
	}
	// change to the sender
	if changeAmount.Sign() > 0 {
		ma := NewUUIDMixAddress([]string{u.UserId}, 1)
		recipients = append(recipients, &TransactionRecipient{
			MixAddress: ma.String(),
			Amount:     changeAmount.String(),
		})
	}

	// build the unsigned raw transaction
	tx, err := buildRawTransaction(ctx, asset, utxos, recipients, extra, references, u)
	if err != nil {
		return nil, fmt.Errorf("buildRawTransaction(%s) => %v", asset, err)
	}
	ver := tx.AsVersioned()
	// verify the raw transaction, the same trace id may have been signed already
	str, err := verifyRawTransactionBySequencer(ctx, traceId, ver, u)
	if err != nil || str.State != "unspent" {
		return str, fmt.Errorf("verifyRawTransactionBySequencer(%s) => %v", traceId, err)
	}

	// sign the raw transaction with user private spend key
	if len(str.Views) != len(ver.Inputs) {
		return nil, fmt.Errorf("invalid view keys count %d %d", len(str.Views), len(ver.Inputs))
	}
	ver, err = signRawTransaction(ctx, ver, str.Views, u.SpendPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("signRawTransaction(%v) => %v", ver, err)
	}

	// send the raw transaction to the sequencer api
	result, err := sendRawTransactionToSequencer(ctx, traceId, ver, u)
	if err != nil {
		return nil, fmt.Errorf("sendRawTransactionToSequencer(%s) => %v", traceId, err)
	}
	if hex.EncodeToString(ver.Marshal()) != result.RawTransaction {
		panic(str.RawTransaction)
	}
	return result, nil
}

func GetTransactionById(ctx context.Context, requestId string) (*SequencerTransactionRequest, error) {
	method, path := "GET", fmt.Sprintf("/safe/transactions/%s", requestId)
	body, err := Request(ctx, method, path, nil, "")
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *SequencerTransactionRequest `json:"data"`
		Error Error                        `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code == 404 {
		return nil, nil
	} else if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func buildRawTransaction(ctx context.Context, asset crypto.Hash, utxos []*Output, recipients []*TransactionRecipient, extra []byte, references []string, u *SafeUser) (*common.Transaction, error) {
	tx := common.NewTransactionV5(asset)
	for _, in := range utxos {
		h, err := crypto.HashFromString(in.TransactionHash)
		if err != nil {
			panic(in.TransactionHash)
		}
		tx.AddInput(h, in.OutputIndex)
	}
	for _, r := range references {
		rh, err := crypto.HashFromString(r)
		if err != nil {
			panic(r)
		}
		tx.References = append(tx.References, rh)
	}

	for i, r := range recipients {
		if r.Destination != "" {
			tx.Outputs = append(tx.Outputs, &common.Output{
				Type:   common.OutputTypeWithdrawalSubmit,
				Amount: common.NewIntegerFromString(r.Amount),
				Withdrawal: &common.WithdrawalData{
					Address: r.Destination,
					Tag:     r.Tag,
				},
			})
			continue
		}
		ma, err := NewMixAddressFromString(r.MixAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid mix address %s", r.MixAddress)
		}
		ghost, err := ma.RequestOrGenerateGhostKeys(ctx, uint(i), u)
		if err != nil {
			return nil, err
		}
		mask, err := crypto.KeyFromString(ghost.Mask)
		if err != nil {
			panic(ghost.Mask)
		}
		tx.Outputs = append(tx.Outputs, &common.Output{
			Type:   common.OutputTypeScript,
			Amount: common.NewIntegerFromString(r.Amount),
			Script: common.NewThresholdScript(ma.Threshold),
			Keys:   ghost.KeysSlice(),
			Mask:   mask,
		})
	}

	if l := tx.AsVersioned().GetExtraLimit(); len(tx.Extra) >= l {
		return nil, fmt.Errorf("large extra %d > %d", len(tx.Extra), l)
	}
	tx.Extra = extra
	return tx, nil
}

type KernelTransactionRequestCreateRequest struct {
	RequestID string `json:"request_id"`
	Raw       string `json:"raw"`
}

func verifyRawTransactionBySequencer(ctx context.Context, traceId string, ver *common.VersionedTransaction, u *SafeUser) (*SequencerTransactionRequest, error) {
	requests := []*KernelTransactionRequestCreateRequest{{
		RequestID: traceId,
		Raw:       hex.EncodeToString(ver.Marshal()),
	}}
	data, err := json.Marshal(requests)
	if err != nil {
		return nil, err
	}
	method, path := "POST", "/safe/transaction/requests"
	token, err := SignAuthenticationToken(method, path, string(data), u)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*SequencerTransactionRequest `json:"data"`
		Error Error                          `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	if len(resp.Data) != 1 {
		return nil, errors.New("invalid response size")
	}
	return resp.Data[0], nil
}

func signRawTransaction(ctx context.Context, ver *common.VersionedTransaction, views []string, spendKey string) (*common.VersionedTransaction, error) {
	msg := ver.PayloadHash()
	spent, err := crypto.KeyFromString(spendKey)
	if err != nil {
		return nil, err
	}
	spenty := sha512.Sum512(spent[:])
	y, err := edwards25519.NewScalar().SetBytesWithClamping(spenty[:32])
	if err != nil {
		return nil, err
	}
	signaturesMap := make([]map[uint16]*crypto.Signature, len(ver.Inputs))
	for i := range ver.Inputs {
		viewBytes, err := crypto.KeyFromString(views[i])
		if err != nil {
			return nil, err
		}
		x, err := edwards25519.NewScalar().SetCanonicalBytes(viewBytes[:])
		if err != nil {
			return nil, err
		}
		t := edwards25519.NewScalar().Add(x, y)
		var key crypto.Key
		copy(key[:], t.Bytes())
		sig := key.Sign(msg)
		sigs := make(map[uint16]*crypto.Signature)
		sigs[0] = &sig // for 1/1 bot transaction
		signaturesMap[i] = sigs
	}
	ver.SignaturesMap = signaturesMap
	return ver, nil
}

func sendRawTransactionToSequencer(ctx context.Context, traceId string, ver *common.VersionedTransaction, u *SafeUser) (*SequencerTransactionRequest, error) {
	requests := []*KernelTransactionRequestCreateRequest{{
		RequestID: traceId,
		Raw:       hex.EncodeToString(ver.Marshal()),
	}}
	data, err := json.Marshal(requests)
	if err != nil {
		return nil, err
	}
	method, path := "POST", "/safe/transactions"
	token, err := SignAuthenticationToken(method, path, string(data), u)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*SequencerTransactionRequest `json:"data"`
		Error Error                          `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	if len(resp.Data) != 1 {
		return nil, errors.New("invalid response size")
	}
	return resp.Data[0], nil
}

func requestUnspentOutputsForRecipients(ctx context.Context, assetId string, recipients []*TransactionRecipient, u *SafeUser) ([]*Output, common.Integer, error) {
	membersHash := HashMembers([]string{u.UserId})
	outputs, err := ListUnspentOutputs(ctx, membersHash, 1, assetId, u)
	if err != nil {
		return nil, common.Zero, err
	}
	if len(outputs) == 0 {
		return nil, common.Zero, &UtxoError{
			TotalInput:  common.Zero,
			TotalOutput: common.Zero,
			OutputSize:  0,
		}
	}

	var totalOutput common.Integer
	for _, r := range recipients {
		amt := common.NewIntegerFromString(r.Amount)
		totalOutput = totalOutput.Add(amt)
	}

	var totalInput common.Integer
	for i, o := range outputs {
		amt := common.NewIntegerFromString(o.Amount)
		totalInput = totalInput.Add(amt)
		if totalInput.Cmp(totalOutput) < 0 {
			continue
		}
		return outputs[:i+1], totalInput.Sub(totalOutput), nil
	}
	return nil, common.Zero, &UtxoError{
		TotalInput:  totalInput,
		TotalOutput: totalOutput,
		OutputSize:  len(outputs),
	}
}
