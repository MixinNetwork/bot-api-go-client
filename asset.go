package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"net/url"
	"slices"

	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/gofrs/uuid/v5"
)

const (
	BTC          = "c6d0c728-2624-429b-8e0d-d9d19b6592fa"
	ETH          = "43d61dcd-e413-450d-80b8-101d5e903357"
	USDT_ERC20   = "4d8c508b-91c5-375b-92b0-ee702ed2dac5"
	USDC_ERC20   = "9b180ab6-6abe-3dc0-a13f-04169eb34bfa"
	USDT_TRC20   = "b91e18ff-a9ae-3dc7-8679-e935d9a4b34b"
	USDT_POLYGON = "218bc6f4-7927-3f8e-8568-3a3725b74361"
	USDT_BSC     = "94213408-4ee7-3150-a9c4-9c5cce421c78"
	USDT_SOLANA  = "cb54aed4-1893-3977-b739-ec7b2e04f0c5"
	USDC_SOLANA  = "de6fa523-c596-398e-b12f-6d6980544b59"
	USDC_BASE    = "2f845564-3898-3d17-8c24-3275e96235b5"
)

type Asset struct {
	Type           string  `json:"type"`
	AssetID        string  `json:"asset_id"`
	ChainID        string  `json:"chain_id"`
	AssetKey       string  `json:"asset_key"`
	Precision      int     `json:"precision"`
	KernelAssetId  string  `json:"kernel_asset_id"`
	DisplaySymbol  string  `json:"display_symbol"`
	DisplayName    string  `json:"display_name"`
	Symbol         string  `json:"symbol"`
	Name           string  `json:"name"`
	IconURL        string  `json:"icon_url"`
	Amount         string  `json:"amount"`
	PriceBTC       string  `json:"price_btc"`
	PriceUSD       string  `json:"price_usd"`
	ChangeBTC      string  `json:"change_btc"`
	ChangeUSD      string  `json:"change_usd"`
	Confirmations  int64   `json:"confirmations"`
	Fee            string  `json:"fee"`
	Reserve        string  `json:"reserve"`
	SnapshotsCount int64   `json:"snapshots_count"`
	Capitalization float64 `json:"capitalization"`
	Liquidity      string  `json:"liquidity"`

	CollectionHash string `json:"collection_hash,omitempty"`
}

type AssetTicker struct {
	Type     string `json:"type"`
	PriceBTC string `json:"price_btc"`
	PriceUSD string `json:"price_usd"`
}

type AssetFee struct {
	Type    string `json:"type"`
	AssetID string `json:"asset_id"`
	Amount  string `json:"amount"`
}

func AssetBalance(ctx context.Context, assetId, uid, sid, sessionKey string) (common.Integer, error) {
	su := &SafeUser{
		UserId:            uid,
		SessionId:         sid,
		SessionPrivateKey: sessionKey,
	}

	if id, _ := uuid.FromString(assetId); assetId == id.String() {
		assetId = crypto.Sha256Hash([]byte(assetId)).String()
	}
	return AssetBalanceWithSafeUser(ctx, assetId, su)
}

func AssetBalanceWithSafeUser(ctx context.Context, kernelAssetId string, su *SafeUser) (common.Integer, error) {
	membersHash := HashMembers([]string{su.UserId})
	outputs, err := ListUnspentOutputs(ctx, membersHash, 1, kernelAssetId, su)
	if err != nil {
		return common.Zero, err
	}
	var total common.Integer
	for _, o := range outputs {
		amt := common.NewIntegerFromString(o.Amount)
		total = total.Add(amt)
	}
	return total, nil
}

func UserAssetBalance(ctx context.Context, userID, assetID, accessToken string) (common.Integer, error) {
	if id, _ := uuid.FromString(assetID); assetID == id.String() {
		assetID = crypto.Sha256Hash([]byte(assetID)).String()
	}

	membersHash := HashMembers([]string{userID})
	outputs, err := ListUnspentOutputsByToken(ctx, membersHash, 1, assetID, accessToken)
	if err != nil {
		return common.Zero, err
	}
	var total common.Integer
	for _, o := range outputs {
		amt := common.NewIntegerFromString(o.Amount)
		total = total.Add(amt)
	}
	return total, nil
}

func ReadAssetFee(ctx context.Context, assetId, destination string, su *SafeUser) ([]*AssetFee, error) {
	params := url.Values{}
	params.Set("destination", destination)
	method, path := "GET", fmt.Sprintf("/safe/assets/%s/fees?%s", assetId, params.Encode())
	token, err := SignAuthenticationToken(method, path, "", su)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, nil, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*AssetFee `json:"data"`
		Error Error       `json:"error"`
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

func FetchAssets(ctx context.Context, assetIds []string, safeUser *SafeUser) ([]*Asset, error) {
	body, err := json.Marshal(assetIds)
	if err != nil {
		return nil, err
	}

	path := "/safe/assets/fetch"
	token, err := SignAuthenticationToken("POST", path, string(body), safeUser)
	if err != nil {
		return nil, err
	}
	result, err := Request(ctx, "POST", path, body, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  []*Asset `json:"data"`
		Error Error    `json:"error"`
	}
	err = json.Unmarshal(result, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func ListAssetWithBalance(ctx context.Context, su *SafeUser) ([]*Asset, error) {
	membersHash := HashMembers([]string{su.UserId})
	offset := uint64(0)
	m := make(map[string]number.Decimal)
	filter := make(map[string]bool)
	for {
		outputs, err := ListOutputs(ctx, membersHash, 1, "", "unspent", offset, 500, su)
		if err != nil {
			log.Println(err)
			continue
		}
		for i, output := range outputs {
			if _, ok := filter[output.OutputID]; ok {
				continue
			}
			filter[output.OutputID] = true
			if aa, ok := m[output.AssetId]; ok {
				m[output.AssetId] = aa.Add(number.FromString(output.Amount))
			} else {
				m[output.AssetId] = number.FromString(output.Amount)
			}
			if i == len(outputs)-1 {
				offset = uint64(outputs[len(outputs)-1].Sequence)
			}
		}
		if len(outputs) < 500 {
			break
		}
	}
	assets := []*Asset{}
	var err error
	if len(m) > 0 {
		assetIds := slices.Collect(maps.Keys(m))
		assets, err = FetchAssets(ctx, assetIds, su)
		if err != nil {
			return nil, err
		}
		for _, asset := range assets {
			asset.Amount = m[asset.AssetID].Persist()
		}
	}
	return assets, nil
}

func (asset *Asset) GetSymbol() string {
	if asset.AssetID == USDT_ERC20 {
		return "USDT (ERC20)"
	} else if asset.AssetID == USDC_ERC20 {
		return "USDC (ERC20)"
	} else if asset.AssetID == USDT_TRC20 {
		return "USDT (TRC20)"
	} else if asset.AssetID == USDT_POLYGON {
		return "USDT (Polygon)"
	} else if asset.AssetID == USDT_BSC {
		return "USDT (BSC)"
	} else if asset.AssetID == USDT_SOLANA {
		return "USDT (Solana)"
	}
	return asset.DisplaySymbol
}
