package bot

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetSearchEscapesName(t *testing.T) {
	for _, tt := range []struct {
		name string
		path string
	}{
		{"USD Coin-1_2.3", "USD%20Coin-1_2.3"},
		{"../a/b?limit=1#fragment", "..%2Fa%2Fb%3Flimit=1%23fragment"},
		{"%2e%2e%2ftop", "%252e%252e%252ftop"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				assert.Equal(t, "/network/assets/search/"+tt.path, r.RequestURI)
				_, _ = w.Write([]byte(`{"data":[{"asset_id":"asset"}]}`))
			}))
			assets, err := AssetSearch(context.Background(), tt.name)
			require.NoError(t, err)
			require.Len(t, assets, 1)
			assert.Equal(t, "asset", assets[0].AssetID)
			assert.Equal(t, 1, requests)
		})
	}
}

func TestAssetSearchRejectsDotSegments(t *testing.T) {
	requests := 0
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	for _, name := range []string{".", ".."} {
		assets, err := AssetSearch(context.Background(), name)
		require.Error(t, err)
		assert.Nil(t, assets)
		var apiError Error
		require.ErrorAs(t, err, &apiError)
		assert.Equal(t, 10002, apiError.Code)
	}
	assert.Zero(t, requests)
}
