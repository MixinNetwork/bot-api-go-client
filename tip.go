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

func GetTipNodeByPath(ctx context.Context, path string) (*TipNodeData, error) {
	url := fmt.Sprintf("/external/tip/%s", path)
	body, err := Request(ctx, "GET", url, nil, "")
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	fmt.Println(string(body))
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
