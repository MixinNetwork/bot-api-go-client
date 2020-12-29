package main

import (
	"encoding/hex"
	"fmt"
	"math/rand"

	"github.com/MixinNetwork/mixin/common"
)

var nodes []string

func init() {
	nodes = []string{
		"http://mixin-node-01.b1.run:8239",
		"http://mixin-node-02.b1.run:8239",
		"http://mixin-node.matpool.io:8239",
		"http://mixin-node0.exinpool.com:8239",
		"http://mixin-node1.exinpool.com:8239",
	}
}
func main() {
	testTransfer()
}

func testTransfer() {
	hash := "348cf0672272159a121ecf28a0aea2a0fd10e7325a676a524ffd2ae9a898e51d"
	index := 0
	outputKeys := "7a7a74ff07b675f8fea6ae78671609e9369cdf05a9a4f0bb1c377a16991c97d4"
	outputMask := "234476f66621e4eb0d4efd8ee3dd93ec8282e7e4f0ce185331452a8b28dc8e24"

	raw := fmt.Sprintf(`{"version":1,"asset": "b9f49cf777dc4d03bc54cd1367eebca319f8603ea1ce18910d09e2c540c630d8","inputs":[{"hash":"%s","index":%d}],"outputs":[{"type":0,"amount":"100","script":"fffe01","keys":["%s"], "mask": "%s"}]}`, hash, index, outputKeys, outputMask)

	spend, err := hex.DecodeString("07b2d5ae306b8fc96d0b40e54b42592d63d786cd13bdbda4fda3eb958987d70b")
	if err != nil {
		fmt.Println(err)
		return
	}
	view, err := hex.DecodeString("a8e0fe81425bcad149f87ab082eb1442e0756269279722d44af7fe58ec19e70c")
	if err != nil {
		fmt.Println(err)
		return
	}
	var account common.Address
	copy(account.PrivateViewKey[:], view[:])
	copy(account.PrivateSpendKey[:], spend[:])

	tx, err := SignTransactionRaw(nodes[rand.Intn(len(nodes))], account, raw)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(tx)
	d := &common.VersionedTransaction{SignedTransaction: *tx}
	signedRaw := hex.EncodeToString(d.Marshal())
	fmt.Println(signedRaw)
}
