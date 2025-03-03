package bot

import (
	"testing"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/stretchr/testify/require"
)

func TestMixinInvoice(t *testing.T) {
	require := require.New(t)

	require.Equal("MIXSK624cFT3CXbbjYxU17CeYWCwj6CZgkp2VsfiRsDMXw4MzpfYKPKKYwLmfDby2z85MLAbSWZbAB1dfPetCxUf7vwwJnToaG8", StorageRecipient().String())
	require.Equal("0.00010000", EstimateStorageCost(make([]byte, 255)).String())
	require.Equal("0.00010000", EstimateStorageCost(make([]byte, 256)).String())
	require.Equal("0.00010000", EstimateStorageCost(make([]byte, 512)).String())
	require.Equal("0.00020000", EstimateStorageCost(make([]byte, 1024)).String())
	require.Equal("0.00020000", EstimateStorageCost(make([]byte, 1025)).String())

	recipient := "MIX4fwusRK88p5GexHWddUQuYJbKMJTAuBvhudgahRXKndvaM8FdPHS2Hgeo7DQxNVoSkKSEDyZeD8TYBhiwiea9PvCzay1A9Vx1C2nugc4iAmhwLGGv4h3GnABeCXHTwWEto9wEe1MWB49jLzy3nuoM81tqE2XnLvUWv"
	mi := NewMixinInvoice(recipient)

	trace1 := "772e6bef-3bff-4fcc-987d-29bafca74d63"
	amt1 := common.NewIntegerFromString("0.12345678")
	ref1, _ := crypto.HashFromString("7ecf9fc49ff4d2e36424b8e53e67aed8cc4e9d08d7cbdca7d8bdb153ed2fcdde")
	mi.AddEntry(trace1, BTC, amt1, []byte("extra one"), nil, []crypto.Hash{ref1})

	trace2 := "3552d116-b29d-4d72-9b24-3ca3b2e0f9c2"
	amt2 := common.NewIntegerFromString("0.23345678")
	ref2, _ := crypto.HashFromString("4a5f79c76872524c6a4a81b174338584e790f09fb059c39cf2a894de1b3c31c6")
	mi.AddEntry(trace2, ETH, amt2, []byte("extra two"), []byte{0}, []crypto.Hash{ref2})

	require.Equal("MINAABzAgQHZ6h4KBj1RqG2zMcql6d8Q8lKyI9GcTl2tgoJBk8YEejG0McoJiRCm44N2dGbZZL6Z6h4KBj1RqG2zMcql6d8Q8lKyI9GcTl2tgoJBk8YEejG0McoJiRCm44N2dGbZZL6Z6h4KBj1RqG2zMcql6d8QwJ3LmvvO_9PzJh9Kbr8p01jxtDHKCYkQpuODdnRm2WS-gowLjEyMzQ1Njc4AAlleHRyYSBvbmUBAH7Pn8Sf9NLjZCS45T5nrtjMTp0I18vcp9i9sVPtL83eNVLRFrKdTXKbJDyjsuD5wkPWHc3kE0UNgLgQHV6QM1cKMC4yMzM0NTY3OAAJZXh0cmEgdHdvAgEAAEpfecdoclJMakqBsXQzhYTnkPCfsFnDnPKolN4bPDHGTTpvYA", mi.String())

	mi, err := NewMixinInvoiceFromString(mi.String())
	require.Nil(err)
	require.Equal(MixinInvoiceVersion, mi.version)
	require.Equal(recipient, mi.Recipient.String())
	require.Len(mi.Entries, 2)

	e1 := mi.Entries[0]
	require.Equal(trace1, e1.TraceId.String())
	require.Equal(BTC, e1.AssetId.String())
	require.Equal(amt1, e1.Amount)
	require.Equal([]byte("extra one"), e1.Extra)
	require.Len(e1.IndexReferences, 0)
	require.Len(e1.HashReferences, 1)
	require.Equal(ref1, e1.HashReferences[0])

	e2 := mi.Entries[1]
	require.Equal(trace2, e2.TraceId.String())
	require.Equal(ETH, e2.AssetId.String())
	require.Equal(amt2, e2.Amount)
	require.Equal([]byte("extra two"), e2.Extra)
	require.Len(e2.IndexReferences, 1)
	require.Equal(byte(0), e2.IndexReferences[0])
	require.Len(e2.HashReferences, 1)
	require.Equal(ref2, e2.HashReferences[0])
}
