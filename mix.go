package bot

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/util/base58"
	"github.com/gofrs/uuid/v5"
)

const (
	MixAddressPrefix  = "MIX"
	MixAddressVersion = byte(2)
)

type MixAddress struct {
	Version     byte
	Threshold   byte
	uuidMembers []uuid.UUID
	xinMembers  []*common.Address
}

func NewUUIDMixAddress(members []string, threshold byte) *MixAddress {
	if len(members) > 255 {
		panic(len(members))
	}
	if int(threshold) == 0 || int(threshold) > len(members) {
		panic(threshold)
	}
	ma := &MixAddress{
		Version:   MixAddressVersion,
		Threshold: threshold,
	}
	for _, s := range members {
		u := uuid.Must(uuid.FromString(s))
		ma.uuidMembers = append(ma.uuidMembers, u)
	}
	return ma
}

func NewMainnetMixAddress(members []string, threshold byte) *MixAddress {
	if len(members) > 255 {
		panic(len(members))
	}
	if int(threshold) == 0 {
		panic(threshold)
	}
	ma := &MixAddress{
		Version:   MixAddressVersion,
		Threshold: threshold,
	}
	for _, s := range members {
		a, err := common.NewAddressFromString(s)
		if err != nil {
			panic(s)
		}
		ma.xinMembers = append(ma.xinMembers, &a)
	}
	return ma
}

func (ma *MixAddress) Members() []string {
	var members []string
	if len(ma.uuidMembers) > 0 {
		for _, u := range ma.uuidMembers {
			members = append(members, u.String())
		}
	} else {
		for _, a := range ma.xinMembers {
			members = append(members, a.String())
		}
	}
	return members
}

func (ma *MixAddress) String() string {
	payload := []byte{ma.Version, ma.Threshold}
	if l := len(ma.uuidMembers); l > 0 {
		if l > 255 {
			panic(l)
		}
		payload = append(payload, byte(l))
		for _, u := range ma.uuidMembers {
			payload = append(payload, u.Bytes()...)
		}
	} else {
		l := len(ma.xinMembers)
		if l > 255 {
			panic(l)
		}
		payload = append(payload, byte(l))
		for _, a := range ma.xinMembers {
			payload = append(payload, a.PublicSpendKey[:]...)
			payload = append(payload, a.PublicViewKey[:]...)
		}
	}

	data := append([]byte(MixAddressPrefix), payload...)
	checksum := crypto.Sha256Hash(data)
	payload = append(payload, checksum[:4]...)
	return MixAddressPrefix + base58.Encode(payload)
}

func NewMixAddressFromString(s string) (*MixAddress, error) {
	var ma MixAddress
	if !strings.HasPrefix(s, MixAddressPrefix) {
		return nil, fmt.Errorf("invalid address prefix %s", s)
	}
	data := base58.Decode(s[len(MixAddressPrefix):])
	if len(data) < 3+16+4 {
		return nil, fmt.Errorf("invalid address length %d", len(data))
	}
	payload := data[:len(data)-4]
	checksum := crypto.Sha256Hash(append([]byte(MixAddressPrefix), payload...))
	if !bytes.Equal(checksum[:4], data[len(data)-4:]) {
		return nil, fmt.Errorf("invalid address checksum %x", checksum[:4])
	}

	total := payload[2]
	ma.Version = payload[0]
	ma.Threshold = payload[1]
	if ma.Version != MixAddressVersion {
		return nil, fmt.Errorf("invalid address version %d", ma.Version)
	}
	if ma.Threshold == 0 || total > 64 {
		return nil, fmt.Errorf("invalid address threshold %d/%d", ma.Threshold, total)
	}

	mp := payload[3:]
	if len(mp) == 16*int(total) {
		for i := 0; i < int(total); i++ {
			id, err := uuid.FromBytes(mp[i*16 : i*16+16])
			if err != nil {
				return nil, fmt.Errorf("invalid uuid member %s", s)
			}
			ma.uuidMembers = append(ma.uuidMembers, id)
		}
	} else if len(mp) == 64*int(total) {
		for i := 0; i < int(total); i++ {
			var a common.Address
			copy(a.PublicSpendKey[:], mp[i*64:i*64+32])
			copy(a.PublicViewKey[:], mp[i*64+32:i*64+64])
			ma.xinMembers = append(ma.xinMembers, &a)
		}
	} else {
		return nil, fmt.Errorf("invalid address members list %s", s)
	}

	return &ma, nil
}

func (ma *MixAddress) RequestOrGenerateGhostKeys(ctx context.Context, outputIndex uint, u *SafeUser) (*GhostKeys, error) {
	if len(ma.xinMembers) > 0 {
		seed := make([]byte, 64)
		crypto.ReadRand(seed)
		r := crypto.NewKeyFromSeed(seed)
		gkr := &GhostKeys{
			Mask: r.Public().String(),
			Keys: make([]string, len(ma.xinMembers)),
		}
		for i, a := range ma.xinMembers {
			k := crypto.DeriveGhostPublicKey(&r, &a.PublicViewKey, &a.PublicSpendKey, uint64(outputIndex))
			gkr.Keys[i] = k.String()
		}
		return gkr, nil
	}

	hint := uuid.Must(uuid.NewV4()).String()
	gkr := &GhostKeyRequest{
		Receivers: ma.Members(),
		Index:     outputIndex,
		Hint:      hint,
	}
	gks, err := RequestSafeGhostKeys(ctx, []*GhostKeyRequest{gkr}, u)
	if err != nil {
		return nil, err
	}
	return gks[0], nil
}
