package bot

import (
	"context"
	"encoding/json"
	"fmt"
)

type TipNodeData struct {
	Commitments []string `json:"commitments"`
	Identity    string   `json:"identity"`
}

func GetTipNodeByPathWithRequestId(ctx context.Context, path, requestId string) (*TipNodeData, error) {
	url := fmt.Sprintf("/external/tip/%s", path)
	body, err := RequestWithId(ctx, "GET", url, nil, "", requestId)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  *TipNodeData `json:"data"`
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

func GetTipNodeByPath(ctx context.Context, path string) (*TipNodeData, error) {
	return GetTipNodeByPathWithRequestId(ctx, path, UuidNewV4().String())
}
