package bot

import (
	"context"
	"encoding/json"
)

type Attachment struct {
	Type         string `json:"type"`
	AttachmentId string `json:"attachment_id"`
	ViewURL      string `json:"view_url"`
	UploadUrl    string `json:"upload_url"`
}

func CreateAttachment(ctx context.Context, uid, sid, key string) (*Attachment, error) {
	token, err := SignAuthenticationToken(uid, sid, key, "POST", "/attachments", "")
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", "/attachments", nil, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  Attachment `json:"data"`
		Error Error      `json:"error"`
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
		return nil, resp.Error
	}
	return &resp.Data, nil
}

func AttachemntShow(ctx context.Context, uid, sid, key, id string) (*Attachment, error) {
	token, err := SignAuthenticationToken(uid, sid, key, "GET", "/attachments/"+id, "")
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "GET", "/attachments/"+id, nil, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  Attachment `json:"data"`
		Error Error      `json:"error"`
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
		return nil, resp.Error
	}
	return &resp.Data, nil
}
