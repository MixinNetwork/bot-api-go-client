package bot

import (
	"context"
	"log"
	"testing"
)

func TestFetchSafeDeposit(t *testing.T) {
	pending, err := FetchSafeDeposit(context.Background())
	log.Println(err)
	log.Println(len(pending))
}