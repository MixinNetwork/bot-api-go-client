package bot

import (
	"encoding/hex"
	"fmt"
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

	invo := "MINAAAzAgIDFJ4rl_85SKOyVF61rP1FStO-4jqB1EYukCoi2unvif93ZrJMGgNMOoOjtDWCZoddA68hUS1a90SvgV7gdzg3MrDJSsiPRnE5drYKCQZPGBHoCjAuMDAwMTAwMDACxgIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIACAAYOD-20Pyt-fvn65OZKK1-RSEPJXu8gZ-_KrLbh0jaFEJwT1GeCFXYciRGTVSct_CZs-f9qAOJCk44SduqqIOak9CK9MzJgm_T-DerY739O2csRC-2sX4Ah-R1QbJ5qezfaGM-K4qQraxtxyK1pfO4T_D6IqgpNIJRL3J5jHPJoJP8FTya0QHGJWGr10Bmw4cKXSpftFg6TCbHHrOyb__WqJJbNqoLpnL5JqkvCV0yaXEdEVZAA6GP_a3o9UaEaXKbkiPnnJFYrMOeLTvnJC7fxsXgA54DiqQea9hWuXJt6knmfpXzSaeokrOc3paYZpr4neiwR9uueG1ehAnwyrk-D2gAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABqfVFxksVo7gioRfc9KXiM8DXDFFshqzRNgGLqlAAACl1cqeBM9dtZC3FLov4yyxWRM_wcGStyJX_QfTnLBAHoFuZmMMO7ck3Fnkn2zEMG5gOmqsygb6PjTitArVl52NB-xRj7VM8N76bY_aBtWBb5gJCRLIR8jhRBtHRGLwQtvwIiv1ImnqRpXf_4TVBvD8Xm9fWqZTeU2cwiMABVqW12S9lFAJ2VKSb3i00EZzskBHMmOB0ma0gbstS0Out45uAggDAgkABAQAAAAKDQELAwwNBAUGBw4OCA8g6ZLRjs9oQLyLUB0LcakAAAAAAAAAAAAAAAAAAAAAAAABGY8fTDpFImPUE7LNF-vLwaDliHNk5iYaEqgXkuoWWj4AAgIFACw8edJqGExsueOKHobsji2WXlxuQ0w_qbeAxQ9DzZVcCjAuMDAwMDAwMDEACWV4dHJhIHR3bwApHoteMAZECozQp_cdSU-_yUrIj0ZxOXa2CgkGTxgR6AowLjAwMDAwMDAyAA14aW4gZXh0cmEgdHdvAgEAAQGHdCqc"
	mi, err = NewMixinInvoiceFromString(invo)
	require.Nil(err)
	fmt.Println(hex.EncodeToString(mi.Entries[0].Extra))
}
