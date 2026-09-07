package bot

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	keepAlivePeriod = 3 * time.Second
	writeWait       = 10 * time.Second
	pongWait        = 10 * time.Second
	pingPeriod      = (pongWait * 9) / 10

	createMessageAction = "CREATE_MESSAGE"
	maximumButtons      = 18
)

const (
	// TODO deprecate plain messages
	MessageCategoryPlainText       = "PLAIN_TEXT"
	MessageCategoryPlainImage      = "PLAIN_IMAGE"
	MessageCategoryPlainData       = "PLAIN_DATA"
	MessageCategoryPlainSticker    = "PLAIN_STICKER"
	MessageCategoryPlainLive       = "PLAIN_LIVE"
	MessageCategoryPlainContact    = "PLAIN_CONTACT"
	MessageCategoryPlainPost       = "PLAIN_POST"
	MessageCategoryPlainLocation   = "PLAIN_LOCATION"
	MessageCategoryPlainTranscript = "PLAIN_TRANSCRIPT"

	MessageCategoryEncryptedPost       = "ENCRYPTED_POST"
	MessageCategoryEncryptedText       = "ENCRYPTED_TEXT"
	MessageCategoryEncryptedImage      = "ENCRYPTED_IMAGE"
	MessageCategoryEncryptedVideo      = "ENCRYPTED_VIDEO"
	MessageCategoryEncryptedLive       = "ENCRYPTED_LIVE"
	MessageCategoryEncryptedAudio      = "ENCRYPTED_AUDIO"
	MessageCategoryEncryptedData       = "ENCRYPTED_DATA"
	MessageCategoryEncryptedSticker    = "ENCRYPTED_STICKER"
	MessageCategoryEncryptedContact    = "ENCRYPTED_CONTACT"
	MessageCategoryEncryptedLocation   = "ENCRYPTED_LOCATION"
	MessageCategoryEncryptedTranscript = "ENCRYPTED_TRANSCRIPT"

	MessageCategorySystemConversation    = "SYSTEM_CONVERSATION"
	MessageCategorySystemAccountSnapshot = "SYSTEM_ACCOUNT_SNAPSHOT"
	MessageCategoryMessageRecall         = "MESSAGE_RECALL"
	MessageCategoryMessagePin            = "MESSAGE_PIN"
	MessageCategoryAppButtonGroup        = "APP_BUTTON_GROUP"
	MessageCategoryAppCard               = "APP_CARD"

	MessageCategorySystemSafeSnapshot    = "SYSTEM_SAFE_SNAPSHOT"
	MessageCategorySystemSafeInscription = "SYSTEM_SAFE_INSCRIPTION"
)

