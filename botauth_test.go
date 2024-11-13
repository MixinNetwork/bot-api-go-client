package bot

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignRequest(t *testing.T) {
	assert := assert.New(t)
	keystore, err := read("./test_config.json")
	assert.Nil(err)
	su := keystore.BuildSafeUser()
	logger := slog.Default()
	ctx := context.Background()
	client := NewDefaultClient(su, logger)

	r, err := http.NewRequest(http.MethodGet, "https://test.com/account_addresses?chain_id=solana&user_id=2f73822c-56e8-4e5d-99a4-cdbf75cd", nil)
	if err != nil {
		t.Fatal(err)
	}
	ts := 1718102244
	s, err := client.SignRequest(ctx, int64(ts), "489cfe0b-08d8-47f4-a330-fff193cc8086", r)
	if err != nil {
		return
	}
	assert.Equal("YTgyZTFhNmEtNzVjOS00MDEzLTgwYmMtMTAxODNlZWY0OWEyBM-CNnETfQGwHzNh4x0N5JsxofbCoCpbc7jikoR7C-Y", s)
}

type keystore struct {
	AppID             string `json:"app_id"`
	SessionID         string `json:"session_id"`
	SessionPrivateKey string `json:"session_private_key"`
	ServerPublicKey   string `json:"server_public_key"`
	SpendPrivateKey   string `json:"pin"`
}

func (k *keystore) BuildSafeUser() *SafeUser {
	return &SafeUser{
		UserId:            k.AppID,
		SessionId:         k.SessionID,
		SessionPrivateKey: k.SessionPrivateKey,
		ServerPublicKey:   k.ServerPublicKey,
		SpendPrivateKey:   k.SpendPrivateKey,
	}
}

func read(path string) (*keystore, error) {
	if strings.HasPrefix(path, "~/") {
		usr, _ := user.Current()
		path = filepath.Join(usr.HomeDir, (path)[2:])
	}
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var keystore keystore
	err = json.Unmarshal(f, &keystore)
	return &keystore, err
}
