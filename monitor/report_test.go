package monitor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	bot "github.com/MixinNetwork/bot-api-go-client/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReport(t *testing.T) {
	require := require.New(t)

	md := &MessageData{
		Name:  "bar",
		Value: "foo",
	}

	am := &AppMessage{
		Project: "rpc-bsc-p-30",
		Status:  200,
		Data: []*MessageData{
			md,
		},
	}

	buf, err := am.Marshal()
	require.Nil(err)
	require.Equal(73, len(buf))
}

func TestAppMessageRoundTrip(t *testing.T) {
	want := &AppMessage{
		Project: "service",
		Status:  503,
		Data:    []*MessageData{{Name: "latency", Value: "2s"}},
	}
	data, err := want.Marshal()
	require.NoError(t, err)
	got, err := UnmarshalAppMessage(data)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	_, err = UnmarshalAppMessage([]byte("status: ["))
	assert.Error(t, err)
}

func TestCheckRetryableError(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("request TIMEOUT"), true},
		{errors.New("Internal Server Error"), true},
		{errors.New("insufficient balance"), true},
		{errors.New("inputs locked by another request"), true},
		{errors.New("spent by other transaction"), true},
		{errors.New("forbidden"), false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, CheckRetryableError(tt.err))
	}
}

func TestReportToMonitorReturnsExistingTransaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasPrefix(r.URL.Path, "/safe/transactions/"))
		_, _ = w.Write([]byte(`{"data":{"request_id":"existing"}}`))
	}))
	defer server.Close()
	bot.SetBaseUri(server.URL)
	t.Cleanup(func() { bot.SetBaseUri(bot.DefaultApiHost) })

	transaction, err := ReportToMonitor(
		context.Background(),
		"asset",
		"1",
		"",
		[]string{"e95b1d4e-4d49-4ac3-9402-988804458adc"},
		1,
		&AppMessage{Project: "service", Status: 200},
		&bot.SafeUser{UserId: "c76310d8-c563-499e-9866-c61ae2cbee11"},
	)
	require.NoError(t, err)
	assert.Nil(t, transaction)
}
