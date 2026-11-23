package bot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/pkg/errors"
)

const defaultUserAgent = "Bot-API-Go-Client/v3.26.1"
const requestIDHeader = "X-Request-Id"

var (
	DefaultApiHost   = "https://api.mixin.one"
	DefaultBlazeHost = "blaze.mixin.one"

	httpClient           *http.Client
	httpUri              string
	blazeUri             string
	userAgent            = defaultUserAgent
	debug                = false
	userAgentWarningOnce sync.Once

	globalUser *SafeUser
)

func RequestWithUser(ctx context.Context, method, path string, body []byte, su *SafeUser) ([]byte, error) {
	if su == nil {
		return doUnsignedRequest(ctx, method, path, body, UuidNewV4().String())
	}
	return doSignedRequest(ctx, method, path, body, su)
}

func doSignedRequest(ctx context.Context, method, path string, body []byte, su *SafeUser) ([]byte, error) {
	return RequestWithId(ctx, method, path, body, UuidNewV4().String(), su)
}

// RequestWithId signs the request using requestID as both the JWT's jti and the
// X-Request-Id header.
func RequestWithId(ctx context.Context, method, path string, body []byte, requestID string, su *SafeUser) ([]byte, error) {
	req, err := newSignedRequestWithID(ctx, method, path, body, requestID, su)
	if err != nil {
		return nil, err
	}
	return doRequest(req)
}

func newSignedRequest(ctx context.Context, method, path string, body []byte, su *SafeUser) (*http.Request, error) {
	return newSignedRequestWithID(ctx, method, path, body, UuidNewV4().String(), su)
}

func newSignedRequestWithID(ctx context.Context, method, path string, body []byte, requestID string, su *SafeUser) (*http.Request, error) {
	req, err := newAPIRequest(ctx, method, path, body, requestID)
	if err != nil {
		return nil, err
	}
	if err := signRequestHeaders(req.Header, req.Method, req.URL.RequestURI(), body, su); err != nil {
		return nil, err
	}
	return req, nil
}

func doUnsignedRequest(ctx context.Context, method, path string, body []byte, requestID string) ([]byte, error) {
	req, err := newAPIRequest(ctx, method, path, body, requestID)
	if err != nil {
		return nil, err
	}
	return doRequest(req)
}

func validateRequestID(requestID string) error {
	if requestID == "" || strings.TrimSpace(requestID) != requestID {
		return errors.New("request ID must be nonempty and have no surrounding whitespace")
	}
	if !utf8.ValidString(requestID) || strings.IndexFunc(requestID, unicode.IsControl) >= 0 {
		return errors.New("request ID must be valid UTF-8 and contain no control characters")
	}
	return nil
}

func newAPIRequest(ctx context.Context, method, path string, body []byte, requestID string) (*http.Request, error) {
	if err := validateRequestID(requestID); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, httpUri+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(requestIDHeader, requestID)
	req.Header.Set("User-Agent", userAgent)
	return req, nil
}

func doRequest(req *http.Request) ([]byte, error) {
	requestID := req.Header.Get(requestIDHeader)
	if debug {
		log.Printf("Request: %s , path: %s, requestId: %s", req.Method, req.URL.RequestURI(), requestID)
	}
	warnDefaultUserAgent(userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, errors.Wrap(ServerError(req.Context(), nil), fmt.Sprintf("response status code %d %s", resp.StatusCode, requestID))
	}
	return io.ReadAll(resp.Body)
}

func SimpleRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	if globalUser == nil {
		return nil, errors.New("API key is not configured; call WithAPIKey first")
	}
	client := *httpClient
	transport, err := NewTransport(
		client.Transport,
		globalUser.UserId,
		globalUser.SessionId,
		globalUser.SessionPrivateKey,
	)
	if err != nil {
		return nil, err
	}
	client.Transport = transport
	req, err := http.NewRequestWithContext(ctx, method, httpUri+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	warnDefaultUserAgent(userAgent)
	resp, err := client.Do(req)
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
	httpClient = &http.Client{Timeout: 10 * time.Second}
	httpUri = DefaultApiHost
	blazeUri = DefaultBlazeHost
	if httpClient.Transport == nil {
		httpClient.Transport = http.DefaultTransport
	}
}

func WithAPIKey(userId, sessionId, p string) {
	globalUser = &SafeUser{
		UserId:            userId,
		SessionId:         sessionId,
		SessionPrivateKey: p,
	}
}

func SetHttpTimeout(timeout time.Duration) {
	httpClient.Timeout = timeout
}

func SetBaseUri(base string) {
	httpUri = base
}

func SetBlazeUri(blaze string) {
	blazeUri = blaze
}

// SetUserAgent sets the User-Agent for HTTP requests. Call it during startup,
// before making requests concurrently.
func SetUserAgent(ua string) {
	userAgent = ua
}

func warnDefaultUserAgent(ua string) {
	if ua == "" || ua == defaultUserAgent {
		userAgentWarningOnce.Do(func() {
			log.Print(`⚠️ bot-api-go-client: User-Agent is empty or uses the SDK default; set your own with bot.SetUserAgent("my-app/1.0")`)
		})
	}
}

func SetDebug(d bool) {
	debug = d
}
