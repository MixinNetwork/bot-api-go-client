package bot

type NetworkAsset struct {
	Type           string `json:"type"`
	AssetID        string `json:"asset_id"`
	ChainID        string `json:"chain_id"`
	AssetKey       string `json:"asset_key"`
	MixinID        string `json:"mixin_id"`
	Symbol         string `json:"symbol"`
	Name           string `json:"name"`
	IconURL        string `json:"icon_url"`
	Amount         string `json:"amount"`
	PriceBTC       string `json:"price_btc"`
	PriceUSD       string `json:"price_usd"`
	ChangeBTC      string `json:"change_btc"`
	ChangeUSD      string `json:"change_usd"`
	Confirmations  int64  `json:"confirmations"`
	Fee            string `json:"fee"`
	Reserve        string `json:"reserve"`
	SnapshotsCount int64  `json:"snapshots_count"`
	Capitalization string `json:"capitalization"`
	Liquidity      string `json:"liquidity"`
}
