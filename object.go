package bot

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/mixin/common"
	"github.com/gofrs/uuid"
	"github.com/vmihailenco/msgpack/v4"
)

type ObjectInput struct {
	Amount  number.Decimal
	TraceId string
	Memo    string
}

func CreateObject(ctx context.Context, in *ObjectInput, uid, sid, sessionKey, pin, pinToken string) (*Snapshot, error) {
	if in.Amount.Exhausted() {
		return nil, fmt.Errorf("amount exhausted")
	}

	encryptedPIN, err := EncryptPIN(pin, pinToken, sid, sessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(map[string]interface{}{
		"amount":   in.Amount,
		"trace_id": in.TraceId,
		"memo":     in.Memo,
		"pin":      encryptedPIN,
	})
	if err != nil {
		return nil, err
	}

	path := "/objects"
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", path, string(data))
	if err != nil {
		return nil, err
	}

	id := UuidNewV4().String()
	body, err := RequestWithId(ctx, "POST", path, data, token, id)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data  *Snapshot `json:"data"`
		Error Error     `json:"error"`
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

func EstimateObjectFee(memo string) number.Decimal {
	extra := EncodeMixinExtra(uuid.Nil.String(), memo)
	return number.FromString(fmt.Sprint(len(extra)/1024 + 2)).Mul(number.FromString("0.0015"))
}

type MixinExtraPack struct {
	T uuid.UUID
	M string `msgpack:",omitempty"`
}

func EncodeMixinExtra(traceId, memo string) []byte {
	id, err := uuid.FromString(traceId)
	if err != nil {
		panic(err)
	}
	p := &MixinExtraPack{T: id, M: memo}
	b := MsgpackMarshalPanic(p)
	if len(b) >= common.ExtraSizeStorageCapacity {
		panic(memo)
	}
	return b
}

func MsgpackMarshalPanic(val interface{}) []byte {
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf).UseCompactEncoding(true).SortMapKeys(true)
	err := enc.Encode(val)
	if err != nil {
		panic(fmt.Errorf("MsgpackMarshalPanic: %#v %s", val, err.Error()))
	}
	return buf.Bytes()
}

func MsgpackUnmarshal(data []byte, val interface{}) error {
	err := msgpack.Unmarshal(data, val)
	if err == nil {
		return err
	}
	return fmt.Errorf("MsgpackUnmarshal: %s %s", hex.EncodeToString(data), err.Error())
}
