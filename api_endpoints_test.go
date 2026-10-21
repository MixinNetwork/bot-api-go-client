package bot

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicAPIEndpoints(t *testing.T) {
	user, _ := newTestSafeUser(t.Name())
	oldUser := globalUser
	t.Cleanup(func() { globalUser = oldUser })
	globalUser = user
	seen := make(map[string]int)
	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.URL.Path]++
		var response string
		switch {
		case r.URL.Path == "/external/fiats":
			response = `{"data":[{"code":"USD","rate":1.25}]}`
		case r.URL.Path == "/network":
			response = `{"data":{"assets":[{"asset_id":"asset","display_symbol":"USDT"}]}}`
		case r.URL.Path == "/network/assets/top":
			response = `{"data":[{"asset_id":"top"}]}`
		case r.URL.Path == "/network/assets/asset":
			response = `{"data":{"asset_id":"asset"}}`
		case r.URL.Path == "/network/ticker":
			assert.Equal(t, "asset", r.URL.Query().Get("asset"))
			response = `{"data":{"price_usd":"2.50"}}`
		case r.URL.Path == "/network/assets/search/usdt":
			response = `{"data":[{"asset_id":"search"}]}`
		case r.URL.Path == "/network/chains":
			response = `{"data":[{"chain_id":"chain","name":"Chain"}]}`
		case r.URL.Path == "/network/chains/chain":
			response = `{"data":{"chain_id":"chain","name":"Chain"}}`
		case r.URL.Path == "/safe/inscriptions/collections/collection/items":
			response = `{"data":[{"inscription_hash":"item","collection_hash":"collection"}]}`
		case r.URL.Path == "/safe/inscriptions/collections/collection":
			response = `{"data":{"collection_hash":"collection","name":"Collection"}}`
		case r.URL.Path == "/safe/inscriptions/items/item":
			response = `{"data":{"inscription_hash":"item","collection_hash":"collection"}}`
		case r.URL.Path == "/safe/deposits":
			response = `{"data":[{"deposit_id":"deposit","amount":"1"}]}`
		case r.URL.Path == "/codes/code":
			response = `{"data":{"request_id":"request"}}`
		case r.URL.Path == "/external/tip/nodes":
			response = `{"data":{"identity":"node","commitments":["commitment"]}}`
		case r.URL.Path == "/external/addresses/check":
			assert.Equal(t, "asset", r.URL.Query().Get("asset"))
			assert.Equal(t, "destination", r.URL.Query().Get("destination"))
			assert.Equal(t, "tag", r.URL.Query().Get("tag"))
			response = `{"data":{"destination":"normalized","tag":"tag"}}`
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	ctx := context.Background()

	fiats, err := Fiats(ctx)
	require.NoError(t, err)
	require.Len(t, fiats, 1)
	assert.Equal(t, "USD", fiats[0].Code)
	fiats, err = GetFiats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1.25, fiats[0].Rate)

	networkAssets, err := ReadNetworkAssets(ctx)
	require.NoError(t, err)
	assert.Equal(t, "asset", networkAssets[0].AssetID)
	top, err := ReadNetworkAssetsTop(ctx)
	require.NoError(t, err)
	assert.Equal(t, "top", top[0].AssetID)
	asset, err := ReadAsset(ctx, "asset")
	require.NoError(t, err)
	assert.Equal(t, "asset", asset.AssetID)
	ticker, err := ReadAssetTicker(ctx, "asset")
	require.NoError(t, err)
	assert.Equal(t, "2.50", ticker.PriceUSD)
	ticker, err = ReadAssetTickerWithOffset(ctx, "asset", "2026-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.Equal(t, "2.50", ticker.PriceUSD)
	assets, err := AssetSearch(ctx, "usdt")
	require.NoError(t, err)
	assert.Equal(t, "search", assets[0].AssetID)

	chains, err := ReadNetworkChains(ctx)
	require.NoError(t, err)
	assert.Equal(t, "chain", chains[0].ChainId)
	chain, err := ReadNetworkChainById(ctx, "chain")
	require.NoError(t, err)
	assert.Equal(t, "Chain", chain.Name)

	collection, err := ReadCollection(ctx, "collection")
	require.NoError(t, err)
	assert.Equal(t, "Collection", collection.Name)
	inscription, err := ReadInscription(ctx, "item")
	require.NoError(t, err)
	assert.Equal(t, "item", inscription.InscriptionHash)
	items, err := ReadCollectionItems(ctx, "collection")
	require.NoError(t, err)
	assert.Equal(t, "collection", items[0].CollectionHash)

	deposits, err := FetchPendingSafeDeposits(ctx)
	require.NoError(t, err)
	assert.Equal(t, "deposit", deposits[0].DepositID)
	code, err := ReadMultisigByCode(ctx, "code")
	require.NoError(t, err)
	assert.Equal(t, "request", code.RequestId)
	tipNode, err := GetTipNodeByPathWithRequestId(ctx, "nodes", "request-id")
	require.NoError(t, err)
	assert.Equal(t, "node", tipNode.Identity)
	tipNode, err = GetTipNodeByPath(ctx, "nodes")
	require.NoError(t, err)
	assert.Equal(t, []string{"commitment"}, tipNode.Commitments)
	address, err := CheckAddress(ctx, "asset", "destination", "tag")
	require.NoError(t, err)
	assert.Equal(t, "normalized", address.Destination)

	assert.Equal(t, 2, seen["/external/fiats"])
	assert.Equal(t, 2, seen["/network/ticker"])
}

