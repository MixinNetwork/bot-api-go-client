package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type ParticipantSessionView struct {
	Type      string `json:"type"`
	UserId    string `json:"user_id"`
	SessionId string `json:"session_id"`
	PublicKey string `json:"public_key"`
}

type Participant struct {
	UserId    string    `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Conversation struct {
	ConversationId string    `json:"conversation_id"`
	CreatorId      string    `json:"creator_id"`
	Category       string    `json:"category"`
	Name           string    `json:"name"`
	IconURL        string    `json:"icon_url"`
	Announcement   string    `json:"announcement"`
	CreatedAt      time.Time `json:"created_at"`

	Participants        []Participant            `json:"participants"`
	ParticipantSessions []ParticipantSessionView `json:"participant_sessions"`
}

func CreateContactConversation(ctx context.Context, participantID, uid, sid, key string) (*Conversation, error) {
	participants := []Participant{
		{
			UserId: participantID,
		},
	}
	return CreateConversation(ctx, "CONTACT", UniqueConversationId(participantID, uid), "", "", participants, uid, sid, key)
}

func CreateConversation(ctx context.Context, category, conversationId string, name, announcement string, participants []Participant, uid, sid, key string) (*Conversation, error) {
	params, err := json.Marshal(map[string]interface{}{
		"category":        category,
		"conversation_id": conversationId,
		"name":            name,
		"announcement":    announcement,
		"participants":    participants,
	})
	if err != nil {
		return nil, err
	}
	if category == "CONTACT" {
		if len(participants) != 1 {
			return nil, fmt.Errorf("bad particpants members length %d", len(participants))
		}
	}
	accessToken, err := SignAuthenticationToken(uid, sid, key, "POST", "/conversations", string(params))
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", "/conversations", params, accessToken)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  Conversation `json:"data"`
		Error Error        `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Error.Code > 0 {
		if resp.Error.Code == 401 {
			return nil, AuthorizationError(ctx)
		} else if resp.Error.Code == 403 {
			return nil, ForbiddenError(ctx)
		}
		return nil, ServerError(ctx, resp.Error)
	}
	return &resp.Data, nil
}

func ConversationShow(ctx context.Context, conversationId string, accessToken string) (*Conversation, error) {
	body, err := Request(ctx, "GET", "/conversations/"+conversationId, nil, accessToken)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  Conversation `json:"data"`
		Error Error        `json:"error"`
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		if resp.Error.Code == 401 {
			return nil, AuthorizationError(ctx)
		} else if resp.Error.Code == 403 {
			return nil, ForbiddenError(ctx)
		}
		return nil, ServerError(ctx, resp.Error)
	}
	return &resp.Data, nil
}

func JoinConversation(ctx context.Context, conversationId, uid, sid, key string) (*Conversation, error) {
	path := fmt.Sprintf("/conversations/%s/join", conversationId)
	accessToken, err := SignAuthenticationToken(uid, sid, key, "POST", path, "")
	if err != nil {
		return nil, err
	}

	body, err := Request(ctx, "POST", path, nil, accessToken)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  Conversation `json:"data"`
		Error Error        `json:"error"`
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Error.Code > 0 {
		if resp.Error.Code == 401 {
			return nil, AuthorizationError(ctx)
		} else if resp.Error.Code == 403 {
			return nil, ForbiddenError(ctx)
		}
		return nil, ServerError(ctx, resp.Error)
	}
	return &resp.Data, nil
}
