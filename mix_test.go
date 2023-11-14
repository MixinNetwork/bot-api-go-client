package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUUIDMixAddress(t *testing.T) {
	require := require.New(t)

	members := []string{"67a87828-18f5-46a1-b6cc-c72a97a77c43"}
	ma := NewUUIDMixAddress(members, 1)
	require.Equal("MIX3QEeg1WkLrjvjxyMQf6Xc8dxs81tpPc", ma.String())
	ma, err := NewMixAddressFromString("MIX3QEeg1WkLrjvjxyMQf6Xc8dxs81tpPc")
	require.Nil(err)
	require.Equal(members, ma.Members())
	require.Equal(byte(1), ma.Threshold)
	require.Equal(byte(2), ma.Version)

	members = []string{
		"67a87828-18f5-46a1-b6cc-c72a97a77c43",
		"c94ac88f-4671-3976-b60a-09064f1811e8",
		"c6d0c728-2624-429b-8e0d-d9d19b6592fa",
		"67a87828-18f5-46a1-b6cc-c72a97a77c43",
		"c94ac88f-4671-3976-b60a-09064f1811e8",
		"c6d0c728-2624-429b-8e0d-d9d19b6592fa",
		"67a87828-18f5-46a1-b6cc-c72a97a77c43",
	}
	ma = NewUUIDMixAddress(members, 4)
	require.Equal("MIX4fwusRK88p5GexHWddUQuYJbKMJTAuBvhudgahRXKndvaM8FdPHS2Hgeo7DQxNVoSkKSEDyZeD8TYBhiwiea9PvCzay1A9Vx1C2nugc4iAmhwLGGv4h3GnABeCXHTwWEto9wEe1MWB49jLzy3nuoM81tqE2XnLvUWv", ma.String())
	ma, err = NewMixAddressFromString("MIX4fwusRK88p5GexHWddUQuYJbKMJTAuBvhudgahRXKndvaM8FdPHS2Hgeo7DQxNVoSkKSEDyZeD8TYBhiwiea9PvCzay1A9Vx1C2nugc4iAmhwLGGv4h3GnABeCXHTwWEto9wEe1MWB49jLzy3nuoM81tqE2XnLvUWv")
	require.Nil(err)
	require.Equal(members, ma.Members())
	require.Equal(byte(4), ma.Threshold)
	require.Equal(byte(2), ma.Version)
}

func TestMainnetMixAddress(t *testing.T) {
	require := require.New(t)

	members := []string{"XIN3BMNy9pQyj5XWDJtTbaBVE2zQ66zBo2weyc43iL286asdqwApWswAzQC5qba26fh3fzHK9iMoxyx1q3Lgj45KJftzGD9q"}
	ma := NewMainnetMixAddress(members, 1)
	require.Equal("MIXPYWwhjxKsbFRzAP2Dcb2mMjj7sQQo4MpCSv3NYaYCdQ2kEcbcimpPT81gaxtuNhunLWPx7Sv7fawjZ8DhRmEj8E2hrQM4Z6e", ma.String())
	ma, err := NewMixAddressFromString("MIXPYWwhjxKsbFRzAP2Dcb2mMjj7sQQo4MpCSv3NYaYCdQ2kEcbcimpPT81gaxtuNhunLWPx7Sv7fawjZ8DhRmEj8E2hrQM4Z6e")
	require.Nil(err)
	require.Equal(members, ma.Members())
	require.Equal(byte(1), ma.Threshold)
	require.Equal(byte(2), ma.Version)

	members = []string{
		"XINGNzunRUMmKGqDhnf1MT8tR7ek6ozg2V6dXFHCCg3tndnSRcAdzET8Fw4ktcQKshzteDmyV2RE8aFiKPz8ewrvsj3s7fvC",
		"XINMd9kCbxEoEetZuDM8gGJS11X3TVrRLwzhnqgMr65qjJBkCncNqSAngESpC7Hddnsw1D9Jo2QJakbFPr8WyrM6VkskGkB8",
		"XINLM7VuMYSjvKiEQPyLpaG7NDLDPngWWFBZpVJjhGamMsgPbmeSsGs3fQzNoqSr6syBTyLM3i69T7iSN8Tru7aQadiKLkSV",
	}
	ma = NewMainnetMixAddress(members, 2)
	require.Equal("MIXBCirWksVv9nuphqbtNRZZvwKsXHHMUnB5hVrVY1P7f4eBdLpDoLwiQoHYPvXia2wFepnX6hJwTjHybzBiroWVEMaFHeRFfLpcU244tzRM8smak9iRAD4PJRHN1MLHRWFtErottp9t7piaRVZBzsQXpSsaSgagj93voQdUuXhuQGZNj3Fme5YYMHfJBWjoRFHis4mnhBgxkyEGRUHAVYnfej2FhrypJmMDu74irRTdj2xjQYr6ovBJSUBYDBcvAyLPE3cEKc4JsPz7b9", ma.String())
	ma, err = NewMixAddressFromString("MIXBCirWksVv9nuphqbtNRZZvwKsXHHMUnB5hVrVY1P7f4eBdLpDoLwiQoHYPvXia2wFepnX6hJwTjHybzBiroWVEMaFHeRFfLpcU244tzRM8smak9iRAD4PJRHN1MLHRWFtErottp9t7piaRVZBzsQXpSsaSgagj93voQdUuXhuQGZNj3Fme5YYMHfJBWjoRFHis4mnhBgxkyEGRUHAVYnfej2FhrypJmMDu74irRTdj2xjQYr6ovBJSUBYDBcvAyLPE3cEKc4JsPz7b9")
	require.Nil(err)
	require.Equal(members, ma.Members())
	require.Equal(byte(2), ma.Threshold)
	require.Equal(byte(2), ma.Version)
}
