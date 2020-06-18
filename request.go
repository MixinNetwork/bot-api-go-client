package bot

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"time"
)

var httpClient *http.Client
var uri string

func Request(ctx context.Context, method, path string, body []byte, accessToken string) ([]byte, error) {
	req, err := http.NewRequest(method, uri+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, ServerError(ctx, nil)
	}
	return ioutil.ReadAll(resp.Body)
}

func init() {
	httpClient = &http.Client{Timeout: 10 * time.Second}
	uri = "https://mixin-api.zeromesh.net"
}

func SetBaseUri(base string) {
	uri = base
}
