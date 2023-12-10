package bot

import (
	"context"
	"encoding/json"

	"github.com/MixinNetwork/mixin/crypto"
)

type SafeUser struct {
	// this is the app or bot uuid of messenger api
	// user id can never change
	UserId string

	// session id could be rotated by the app owner
	SessionId string

	// session private key rotates with the session id
	// this key is used for all authentication of messenger api
	SessionPrivateKey string

	// server public key rotates with the session id
	// server public key is used to verify signature of server response
	// could also be used to do ecdh with session private key
	ServerPublicKey string

	// spend private key is used to query or send money
	// this is the mixin kernel spend private key
	// spend private key can never change
	SpendPrivateKey string
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