func TestAuthenticatedAPIEndpoints(t *testing.T) {
	user, _ := newTestSafeUser(t.Name())
	spendSeed := sha256.Sum256([]byte("spend key"))
	serverSeed := sha256.Sum256([]byte("server key"))
	serverPrivate := ed25519.NewKeyFromSeed(serverSeed[:])
	user.SpendPrivateKey = hex.EncodeToString(spendSeed[:])
	user.ServerPublicKey = hex.EncodeToString(serverPrivate.Public().(ed25519.PublicKey))
	oldUser := globalUser
	t.Cleanup(func() { globalUser = oldUser })
	globalUser = user

	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authorization := r.Header.Get("Authorization"); authorization != "" {
			assert.True(t, strings.HasPrefix(authorization, "Bearer"), r.URL.Path)
		}
		var response string
		switch {
		case r.URL.Path == "/attachments" || strings.HasPrefix(r.URL.Path, "/attachments/"):
			response = `{"data":{"attachment_id":"attachment","view_url":"view"}}`
		case r.URL.Path == "/addresses":
			response = `{"data":{"address_id":"address","asset_id":"asset","destination":"destination"}}`
		case r.URL.Path == "/addresses/address/delete":
			response = `{}`
		case r.URL.Path == "/addresses/address":
			response = `{"data":{"address_id":"address"}}`
		case r.URL.Path == "/assets/asset/addresses":
			response = `{"data":[{"address_id":"address"}]}`
		case r.URL.Path == "/apps/e95b1d4e-4d49-4ac3-9402-988804458adc" || r.URL.Path == "/apps/app/transfer":
			response = `{"data":{"app_id":"app","name":"App"}}`
		case r.URL.Path == "/safe/assets/asset/fees":
			response = `{"data":[{"asset_id":"asset","amount":"0.1"}]}`
		case r.URL.Path == "/safe/fees":
			response = `{"data":[{"asset_id":"asset","fee_amount":"0.1"}]}`
		case r.URL.Path == "/safe/assets/fetch":
			response = `{"data":[{"asset_id":"asset","display_symbol":"AST"}]}`
		case r.URL.Path == "/safe/deposit/entries":
			response = `{"data":[{"entry_id":"entry","chain_id":"chain"}]}`
		case r.URL.Path == "/safe/keys":
			response = `{"data":[{"mask":"mask","keys":[]}]}`
		case r.URL.Path == "/sessions/fetch":
			response = `{"data":[{"user_id":"recipient","session_id":"session","public_key":"public"}]}`
		case r.URL.Path == "/turn":
			response = `{"data":[{"url":"turn:example","username":"user"}]}`
		case r.URL.Path == "/safe/outputs/output":
			response = `{"data":{"output_id":"output","amount":"2"}}`
		case r.URL.Path == "/safe/outputs":
			assert.Equal(t, "members", r.URL.Query().Get("members"))
			response = `{"data":[{"output_id":"output","asset_id":"asset","amount":"2","sequence":1}]}`
		case r.URL.Path == "/safe/snapshots/notifications":
			response = `{"data":{"message_id":"message"}}`
		case r.URL.Path == "/safe/snapshots/snapshot":
			response = `{"data":{"snapshot_id":"snapshot","amount":"1"}}`
		case r.URL.Path == "/safe/snapshots":
			response = `{"data":[{"snapshot_id":"snapshot","amount":"1"}]}`
		case r.URL.Path == "/snapshots/trace/trace" || r.URL.Path == "/snapshots/snapshot" || r.URL.Path == "/network/snapshots/snapshot":
			response = `{"data":{"snapshot_id":"snapshot","amount":"1"}}`
		case r.URL.Path == "/snapshots":
			response = `{"data":[{"snapshot_id":"snapshot","amount":"1"}]}`
		case r.URL.Path == "/network/snapshots":
			response = `{"data":[{"snapshot_id":"snapshot","amount":"1"}]}`
		case r.URL.Path == "/users" || r.URL.Path == "/users/user" || r.URL.Path == "/search/query" || r.URL.Path == "/me" || r.URL.Path == "/me/preferences" || r.URL.Path == "/relationships":
			response = `{"data":{"user_id":"user","full_name":"User"}}`
		case r.URL.Path == "/users/fetch":
			response = `{"data":[{"user_id":"user"}]}`
		case r.URL.Path == "/safe/me":
			response = `{"data":{"user_id":"user","has_pin":true}}`
		case r.URL.Path == "/pin/update":
			response = `{}`
		case r.URL.Path == "/pin/verify":
			response = `{"data":{"user_id":"user"}}`
		case r.URL.Path == "/conversations":
			response = `{"data":{"conversation_id":"conversation","name":"Conversation"}}`
		case strings.HasPrefix(r.URL.Path, "/conversations/conversation"):
			response = `{"data":{"conversation_id":"conversation","name":"Conversation"}}`
		case r.URL.Path == "/multisigs" || r.URL.Path == "/multisigs/outputs":
			response = `{"data":[{"utxo_id":"utxo","amount":"1"}]}`
		case r.URL.Path == "/multisigs/requests":
			response = `{"data":{"request_id":"request"}}`
		case r.URL.Path == "/multisigs/requests/request/sign":
			response = `{"data":{"request_id":"request","state":"signed"}}`
		case r.URL.Path == "/multisigs/requests/request/cancel" || r.URL.Path == "/multisigs/requests/request/unlock":
			response = `{}`
		case r.URL.Path == "/safe/multisigs":
			response = `{"data":[{"request_id":"request"}]}`
		case r.URL.Path == "/safe/multisigs/request":
			response = `{"data":{"request_id":"request"}}`
		case r.URL.Path == "/messages" || r.URL.Path == "/acknowledgements":
			response = `{}`
		case r.URL.Path == "/external/kernel":
			response = `{"data":{"ok":true}}`
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	ctx := context.Background()

	attachment, err := CreateAttachment(ctx, user)
	require.NoError(t, err)
	assert.Equal(t, "attachment", attachment.AttachmentId)
	attachment, err = AttachmentShow(ctx, "attachment", user)
	require.NoError(t, err)
	assert.Equal(t, "view", attachment.ViewURL)

	address, err := CreateAddress(ctx, &AddressInput{ChainId: "chain", AssetId: "asset", Destination: "destination"}, user)
	require.NoError(t, err)
	assert.Equal(t, "address", address.AddressId)
	address, err = ReadAddress(ctx, "address", user)
	require.NoError(t, err)
	assert.Equal(t, "address", address.AddressId)
	addresses, err := GetAddressesByAssetId(ctx, "asset", user)
	require.NoError(t, err)
	assert.Equal(t, "address", addresses[0].AddressId)
	require.NoError(t, DeleteAddress(ctx, "address", user))

	app, err := UpdateApp(ctx, &UpdateAppInput{Name: "App"}, user)
	require.NoError(t, err)
	assert.Equal(t, "app", app.AppId)
	app, err = TransferAppOwnership(ctx, "app", "receiver", user)
	require.NoError(t, err)
	assert.Equal(t, "App", app.Name)

	fees, err := ReadAssetFee(ctx, "asset", "destination", user)
	require.NoError(t, err)
	assert.Equal(t, "0.1", fees[0].Amount)
	safeFees, err := ReadSafeFees(ctx, user)
	require.NoError(t, err)
	assert.Equal(t, "0.1", safeFees[0].FeeAmount)
	assets, err := FetchAssets(ctx, []string{"asset"}, user)
	require.NoError(t, err)
	assert.Equal(t, "AST", assets[0].DisplaySymbol)

	entries, err := CreateDepositEntry(ctx, "chain", []string{"member"}, 1, user)
	require.NoError(t, err)
	assert.Equal(t, "entry", entries[0].EntryID)
	ghosts, err := RequestSafeGhostKeys(ctx, []*GhostKeyRequest{{Receivers: []string{"member"}, Index: 1}}, user)
	require.NoError(t, err)
	assert.Equal(t, "mask", ghosts[0].Mask)
	sessions, err := FetchUserSessions(ctx, []string{"recipient"}, user)
	require.NoError(t, err)
	assert.Equal(t, "session", sessions[0].SessionId)
	turns, err := GetTurnServer(ctx, user)
	require.NoError(t, err)
	assert.Equal(t, "turn:example", turns[0].Url)

	outputs, err := ListOutputs(ctx, "members", 1, "asset", OutputStateUnspent, 1, 10, user)
	require.NoError(t, err)
	assert.Equal(t, "output", outputs[0].OutputID)
	outputs, err = ListUnspentOutputs(ctx, "members", 1, "asset", user)
	require.NoError(t, err)
	outputs, err = ListOutputsByToken(ctx, "members", 1, "asset", OutputStateSpent, 2, 5, "token")
	require.NoError(t, err)
	outputs, err = ListUnspentOutputsByToken(ctx, "members", 1, "asset", "token")
	require.NoError(t, err)
	output, err := GetOutput(ctx, "output", user)
	require.NoError(t, err)
	assert.Equal(t, "2", output.Amount)

	testSnapshotEndpoints(t, ctx, user)
	testUserEndpoints(t, ctx, user, serverPrivate.Public().(ed25519.PublicKey))
	testConversationEndpoints(t, ctx, user)
	testMultisigEndpoints(t, ctx, user)

	message := &MessageRequest{RecipientId: "recipient", MessageId: "message", Category: "PLAIN_TEXT", DataBase64: "dGVzdA"}
	require.NoError(t, PostMessageRequest(ctx, message, user))
	require.NoError(t, PostMessages(ctx, []*MessageRequest{message}, user))
	require.NoError(t, PostMessage(ctx, "conversation", "recipient", "message", "PLAIN_TEXT", "dGVzdA", user))
	require.NoError(t, PostAcknowledgements(ctx, []*ReceiptAcknowledgementRequest{{MessageId: "message", Status: "READ"}}, user))
	kernelResponse, err := CallKernelRPC(ctx, user, "getinfo", 1)
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":{"ok":true}}`, string(kernelResponse))
}

func testSnapshotEndpoints(t *testing.T, ctx context.Context, user *SafeUser) {
	t.Helper()
	safeSnapshots, err := SafeSnapshots(ctx, 10, "app", "asset", "opponent", "offset", user)
	require.NoError(t, err)
	assert.Equal(t, "snapshot", safeSnapshots[0].SnapshotID)
	safeSnapshots, err = SafeSnapshotsByToken(ctx, 10, "app", "asset", "opponent", "offset", "token")
	require.NoError(t, err)
	safeSnapshot, err := SafeSnapshotById(ctx, "snapshot", user)
	require.NoError(t, err)
	assert.Equal(t, "1", safeSnapshot.Amount)
	safeSnapshot, err = SafeSnapshotByToken(ctx, "snapshot", "token")
	require.NoError(t, err)
	notification, err := SafeNotifySnapshot(ctx, "hash", 1, "receiver", user)
	require.NoError(t, err)
	assert.Equal(t, "message", notification.MessageId)

	legacy, err := Snapshots(ctx, 10, "offset", "asset", "DESC", user.UserId, user.SessionId, user.SessionPrivateKey)
	require.NoError(t, err)
	assert.Equal(t, "snapshot", legacy[0].SnapshotId)
	legacy, err = SnapshotsByToken(ctx, 10, "offset", "asset", "DESC", "token")
	require.NoError(t, err)
	legacySnapshot, err := SnapshotById(ctx, "snapshot", user.UserId, user.SessionId, user.SessionPrivateKey)
	require.NoError(t, err)
	assert.Equal(t, "snapshot", legacySnapshot.SnapshotId)
	legacySnapshot, err = SnapshotByTraceId(ctx, "trace", user.UserId, user.SessionId, user.SessionPrivateKey)
	require.NoError(t, err)
	legacySnapshot, err = SnapshotByToken(ctx, "snapshot", "token")
	require.NoError(t, err)
	legacySnapshot, err = NetworkSnapshot(ctx, "snapshot")
	require.NoError(t, err)
	legacySnapshot, err = NetworkSnapshotByToken(ctx, "snapshot", "token")
	require.NoError(t, err)
	short, err := NetworkSnapshots(ctx, 10, "offset", "asset", "ASC")
	require.NoError(t, err)
	assert.Equal(t, "snapshot", short[0].SnapshotId)
	short, err = NetworkSnapshotsByToken(ctx, 10, "offset", "asset", "invalid-order", user.UserId, user.SessionId, user.SessionPrivateKey)
	require.NoError(t, err)
}

func testUserEndpoints(t *testing.T, ctx context.Context, user *SafeUser, serverPublicKey ed25519.PublicKey) {
	t.Helper()
	created, err := CreateUserSimple(ctx, "session-public", "User")
	require.NoError(t, err)
	assert.Equal(t, "user", created.UserId)
	created, err = CreateUser(ctx, "session-public", "User", user)
	require.NoError(t, err)
	fetched, err := GetUser(ctx, "user", user)
	require.NoError(t, err)
	assert.Equal(t, "User", fetched.FullName)
	users, err := GetUsers(ctx, []string{"user"}, user)
	require.NoError(t, err)
	assert.Equal(t, "user", users[0].UserId)
	fetched, err = SearchUser(ctx, "query", user)
	require.NoError(t, err)

	me, err := UserMeWithRequestID(ctx, "token", "request-id")
	require.NoError(t, err)
	assert.True(t, me.HasPIN)
	me, err = UserMe(ctx, "token")
	require.NoError(t, err)
	me, err = RequestUserMe(ctx, user)
	require.NoError(t, err)
	updated, err := UpdateUserMe(ctx, "User", "avatar", user)
	require.NoError(t, err)
	assert.Equal(t, "user", updated.UserId)
	updated, err = UpdatePreference(ctx, PreferenceSourceAll, PreferenceSourceContacts, "USD", 1.5, user)
	require.NoError(t, err)
	updated, err = Relationship(ctx, "user", RelationshipActionAdd, user)
	require.NoError(t, err)

	require.NoError(t, UpdatePin(ctx, "old", "new", user))
	require.NoError(t, UpdateTipPin(ctx, "00", hex.EncodeToString(serverPublicKey), user))
	verified, err := VerifyPIN(ctx, "00", user)
	require.NoError(t, err)
	assert.Equal(t, "user", verified.UserId)
	verified, err = VerifyPINTip(ctx, user)
	require.NoError(t, err)
}

func testConversationEndpoints(t *testing.T, ctx context.Context, user *SafeUser) {
	t.Helper()
	conversation, err := CreateContactConversation(ctx, "recipient", user)
	require.NoError(t, err)
	assert.Equal(t, "conversation", conversation.ConversationId)
	conversation, err = CreateGroupConversation(ctx, "Group", "Announcement", []Participant{{UserId: "recipient"}}, user)
	require.NoError(t, err)
	conversation, err = ConversationShow(ctx, "conversation", user)
	require.NoError(t, err)
	conversation, err = ConversationShowByToken(ctx, "conversation", "token")
	require.NoError(t, err)
	conversation, err = JoinConversation(ctx, "conversation", user)
	require.NoError(t, err)
	conversation, err = RotateConversation(ctx, "conversation", user)
	require.NoError(t, err)
	conversation, err = UpdateParticipants(ctx, "conversation", "ADD", []Participant{{UserId: "recipient"}}, user)
	require.NoError(t, err)
	_, err = createConversation(ctx, "CONTACT", "conversation", "", "", nil, "", user)
	require.Error(t, err)
}

func testMultisigEndpoints(t *testing.T, ctx context.Context, user *SafeUser) {
	t.Helper()
	legacy, err := ReadMultisigsLegacy(ctx, 10, "offset", user)
	require.NoError(t, err)
	assert.Equal(t, "utxo", legacy[0].UTXOId)
	legacy, err = ReadMultisigs(ctx, 10, "offset", "members", "1", "unspent", user)
	require.NoError(t, err)
	request, err := CreateMultisig(ctx, "sign", "raw", user)
	require.NoError(t, err)
	assert.Equal(t, "request", request.RequestId)
	request, err = SignMultisig(ctx, "request", "pin", user)
	require.NoError(t, err)
	assert.Equal(t, "signed", request.State)
	require.NoError(t, CancelMultisig(ctx, "request", user))
	require.NoError(t, UnlockMultisig(ctx, "request", "pin", user))

	safeRequests, err := CreateSafeMultisigRequest(ctx, nil, user)
	require.NoError(t, err)
	assert.Equal(t, "request", safeRequests[0].RequestID)
	safeRequest, err := FetchSafeMultisigRequest(ctx, "request", user)
	require.NoError(t, err)
	assert.Equal(t, "request", safeRequest.RequestID)
}

func TestAPIEndpointErrorResponses(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		response string
		wantCode int
	}{
		{name: "malformed response", response: `{`, wantCode: 10002},
		{name: "API response", response: `{"error":{"status":202,"code":1234,"description":"bad request"}}`, wantCode: 1234},
		{name: "server response", status: http.StatusBadGateway, response: `unavailable`, wantCode: 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.status != 0 {
					w.WriteHeader(tt.status)
				}
				_, _ = fmt.Fprint(w, tt.response)
			}))
			_, err := Fiats(context.Background())
			require.Error(t, err)
			var apiError Error
			require.ErrorAs(t, err, &apiError)
			assert.Equal(t, tt.wantCode, apiError.Code)
		})
	}
}
