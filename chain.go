package bot

import (
	"context"
	"encoding/json"
	"time"
)

const (
	BitcoinChainId         = "c6d0c728-2624-429b-8e0d-d9d19b6592fa"
	BitcoinCashChainId     = "fd11b6e3-0b87-41f1-a41f-f0e9b49e5bf0"
	BitcoinSVChainId       = "574388fd-b93f-4034-a682-01c2bc095d17"
	LitecoinChainId        = "76c802a2-7c88-447f-a93e-c29c9e5dd9c8"
	EthereumChainId        = "43d61dcd-e413-450d-80b8-101d5e903357"
	EthereumClassicChainId = "2204c1ee-0ea2-4add-bb9a-b3719cfff93a"
	BSCChainId             = "1949e683-6a08-49e2-b087-d6b72398588f"
	PolygonChainId         = "b7938396-3f94-4e0a-9179-d3440718156f"
	MVMChainId             = "a0ffd769-5850-4b48-9651-d2ae44a3e64d"
	DecredChainId          = "8f5caf2a-283d-4c85-832a-91e83bbf290b"
	RippleChainId          = "23dfb5a5-5d7b-48b6-905f-3970e3176e27"
	SiacoinChainId         = "990c4c29-57e9-48f6-9819-7d986ea44985"
	EOSChainId             = "6cfe566e-4aad-470b-8c9a-2fd35b49c68d"
	DogecoinChainId        = "6770a1e5-6086-44d5-b60f-545f9d9e8ffd"
	DashChainId            = "6472e7e3-75fd-48b6-b1dc-28d294ee1476"
	ZcashChainId           = "c996abc9-d94e-4494-b1cf-2a3fd3ac5714"
	NEMChainId             = "27921032-f73e-434e-955f-43d55672ee31"
	ArweaveChainId         = "882eb041-64ea-465f-a4da-817bd3020f52"
	HorizenChainId         = "a2c5d22b-62a2-4c13-b3f0-013290dbac60"
	TRONChainId            = "25dabac5-056a-48ff-b9f9-f67395dc407c"
	StellarChainId         = "56e63c06-b506-4ec5-885a-4a5ac17b83c1"
	MassGridChainId        = "b207bce9-c248-4b8e-b6e3-e357146f3f4c"
	BytomChainId           = "443e1ef5-bc9b-47d3-be77-07f328876c50"
	BytomPoSChainId        = "71a0e8b5-a289-4845-b661-2b70ff9968aa"
	CosmosChainId          = "7397e9f1-4e42-4dc8-8a3b-171daaadd436"
	AkashChainId           = "9c612618-ca59-4583-af34-be9482f5002d"
	BinanceChainId         = "17f78d7c-ed96-40ff-980c-5dc62fecbc85"
	MoneroChainId          = "05c5ac01-31f9-4a69-aa8a-ab796de1d041"
	StarcoinChainId        = "c99a3779-93df-404d-945d-eddc440aa0b2"
	BitSharesChainId       = "05891083-63d2-4f3d-bfbe-d14d7fb9b25a"
	RavencoinChainId       = "6877d485-6b64-4225-8d7e-7333393cb243"
	GrinChainId            = "1351e6bd-66cf-40c1-8105-8a8fe518a222"
	VCashChainId           = "c3b9153a-7fab-4138-a3a4-99849cadc073"
	HandshakeChainId       = "13036886-6b83-4ced-8d44-9f69151587bf"
	NervosChainId          = "d243386e-6d84-42e6-be03-175be17bf275"
	TezosChainId           = "5649ca42-eb5f-4c0e-ae28-d9a4e77eded3"
	NamecoinChainId        = "f8b77dc0-46fd-4ea1-9821-587342475869"
	SolanaChainId          = "64692c23-8971-4cf4-84a7-4dd1271dd887"
	NearChainId            = "d6ac94f7-c932-4e11-97dd-617867f0669e"
	FilecoinChainId        = "08285081-e1d8-4be6-9edc-e203afa932da"
	MobileCoinChainId      = "eea900a8-b327-488c-8d8d-1428702fe240"
	PolkadotChainId        = "54c61a72-b982-4034-a556-0d99e3c21e39"
	KusamaChainId          = "9d29e4f6-d67c-4c4b-9525-604b04afbe9f"
	AlgorandChainId        = "706b6f84-3333-4e55-8e89-275e71ce9803"
	AvalancheChainId       = "cbc77539-0a20-4666-8c8a-4ded62b36f0a"
	XDCChainId             = "b12bb04a-1cea-401c-a086-0be61f544889"
	AptosChainId           = "d2c1c7e1-a1a9-4f88-b282-d93b0a08b42b"
	TONChainId             = "ef660437-d915-4e27-ad3f-632bfb6ba0ee"
)

type NetworkChain struct {
	Type                   string    `json:"type"`
	ChainId                string    `json:"chain_id"`
	Name                   string    `json:"name"`
	Symbol                 string    `json:"symbol"`
	IconURL                string    `json:"icon_url"`
	ManagedBlockHeight     int64     `json:"managed_block_height"`
	DepositBlockHeight     int64     `json:"deposit_block_height"`
	ExternalBlockHeight    int64     `json:"external_block_height"`
	Threshold              int       `json:"threshold"`
	WithdrawalTimestamp    time.Time `json:"withdrawal_timestamp"`
	WithdrawalPendingCount int64     `json:"withdrawal_pending_count"`
	WithdrawalFee          string    `json:"withdrawal_fee"`
	IsSynchronized         bool      `json:"is_synchronized"`
}

func ReadNetworkChainById(ctx context.Context, chainId string) (*NetworkChain, error) {
	body, err := Request(ctx, "GET", "/network/chains/"+chainId, nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  *NetworkChain `json:"data"`
		Error Error         `json:"error"`
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

func ReadNetworkChains(ctx context.Context, chainId string) ([]*NetworkChain, error) {
	body, err := Request(ctx, "GET", "/network/chains", nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  []*NetworkChain `json:"data"`
		Error Error           `json:"error"`
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
