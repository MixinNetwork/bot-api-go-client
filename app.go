package bot

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type App struct {
	Type             string    `json:"type"`
	AppId            string    `json:"app_id"`
	AppNumber        string    `json:"app_number"`
	RedirectURI      string    `json:"redirect_uri"`
	HomeURI          string    `json:"home_uri"`
	Name             string    `json:"name"`
	IconURL          string    `json:"icon_url"`
	Description      string    `json:"description"`
	Capabilities     []string  `json:"capabilities"`
	ResourcePatterns []string  `json:"resource_patterns"`
	Category         string    `json:"category"`
	CreatorId        string    `json:"creator_id"`
	UpdatedAt        time.Time `json:"updated_at"`
	IsVerified       bool      `json:"is_verified"`
}

func Migrate(ctx context.Context, receiver, uid, sid, sessionKey, pin, pinToken string) (*App, error) {
	tipBody := TipBodyForOwnershipTransfer(receiver)

	pinBuf, err := hex.DecodeString(pin)
	if err != nil {
		return nil, err
	}
	sigBuf := ed25519.Sign(ed25519.PrivateKey(pinBuf), tipBody)
	encryptedPIN, err := EncryptEd25519PIN(hex.EncodeToString(sigBuf), pinToken, sessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}
	log.Println("data:", receiver, encryptedPIN, uid, sid, sessionKey)
	data, err := json.Marshal(map[string]string{
		"user_id":    receiver,
		"pin_base64": encryptedPIN,
	})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/apps/%s/transfer", uid)
	token, err := SignAuthenticationToken(uid, sid, sessionKey, "POST", path, string(data))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", path, data, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  *App  `json:"data"`
		Error Error `json:"error"`
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
