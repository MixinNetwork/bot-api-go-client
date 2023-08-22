package bot

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"nhooyr.io/websocket"
)

func TestOauthWS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	opts := &websocket.DialOptions{
		Subprotocols: []string{"Mixin-OAuth-1"},
	}
	c, _, err := websocket.Dial(ctx, "wss://blaze.mixin.one", opts)
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	go readPumpTest(ctx, c)

	blazeMessage, _ := json.Marshal(BlazeMessage{
		Id:     uuid.Must(uuid.NewV4()).String(),
		Action: "REFRESH_OAUTH_CODE",
		Params: map[string]interface{}{
			"client_id": "67a87828-18f5-46a1-b6cc-c72a97a77c43",
			"scope":     "PROFILE:READ",
		},
	})
	err = writeGzipToConnTest(ctx, c, blazeMessage)
	log.Println(err)
	time.Sleep(3 * time.Second)
}

type BlazeMessageTest struct {
	Id     string                 `json:"id"`
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params,omitempty"`
	Data   interface{}            `json:"data,omitempty"`
}

func parseMessageTest(ctx context.Context, wsReader io.Reader) error {
	var message BlazeMessageTest
	gzReader, err := gzip.NewReader(wsReader)
	if err != nil {
		return err
	}
	defer gzReader.Close()
	if err = json.NewDecoder(gzReader).Decode(&message); err != nil {
		return err
	}
	log.Printf("parseMessage %#v", message)
	return nil
}

func readPumpTest(ctx context.Context, conn *websocket.Conn) error {
	for {
		messageType, wsReader, err := conn.Reader(ctx)
		if err != nil {
			return err
		}

		if messageType != websocket.MessageBinary {
			return fmt.Errorf("messageType %d", messageType)
		}

		err = parseMessageTest(ctx, wsReader)
		if err != nil {
			return err
		}
	}
}

func writeGzipToConnTest(ctx context.Context, conn *websocket.Conn, msg []byte) error {
	wsWriter, err := conn.Writer(ctx, websocket.MessageBinary)
	if err != nil {
		return err
	}
	gzWriter, err := gzip.NewWriterLevel(wsWriter, 3)
	if err != nil {
		return err
	}
	if _, err := gzWriter.Write(msg); err != nil {
		return err
	}

	if err := gzWriter.Close(); err != nil {
		return err
	}
	return wsWriter.Close()
}
