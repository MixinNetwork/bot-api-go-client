package bot

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/pkg/errors"
)

const (
	OperationTypeAddUser     = 1
	OperationTypeSystemCall  = 2
	OperationTypeUserDeposit = 3

	SolanaChainID = "64692c23-8971-4cf4-84a7-4dd1271dd887"
	computerUri   = "https://computer.mixin.one"
)

var computerClient = &http.Client{Timeout: 10 * time.Second}

type ComputerInfoResponse struct {
	ObserverId string `json:"observer"`
	Payer      string `json:"payer"`
	Height     int64  `json:"height"`
	Members    struct {
		AppId     string   `json:"app_id"`
		Members   []string `json:"members"`
		Threshold int      `json:"threshold"`
	} `json:"members"`
	Params struct {
		Operation struct {
			Asset string `json:"asset"`
			Price string `json:"price"`
		} `json:"operation"`
	} `json:"params"`

	Error `json:"error"`
}

type ComputerUserResponse struct {
	ID           string `json:"id"`
	ChainAddress string `json:"chain_address"`
	MixAddress   string `json:"mix_address"`

	Error `json:"error"`
}

type ComputerDeployedAsset struct {
	AssetID  string `json:"asset_id"`
	Address  string `json:"address"`
	Decimals int64  `json:"decimals"`
	IconURL  string `json:"uri"`
}

type ComputerSystemCall struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	UserID       string `json:"user_id"`
	NonceAccount string `json:"nonce_account"`
	Raw          string `json:"raw"`
	State        string `json:"state"`
	Hash         string `json:"hash"`
}

type ComputerSystemCallResponse struct {
	ComputerSystemCall
	Reason   string               `json:"reason"`
	SubCalls []ComputerSystemCall `json:"subs"`

	Error `json:"error"`
}

type ComputerNonceAccountResponse struct {
	Mix          string `json:"mix"`
	NonceAddress string `json:"nonce_address"`
	NonceHash    string `json:"nonce_hash"`

	Error `json:"error"`
}

type ComputerFeeResponse struct {
	FeeID     string `json:"fee_id"`
	XINAmount string `json:"xin_amount"`

	Error `json:"error"`
}

func computerRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(method, computerUri+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := computerClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, errors.Wrap(ServerError(ctx, nil), fmt.Sprintf("response status code %d", resp.StatusCode))
	}
	return io.ReadAll(resp.Body)
}

func GetComputerInfo(ctx context.Context) (*ComputerInfoResponse, error) {
	body, err := computerRequest(ctx, "GET", "/", nil)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp *ComputerInfoResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp, nil
}

func GetComputerUser(ctx context.Context, addr string) (*ComputerUserResponse, error) {
	body, err := computerRequest(ctx, "GET", "/users/"+addr, nil)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp *ComputerUserResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		if resp.Error.Code == 404 {
			return nil, nil
		}
		return nil, resp.Error
	}
	return resp, nil
}

func GetComputerDeployedAssets(ctx context.Context) ([]*ComputerDeployedAsset, error) {
	body, err := computerRequest(ctx, "GET", "/deployed_assets", nil)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp []*ComputerDeployedAsset
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	return resp, nil
}

func GetComputerSystemCall(ctx context.Context, id string) (*ComputerSystemCallResponse, error) {
	body, err := computerRequest(ctx, "GET", "/system_calls/"+id, nil)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp *ComputerSystemCallResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		if resp.Error.Code == 404 {
			return nil, nil
		}
		return nil, resp.Error
	}
	return resp, nil
}

func ComputerDeployExternalAsset(ctx context.Context, assets []string) error {
	for _, asset := range assets {
		if asset != SolanaChainID {
			continue
		}
		return fmt.Errorf("cannot deploy asset from Solana: %s", asset)
	}
	data, err := json.Marshal(assets)
	if err != nil {
		return err
	}
	_, err = computerRequest(ctx, "POST", "/deployed_assets", data)
	if err != nil {
		return ServerError(ctx, err)
	}
	return err
}

func LockComputerNonceAccount(ctx context.Context, mix string) (*ComputerNonceAccountResponse, error) {
	data, err := json.Marshal(map[string]string{
		"mix": mix,
	})
	if err != nil {
		return nil, err
	}
	body, err := computerRequest(ctx, "POST", "/nonce_accounts", data)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp *ComputerNonceAccountResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	return resp, nil
}

func GetFeeOnXINBasedOnSOL(ctx context.Context, solAmount string) (*ComputerFeeResponse, error) {
	data, err := json.Marshal(map[string]string{
		"sol_amount": solAmount,
	})
	if err != nil {
		return nil, err
	}
	body, err := computerRequest(ctx, "POST", "/fee", data)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp *ComputerFeeResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	return resp, nil
}

func RegisterComputer(ctx context.Context, su *SafeUser) (*SequencerTransactionRequest, error) {
	info, err := GetComputerInfo(ctx)
	if err != nil {
		return nil, err
	}
	mix := NewUUIDMixAddress([]string{su.UserId}, 1).String()
	memo := EncodeMtgExtra(info.Members.AppId, EncodeOperationMemo(OperationTypeAddUser, []byte(mix)))

	trace := UniqueObjectId(mix, "computer_register")
	rs := []*TransactionRecipient{
		{
			MixAddress: NewUUIDMixAddress(info.Members.Members, byte(info.Members.Threshold)),
			Amount:     info.Params.Operation.Price,
		},
	}
	return SendTransaction(ctx, info.Params.Operation.Asset, rs, trace, []byte(memo), nil, su)
}

func EncodeOperationMemo(operation byte, extra []byte) []byte {
	memo := []byte{operation}
	memo = append(memo, extra...)
	return memo
}

func EncodeMtgExtra(appID string, extra []byte) string {
	data := uuid.Must(uuid.FromString(appID)).Bytes()
	data = append(data, extra...)
	return base64.RawURLEncoding.EncodeToString(data)
}