type BlazeMessage struct {
	Id     string          `json:"id"`
	Action string          `json:"action"`
	Params map[string]any  `json:"params,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

type MessageView struct {
	ConversationId   string    `json:"conversation_id"`
	UserId           string    `json:"user_id"`
	MessageId        string    `json:"message_id"`
	Category         string    `json:"category"`
	DataBase64       string    `json:"data_base64"`
	RepresentativeId string    `json:"representative_id"`
	QuoteMessageId   string    `json:"quote_message_id"`
	Status           string    `json:"status"`
	Source           string    `json:"source"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type TransferView struct {
	Type          string    `json:"type"`
	SnapshotId    string    `json:"snapshot_id"`
	CounterUserId string    `json:"counter_user_id"`
	AssetId       string    `json:"asset_id"`
	Amount        string    `json:"amount"`
	TraceId       string    `json:"trace_id"`
	Memo          string    `json:"memo"`
	CreatedAt     time.Time `json:"created_at"`
}

type TransferSafeView struct {
	Type            string    `json:"type"`
	SnapshotId      string    `json:"snapshot_id"`
	UserId          string    `json:"user_id"`
	OpponentId      string    `json:"opponent_id"`
	TransactionHash string    `json:"transaction_hash"`
	AssetId         string    `json:"asset_id"`
	Amount          string    `json:"amount"`
	Memo            string    `json:"memo"`
	CreatedAt       time.Time `json:"created_at"`
	DepositHash     string    `json:"deposit_hash"`     // deposit only
	InscriptionHash string    `json:"inscription_hash"` // inscription only
}

type AppButtonView struct {
	Label  string `json:"label"`
	Action string `json:"action"`
	Color  string `json:"color"`
}

type AppCardAction = AppButtonView

type AppCardView struct {
	AppID       string          `json:"app_id"`
	CoverURL    string          `json:"cover_url"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Actions     []AppCardAction `json:"actions"`
	Shareable   bool            `json:"shareable"`
}

type messageContext struct {
	transactions *tmap
	readBuffer   chan MessageView
	writeBuffer  chan []byte

	deadMu sync.Mutex
	dead   map[string]time.Time
}

type SystemConversationPayload struct {
	Action        string `json:"action"`
	ParticipantId string `json:"participant_id"`
	UserId        string `json:"user_id,omitempty"`
	Role          string `json:"role,omitempty"`
}

type BlazeClient struct {
	mc     *messageContext
	uid    string
	sid    string
	key    string
	dailer *websocket.Dialer
}

type BlazeListener interface {
	OnMessage(ctx context.Context, msg MessageView, userId string) error
	OnAckReceipt(ctx context.Context, msg MessageView, userId string) error
	SyncAck() bool
}

func NewBlazeClientWithSafeUser(user *SafeUser) *BlazeClient {
	return NewBlazeClient(user.UserId, user.SessionId, user.SessionPrivateKey)
}

func NewBlazeClient(uid, sid, key string) *BlazeClient {
	client := BlazeClient{
		mc: &messageContext{
			transactions: newTmap(),
			readBuffer:   make(chan MessageView, 102400),
			writeBuffer:  make(chan []byte, 102400),
		},
		uid: uid,
		sid: sid,
		key: key,
	}
	client.SetupDailer(nil)
	return &client
}

func (b *BlazeClient) SetupDailer(dailer *websocket.Dialer) {
	if dailer == nil {
		dailer = &websocket.Dialer{}
	}
	dailer.Subprotocols = []string{"Mixin-Blaze-1"}
	if dailer.HandshakeTimeout == 0 {
		dailer.HandshakeTimeout = time.Second * 5
	}
	b.dailer = dailer
}

func (b *BlazeClient) Loop(ctx context.Context, listener BlazeListener) error {
	conn, err := b.connectMixinBlaze(ctx)
	if err != nil {
		return err
	}
	pumpCtx, cancelPumps := context.WithCancel(ctx)
	readDone := make(chan struct{})
	var pumps sync.WaitGroup
	defer func() {
		cancelPumps()
		conn.Close()
		// Finish this connection's pumps before the client can reconnect.
		pumps.Wait()
	}()
	pumps.Add(2)
	go func() {
		defer pumps.Done()
		writePump(pumpCtx, conn, b.mc)
	}()
	go func() {
		defer pumps.Done()
		defer close(readDone)
		defer cancelPumps()
		readPump(pumpCtx, conn, b.mc)
	}()

	if err = writeMessageAndWait(ctx, b.mc, "LIST_PENDING_MESSAGES", nil); err != nil {
		return blazeRequestError(ctx, err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-readDone:
			return ctx.Err()
		case msg := <-b.mc.readBuffer:
			if msg.Source == "ACKNOWLEDGE_MESSAGE_RECEIPT" {
				err = listener.OnAckReceipt(ctx, msg, b.uid)
				if err != nil {
					return err
				}
			} else {
				err = listener.OnMessage(ctx, msg, b.uid)
				if err != nil {
					return err
				}
				if listener.SyncAck() {
					params := map[string]any{"message_id": msg.MessageId, "status": "READ"}
					if err = writeMessageAndWait(ctx, b.mc, "ACKNOWLEDGE_MESSAGE_RECEIPT", params); err != nil {
						return blazeRequestError(ctx, err)
					}
				}
			}
		}
	}
}

func (b *BlazeClient) SendMessage(ctx context.Context, conversationId, recipientId, messageId, category, content, representativeId string) error {
	params := map[string]any{
		"conversation_id":   conversationId,
		"recipient_id":      recipientId,
		"message_id":        messageId,
		"category":          category,
		"data_base64":       base64.RawURLEncoding.EncodeToString([]byte(content)),
		"representative_id": representativeId,
	}
	return blazeRequestError(ctx, writeMessageAndWait(ctx, b.mc, createMessageAction, params))
}

func (b *BlazeClient) SendPlainText(ctx context.Context, msg MessageView, content string) error {
	params := map[string]any{
		"conversation_id": msg.ConversationId,
		"recipient_id":    msg.UserId,
		"message_id":      UuidNewV4().String(),
		"category":        MessageCategoryPlainText,
		"data_base64":     base64.RawURLEncoding.EncodeToString([]byte(content)),
	}
	return blazeRequestError(ctx, writeMessageAndWait(ctx, b.mc, createMessageAction, params))
}

func (b *BlazeClient) SendRecallMessage(ctx context.Context, conversationId, recipientId, recallMessageId string) error {
	c := RecallMessagePayload{
		MessageId: recallMessageId,
	}
	a, _ := json.Marshal(c)
	params := map[string]any{
		"conversation_id": conversationId,
		"recipient_id":    recipientId,
		"message_id":      UuidNewV4().String(),
		"category":        MessageCategoryMessageRecall,
		"data_base64":     base64.RawURLEncoding.EncodeToString(a),
	}
	return blazeRequestError(ctx, writeMessageAndWait(ctx, b.mc, createMessageAction, params))
}

func (b *BlazeClient) SendPost(ctx context.Context, msg MessageView, content string) error {
	params := map[string]any{
		"conversation_id": msg.ConversationId,
		"recipient_id":    msg.UserId,
		"message_id":      UuidNewV4().String(),
		"category":        MessageCategoryPlainPost,
		"data_base64":     base64.RawURLEncoding.EncodeToString([]byte(content)),
	}
	return blazeRequestError(ctx, writeMessageAndWait(ctx, b.mc, createMessageAction, params))
}

func (b *BlazeClient) SendContact(ctx context.Context, conversationId, recipientId, contactId string) error {
	contactMap := map[string]string{"user_id": contactId}
	contactData, _ := json.Marshal(contactMap)
	params := map[string]any{
		"conversation_id": conversationId,
		"recipient_id":    recipientId,
		"message_id":      UuidNewV4().String(),
		"category":        MessageCategoryPlainContact,
		"data_base64":     base64.RawURLEncoding.EncodeToString(contactData),
	}
	return blazeRequestError(ctx, writeMessageAndWait(ctx, b.mc, createMessageAction, params))
}

func (b *BlazeClient) SendAppCard(ctx context.Context, conversationId, recipientId, title, description, action, iconUrl string) error {
	data, err := json.Marshal(map[string]string{
		"title":       title,
		"description": description,
		"action":      action,
		"icon_url":    iconUrl,
	})
	if err != nil {
		return err
	}
	params := map[string]any{
		"conversation_id": conversationId,
		"recipient_id":    recipientId,
		"message_id":      UuidNewV4().String(),
		"category":        MessageCategoryAppCard,
		"data_base64":     base64.RawURLEncoding.EncodeToString(data),
	}
	return blazeRequestError(ctx, writeMessageAndWait(ctx, b.mc, createMessageAction, params))
}

func (b *BlazeClient) SendAppButton(ctx context.Context, conversationId, recipientId, label, action, color string) error {
	btns, err := json.Marshal([]any{map[string]string{
		"label":  label,
		"action": action,
		"color":  color,
	}})
	if err != nil {
		return err
	}
	params := map[string]any{
		"conversation_id": conversationId,
		"recipient_id":    recipientId,
		"message_id":      UuidNewV4().String(),
		"category":        MessageCategoryAppButtonGroup,
		"data_base64":     base64.RawURLEncoding.EncodeToString(btns),
	}
	return blazeRequestError(ctx, writeMessageAndWait(ctx, b.mc, createMessageAction, params))
}

func (b *BlazeClient) SendGroupAppButton(ctx context.Context, conversationId, recipientId string, buttons []*AppButtonView) error {
	if len(buttons) > maximumButtons {
		return fmt.Errorf("too many buttons, maximum is %d", maximumButtons)
	}
	btns, err := json.Marshal(buttons)
	if err != nil {
		return err
	}
	params := map[string]any{
		"conversation_id": conversationId,
		"recipient_id":    recipientId,
		"message_id":      UuidNewV4().String(),
		"category":        MessageCategoryAppButtonGroup,
		"data_base64":     base64.RawURLEncoding.EncodeToString(btns),
	}
	return blazeRequestError(ctx, writeMessageAndWait(ctx, b.mc, createMessageAction, params))
}

func (b *BlazeClient) connectMixinBlaze(ctx context.Context) (*websocket.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	user := &SafeUser{
		UserId:            b.uid,
		SessionId:         b.sid,
		SessionPrivateKey: b.key,
	}
	token, err := SignAuthenticationToken("GET", "/", "", user)
	if err != nil {
		return nil, err
	}
	header := make(http.Header)
	header.Add("Authorization", "Bearer "+token)
	u := url.URL{Scheme: "wss", Host: blazeUri, Path: "/"}
	// The WebSocket dialer uses socket deadlines while reading the HTTP
	// upgrade response. Also close the socket on explicit cancellation.
	var stopClose func() bool
	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			stopClose = context.AfterFunc(ctx, func() { _ = info.Conn.Close() })
		},
	})
	defer func() {
		if stopClose != nil {
			stopClose()
		}
	}()
	conn, _, err := b.dailer.DialContext(ctx, u.String(), header)
	if ctxErr := ctx.Err(); ctxErr != nil {
		if conn != nil {
			conn.Close()
		}
		return nil, ctxErr
	}
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			blazeUri = DefaultBlazeHost
		}
		return nil, err
	}
	return conn, nil
}

func readPump(ctx context.Context, conn *websocket.Conn, mc *messageContext) error {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		err := conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			return BlazeServerError(ctx, err)
		}
		return nil
	})

	for {
		err := conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			return BlazeServerError(ctx, err)
		}
		messageType, wsReader, err := conn.NextReader()
		if err != nil {
			return BlazeServerError(ctx, err)
		}
		if messageType != websocket.BinaryMessage {
			return BlazeServerError(ctx, fmt.Errorf("invalid message type %d", messageType))
		}
		err = parseMessage(ctx, mc, wsReader)
		if err != nil {
			return BlazeServerError(ctx, err)
		}
	}
}

func writePump(ctx context.Context, conn *websocket.Conn, mc *messageContext) error {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()
	for {
		if ctx.Err() != nil {
			return nil
		}
		select {
		case data := <-mc.writeBuffer:
			err := writeGzipToConn(conn, data)
			if err != nil {
				return BlazeServerError(ctx, err)
			}
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				return BlazeServerError(ctx, err)
			}
		}
	}
}

func blazeRequestError(ctx context.Context, err error) error {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if _, ok := err.(Error); ok {
		return err
	}
	if e, ok := err.(*Error); ok && e != nil {
		return *e
	}
	// Preserve the existing error format for local failures, while
	// allowing callers to act on API errors and context cancellation.
	return BlazeServerError(ctx, err)
}

func writeMessageAndWait(ctx context.Context, mc *messageContext, action string, params map[string]any) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		resp := make(chan BlazeMessage, 1)
		id := UuidNewV4().String()
		mc.transactions.set(id, func(t BlazeMessage) error {
			// The waiter may have already returned. Never block or fail the
			// shared reader when delivering an individual request's response.
			select {
			case resp <- t:
			default:
			}
			return nil
		})
		blazeMessage, err := json.Marshal(BlazeMessage{Id: id, Action: action, Params: params})
		if err != nil {
			mc.transactions.retrieve(id)
			return err
		}
		writeTimer := time.NewTimer(keepAlivePeriod)
		select {
		case <-ctx.Done():
			writeTimer.Stop()
			mc.transactions.retrieve(id)
			mc.deadRemember(id)
			return ctx.Err()
		case <-writeTimer.C:
			mc.transactions.retrieve(id)
			mc.deadRemember(id)
			return fmt.Errorf("timeout to write %s %v", action, params)
		case mc.writeBuffer <- blazeMessage:
			writeTimer.Stop()
		}

		responseTimer := time.NewTimer(keepAlivePeriod)
		select {
		case <-ctx.Done():
			responseTimer.Stop()
			mc.transactions.retrieve(id)
			mc.deadRemember(id)
			return ctx.Err()
		case <-responseTimer.C:
			mc.transactions.retrieve(id)
			mc.deadRemember(id)
			return fmt.Errorf("timeout to wait %s %v", action, params)
		case response := <-resp:
			responseTimer.Stop()
			if response.Error == nil {
				return nil
			}
			// Retry rate limiting and server failures. Return rejected
			// requests to the caller with their original error details.
			e := response.Error
			retryable := e.Code == http.StatusTooManyRequests ||
				e.Code >= 500 && e.Code < 600 || e.Status >= 500 && e.Status < 600 ||
				e.Code == 7000 || e.Code == 7001
			if !retryable {
				return *e
			}
		}
		retryTimer := time.NewTimer(keepAlivePeriod)
		select {
		case <-ctx.Done():
			retryTimer.Stop()
			return ctx.Err()
		case <-retryTimer.C:
		}
	}
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
	return wsWriter.Close()
}

func parseMessage(ctx context.Context, mc *messageContext, wsReader io.Reader) error {
	var message BlazeMessage
	gzReader, err := gzip.NewReader(wsReader)
	if err != nil {
		return err
	}
	defer gzReader.Close()
	if err = json.NewDecoder(gzReader).Decode(&message); err != nil {
		return err
	}
	transaction := mc.transactions.retrieve(message.Id)
	if transaction != nil {
		return transaction(message)
	}
	// Responses can arrive after their waiter has timed out or been canceled.
	// Drop them instead of delivering them as incoming messages. Error and
	// empty responses are not incoming messages either.
	if mc.deadForget(message.Id) {
		return nil
	}
	if message.Error != nil || len(message.Data) == 0 {
		return nil
	}

	if message.Action != "CREATE_MESSAGE" && message.Action != "ACKNOWLEDGE_MESSAGE_RECEIPT" {
		return nil
	}

	var msg MessageView
	if err = json.Unmarshal(message.Data, &msg); err != nil {
		return err
	}
	if msg.MessageId == "" {
		return nil
	}
	timer := time.NewTimer(keepAlivePeriod)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		timer.Reset(keepAlivePeriod)
		return fmt.Errorf("timeout to handle %s %s", msg.Category, msg.MessageId)
	case mc.readBuffer <- msg:
	}
	return nil
}

type tmap struct {
	mutex sync.Mutex
	m     map[string]mixinTransaction
}

type mixinTransaction func(BlazeMessage) error

func newTmap() *tmap {
	return &tmap{
		m: make(map[string]mixinTransaction),
	}
}

func (m *tmap) retrieve(key string) mixinTransaction {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	defer delete(m.m, key)
	return m.m[key]
}

func (m *tmap) set(key string, t mixinTransaction) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.m[key] = t
}

// deadRemember marks a request id whose waiter has abandoned it. Responses to
// abandoned requests must be dropped instead of delivered as incoming messages.
func (mc *messageContext) deadRemember(id string) {
	mc.deadMu.Lock()
	defer mc.deadMu.Unlock()
	if mc.dead == nil {
		mc.dead = make(map[string]time.Time)
	}
	now := time.Now()
	for k, t := range mc.dead {
		if now.Sub(t) > time.Minute {
			delete(mc.dead, k)
		}
	}
	mc.dead[id] = now
}

// deadForget reports and removes a request id from the dead set.
func (mc *messageContext) deadForget(id string) bool {
	mc.deadMu.Lock()
	defer mc.deadMu.Unlock()
	if mc.dead == nil {
		return false
	}
	_, ok := mc.dead[id]
	delete(mc.dead, id)
	return ok
}
