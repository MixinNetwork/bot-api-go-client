package bot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

type transport struct {
	roundTripper  http.RoundTripper
	authenticator *Authenticator
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
		roundTripper:  roundTripper,
		authenticator: NewAuthenticator(uid, sid, privateKey),
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
	if req.Body != nil {
		clone.Body = io.NopCloser(bytes.NewReader(body))
	}
	jwt, err := t.authenticator.BuildJWT(
		clone.Method, clone.URL.RequestURI(), string(body),
	)
	if err != nil {
		return nil, err
	}
	clone.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))
	return t.roundTripper.RoundTrip(clone)
}
