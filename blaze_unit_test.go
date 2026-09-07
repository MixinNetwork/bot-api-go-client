package bot

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlazeClientMessageHelpers(t *testing.T) {
	user, _ := newTestSafeUser(t.Name())
	client := NewBlazeClientWithSafeUser(user)
	assert.Equal(t, user.UserId, client.uid)
	assert.Equal(t, []string{"Mixin-Blaze-1"}, client.dailer.Subprotocols)
	assert.Equal(t, 5*time.Second, client.dailer.HandshakeTimeout)

	dialer := &websocket.Dialer{HandshakeTimeout: time.Second}
	client.SetupDailer(dialer)
	assert.Same(t, dialer, client.dailer)
	assert.Equal(t, []string{"Mixin-Blaze-1"}, dialer.Subprotocols)
	assert.Equal(t, time.Second, dialer.HandshakeTimeout)

	received := make(chan BlazeMessage, 10)
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for {
			select {
			case raw := <-client.mc.writeBuffer:
				var message BlazeMessage
				if json.Unmarshal(raw, &message) != nil {
					continue
				}
				received <- message
				if transaction := client.mc.transactions.retrieve(message.Id); transaction != nil {
					_ = transaction(BlazeMessage{Id: message.Id})
				}
			case <-stop:
				return
			}
		}
	}()
	ctx := context.Background()
	view := MessageView{ConversationId: "conversation", UserId: "recipient"}

	require.NoError(t, client.SendMessage(ctx, "conversation", "recipient", "message", MessageCategoryPlainText, "hello", "representative"))
	require.NoError(t, client.SendPlainText(ctx, view, "hello"))
	require.NoError(t, client.SendRecallMessage(ctx, "conversation", "recipient", "old-message"))
	require.NoError(t, client.SendPost(ctx, view, "post"))
	require.NoError(t, client.SendContact(ctx, "conversation", "recipient", "contact"))
	require.NoError(t, client.SendAppCard(ctx, "conversation", "recipient", "title", "description", "https://example.test", "icon"))
	require.NoError(t, client.SendAppButton(ctx, "conversation", "recipient", "label", "https://example.test", "#fff"))
	require.NoError(t, client.SendGroupAppButton(ctx, "conversation", "recipient", []*AppButtonView{{Label: "label"}}))

	for i := range 8 {
		message := <-received
		assert.Equal(t, createMessageAction, message.Action)
		assert.NotEmpty(t, message.Id)
		if i == 0 {
			assert.Equal(t, "hello", string(decodeBlazeData(t, message.Params["data_base64"])))
		}
	}
	tooMany := make([]*AppButtonView, maximumButtons+1)
	err := client.SendGroupAppButton(ctx, "conversation", "recipient", tooMany)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum is 18")
}

func TestBlazeRetriesTransientErrors(t *testing.T) {
	for _, failure := range []Error{
		{Status: 202, Code: 429},
		{Code: 500},
		{Code: 503},
		{Status: 503},
		{Code: 7000},
		{Code: 7001},
	} {
		t.Run(fmt.Sprintf("%d/%d", failure.Status, failure.Code), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				client := NewBlazeClient("user", "session", "key")
				done := make(chan error, 1)
				go func() {
					done <- client.SendPlainText(context.Background(), MessageView{ConversationId: "conversation", UserId: "recipient"}, "hello")
				}()
				var first, retry BlazeMessage
				require.NoError(t, json.Unmarshal(<-client.mc.writeBuffer, &first))
				require.NoError(t, parseMessage(context.Background(), client.mc, gzipBlazeMessage(t, BlazeMessage{Id: first.Id, Error: &failure})))
				synctest.Wait()
				assert.Empty(t, client.mc.writeBuffer, "do not immediately retry a throttled or failed server")
				require.NoError(t, json.Unmarshal(<-client.mc.writeBuffer, &retry))
				assert.NotEqual(t, first.Id, retry.Id)
				assert.Equal(t, first.Action, retry.Action)
				assert.Equal(t, first.Params, retry.Params, "retries must preserve the message ID to avoid duplicate delivery")
				require.NoError(t, parseMessage(context.Background(), client.mc, gzipBlazeMessage(t, BlazeMessage{Id: retry.Id})))
				require.NoError(t, <-done)
				assert.Empty(t, client.mc.transactions.m)
			})
		})
	}
}

