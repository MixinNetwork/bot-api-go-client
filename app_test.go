package bot

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type appRoundTripFunc func(*http.Request) (*http.Response, error)

func (f appRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestUpdateAppUsesSafeUserID(t *testing.T) {
	originalClient := httpClient
	originalURI := httpUri
	t.Cleanup(func() {
		httpClient = originalClient
		httpUri = originalURI
	})

	appID := "11111111-1111-1111-1111-111111111111"
	var seed [ed25519.SeedSize]byte
	user := &SafeUser{
		UserId:            appID,
		SessionId:         "22222222-2222-2222-2222-222222222222",
		SessionPrivateKey: hex.EncodeToString(seed[:]),
	}
	input := &UpdateAppInput{
		RedirectURI:      "https://example.com/oauth",
		HomeURI:          "https://example.com",
		Name:             "Example App",
		Description:      "Example app description",
		IconBase64:       "base64-icon",
		Category:         "TOOLS",
		Capabilities:     []string{"CONTACT", "GROUP", "IMMERSIVE"},
		ResourcePatterns: []string{"https://example.com/*"},
	}

	httpUri = "https://api.test"
	httpClient = &http.Client{
		Transport: appRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != "POST" {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/apps/"+appID {
				t.Fatalf("path = %s, want /apps/%s", r.URL.Path, appID)
			}
			if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
				t.Fatalf("Authorization = %q, want bearer token", got)
			}

			var got UpdateAppInput
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if got.Name != input.Name || got.Description != input.Description || got.RedirectURI != input.RedirectURI || got.HomeURI != input.HomeURI || got.IconBase64 != input.IconBase64 || got.Category != input.Category {
				t.Fatalf("request body = %#v, want %#v", got, *input)
			}
			if !reflect.DeepEqual(got.Capabilities, input.Capabilities) || !reflect.DeepEqual(got.ResourcePatterns, input.ResourcePatterns) {
				t.Fatalf("request body = %#v, want %#v", got, *input)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"type":"app","app_id":"` + appID + `","name":"Example App","description":"Example app description"},"error":{}}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	app, err := UpdateApp(context.Background(), input, user)
	if err != nil {
		t.Fatal(err)
	}
	if app == nil || app.AppId != appID {
		t.Fatalf("app = %#v, want app_id %s", app, appID)
	}
}
