package bot

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestWithID(t *testing.T) {
	user, privateKey := newTestSafeUser(t.Name())
	requestBody := []byte(`{"hello":"world"}`)
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/resource%20name?order=desc", r.URL.RequestURI())
		assert.Equal(t, requestBody, body)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assertTestRequestAuthentication(t, r, body, user, privateKey.Public().(ed25519.PublicKey))
		assert.Equal(t, "request-id", r.Header.Get("X-Request-Id"))
		assert.Equal(t, defaultUserAgent, r.Header.Get("User-Agent"))
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	SetBaseUri(httpUri + "/api")
	body, err := RequestWithId(context.Background(), http.MethodPost, "/resource name?order=desc", requestBody, "request-id", user)
	require.NoError(t, err)
	assert.JSONEq(t, `{"ok":true}`, string(body))
}

func TestRequestWithIDRejectsInvalidID(t *testing.T) {
	user, _ := newTestSafeUser(t.Name())
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request with an invalid ID reached the server")
	}))
	for _, id := range []string{"", " ", " padded-id ", "invalid\xff", "bad\nheader"} {
		_, err := RequestWithId(context.Background(), http.MethodGet, "/resource", nil, id, user)
		require.ErrorContains(t, err, "request ID")
	}
	_, err := RequestWithId(context.Background(), http.MethodGet, "/resource", nil, "valid-id", nil)
	require.ErrorContains(t, err, "safe user is nil")
}

func TestRequestWithUser(t *testing.T) {
	user, privateKey := newTestSafeUser(t.Name())
	for _, tt := range []struct {
		name string
		user *SafeUser
	}{
		{name: "unsigned"},
		{name: "signed", user: user},
	} {
		t.Run(tt.name, func(t *testing.T) {
			type receivedRequest struct {
				request *http.Request
				body    []byte
				err     error
			}
			requests := make(chan receivedRequest, 1)
			useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				requests <- receivedRequest{request: r, body: body, err: err}
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			globalUser, _ = newTestSafeUser(t.Name() + "/global")
			requestBody := []byte(`{"hello":"world"}`)
			const path = "/resource/a%2Fb?name=hello%20world&order=desc"
			var previousRequestID string
			for range 2 {
				body, err := RequestWithUser(context.Background(), http.MethodPost, path, requestBody, tt.user)
				require.NoError(t, err)
				assert.JSONEq(t, `{"ok":true}`, string(body))

				received := <-requests
				require.NoError(t, received.err)
				assert.Equal(t, http.MethodPost, received.request.Method)
				assert.Equal(t, path, received.request.URL.RequestURI())
				assert.Equal(t, requestBody, received.body)
				assert.Equal(t, "application/json", received.request.Header.Get("Content-Type"))
				assertTestRequestAuthentication(t, received.request, received.body, tt.user, privateKey.Public().(ed25519.PublicKey))
				requestID := received.request.Header.Get("X-Request-Id")
				id, err := uuid.FromString(requestID)
				require.NoError(t, err)
				assert.Equal(t, uuid.V4, id.Version())
				assert.NotEqual(t, previousRequestID, requestID)
				previousRequestID = requestID
			}
		})
	}
}

func TestRequestWithUserRejectsInvalidPrivateKey(t *testing.T) {
	for _, key := range []string{"", "not-hex", "00"} {
		t.Run("key="+key, func(t *testing.T) {
			oldClient := httpClient
			t.Cleanup(func() { httpClient = oldClient })
			calls := 0
			httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: r}, nil
			})}
			user := &SafeUser{SessionPrivateKey: key}
			body, err := RequestWithUser(context.Background(), http.MethodGet, "/resource", nil, user)
			require.ErrorContains(t, err, "invalid ed25519 private key")
			assert.Nil(t, body)
			assert.Zero(t, calls, "signing failures must not send an unauthenticated request")
		})
	}
}

func TestRequestHonorsContextCancellation(t *testing.T) {
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	user, _ := newTestSafeUser(t.Name())
	_, err := RequestWithId(ctx, http.MethodGet, "/blocked", nil, "request-id", user)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRequestReturnsServerErrorFor5xx(t *testing.T) {
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))

	user, _ := newTestSafeUser(t.Name())
	_, err := RequestWithId(context.Background(), http.MethodGet, "/resource", nil, "known-id", user)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "response status code 503 known-id")
	assert.Contains(t, err.Error(), `"code":500`)
}

func TestSimpleRequestSignsRequestURIWithoutMutatingClient(t *testing.T) {
	user, privateKey := newTestSafeUser(t.Name())
	var calls int
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		tokenString := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			assert.Equal(t, jwt.SigningMethodEdDSA, token.Method)
			return privateKey.Public().(ed25519.PublicKey), nil
		})
		require.NoError(t, err)
		require.True(t, parsed.Valid)
		claims := parsed.Claims.(jwt.MapClaims)
		requestID := r.Header.Get("X-Request-Id")
		assert.NotEmpty(t, requestID)
		assert.Equal(t, requestID, claims["jti"])
		sum := sha256.Sum256(append([]byte(http.MethodPost+"/signed?part=two"), body...))
		assert.Equal(t, hex.EncodeToString(sum[:]), claims["sig"])
		assert.Equal(t, user.UserId, claims["uid"])
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	globalUser = user
	originalTransport := httpClient.Transport

	for range 2 {
		body, err := SimpleRequest(context.Background(), http.MethodPost, "/signed?part=two", []byte("payload"))
		require.NoError(t, err)
		assert.JSONEq(t, `{"ok":true}`, string(body))
	}
	assert.Equal(t, 2, calls)
	assert.Equal(t, originalTransport, httpClient.Transport, "SimpleRequest must not wrap the shared transport repeatedly")
}

