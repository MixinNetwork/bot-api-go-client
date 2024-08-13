package bot

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"filippo.io/edwards25519"
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/gofrs/uuid/v5"
)

type TransactionRecipient struct {
	MixAddress *MixAddress
	Amount     string

	Destination string
	Tag         string
}

type TransactionReceiver struct {
	Members     []string    `json:"members"`
	MembersHash crypto.Hash `json:"members_hash"`
	Threshold   uint8       `json:"threshold"`
}

type SequencerTransactionRequest struct {
	RequestID        string                 `json:"request_id"`
	TransactionHash  string                 `json:"transaction_hash"`
	Asset            string                 `json:"asset"`
	Amount           string                 `json:"amount"`
	Extra            string                 `json:"extra"`
	Receivers        []*TransactionReceiver `json:"receivers"`
	Senders          []string               `json:"senders"`
	SendersHash      string                 `json:"senders_hash"`
	SendersThreshold uint8                  `json:"senders_threshold"`
	Signers          []string               `json:"signers"`
	State            string                 `json:"state"`
	RawTransaction   string                 `json:"raw_transaction"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	SnapshotID       string                 `json:"snapshot_id"`
	SnapshotHash     string                 `json:"snapshot_hash"`
	SnapshotAt       time.Time              `json:"snapshot_at"`

	Views []string `json:"views"`
}

type UtxoInsufficientError struct {
	TotalInput  common.Integer
	TotalOutput common.Integer
	OutputSize  int
}

func (ue *UtxoInsufficientError) Error() string {
	return fmt.Sprintf("insufficient outputs %s@%d %s", ue.TotalInput, ue.OutputSize, ue.TotalOutput)
}

func SendTransferTransaction(ctx context.Context, assetId, receiver, amount, traceId string, extra []byte, u *SafeUser) (*SequencerTransactionRequest, error) {
	if uuid.FromStringOrNil(receiver).String() != receiver {
		return nil, fmt.Errorf("invalid receiver %s", receiver)
	}
	ma := NewUUIDMixAddress([]string{receiver}, 1)
	tr := &TransactionRecipient{
		MixAddress: ma,
		Amount:     amount,
	}
	return SendTransaction(ctx, assetId, []*TransactionRecipient{tr}, traceId, extra, nil, u)
}

func SendTransactionUntilSufficient(ctx context.Context, assetId, receiver, amount, traceId string, extra []byte, u *SafeUser) (*SequencerTransactionRequest, error) {
	for {
		str, err := SendTransferTransaction(ctx, assetId, receiver, amount, traceId, extra, u)
		if err == nil {
			return str, nil
		}
		if ue, ok := err.(*UtxoInsufficientError); ok {
			log.Println(ue)
			time.Sleep(2 * time.Second)
		} else {
			return nil, err
		}
	}
}

func SendTransactionSplitChangeOutput(ctx context.Context, assetId string, recipients []*TransactionRecipient, traceId string, extra []byte, references []string, splitAmount uint64, splitCount int, u *SafeUser) (*SequencerTransactionRequest, error) {
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
		if splitCount > 0 && changeAmount.Cmp(common.NewInteger(splitAmount)) > 0 {
			if splitCount > (256 - len(recipients)) {
				return nil, fmt.Errorf("invalid split count %d", splitCount)
			}
			if splitCount%2 != 0 {
				return nil, fmt.Errorf("invalid split count %d", splitCount)
			}

			var rs []*TransactionRecipient
			totalAmount := common.Zero
			splitAmt := common.NewInteger(splitAmount)
			for i := 0; i < splitCount; i++ {
				amount := splitAmt
				if changeAmount.Sub(totalAmount).Cmp(splitAmt) < 0 {
					amount = changeAmount.Sub(totalAmount)
				}
				if i == splitCount-1 {
					amount = changeAmount.Sub(totalAmount)
				}
				if amount.Cmp(common.Zero) > 0 {
					recipient := &TransactionRecipient{
						MixAddress: ma,
						Amount:     amount.String(),
					}
					rs = append(rs, recipient)
				}
				if amount.Cmp(splitAmt) < 0 {
					break
				}
				totalAmount = totalAmount.Add(amount)
			}
			// validate change amount
			validateAmount := common.Zero
			for _, r := range rs {
				validateAmount = validateAmount.Add(common.NewIntegerFromString(r.Amount))
			}
			if validateAmount.Cmp(changeAmount) != 0 {
				return nil, fmt.Errorf("invalid split change amount %s != %s", validateAmount, changeAmount)
			}
			recipients = append(recipients, rs...)
		} else {
			recipients = append(recipients, &TransactionRecipient{
				MixAddress: ma,
				Amount:     changeAmount.String(),
			})
		}
	}
	return sendTransaction(ctx, asset, utxos, recipients, traceId, extra, references, u)
}

func SendTransaction(ctx context.Context, assetId string, recipients []*TransactionRecipient, traceId string, extra []byte, references []string, u *SafeUser) (*SequencerTransactionRequest, error) {
	return SendTransactionSplitChangeOutput(ctx, assetId, recipients, traceId, extra, references, 0, 0, u)
}

func SendTransactionWithOutputs(ctx context.Context, assetId string, recipients []*TransactionRecipient, outputs []*Output, traceId string, extra []byte, references []string, u *SafeUser) (*SequencerTransactionRequest, error) {
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
	return sendTransaction(ctx, asset, outputs, recipients, traceId, extra, references, u)
}

func SendTransactionWithOutput(ctx context.Context, assetId string, recipients []*TransactionRecipient, utxo *Output, traceId string, extra []byte, references []string, u *SafeUser) (*SequencerTransactionRequest, error) {
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

	var totalOutput common.Integer
	for _, r := range recipients {
		amt := common.NewIntegerFromString(r.Amount)
		totalOutput = totalOutput.Add(amt)
	}
	totalInput := common.NewIntegerFromString(utxo.Amount)
	changeAmount := totalInput.Sub(totalOutput)
	if changeAmount.Sign() < 0 {
		return nil, &UtxoInsufficientError{
			TotalInput:  totalInput,
			TotalOutput: totalOutput,
			OutputSize:  1,
		}
	}
	if changeAmount.Sign() > 0 {
		ma := NewUUIDMixAddress([]string{u.UserId}, 1)
		recipients = append(recipients, &TransactionRecipient{
			MixAddress: ma,
			Amount:     changeAmount.String(),
		})
	}
	return sendTransaction(ctx, asset, []*Output{utxo}, recipients, traceId, extra, references, u)
}

func GetTransactionById(ctx context.Context, requestId string) (*SequencerTransactionRequest, error) {
	return GetTransactionByIdWithSafeUser(ctx, requestId, nil)
}

func GetTransactionByIdWithSafeUser(ctx context.Context, requestId string, su *SafeUser) (*SequencerTransactionRequest, error) {
	method, path := "GET", fmt.Sprintf("/safe/transactions/%s", requestId)
	var accessToken string
	var err error
	if su != nil {
		accessToken, err = SignAuthenticationToken(method, path, "", su)
		if err != nil {
			return nil, err
		}
	}
	body, err := Request(ctx, method, path, nil, accessToken)
	if err != nil {
		return nil, err
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

func sendTransaction(ctx context.Context, asset crypto.Hash, utxos []*Output, recipients []*TransactionRecipient, traceId string, extra []byte, references []string, u *SafeUser) (*SequencerTransactionRequest, error) {
	// build the unsigned raw transaction
	tx, err := BuildRawTransaction(ctx, asset, utxos, recipients, extra, references, traceId, u)
	if err != nil {
		return nil, fmt.Errorf("BuildRawTransaction(%s) => %v", asset, err)
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
	ver, err = signRawTransaction(ver, str.Views, u.SpendPrivateKey)
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

func BuildRawTransaction(ctx context.Context, asset crypto.Hash, utxos []*Output, recipients []*TransactionRecipient, extra []byte, references []string, traceId string, u *SafeUser) (*common.Transaction, error) {
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

	gkm, err := RequestGhostRecipientsWithTraceId(ctx, recipients, traceId, u)
	if err != nil {
		return nil, err
	}
	for i, r := range recipients {
		if r.Destination == "" && r.MixAddress == nil {
			panic(r)
		}
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

		g := gkm[i]
		mask, err := crypto.KeyFromString(g.Mask)
		if err != nil {
			panic(g.Mask)
		}
		tx.Outputs = append(tx.Outputs, &common.Output{
			Type:   common.OutputTypeScript,
			Amount: common.NewIntegerFromString(r.Amount),
			Script: common.NewThresholdScript(r.MixAddress.Threshold),
			Keys:   g.KeysSlice(),
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

func VerifyRawTransaction(ctx context.Context, requests []*KernelTransactionRequestCreateRequest, u *SafeUser) ([]*SequencerTransactionRequest, error) {
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
	return resp.Data, nil
}

func verifyRawTransactionBySequencer(ctx context.Context, traceId string, ver *common.VersionedTransaction, u *SafeUser) (*SequencerTransactionRequest, error) {
	requests := []*KernelTransactionRequestCreateRequest{{
		RequestID: traceId,
		Raw:       hex.EncodeToString(ver.Marshal()),
	}}
	verified, err := VerifyRawTransaction(ctx, requests, u)
	if err != nil {
		return nil, err
	}

	if len(verified) != 1 {
		return nil, errors.New("invalid response size")
	}
	return verified[0], nil
}

func signRawTransaction(ver *common.VersionedTransaction, views []string, spendKey string) (*common.VersionedTransaction, error) {
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

func SendRawTransaction(ctx context.Context, requests []*KernelTransactionRequestCreateRequest, u *SafeUser) ([]*SequencerTransactionRequest, error) {
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
	return resp.Data, nil
}

func sendRawTransactionToSequencer(ctx context.Context, traceId string, ver *common.VersionedTransaction, u *SafeUser) (*SequencerTransactionRequest, error) {
	requests := []*KernelTransactionRequestCreateRequest{{
		RequestID: traceId,
		Raw:       hex.EncodeToString(ver.Marshal()),
	}}
	txs, err := SendRawTransaction(ctx, requests, u)
	if err != nil {
		return nil, err
	}
	if len(txs) != 1 {
		return nil, errors.New("invalid response size")
	}
	return txs[0], nil
}

func requestUnspentOutputsForRecipients(ctx context.Context, assetId string, recipients []*TransactionRecipient, u *SafeUser) ([]*Output, common.Integer, error) {
	var totalOutput common.Integer
	for _, r := range recipients {
		amt := common.NewIntegerFromString(r.Amount)
		totalOutput = totalOutput.Add(amt)
	}

	membersHash := HashMembers([]string{u.UserId})
	unspentOutputs, err := ListUnspentOutputs(ctx, membersHash, 1, assetId, u)
	if err != nil {
		return nil, common.Zero, err
	}
	if len(unspentOutputs) == 0 {
		return nil, common.Zero, &UtxoInsufficientError{
			TotalInput:  common.Zero,
			TotalOutput: totalOutput,
			OutputSize:  0,
		}
	}

	var totalInput common.Integer
	for i, o := range unspentOutputs {
		amt := common.NewIntegerFromString(o.Amount)
		totalInput = totalInput.Add(amt)
		if totalInput.Cmp(totalOutput) < 0 {
			continue
		}
		return unspentOutputs[:i+1], totalInput.Sub(totalOutput), nil
	}
	return nil, common.Zero, &UtxoInsufficientError{
		TotalInput:  totalInput,
		TotalOutput: totalOutput,
		OutputSize:  len(unspentOutputs),
	}
}

func RequestGhostRecipientsWithTraceId(ctx context.Context, recipients []*TransactionRecipient, traceId string, u *SafeUser) (map[int]*GhostKeys, error) {
	traceHash := crypto.Blake3Hash([]byte(traceId))
	privSpend, err := crypto.KeyFromString(u.SpendPrivateKey)
	if err != nil {
		panic(err)
	}
	gkm := make(map[int]*GhostKeys, len(recipients))
	var uuidGkrs []*GhostKeyRequest
	for i, r := range recipients {
		if r.MixAddress == nil {
			continue
		}
		ma := r.MixAddress
		seedHash := crypto.Blake3Hash(append(traceHash[:], big.NewInt(int64(i)).Bytes()...))
		if len(ma.xinMembers) > 0 {
			privHash := crypto.Blake3Hash(append(seedHash[:], privSpend[:]...))
			r := crypto.NewKeyFromSeed(append(traceHash[:], privHash[:]...))
			gk := &GhostKeys{
				Mask: r.Public().String(),
				Keys: make([]string, len(ma.xinMembers)),
			}
			for j, a := range ma.xinMembers {
				k := crypto.DeriveGhostPublicKey(&r, &a.PublicViewKey, &a.PublicSpendKey, uint64(i))
				gk.Keys[j] = k.String()
			}
			gkm[i] = gk
		} else {
			hint := UniqueObjectId(traceHash.String(), seedHash.String())
			uuidGkrs = append(uuidGkrs, &GhostKeyRequest{
				Receivers: ma.Members(),
				Index:     uint(i),
				Hint:      hint,
			})
		}
	}
	if len(uuidGkrs) > 0 {
		uuidGks, err := RequestSafeGhostKeys(ctx, uuidGkrs, u)
		if err != nil {
			return nil, err
		}
		for i, g := range uuidGks {
			index := uuidGkrs[i].Index
			gkm[int(index)] = g
		}
	}
	return gkm, nil
}
