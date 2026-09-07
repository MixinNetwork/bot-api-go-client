package bot

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputerAPIEndpoints(t *testing.T) {
	oldClient := computerClient
	t.Cleanup(func() { computerClient = oldClient })
	requests := 0
	computerClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		var response string
		switch r.URL.Path {
		case "/":
			response = `{"observer":"observer","height":42,"members":{"app_id":"app","members":["member"],"threshold":1},"params":{"operation":{"asset":"asset","price":"1"}}}`
		case "/users/mix":
			response = `{"id":"1","mix_address":"mix","chain_address":"chain"}`
		case "/users/missing":
			response = `{"error":{"code":404}}`
		case "/deployed_assets":
			if r.Method == http.MethodPost {
				response = `{}`
			} else {
				response = `[{"asset_id":"asset","address":"address"}]`
			}
		case "/system_calls/call":
			response = `{"id":"call","state":"done"}`
		case "/system_calls/missing":
			response = `{"error":{"code":404}}`
		case "/nonce_accounts":
			response = `{"mix":"mix","nonce_address":"nonce"}`
		case "/fee":
			response = `{"fee_id":"fee","xin_amount":"2"}`
		default:
			t.Fatalf("unexpected computer request %s %s", r.Method, r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(response)),
			Request:    r,
		}, nil
	})}
	ctx := context.Background()

	info, err := GetComputerInfo(ctx)
	require.NoError(t, err)
	assert.Equal(t, "observer", info.ObserverId)
	user, err := GetComputerUser(ctx, "mix")
	require.NoError(t, err)
	assert.Equal(t, "chain", user.ChainAddress)
	user, err = GetComputerUser(ctx, "missing")
	require.NoError(t, err)
	assert.Nil(t, user)
	assets, err := GetComputerDeployedAssets(ctx)
	require.NoError(t, err)
	assert.Equal(t, "asset", assets[0].AssetID)
	call, err := GetComputerSystemCall(ctx, "call")
	require.NoError(t, err)
	assert.Equal(t, "done", call.State)
	call, err = GetComputerSystemCall(ctx, "missing")
	require.NoError(t, err)
	assert.Nil(t, call)
	require.NoError(t, ComputerDeployExternalAsset(ctx, []string{"asset"}))
	beforeRejected := requests
	err = ComputerDeployExternalAsset(ctx, []string{SolanaChainId})
	require.Error(t, err)
	assert.Equal(t, beforeRejected, requests)
	nonce, err := LockComputerNonceAccount(ctx, "mix")
	require.NoError(t, err)
	assert.Equal(t, "nonce", nonce.NonceAddress)
	fee, err := GetFeeOnXINBasedOnSOL(ctx, "1")
	require.NoError(t, err)
	assert.Equal(t, "2", fee.XINAmount)
	assert.Equal(t, UniqueObjectId(SolanaChainId, "address"), assets[0].GetSolanaAssetId())
}

func TestComputerRequestHonorsContext(t *testing.T) {
	oldClient := computerClient
	t.Cleanup(func() { computerClient = oldClient })
	computerClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := computerRequest(ctx, http.MethodGet, "/", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestComputerResponseErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantCode   int
	}{
		{name: "bad JSON", statusCode: http.StatusOK, body: "{", wantCode: 10002},
		{name: "API error", statusCode: http.StatusOK, body: `{"error":{"code":123,"description":"bad"}}`, wantCode: 123},
		{name: "server error", statusCode: http.StatusBadGateway, body: "bad gateway", wantCode: 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldClient := computerClient
			t.Cleanup(func() { computerClient = oldClient })
			computerClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(tt.body)),
					Request:    r,
				}, nil
			})}
			_, err := GetComputerInfo(context.Background())
			require.Error(t, err)
			var apiError Error
			require.ErrorAs(t, err, &apiError)
			assert.Equal(t, tt.wantCode, apiError.Code)
		})
	}
}

func TestComputerEncodingHelpers(t *testing.T) {
	encoded, err := ComputerUserIDToBytes("281474976710657")
	require.NoError(t, err)
	assert.Equal(t, "0001000000000001", hex.EncodeToString(encoded))
	for _, invalid := range []string{"", "-1", "18446744073709551616"} {
		_, err := ComputerUserIDToBytes(invalid)
		assert.Error(t, err)
	}

	const appID = "0acfe278-714f-4cfc-ae52-70ce34e3eb11"
	const contractID = "ded9e592-111a-4272-a5b7-9e18e627ba3c"
	const functionID = "1055985c-5759-3839-b5b5-977915ac424d"
	extra, err := BuildSystemCallExtra("281474976710657", contractID, true, functionID)
	require.NoError(t, err)
	assert.Equal(t, byte(1), extra[24])
	_, err = BuildSystemCallExtra("bad", contractID, false, functionID)
	assert.Error(t, err)
	_, err = BuildSystemCallExtra("1", "bad", false, functionID)
	assert.Error(t, err)
	_, err = BuildSystemCallExtra("1", contractID, false, "bad")
	assert.Error(t, err)

	operation := EncodeOperationMemo(OperationTypeSystemCall, extra)
	assert.Equal(t, byte(OperationTypeSystemCall), operation[0])
	memo := EncodeMtgExtra(appID, operation)
	decodedAppID, decoded := DecodeComputerExtraBase64(memo)
	assert.Equal(t, appID, decodedAppID)
	assert.Equal(t, operation, decoded)
	id, rest := DecodeComputerExtraBase64("bad")
	assert.Empty(t, id)
	assert.Nil(t, rest)
	short := base64.RawURLEncoding.EncodeToString([]byte("short"))
	id, rest = DecodeComputerExtraBase64(short)
	assert.Empty(t, id)
	assert.Nil(t, rest)
}