func TestSimpleRequestRequiresAPIKey(t *testing.T) {
	oldUser := globalUser
	globalUser = nil
	t.Cleanup(func() { globalUser = oldUser })

	_, err := SimpleRequest(context.Background(), http.MethodGet, "/resource", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithAPIKey")
}

func TestTransportPreservesBodyAndOriginalRequest(t *testing.T) {
	user, privateKey := newTestSafeUser(t.Name())
	var forwarded *http.Request
	base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		forwarded = r
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("ok")),
			Request:    r,
		}, nil
	})
	transport, err := NewTransport(base, user.UserId, user.SessionId, user.SessionPrivateKey)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPut, "https://example.test/items?a=1", bytes.NewBufferString("body"))
	require.NoError(t, err)
	req.Header.Set("X-Original", "yes")
	req.Header["x-request-id"] = []string{"caller-request-id"}
	req.Header["authorization"] = []string{"stale-token"}

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Empty(t, req.Header.Get("Authorization"), "RoundTrip must not mutate caller headers")
	originalBody, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, "body", string(originalBody))
	require.NotNil(t, forwarded)
	assert.NotContains(t, forwarded.Header, "x-request-id")
	assert.NotContains(t, forwarded.Header, "authorization")
	forwardedBody, err := io.ReadAll(forwarded.Body)
	require.NoError(t, err)
	assert.Equal(t, "body", string(forwardedBody))

	tokenString := strings.TrimPrefix(forwarded.Header.Get("Authorization"), "Bearer ")
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return privateKey.Public().(ed25519.PublicKey), nil
	})
	require.NoError(t, err)
	claims := parsed.Claims.(jwt.MapClaims)
	assert.Equal(t, "caller-request-id", forwarded.Header.Get("X-Request-Id"))
	assert.Equal(t, "caller-request-id", claims["jti"])
	sum := sha256.Sum256([]byte(http.MethodPut + "/items?a=1" + "body"))
	assert.Equal(t, hex.EncodeToString(sum[:]), claims["sig"])

	req.Header.Set("X-Request-Id", "different-id")
	forwarded = nil
	_, err = transport.RoundTrip(req)
	require.ErrorContains(t, err, "conflicting request ID")
	assert.Nil(t, forwarded)
}

func TestTransportHandlesNilAndUnreadableBodies(t *testing.T) {
	user, privateKey := newTestSafeUser(t.Name())
	called := false
	base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		assert.Equal(t, http.MethodGet, r.Method)
		assertTestRequestAuthentication(t, r, nil, user, privateKey.Public().(ed25519.PublicKey))
		return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: http.NoBody, Request: r}, nil
	})
	transport, err := NewTransport(base, user.UserId, user.SessionId, user.SessionPrivateKey)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodGet, "https://example.test/items", nil)
	require.NoError(t, err)
	req.Method = "" // net/http sends an empty method as GET.
	_, err = transport.RoundTrip(req)
	require.NoError(t, err)
	assert.True(t, called)

	req.Body = readErrorCloser{}
	_, err = transport.RoundTrip(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read")
}

func TestTransportCancelsBlockedBodyRead(t *testing.T) {
	for _, cancellation := range []string{"already canceled", "during read", "client timeout"} {
		t.Run(cancellation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				user, _ := newTestSafeUser(t.Name())
				base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
					t.Error("canceled request must not reach the underlying transport")
					return &http.Response{StatusCode: 200, Body: http.NoBody, Request: r}, nil
				})
				transport, err := NewTransport(base, user.UserId, user.SessionId, user.SessionPrivateKey)
				require.NoError(t, err)
				client := &http.Client{Transport: transport}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				reader, writer := io.Pipe()
				defer writer.Close()
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://example.test/blocked", reader)
				require.NoError(t, err)
				if cancellation == "already canceled" {
					cancel()
				}
				if cancellation == "client timeout" {
					client.Timeout = time.Second
				}
				done := make(chan error, 1)
				go func() {
					resp, err := client.Do(req)
					if resp != nil {
						_ = resp.Body.Close()
					}
					done <- err
				}()
				synctest.Wait()
				wantErr := context.Canceled
				if cancellation == "client timeout" {
					wantErr = context.DeadlineExceeded
					time.Sleep(time.Second)
				} else {
					cancel()
				}
				synctest.Wait()
				select {
				case err := <-done:
					assert.ErrorIs(t, err, wantErr)
				default:
					t.Error("request remained blocked reading the body after cancellation")
					_ = writer.Close()
					<-done
				}
			})
		})
	}
}

func TestRequestSetters(t *testing.T) {
	oldTimeout, oldURI, oldBlaze, oldAgent, oldDebug, oldUser := httpClient.Timeout, httpUri, blazeUri, userAgent, debug, globalUser
	t.Cleanup(func() {
		httpClient.Timeout, httpUri, blazeUri, userAgent, debug = oldTimeout, oldURI, oldBlaze, oldAgent, oldDebug
		globalUser = oldUser
	})

	SetHttpTimeout(3 * time.Second)
	SetBaseUri("https://api.example.test")
	SetBlazeUri("blaze.example.test")
	SetUserAgent("test-agent")
	SetDebug(true)
	WithAPIKey("user", "session", "private")
	assert.Equal(t, 3*time.Second, httpClient.Timeout)
	assert.Equal(t, "https://api.example.test", httpUri)
	assert.Equal(t, "blaze.example.test", blazeUri)
	assert.Equal(t, "test-agent", userAgent)
	assert.True(t, debug)
	assert.Equal(t, "user", globalUser.UserId)
}

type readErrorCloser struct{}

func (readErrorCloser) Read([]byte) (int, error) { return 0, errors.New("cannot read") }
func (readErrorCloser) Close() error             { return nil }
