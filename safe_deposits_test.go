package bot

import (
	"context"
	"log"
	"testing"
)

func TestFetchSafeDeposit(t *testing.T) {
	requireLiveAPI(t)
	pending, err := FetchPendingSafeDeposits(context.Background())
	log.Println(err)
	log.Println(len(pending))
}
