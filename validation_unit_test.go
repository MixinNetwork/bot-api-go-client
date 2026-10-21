package bot

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyPINTipReturnsSigningError(t *testing.T) {
	user, _ := newTestSafeUser(t.Name())
	user.SpendPrivateKey = "invalid"
	_, err := VerifyPINTip(context.Background(), user)
	require.Error(t, err)
}

func TestRequestOrGenerateGhostKeysRejectsEmptyResponse(t *testing.T) {
	user, _ := newTestSafeUser(t.Name())
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	address := NewUUIDMixAddress([]string{"e95b1d4e-4d49-4ac3-9402-988804458adc"}, 1)
	_, err := address.RequestOrGenerateGhostKeys(context.Background(), 0, user)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no ghost keys")
}

func TestGenerateUserChecksumDoesNotReorderSessions(t *testing.T) {
	sessions := []*Session{{SessionID: "z"}, {SessionID: "a"}}
	original := slices.Clone(sessions)
	first := GenerateUserChecksum(sessions)
	second := GenerateUserChecksum([]*Session{{SessionID: "a"}, {SessionID: "z"}})
	assert.Equal(t, first, second)
	assert.Equal(t, original, sessions)
	assert.Empty(t, GenerateUserChecksum(nil))
}
