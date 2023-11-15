package bot

import (
	"context"
	"encoding/json"

	"github.com/MixinNetwork/mixin/crypto"
)

type SafeUser struct {
	UserId     string
	SessionId  string
	SessionKey string
	UserKey    string
	SpendKey   string
}

type GhostKeys struct {
	Type string   `json:"type"`
	Mask string   `json:"mask"`
	Keys []string `json:"keys"`
}

type GhostKeyRequest struct {
	Receivers []string `json:"receivers"`
	Index     uint     `json:"index"`
	Hint      string   `json:"hint"`
}

func RequestSafeGhostKeys(ctx context.Context, gkr []*GhostKeyRequest, uid, sid, sessionKey string) ([]*GhostKeys, error) {
	data, err := json.Marshal(gkr)
	if err != nil {
		return nil, err
	}
	method, path := "POST", "/safe/keys"
	token, err := SignAuthenticationToken(uid, sid, sessionKey, method, path, string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, method, path, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*GhostKeys `json:"data"`
		Error Error        `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}

func (g GhostKeys) KeysSlice() []*crypto.Key {
	keys := make([]*crypto.Key, len(g.Keys))
	for i, k := range g.Keys {
		key, err := crypto.KeyFromString(k)
		if err != nil {
			panic(k)
		}
		keys[i] = &key
	}
	return keys
}
