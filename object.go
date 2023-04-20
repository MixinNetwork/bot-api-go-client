package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MixinNetwork/go-number"
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
	body, err := Request(ctx, "POST", path, data, token)
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