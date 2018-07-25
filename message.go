package bot

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"io"
	"strings"
)

func PostMessage(ctx context.Context, conversationId, recipientId, messageId, category, data string, clientId, sessionId, secret string) error {
	msg, err := json.Marshal(map[string]interface{}{
		"conversation_id": conversationId,
		"recipient_id":    recipientId,
		"message_id":      messageId,
		"category":        category,
		"data":            data,
	})
	if err != nil {
		return err
	}
	accessToken, err := SignAuthenticationToken(clientId, sessionId, secret, "POST", "/messages", string(msg))
	if err != nil {
		return err
	}
	body, err := Request(ctx, "POST", "/messages", msg, accessToken)
	if err != nil {
		return err
	}
	var resp struct {
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}
	if resp.Error.Code > 0 {
		return resp.Error
	}
	return nil
}

func UniqueConversationId(userId, recipientId string) string {
	minId, maxId := userId, recipientId
	if strings.Compare(userId, recipientId) > 0 {
		maxId, minId = userId, recipientId
	}
	h := md5.New()
	io.WriteString(h, minId)
	io.WriteString(h, maxId)
	sum := h.Sum(nil)
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	return UuidFromBytes(sum)
}
