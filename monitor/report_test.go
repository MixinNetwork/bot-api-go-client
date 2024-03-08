package monitor

import (
	"testing"

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
