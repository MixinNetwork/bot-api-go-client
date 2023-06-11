package bot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	DefaultApiHost   = "https://api.mixin.one"
	DefaultBlazeHost = "blaze.mixin.one"

	ZeromeshApiHost   = "https://mixin-api.zeromesh.net"
	ZeromeshBlazeHost = "mixin-blaze.zeromesh.net"
	httpClient        *http.Client
	httpUri           string
	blazeUri          string

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
	resp, err := httpClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "Client.Timeout") {
			if httpUri == DefaultApiHost {
				httpUri = ZeromeshApiHost
			} else {
				httpUri = DefaultApiHost
			}
		}
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
		if strings.Contains(err.Error(), "Client.Timeout") {
			if httpUri == DefaultApiHost {
				httpUri = ZeromeshApiHost
			} else {
				httpUri = DefaultApiHost
			}
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, errors.Wrap(ServerError(ctx, nil), fmt.Sprintf("response status code %d", resp.StatusCode))
	}
	return io.ReadAll(resp.Body)
}

func init() {
	httpClient = &http.Client{Timeout: 10 * time.Second}
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
