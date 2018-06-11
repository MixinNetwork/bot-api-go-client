package bot

import (
	"context"
	"encoding/json"
)

type App struct {
	Type         string   `json:"type"`
	AppId        string   `json:"app_id"`
	AppNumber    string   `json:"app_number"`
	RedirectURI  string   `json:"redirect_uri"`
	HomeURI      string   `json:"home_uri"`
	Name         string   `json:"name"`
	IconURL      string   `json:"icon_url"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilites"`
	PlainSecret  string   `json:"app_secret"`
	CreatorId    string   `json:"creator_id"`
}

func GetApp(ctx context.Context, appId, clientId, sessionId, privateKey string) (App, error) {
	path := "/apps/" + appId
	accessToken, err := SignAuthenticationToken(clientId, sessionId, privateKey, "GET", path, "")
	if err != nil {
		return App{}, err
	}
	body, err := Request(ctx, "GET", path, nil, accessToken)
	if err != nil {
		return App{}, err
	}
	var resp struct {
		Data  App   `json:"data"`
		Error Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return App{}, err
	}
	return resp.Data, nil
}