func TestBlazeReturnsPermanentErrors(t *testing.T) {
	for _, code := range []int{400, 401, 403, 404, 10002, 10006, 20116} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				client := NewBlazeClient("user", "session", "key")
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				done := make(chan error, 1)
				go func() {
					done <- client.SendMessage(ctx, "conversation", "recipient", "message", MessageCategoryPlainText, "hello", "")
				}()
				var request BlazeMessage
				require.NoError(t, json.Unmarshal(<-client.mc.writeBuffer, &request))
				want := &Error{Status: 202, Code: code, Description: "request rejected", Extra: "details"}
				require.NoError(t, parseMessage(ctx, client.mc, gzipBlazeMessage(t, BlazeMessage{Id: request.Id, Action: request.Action, Error: want})))
				synctest.Wait()
				select {
				case err := <-done:
					var actual Error
					if assert.ErrorAs(t, err, &actual) {
						assert.Equal(t, *want, actual, "callers need the original API error")
					}
				default:
					t.Error("a rejected request must return instead of retrying")
					cancel()
					<-done
				}
				assert.Empty(t, client.mc.writeBuffer, "permanent errors must not be retried")
				assert.Empty(t, client.mc.transactions.m)
			})
		})
	}
}

func TestBlazeRetryCanBeCanceled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := NewBlazeClient("user", "session", "key")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() {
			done <- client.SendPlainText(ctx, MessageView{ConversationId: "conversation", UserId: "recipient"}, "hello")
		}()
		var request BlazeMessage
		require.NoError(t, json.Unmarshal(<-client.mc.writeBuffer, &request))
		require.NoError(t, parseMessage(ctx, client.mc, gzipBlazeMessage(t, BlazeMessage{Id: request.Id, Error: &Error{Status: 202, Code: 429}})))
		synctest.Wait()
		assert.Empty(t, client.mc.writeBuffer, "retries must allow time for the server to recover")
		cancel()
		require.ErrorIs(t, <-done, context.Canceled)
		assert.Empty(t, client.mc.transactions.m)
	})
}

func TestBlazeSendTimeoutPreservesServerError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := NewBlazeClient("user", "session", "key")
		err := client.SendPlainText(context.Background(), MessageView{}, "message body")
		require.Error(t, err)
		assert.JSONEq(t, `{"status":500,"code":7000,"description":"Blaze server error."}`, err.Error())
		assert.Empty(t, client.mc.transactions.m)
	})
}

func TestBlazePreCanceledRequestDoesNotQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for range 64 {
		mc := testMessageContext()
		err := writeMessageAndWait(ctx, mc, createMessageAction, nil)
		require.ErrorIs(t, err, context.Canceled)
		require.Empty(t, mc.writeBuffer, "an already-canceled request must not enqueue a message")
		require.Empty(t, mc.transactions.m)
	}
}

func TestBlazeIgnoresLateResponses(t *testing.T) {
	for _, action := range []string{createMessageAction, "ACKNOWLEDGE_MESSAGE_RECEIPT"} {
		for _, canceled := range []bool{false, true} {
			name := "timeout"
			if canceled {
				name = "canceled"
			}
			t.Run(action+"/"+name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					mc := testMessageContext()
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					done := make(chan error, 1)
					go func() { done <- writeMessageAndWait(ctx, mc, action, nil) }()
					var request BlazeMessage
					require.NoError(t, json.Unmarshal(<-mc.writeBuffer, &request))
					if canceled {
						cancel()
						require.ErrorIs(t, <-done, context.Canceled)
					} else {
						require.ErrorContains(t, <-done, "timeout to wait")
					}
					assert.Empty(t, mc.transactions.m, "expired callbacks must be released")

					go func() { done <- writeMessageAndWait(context.Background(), mc, action, nil) }()
					var pending BlazeMessage
					require.NoError(t, json.Unmarshal(<-mc.writeBuffer, &pending))
					for _, reply := range []BlazeMessage{
						{Error: &Error{Code: 403}},
						{},
						{Data: json.RawMessage("null")},
						{Data: json.RawMessage("{}")},
						{Data: mustJSON(t, MessageView{MessageId: "late", Category: MessageCategoryPlainText})},
					} {
						abandoned, abort := context.WithCancel(context.Background())
						done := make(chan error, 1)
						go func() { done <- writeMessageAndWait(abandoned, mc, action, nil) }()
						var sent BlazeMessage
						require.NoError(t, json.Unmarshal(<-mc.writeBuffer, &sent))
						abort()
						require.ErrorIs(t, <-done, context.Canceled)
						reply.Id = sent.Id
						reply.Action = action
						err := parseMessage(context.Background(), mc, gzipBlazeMessage(t, reply))
						assert.NoError(t, err, "late response must not stop the reader")
						select {
						case message := <-mc.readBuffer:
							t.Errorf("late response delivered as incoming message: %+v", message)
						default:
						}
					}
					require.NoError(t, parseMessage(context.Background(), mc, gzipBlazeMessage(t, BlazeMessage{Id: request.Id, Action: action})))
					assert.Empty(t, mc.dead, "late responses must release their abandoned request ids")

					require.NoError(t, parseMessage(context.Background(), mc, gzipBlazeMessage(t, BlazeMessage{Id: pending.Id, Action: action})))
					require.NoError(t, <-done, "unrelated requests must still complete")
					incoming := MessageView{MessageId: "incoming", Category: MessageCategoryPlainText}
					err := parseMessage(context.Background(), mc, gzipBlazeMessage(t, BlazeMessage{Action: createMessageAction, Data: mustJSON(t, incoming)}))
					require.NoError(t, err)
					assert.Equal(t, incoming, <-mc.readBuffer)
				})
			})
		}
	}
}

func TestBlazeResponseCallbackAfterCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		for range 64 {
			mc := testMessageContext()
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { done <- writeMessageAndWait(ctx, mc, createMessageAction, nil) }()
			var request BlazeMessage
			require.NoError(t, json.Unmarshal(<-mc.writeBuffer, &request))

			// Model cancellation after parseMessage has retrieved the callback.
			callback := mc.transactions.retrieve(request.Id)
			require.NotNil(t, callback)
			cancel()
			require.ErrorIs(t, <-done, context.Canceled)
			require.NoError(t, callback(BlazeMessage{Id: request.Id}), "request cancellation must not stop the reader")
			assert.Empty(t, mc.readBuffer)
		}
	})
}

func TestParseBlazeMessage(t *testing.T) {
	t.Run("transaction response", func(t *testing.T) {
		mc := testMessageContext()
		called := false
		mc.transactions.set("id", func(message BlazeMessage) error {
			called = true
			assert.Equal(t, "id", message.Id)
			return nil
		})
		err := parseMessage(context.Background(), mc, gzipBlazeMessage(t, BlazeMessage{Id: "id", Action: "RESPONSE"}))
		require.NoError(t, err)
		assert.True(t, called)
		assert.Nil(t, mc.transactions.retrieve("id"))
	})

	t.Run("incoming message", func(t *testing.T) {
		mc := testMessageContext()
		data, err := json.Marshal(MessageView{MessageId: "message", Category: MessageCategoryPlainText})
		require.NoError(t, err)
		err = parseMessage(context.Background(), mc, gzipBlazeMessage(t, BlazeMessage{Action: createMessageAction, Data: data}))
		require.NoError(t, err)
		assert.Equal(t, "message", (<-mc.readBuffer).MessageId)
	})

	t.Run("ignored action", func(t *testing.T) {
		err := parseMessage(context.Background(), testMessageContext(), gzipBlazeMessage(t, BlazeMessage{Action: "UNKNOWN"}))
		require.NoError(t, err)
	})

	t.Run("invalid gzip", func(t *testing.T) {
		err := parseMessage(context.Background(), testMessageContext(), bytes.NewReader([]byte("invalid")))
		assert.Error(t, err)
	})
}

func TestEncryptedMessageError(t *testing.T) {
	err := (&EncryptedMessageError{Responses: []*EncryptedMessageResponse{{MessageId: "one"}, nil, {MessageId: "two"}}}).Error()
	assert.Contains(t, err, "one, two")
	assert.False(t, errors.Is(&EncryptedMessageError{}, context.Canceled))
}

func testMessageContext() *messageContext {
	return &messageContext{
		transactions: newTmap(),
		readBuffer:   make(chan MessageView, 1),
		writeBuffer:  make(chan []byte, 1),
	}
}

func gzipBlazeMessage(t *testing.T, message BlazeMessage) *bytes.Reader {
	t.Helper()
	data, err := json.Marshal(message)
	require.NoError(t, err)
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err = writer.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return bytes.NewReader(compressed.Bytes())
}

func decodeBlazeData(t *testing.T, value any) []byte {
	t.Helper()
	encoded, ok := value.(string)
	require.True(t, ok)
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	return data
}
