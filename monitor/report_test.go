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
		Score: 200,
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
	require.Equal(`p: rpc-bsc-p-30
s: 200
d:
    - "n": bar
      v: foo
      s: 200
`, string(buf))
	require.Equal(67, len(buf))
}
