package bot

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
)

var httpClient *http.Client

func Request(ctx context.Context, method, path string, body []byte, accessToken string) ([]byte, error) {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	req, err := http.NewRequest(method, "https://api.mixin.one"+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
