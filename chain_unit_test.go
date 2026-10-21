package bot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChainHelpers(t *testing.T) {
	names := map[string]string{
		EOSChainId:             "EOS",
		RippleChainId:          "Ripple",
		SiacoinChainId:         "Siacoin",
		EthereumChainId:        "Ethereum",
		EthereumClassicChainId: "Ethereum Classic",
		BSCChainId:             "BNB Smart Chain",
		PolygonChainId:         "Polygon",
		BaseChainId:            "Base",
		OptimismChainId:        "OP Mainnet",
		ArbitrumChainId:        "Arbitrum One",
		HyperEVMChainId:        "HyperEVM",
		XLayerChainId:          "X Layer",
		RobinhoodChainId:       "Robinhood",
		BitcoinChainId:         "Bitcoin",
		HandshakeChainId:       "Handshake",
		BitcoinCashChainId:     "Bitcoin Cash",
		BitcoinSVChainId:       "Bitcoin SV",
		LitecoinChainId:        "Litecoin",
		DecredChainId:          "Decred",
		DogecoinChainId:        "Dogecoin",
		DashChainId:            "Dash",
		ZcashChainId:           "Zcash",
		AvalancheXChainId:      "Avalanche X-Chain",
		AvalancheCChainId:      "Avalanche C-Chain",
		MarsChainChainId:       "MarsChain",
		MoneroChainId:          "Monero",
		NEMChainId:             "NEM",
		HorizenChainId:         "Horizen",
		MassGridChainId:        "MassGrid",
		BytomChainId:           "Bytom",
		BytomPoSChainId:        "Bytom",
		TRONChainId:            "TRON",
		TONChainId:             "TON",
		StellarChainId:         "Stellar",
		CosmosChainId:          "Cosmos",
		StarcoinChainId:        "Starcoin",
		AkashChainId:           "Akash",
		BinanceChainId:         "BNB Beacon Chain",
		BitSharesChainId:       "Bitshares",
		TezosChainId:           "Tezos",
		RavencoinChainId:       "Ravencoin",
		NamecoinChainId:        "Namecoin",
		NervosChainId:          "Nervos",
		GrinChainId:            "Grin",
		VCashChainId:           "VCash",
		FilecoinChainId:        "Filecoin",
		PolkadotChainId:        "Polkadot",
		KusamaChainId:          "Kusama",
		ArweaveChainId:         "Arweave",
		MobileCoinChainId:      "MobileCoin",
		SolanaChainId:          "Solana",
		NearChainId:            "NEAR",
		AlgorandChainId:        "Algorand",
		XDCChainId:             "XDC Network",
		AptosChainId:           "Aptos",
		SuiChainId:             "Sui",
		PearlChainId:           "Pearl",
	}
	for chainID, want := range names {
		assert.Equal(t, want, GetChainName(chainID), chainID)
		assert.True(t, IsChainId(chainID), chainID)
	}
	assert.Equal(t, "Not Supported Chain", GetChainName("unknown"))
	assert.False(t, IsChainId("unknown"))

	full := GetFullChains()
	for chainID := range full {
		assert.True(t, IsChainId(chainID), chainID)
	}
	full["unknown"] = true
	assert.False(t, IsChainId("unknown"), "GetFullChains must return an independent map")
}
