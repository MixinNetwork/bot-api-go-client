package bot

import (
	"context"
	"crypto/ed25519"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"
)

type User struct {
	UserId         string `json:"user_id"`
	SessionId      string `json:"session_id"`
	PinToken       string `json:"pin_token"`
	PINTokenBase64 string `json:"pin_token_base64"`
	IdentityNumber string `json:"identity_number"`
	HasSafe        bool   `json:"has_safe"`
	TIPKeyBase64   string `json:"tip_key_base64"`
	Phone          string `json:"phone"`
	FullName       string `json:"full_name"`
	AvatarURL      string `json:"avatar_url"`
	DeviceStatus   string `json:"device_status"`
	CreatedAt      string `json:"created_at"`
}

const (
	RelationshipActionAdd     = "ADD"
	RelationshipActionUpdate  = "UPDATE"
	RelationshipActionRemove  = "REMOVE"
	RelationshipActionBlock   = "BLOCK"
	RelationshipActionUnblock = "UNBLOCK"

	PreferenceSourceAll      = "EVERYBODY"
	PreferenceSourceContacts = "CONTACTS"
	PreferenceSourceNobody   = "NOBODY"
)

func CreateUserSimple(ctx context.Context, sessionSecret, fullName string) (*User, error) {
	data, _ := json.Marshal(map[string]string{
		"session_secret": sessionSecret,
		"full_name":      fullName,
	})
	body, err := SimpleRequest(ctx, "POST", "/users", data)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *User `json:"data"`
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

func CreateUser(ctx context.Context, sessionSecret, fullName string, su *SafeUser) (*User, error) {
	data, _ := json.Marshal(map[string]string{
		"session_secret": sessionSecret,
		"full_name":      fullName,
	})
	token, err := SignAuthenticationToken("POST", "/users", string(data), su)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", "/users", data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *User `json:"data"`
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

func GetUser(ctx context.Context, userId string, su *SafeUser) (*User, error) {
	url := fmt.Sprintf("/users/%s", userId)
	token, err := SignAuthenticationToken("GET", url, "", su)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", url, nil, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *User `json:"data"`
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

func SearchUser(ctx context.Context, mixinId string, su *SafeUser) (*User, error) {
	url := fmt.Sprintf("/search/%s", mixinId)
	token, err := SignAuthenticationToken("GET", url, "", su)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", url, nil, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *User `json:"data"`
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

func UpdateTipPin(ctx context.Context, pin, pubTip string, su *SafeUser) error {
	oldEncryptedPin, err := EncryptEd25519PIN(pin, uint64(time.Now().UnixNano()), su)
	if err != nil {
		return err
	}

	pubTipBuf, err := hex.DecodeString(pubTip)
	if err != nil {
		return err
	}
	if len(pubTipBuf) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key")
	}

	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, 1)
	pubTipBuf = append(pubTipBuf, counter...)
	encryptedPin, err := EncryptEd25519PIN(hex.EncodeToString(pubTipBuf), uint64(time.Now().UnixNano()), su)
	if err != nil {
		return err
	}

	return UpdatePin(ctx, oldEncryptedPin, encryptedPin, su)
}

func UpdatePin(ctx context.Context, oldEncryptedPin, encryptedPin string, su *SafeUser) error {
	data, _ := json.Marshal(map[string]string{
		"old_pin_base64": oldEncryptedPin,
		"pin_base64":     encryptedPin,
	})

	token, err := SignAuthenticationToken("POST", "/pin/update", string(data), su)
	if err != nil {
		return err
	}
	body, err := Request(ctx, "POST", "/pin/update", data, token)
	if err != nil {
		return ServerError(ctx, err)
	}
	var resp struct {
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return ServerError(ctx, err)
	}
	if resp.Error.Code > 0 {
		return resp.Error
	}
	return nil
}

func UserMeWithRequestID(ctx context.Context, accessToken, requestID string) (*User, error) {
	body, err := RequestWithId(ctx, "GET", "/safe/me", nil, accessToken, requestID)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *User `json:"data"`
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

func UserMe(ctx context.Context, accessToken string) (*User, error) {
	return UserMeWithRequestID(ctx, accessToken, UuidNewV4().String())
}

func RequestUserMe(ctx context.Context, su *SafeUser) (*User, error) {
	path := "/safe/me"
	token, err := SignAuthenticationToken("GET", path, "", su)
	if err != nil {
		return nil, err
	}
	return UserMe(ctx, token)
}

func UpdateUserMe(ctx context.Context, fullName, avatarBase64 string, su *SafeUser) (*User, error) {
	data, err := json.Marshal(map[string]interface{}{
		"full_name":     fullName,
		"avatar_base64": avatarBase64,
	})
	if err != nil {
		return nil, err
	}

	path := "/me"
	token, err := SignAuthenticationToken("POST", path, string(data), su)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", path, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *User `json:"data"`
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

func UpdatePreference(ctx context.Context, messageSource, conversationSource, currency string, threshold float64, su *SafeUser) (*User, error) {
	data, err := json.Marshal(map[string]interface{}{
		"receive_message_source":          messageSource,
		"accept_conversation_source":      conversationSource,
		"fiat_currency":                   currency,
		"transfer_notification_threshold": threshold,
	})
	if err != nil {
		return nil, err
	}
	path := "/me/preferences"
	token, err := SignAuthenticationToken("POST", path, string(data), su)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", path, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *User `json:"data"`
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

func Relationship(ctx context.Context, userId, action string, su *SafeUser) (*User, error) {
	data, err := json.Marshal(map[string]interface{}{
		"user_id": userId,
		"action":  action,
	})
	if err != nil {
		return nil, err
	}

	path := "/relationships"
	token, err := SignAuthenticationToken("POST", path, string(data), su)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", path, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *User `json:"data"`
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

type Session struct {
	UserID    string
	SessionID string
	PublicKey string
}

func GenerateUserChecksum(sessions []*Session) string {
	if len(sessions) < 1 {
		return ""
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].SessionID < sessions[j].SessionID
	})
	h := md5.New()
	for _, s := range sessions {
		io.WriteString(h, s.SessionID)
	}
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:])
}
