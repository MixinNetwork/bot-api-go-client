package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetBalancePaginationAndAggregation(t *testing.T) {
	user, _ := newTestSafeUser(t.Name())
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/safe/outputs":
			if r.URL.Query().Get("asset") == "asset" {
				if r.URL.Query().Get("offset") == "" {
					outputs := make([]*Output, 500)
					for i := range outputs {
						outputs[i] = &Output{OutputID: fmt.Sprintf("output-%d", i), AssetId: "asset", Amount: "1", Sequence: int64(i + 1)}
					}
					require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"data": outputs}))
					return
				}
				assert.Equal(t, "500", r.URL.Query().Get("offset"))
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"data": []*Output{{OutputID: "last", AssetId: "asset", Amount: "2", Sequence: 501}}}))
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"data": []*Output{
				{OutputID: "one", AssetId: "asset", Amount: "1.2", Sequence: 1},
				{OutputID: "two", AssetId: "asset", Amount: "2.3", Sequence: 2},
			}}))
		case "/safe/assets/fetch":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"data": []*Asset{{AssetID: "asset", DisplaySymbol: "AST"}}}))
		default:
			t.Fatalf("unexpected request %s", r.URL.RequestURI())
		}
	}))

	balance, err := AssetBalanceWithSafeUser(context.Background(), "asset", user)
	require.NoError(t, err)
	assert.Equal(t, "502.00000000", balance.String())
	balance, err = AssetBalance(context.Background(), "asset", user.UserId, user.SessionId, user.SessionPrivateKey)
	require.NoError(t, err)
	assert.Equal(t, "502.00000000", balance.String())
	assets, err := ListAssetWithBalance(context.Background(), user)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	assert.Equal(t, "3.5", assets[0].Amount)
}

func TestAssetBalanceReturnsOutputErrors(t *testing.T) {
	user, _ := newTestSafeUser(t.Name())
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error":{"status":202,"code":1234,"description":"failed"}}`))
	}))

	_, err := AssetBalanceWithSafeUser(context.Background(), "asset", user)
	require.Error(t, err)
	var apiError Error
	require.ErrorAs(t, err, &apiError)
	assert.Equal(t, 1234, apiError.Code)

	_, err = ListAssetWithBalance(context.Background(), user)
	require.Error(t, err)
}

func TestAssetBalanceRejectsStalledPagination(t *testing.T) {
	user, _ := newTestSafeUser(t.Name())
	requests := 0
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		outputs := make([]*Output, 500)
		for i := range outputs {
			outputs[i] = &Output{OutputID: fmt.Sprintf("output-%d", i), Amount: "1", Sequence: 0}
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"data": outputs}))
	}))

	_, err := AssetBalanceWithSafeUser(context.Background(), "asset", user)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pagination did not advance")
	assert.Equal(t, 1, requests)
}

func TestUserAssetBalance(t *testing.T) {
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"output_id":"one","amount":"1.25"},{"output_id":"two","amount":"2.75"}]}`))
	}))
	balance, err := UserAssetBalance(context.Background(), "user", "asset", "token")
	require.NoError(t, err)
	assert.Equal(t, "4.00000000", balance.String())
}

func TestAssetSymbols(t *testing.T) {
	tests := map[string]string{
		USDT_ERC20:   "USDT (ERC20)",
		USDC_ERC20:   "USDC (ERC20)",
		USDT_TRC20:   "USDT (TRC20)",
		USDT_POLYGON: "USDT (Polygon)",
		USDT_BSC:     "USDT (BSC)",
		USDT_SOLANA:  "USDT (Solana)",
	}
	for assetID, want := range tests {
		assert.Equal(t, want, (&Asset{AssetID: assetID}).GetSymbol())
		assert.Equal(t, want, (&AssetNetwork{AssetID: assetID}).GetSymbol())
	}
	assert.Equal(t, "CUSTOM", (&Asset{AssetID: "custom", DisplaySymbol: "CUSTOM"}).GetSymbol())
	assert.Equal(t, "CUSTOM", (&AssetNetwork{AssetID: "custom", DisplaySymbol: "CUSTOM"}).GetSymbol())
}
