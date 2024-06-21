package bot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func testRequestWithdrawalFees(t *testing.T) {
	require := require.New(t)

	su := &SafeUser{
		UserId:            "",
		SessionId:         "",
		SessionPrivateKey: "",
	}
	fees, err := RequestWithdrawalFees(context.Background(), "c6d0c728-2624-429b-8e0d-d9d19b6592fa", su)
	require.Nil(err)
	require.Len(fees, 1)
}
