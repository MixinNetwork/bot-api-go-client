package bot

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlazeLoopCancelsDial(t *testing.T) {
	oldURI := blazeUri
	t.Cleanup(func() { blazeUri = oldURI })
	synctest.Test(t, func(t *testing.T) {
		user, _ := newTestSafeUser(t.Name())
		client := NewBlazeClientWithSafeUser(user)
		started := make(chan struct{})
		client.SetupDailer(&websocket.Dialer{NetDialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			close(started)
			<-ctx.Done()
			return nil, ctx.Err()
		}})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- client.Loop(ctx, blazeTestListener(func(MessageView) error { return nil })) }()
		<-started
		cancel()
		synctest.Wait()
		select {
		case err := <-done:
			assert.ErrorIs(t, err, context.Canceled)
		default:
			t.Error("dial continued after the loop context was canceled")
			<-done
		}
	})
}

func TestBlazeLoopCancelsHandshake(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	client := blazeClientForTestServer(t, server)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- client.Loop(ctx, blazeTestListener(func(MessageView) error { return nil })) }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("WebSocket handshake did not start")
	}
	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Error("WebSocket handshake continued after cancellation")
		server.CloseClientConnections()
		<-done
	}
}

func TestBlazeLoopCancelsIdleConnection(t *testing.T) {
	client, closeConnections := newBlazeLoopTestClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- client.Loop(ctx, blazeTestListener(func(MessageView) error {
			close(ready)
			return nil
		}))
	}()
	select {
	case <-ready:
	case err := <-done:
		t.Fatalf("loop ended before receiving a message: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("loop did not receive a message")
	}
	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Error("idle loop did not stop after cancellation")
		closeConnections()
		<-done
	}
}

func TestBlazeLoopReconnectAfterListenerError(t *testing.T) {
	client, _ := newBlazeLoopTestClient(t)
	wantErr := errors.New("listener stopped")
	for range 5 {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := client.Loop(ctx, blazeTestListener(func(MessageView) error { return wantErr }))
		cancel()
		require.ErrorIs(t, err, wantErr, "shutdown from a previous loop must not terminate a new connection")
	}
}

func TestParseBlazeMessageCancelsBlockedDelivery(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		mc := testMessageContext()
		mc.readBuffer <- MessageView{MessageId: "queued"}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		message := BlazeMessage{Action: createMessageAction, Data: mustJSON(t, MessageView{MessageId: "incoming"})}
		raw := gzipBlazeMessage(t, message)
		done := make(chan error, 1)
		go func() { done <- parseMessage(ctx, mc, raw) }()
		synctest.Wait()
		cancel()
		synctest.Wait()
		select {
		case err := <-done:
			assert.ErrorIs(t, err, context.Canceled)
		default:
			t.Error("reader remained blocked on a full message buffer after cancellation")
			<-done
		}
	})
}

type blazeTestListener func(MessageView) error

func (f blazeTestListener) OnMessage(_ context.Context, message MessageView, _ string) error {
	return f(message)
}

func (f blazeTestListener) OnAckReceipt(_ context.Context, message MessageView, _ string) error {
	return f(message)
}

func (blazeTestListener) SyncAck() bool { return false }

func blazeClientForTestServer(t *testing.T, server *httptest.Server) *BlazeClient {
	t.Helper()
	oldURI := blazeUri
	blazeUri = strings.TrimPrefix(server.URL, "https://")
	t.Cleanup(func() { blazeUri = oldURI })
	user, _ := newTestSafeUser(t.Name())
	client := NewBlazeClientWithSafeUser(user)
	client.SetupDailer(&websocket.Dialer{
		TLSClientConfig: server.Client().Transport.(*http.Transport).TLSClientConfig.Clone(),
	})
	return client
}

func newBlazeLoopTestClient(t *testing.T) (*BlazeClient, func()) {
	t.Helper()
	var mu sync.Mutex
	var connections []*websocket.Conn
	closeConnections := func() {
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range connections {
			_ = conn.Close()
		}
	}
	upgrader := websocket.Upgrader{Subprotocols: []string{"Mixin-Blaze-1"}}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer conn.Close()
		mu.Lock()
		connections = append(connections, conn)
		mu.Unlock()
		for {
			_, reader, err := conn.NextReader()
			if err != nil {
				return
			}
			gz, err := gzip.NewReader(reader)
			if !assert.NoError(t, err) {
				return
			}
			var request BlazeMessage
			err = json.NewDecoder(gz).Decode(&request)
			_ = gz.Close()
			if !assert.NoError(t, err) {
				return
			}
			response := mustJSON(t, BlazeMessage{Id: request.Id, Action: request.Action})
			if writeGzipToConn(conn, response) != nil {
				return
			}
			if request.Action == "LIST_PENDING_MESSAGES" {
				incoming := BlazeMessage{Action: createMessageAction, Data: mustJSON(t, MessageView{MessageId: UuidNewV4().String()})}
				if writeGzipToConn(conn, mustJSON(t, incoming)) != nil {
					return
				}
			}
		}
	}))
	t.Cleanup(func() {
		closeConnections()
		server.Close()
	})
	return blazeClientForTestServer(t, server), closeConnections
}
