package bot

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"io"
	"strings"
)

type LiveMessagePayload struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	ThumbUrl string `json:"thumb_url"`
	Url      string `json:"url"`
}

type RecallMessagePayload struct {
	MessageId string `json:"message_id"`
}

type MessageRequest struct {
	ConversationId   string `json:"conversation_id"`
	RecipientId      string `json:"recipient_id"`
	MessageId        string `json:"message_id"`
	Category         string `json:"category"`
	Data             string `json:"data"`
	RepresentativeId string `json:"representative_id"`
	QuoteMessageId   string `json:"quote_message_id"`
}

func PostMessages(ctx context.Context, messages []*MessageRequest, clientId, sessionId, secret string) error {
	msg, err := json.Marshal(messages)
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

func PostMessage(ctx context.Context, conversationId, recipientId, messageId, category, data string, clientId, sessionId, secret string) error {
	request := MessageRequest{
		ConversationId: conversationId,
		RecipientId:    recipientId,
		MessageId:      messageId,
		Category:       category,
		Data:           data,
	}
	return PostMessages(ctx, []*MessageRequest{&request}, clientId, sessionId, secret)
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
	id, _ := UuidFromBytes(sum)
	return id.String()
}

func Chunked(source []interface{}, size int) [][]interface{} {
	var result [][]interface{}
	index := 0
	for index < len(source) {
		end := index + size
		if end >= len(source) {
			end = len(source)
		}
		result = append(result, source[index:end])
		index += size
	}
	return result
}
