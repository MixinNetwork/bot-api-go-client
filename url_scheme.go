package bot

import (
	"encoding/base64"
	"net/url"

	"github.com/MixinNetwork/go-number"
)

const _urlScheme = "mixin"

// SchemeUsers scheme of a user
//
//	userId required
//
// https://developers.mixin.one/docs/schema#popups-user-profile
func SchemeUsers(userId string) string {
	u := url.URL{
		Scheme: _urlScheme,
		Host:   "users",
		Path:   userId,
	}

	return u.String()
}

// SchemeTransfer scheme of a transfer
//
//	userId required
//
// https://developers.mixin.one/docs/schema#invoke-transfer-page
func SchemeTransfer(userId string) string {
	u := url.URL{
		Scheme: _urlScheme,
		Host:   "transfer",
		Path:   userId,
	}

	return u.String()
}

// SchemePay scheme of a pay
//
//	assetId required
//	recipientId required, receiver's user id
//	amount require, transfer amount
//	traceId optional, UUID, prevent duplicate payment
//	memo optional, transaction memo
//
// https://developers.mixin.one/docs/schema#invoke-payment-page
func SchemePay(assetId, traceId, recipientId, memo string, amount number.Decimal) string {
	q := url.Values{}
	q.Set("asset", assetId)
	q.Set("trace", traceId)
	q.Set("amount", amount.String())
	q.Set("recipient", recipientId)
	q.Set("memo", memo)

	u := url.URL{
		Scheme:   _urlScheme,
		Host:     "pay",
		RawQuery: q.Encode(),
	}

	return u.String()
}

// SchemeCodes scheme of a code
//
//	code required
//
// https://developers.mixin.one/docs/schema#popus-code-info
func SchemeCodes(codeId string) string {
	u := url.URL{
		Scheme: _urlScheme,
		Host:   "codes",
		Path:   codeId,
	}

	return u.String()
}

// SchemeSnapshots scheme of a snapshot
//
//	snapshotId required if no traceId
//	traceId required if no snapshotId
//
// https://developers.mixin.one/docs/schema#transfer-details-interface
func SchemeSnapshots(snapshotId, traceId string) string {
	u := url.URL{
		Scheme: _urlScheme,
		Host:   "snapshots",
	}

	if snapshotId != "" {
		u.Path = snapshotId
	}

	if traceId != "" {
		query := url.Values{}
		query.Set("trace", traceId)
		u.RawQuery = query.Encode()
	}

	return u.String()
}

// SchemeConversations scheme of a conversation
//
//	userID optional, for user conversation only, if there's not conversation with the user, messenger will create the conversation first
//
// https://developers.mixin.one/docs/schema#open-an-conversation
func SchemeConversations(conversationID, userID string) string {
	u := url.URL{
		Scheme: _urlScheme,
		Host:   "conversations",
	}

	if conversationID != "" {
		u.Path = conversationID
	}

	if userID != "" {
		query := url.Values{}
		query.Set("user", userID)
		u.RawQuery = query.Encode()
	}

	return u.String()
}

// SchemeApps scheme of an app
//
//	appID required, userID of an app
//	action optional, action about this scheme, default is "open"
//	params optional, parameters of any name or type can be passed when opening the bot homepage to facilitate the development of features like invitation codes, visitor tracking, etc
//
// https://developers.mixin.one/docs/schema#popups-bot-profile
func SchemeApps(appID, action string, params map[string]string) string {
	u := url.URL{
		Scheme: _urlScheme,
		Host:   "apps",
	}

	if appID != "" {
		u.Path = appID
	}

	query := url.Values{}
	if action != "" {
		query.Set("action", action)
	} else {
		query.Set("action", "open")
	}
	for k, v := range params {
		query.Set(k, v)
	}
	u.RawQuery = query.Encode()

	return u.String()
}

type SendSchemeCategory = string

const (
	SendSchemeCategoryText    SendSchemeCategory = "text"
	SendSchemeCategoryImage   SendSchemeCategory = "image"
	SendSchemeCategoryContact SendSchemeCategory = "contact"
	SendSchemeCategoryAppCard SendSchemeCategory = "app_card"
	SendSchemeCategoryLive    SendSchemeCategory = "live"
	SendSchemeCategoryPost    SendSchemeCategory = "post"
)

// SchemeSend scheme of a share
//
//	category required, category of shared content
//	data required, shared content
//	conversationID optional, If you specify conversation and it is the conversation of the user's current session, the confirmation box shown above will appear, the message will be sent after the user clicks the confirmation; if the conversation is not specified or is not the conversation of the current session, an interface where the user chooses which session to share with will show up.
//
// https://developers.mixin.one/docs/schema#sharing
func SchemeSend(category SendSchemeCategory, data []byte, conversationID string) string {
	u := url.URL{
		Scheme: _urlScheme,
		Host:   "send",
	}
	query := url.Values{}
	query.Set("category", category)
	if len(data) > 0 {
		query.Set("data", url.QueryEscape(base64.StdEncoding.EncodeToString(data)))
	}
	if conversationID != "" {
		query.Set("conversation", conversationID)
	}
	u.RawQuery = query.Encode()

	return u.String()
}
