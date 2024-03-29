package bot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

var (
	DefaultApiHost   = "https://api.mixin.one"
	DefaultBlazeHost = "blaze.mixin.one"

	httpClient *http.Client
	httpUri    string
	blazeUri   string
	userAgent  = "Bot-API-Go-Client"

	uid        string
	sid        string
	privateKey string
)

func Request(ctx context.Context, method, path string, body []byte, accessToken string) ([]byte, error) {
	return RequestWithId(ctx, method, path, body, accessToken, UuidNewV4().String())
}

func RequestWithId(ctx context.Context, method, path string, body []byte, accessToken, requestID string) ([]byte, error) {
	req, err := http.NewRequest(method, httpUri+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Request-Id", requestID)
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, errors.Wrap(ServerError(ctx, nil), fmt.Sprintf("response status code %d", resp.StatusCode))
	}
	return io.ReadAll(resp.Body)
}

func SimpleRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	transport, err := NewTransport(
		httpClient.Transport,
		uid,
		sid,
		privateKey,
	)
	if err != nil {
		return nil, err
	}
	httpClient.Transport = transport
	req, err := http.NewRequest(method, httpUri+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, errors.Wrap(ServerError(ctx, nil), fmt.Sprintf("response status code %d", resp.StatusCode))
	}
	return io.ReadAll(resp.Body)
}

func init() {
	httpClient = &http.Client{Timeout: 30 * time.Second}
	httpUri = DefaultApiHost
	blazeUri = DefaultBlazeHost
	if httpClient.Transport == nil {
		httpClient.Transport = http.DefaultTransport
	}
}

func WithAPIKey(userId, sessionId, p string) {
	uid = userId
	sid = sessionId
	privateKey = p
}

func SetBaseUri(base string) {
	httpUri = base
}

func SetBlazeUri(blaze string) {
	blazeUri = blaze
}

func SetUserAgent(ua string) {
	userAgent = ua
}
