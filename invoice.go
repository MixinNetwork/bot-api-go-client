package bot

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/gofrs/uuid/v5"
)

const (
	MixinInvoiceVersion = byte(0)
	MixinInvoicePrefix  = "MIN"
)

type MixinInvoice struct {
	version   byte
	recipient *MixAddress
	Entries   []*InvoiceEntry
}

type InvoiceEntry struct {
	traceId            uuid.UUID
	assetId            uuid.UUID
	amount             common.Integer
	extra              []byte
	indexReferences    []byte
	externalReferences []crypto.Hash
}

func NewMixinInvoice(recipient string) *MixinInvoice {
	r, err := NewMixAddressFromString(recipient)
	if err != nil {
		panic(recipient)
	}
	mi := &MixinInvoice{
		version:   MixinInvoiceVersion,
		recipient: r,
	}
	return mi
}

func (mi *MixinInvoice) AddEntry(traceId, assetId string, amount common.Integer, extra []byte, indexReferences []byte, externalReferences []crypto.Hash) {
	e := &InvoiceEntry{
		traceId:            uuid.Must(uuid.FromString(traceId)),
		assetId:            uuid.Must(uuid.FromString(assetId)),
		amount:             amount,
		extra:              extra,
		externalReferences: externalReferences,
	}
	if len(e.externalReferences)+len(indexReferences) > common.ReferencesCountLimit {
		panic("too many references")
	}
	for _, ir := range indexReferences {
		if int(ir) >= len(mi.Entries) {
			panic(len(mi.Entries))
		}
		e.indexReferences = append(e.indexReferences, ir)
	}
	mi.Entries = append(mi.Entries, e)
}

func (mi *MixinInvoice) BytesUnchecked() []byte {
	enc := common.NewEncoder()
	enc.Write([]byte{mi.version})
	rb := mi.recipient.BytesUnchecked()
	if len(rb) > 1024 {
		panic(len(rb))
	}
	enc.WriteUint16(uint16(len(rb)))
	enc.Write(rb)

	if len(mi.Entries) > 128 {
		panic(len(mi.Entries))
	}
	enc.Write([]byte{byte(len(mi.Entries))})

	for _, e := range mi.Entries {
		enc.Write(e.traceId.Bytes())
		enc.Write(e.assetId.Bytes())
		ab := len(e.amount.String())
		if ab > 128 {
			panic(e.amount.String())
		}
		enc.Write([]byte{byte(len(e.amount.String()))})
		enc.Write([]byte(e.amount.String()))
		if len(e.extra) >= common.ExtraSizeStorageCapacity {
			panic(len(e.extra))
		}
		enc.WriteInt(len(e.extra))
		enc.Write(e.extra)

		rl := len(e.indexReferences) + len(e.externalReferences)
		if rl > common.ReferencesCountLimit {
			panic(rl)
		}
		enc.Write([]byte{byte(rl)})
		for _, ir := range e.indexReferences {
			enc.Write([]byte{1, ir})
		}
		for _, er := range e.externalReferences {
			enc.Write([]byte{0})
			enc.Write(er[:])
		}
	}

	return enc.Bytes()
}

func (mi *MixinInvoice) String() string {
	payload := mi.BytesUnchecked()
	data := append([]byte(MixinInvoicePrefix), payload...)
	checksum := crypto.Sha256Hash(data)
	payload = append(payload, checksum[:4]...)
	return MixinInvoicePrefix + base64.RawURLEncoding.EncodeToString(payload)
}

func NewMixinInvoiceFromString(s string) (*MixinInvoice, error) {
	var mi MixinInvoice
	if !strings.HasPrefix(s, MixinInvoicePrefix) {
		return nil, fmt.Errorf("invalid invoice prefix %s", s)
	}
	data, err := base64.RawURLEncoding.DecodeString(s[len(MixinInvoicePrefix):])
	if err != nil {
		return nil, fmt.Errorf("invalid invoice base64 %v", err)
	}
	if len(data) < 3+23+1 {
		return nil, fmt.Errorf("invalid invoice length %d", len(data))
	}
	payload := data[:len(data)-4]
	checksum := crypto.Sha256Hash(append([]byte(MixinInvoicePrefix), payload...))
	if !bytes.Equal(checksum[:4], data[len(data)-4:]) {
		return nil, fmt.Errorf("invalid invoice checksum %x", checksum[:4])
	}

	dec := common.NewDecoder(payload)
	mi.version, err = dec.ReadByte()
	if err != nil || mi.version != MixinInvoiceVersion {
		return nil, fmt.Errorf("invalid invoice version %d %v", mi.version, err)
	}
	rbl, err := dec.ReadUint16()
	if err != nil {
		return nil, err
	}
	rb := make([]byte, rbl)
	err = dec.Read(rb)
	if err != nil {
		return nil, err
	}
	mi.recipient, err = NewMixAddressFromBytesUnchecked(rb)
	if err != nil {
		return nil, err
	}

	el, err := dec.ReadByte()
	if err != nil {
		return nil, err
	}
	for ; el > 0; el-- {
		var e InvoiceEntry
		b := make([]byte, 16)
		err = dec.Read(b)
		if err != nil {
			return nil, err
		}
		e.traceId = uuid.Must(uuid.FromBytes(b))
		err = dec.Read(b)
		if err != nil {
			return nil, err
		}
		e.assetId = uuid.Must(uuid.FromBytes(b))

		al, err := dec.ReadByte()
		if err != nil {
			return nil, err
		}
		b = make([]byte, al)
		err = dec.Read(b)
		if err != nil {
			return nil, err
		}
		e.amount = common.NewIntegerFromString(string(b))

		e.extra, err = dec.ReadBytes()
		if err != nil {
			return nil, err
		}

		rl, err := dec.ReadByte()
		if err != nil {
			return nil, err
		}
		if rl > common.ReferencesCountLimit {
			return nil, fmt.Errorf("too many references %d", rl)
		}
		for ; rl > 0; rl-- {
			rv, err := dec.ReadByte()
			if err != nil {
				return nil, err
			}
			switch rv {
			case 0:
				var b crypto.Hash
				err = dec.Read(b[:])
				if err != nil {
					return nil, err
				}
				e.externalReferences = append(e.externalReferences, b)
			case 1:
				ir, err := dec.ReadByte()
				if err != nil {
					return nil, err
				}
				if int(ir) >= len(mi.Entries) {
					return nil, fmt.Errorf("invalid reference index %d", ir)
				}
				e.indexReferences = append(e.indexReferences, ir)
			default:
				return nil, fmt.Errorf("invalid reference type %d", rv)
			}
		}
		mi.Entries = append(mi.Entries, &e)
	}
	return &mi, nil
}
