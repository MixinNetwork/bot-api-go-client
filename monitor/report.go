package monitor

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/MixinNetwork/bot-api-go-client/v3"
	yaml "gopkg.in/yaml.v3"
)

const (
	AppMessageRunningPeriodShort = "p"
	AppMessageRunningTimeShort   = "t"

	AppMessageRunningPeriod = "period"
	AppMessageRunningTime   = "time"
)

type MessageData struct {
	Name  string `yaml:"name" json:"name"`
	Value string `yaml:"value" json:"value"`
}

// Tag: service name
// Running: period, time
// Duration: 0m 10m, 30m, 60m
// project example:
// 1. rpc-bsc|p|30|rpc
// 2. rpc-deposit|t|0|rpc
type AppMessage struct {
	Project string         `yaml:"project"`
	Status  int            `yaml:"status"`
	Data    []*MessageData `yaml:"data"`
}

func UnmarshalAppMessage(b []byte) (*AppMessage, error) {
	var m *AppMessage
	err := yaml.Unmarshal(b, &m)
	return m, err
}

func (m *AppMessage) Marshal() ([]byte, error) {
	b, err := yaml.Marshal(m)
	return b, err
}

func ReportToMonitor(ctx context.Context, asset, amount, trace string, receivers []string, threshold int, msg *AppMessage, u *bot.SafeUser) (*bot.SequencerTransactionRequest, error) {
	minutes := time.Now().UTC().Unix() / 60
	memo, err := msg.Marshal()
	if err != nil {
		return nil, err
	}
	ma := bot.NewUUIDMixAddress(receivers, byte(threshold))
	tr := &bot.TransactionRecipient{MixAddress: ma.String(), Amount: amount}
	if trace == "" {
		trace = bot.UniqueObjectId(ma.String(), asset, amount, u.UserId, hex.EncodeToString(memo), fmt.Sprint(minutes))
	}
	old, err := bot.GetTransactionById(ctx, trace)
	if err != nil || old != nil {
		return nil, err
	}
	return bot.SendTransaction(ctx, asset, []*bot.TransactionRecipient{tr}, trace, memo, nil, u)
}

func CheckRetryableError(err error) bool {
	if err == nil {
		return false
	}
	reason := strings.ToLower(err.Error())
	switch {
	case strings.Contains(reason, "timeout"):
	case strings.Contains(reason, "internal server"):
	case strings.Contains(reason, "insufficient"):
	case strings.Contains(reason, "inputs locked by"): // concurrent utxo query
	case strings.Contains(reason, "by other transaction"): // concurrent utxo query
	default:
		return false
	}
	return true
}