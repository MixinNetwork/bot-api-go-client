package blaze

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mixinmessenger/bot-api-go-client/bot"
	"github.com/mixinmessenger/bot-api-go-client/session"
	"github.com/mixinmessenger/bot-api-go-client/uuid"
)

const keepAlivePeriod = 3 * time.Second
const writeWait = 10 * time.Second

type MessageListener interface {
	OnMessage(ctx context.Context, mc *MessageContext, msg MessageView) error
}

const (
	MessageCategoryPlainText             = "PLAIN_TEXT"
	MessageCategoryPlainImage            = "PLAIN_IMAGE"
	MessageCategoryPlainData             = "PLAIN_DATA"
	MessageCategoryPlainSticker          = "PLAIN_STICKER"
	MessageCategorySystemConversation    = "SYSTEM_CONVERSATION"
	MessageCategorySystemAccountSnapshot = "SYSTEM_ACCOUNT_SNAPSHOT"
)

type BlazeMessage struct {
	Id     string                 `json:"id"`
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params,omitempty"`
	Data   interface{}            `json:"data,omitempty"`
	Error  *session.Error         `json:"error,omitempty"`
}

type MessageView struct {
	ConversationId string    `json:"conversation_id"`
	UserId         string    `json:"user_id"`
	MessageId      string    `json:"message_id"`
	Category       string    `json:"category"`
	Data           string    `json:"data"`
	Status         string    `json:"status"`
	Source         string    `json:"source"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type MessageContext struct {
	ReadDone    chan bool
	WriteDone   chan bool
	ReadBuffer  chan MessageView
	WriteBuffer chan []byte
}

type systemConversationPayload struct {
	Action        string `json:"action"`
	ParticipantId string `json:"participant_id"`
	UserId        string `json:"user_id,omitempty"`
	Role          string `json:"role,omitempty"`
}

func Loop(ctx context.Context, listener MessageListener, uid, sid, key string) error {
	conn, err := connectMixinBlaze(uid, sid, key)
	if err != nil {
		return err
	}
	defer conn.Close()

	mc := &MessageContext{
		ReadDone:    make(chan bool, 1),
		WriteDone:   make(chan bool, 1),
		ReadBuffer:  make(chan MessageView, 102400),
		WriteBuffer: make(chan []byte, 102400),
	}
	go writePump(ctx, conn, mc)
	go readPump(ctx, conn, mc)
	if err = writeMessageAndWait(ctx, mc, "LIST_PENDING_MESSAGES", nil); err != nil {
		return session.BlazeServerError(ctx, err)
	}
	for {
		select {
		case <-mc.ReadDone:
			return nil
		case msg := <-mc.ReadBuffer:
			params := map[string]interface{}{"message_id": msg.MessageId, "status": "READ"}
			if err = writeMessageAndWait(ctx, mc, "ACKNOWLEDGE_MESSAGE_RECEIPT", params); err != nil {
				return session.BlazeServerError(ctx, err)
			}
			err = listener.OnMessage(ctx, mc, msg)
			if err != nil {
				return err
			}
		}
	}
}

func SendPlainText(ctx context.Context, mc *MessageContext, msg MessageView, btns string) error {
	params := map[string]interface{}{
		"conversation_id": msg.ConversationId,
		"recipient_id":    msg.UserId,
		"message_id":      uuid.NewV4().String(),
		"category":        "PLAIN_TEXT",
		"data":            base64.StdEncoding.EncodeToString([]byte(btns)),
	}
	if err := writeMessageAndWait(ctx, mc, "CREATE_MESSAGE", params); err != nil {
		return session.BlazeServerError(ctx, err)
	}
	return nil
}

func connectMixinBlaze(uid, sid, key string) (*websocket.Conn, error) {
	token, err := bot.SignAuthenticationToken(uid, sid, key, "GET", "/", "")
	if err != nil {
		return nil, err
	}
	header := make(http.Header)
	header.Add("Authorization", "Bearer "+token)
	u := url.URL{Scheme: "wss", Host: "blaze.mixin.one", Path: "/"}
	dialer := &websocket.Dialer{
		Subprotocols: []string{"Mixin-Blaze-1"},
	}
	conn, _, err := dialer.Dial(u.String(), header)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func readPump(ctx context.Context, conn *websocket.Conn, mc *MessageContext) error {
	defer func() {
		conn.Close()
		mc.WriteDone <- true
		mc.ReadDone <- true
	}()

	for {
		messageType, wsReader, err := conn.NextReader()
		if err != nil {
			return err
		}
		if messageType != websocket.BinaryMessage {
			return session.BlazeServerError(ctx, fmt.Errorf("invalid message type %d", messageType))
		}
		err = parseMessage(ctx, mc, wsReader)
		if err != nil {
			return session.BlazeServerError(ctx, err)
		}
	}
}

func writePump(ctx context.Context, conn *websocket.Conn, mc *MessageContext) error {
	defer conn.Close()
	for {
		select {
		case data := <-mc.WriteBuffer:
			err := writeGzipToConn(conn, data)
			if err != nil {
				return err
			}
		case <-mc.WriteDone:
			return nil
		}
	}
}

func writeMessageAndWait(ctx context.Context, mc *MessageContext, action string, params map[string]interface{}) error {
	blazeMessage, err := json.Marshal(BlazeMessage{Id: uuid.NewV4().String(), Action: action, Params: params})
	if err != nil {
		return err
	}

	select {
	case <-time.After(keepAlivePeriod):
		return fmt.Errorf("timeout to write %s %v", action, params)
	case mc.WriteBuffer <- blazeMessage:
	}
	return nil
}

func writeGzipToConn(conn *websocket.Conn, msg []byte) error {
	conn.SetWriteDeadline(time.Now().Add(writeWait))
	wsWriter, err := conn.NextWriter(websocket.BinaryMessage)
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
	if err := wsWriter.Close(); err != nil {
		return err
	}
	return nil
}

func parseMessage(ctx context.Context, mc *MessageContext, wsReader io.Reader) error {
	var message BlazeMessage
	gzReader, err := gzip.NewReader(wsReader)
	if err != nil {
		return err
	}
	defer gzReader.Close()
	if err = json.NewDecoder(gzReader).Decode(&message); err != nil {
		return err
	}
	if message.Action != "CREATE_MESSAGE" {
		return nil
	}
	data, err := json.Marshal(message.Data)
	if err != nil {
		return err
	}
	var msg MessageView
	if err = json.Unmarshal(data, &msg); err != nil {
		return err
	}
	select {
	case <-time.After(keepAlivePeriod):
		return fmt.Errorf("timeout to handle %s %s", msg.Category, msg.MessageId)
	case mc.ReadBuffer <- msg:
	}
	return nil
}
