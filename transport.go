package bot

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
)

type transport struct {
	roundTripper http.RoundTripper
	user         SafeUser
}

// NewTransport returns a new transport based on the given inputs.
func NewTransport(
	roundTripper http.RoundTripper,
	uid,
	sid,
	privateKey string) (*transport, error) {
	if roundTripper == nil {
		roundTripper = http.DefaultTransport
	}
	return &transport{
		roundTripper: roundTripper,
		user:         SafeUser{UserId: uid, SessionId: sid, SessionPrivateKey: privateKey},
	}, nil
}

// RoundTrip implements the http.RoundTripper interface and wraps
// the base round tripper with logic to inject the API key auth-based HTTP headers
// into the request. Reference: https://pkg.go.dev/net/http#RoundTripper
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if err := ctx.Err(); err != nil {
		if req.Body != nil {
			_ = req.Body.Close()
		}
		return nil, err
	}
	var body []byte
	if req.Body != nil {
		originalBody := req.Body
		// Signing reads the body before the base transport can handle
		// cancellation. Closing it here also unblocks streaming bodies.
		stop := context.AfterFunc(ctx, func() { _ = originalBody.Close() })
		var err error
		body, err = io.ReadAll(originalBody)
		if stop() {
			_ = originalBody.Close()
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err != nil {
			return nil, err
		}
		// Keep the original request reusable if a caller needs to retry it.
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	clone := req.Clone(req.Context())
	if clone.Method == "" {
		clone.Method = http.MethodGet
	}
	if req.Body != nil {
		clone.Body = io.NopCloser(bytes.NewReader(body))
	}
	if clone.Header == nil {
		clone.Header = make(http.Header)
	}
	if err := prepareSigningHeaders(clone.Header); err != nil {
		return nil, err
	}
	if err := signRequestHeaders(clone.Header, clone.Method, clone.URL.RequestURI(), body, &t.user); err != nil {
		return nil, err
	}
	return t.roundTripper.RoundTrip(clone)
}

func prepareSigningHeaders(header http.Header) error {
	// A caller may populate the header map directly with noncanonical keys.
	// Send exactly one request ID and one Authorization header.
	var requestID string
	for name, values := range header {
		switch {
		case strings.EqualFold(name, requestIDHeader):
			for _, value := range values {
				if value == "" {
					continue
				}
				if requestID != "" && requestID != value {
					return errors.New("conflicting request ID headers")
				}
				requestID = value
			}
			delete(header, name)
		case strings.EqualFold(name, "Authorization"):
			delete(header, name)
		}
	}
	if requestID == "" {
		requestID = UuidNewV4().String()
	}
	header.Set(requestIDHeader, requestID)
	return nil
}
