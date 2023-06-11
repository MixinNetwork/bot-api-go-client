package bot

import (
	"fmt"
	"io"
	"net/http"
	"strings"
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
	return &transport{
		roundTripper:  roundTripper,
		authenticator: NewAuthenticator(uid, sid, privateKey),
	}, nil
}

// RoundTrip implements the http.RoundTripper interface and wraps
// the base round tripper with logic to inject the API key auth-based HTTP headers
// into the request. Reference: https://pkg.go.dev/net/http#RoundTripper
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(strings.NewReader(string(body)))
	jwt, err := t.authenticator.BuildJWT(
		req.Method, req.URL.Path, string(body),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))
	return t.roundTripper.RoundTrip(req)
}
