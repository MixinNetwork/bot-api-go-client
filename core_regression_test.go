package bot

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"slices"
	"testing"

	"github.com/MixinNetwork/go-number"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAesRejectsInvalidInputs(t *testing.T) {
	_, err := AesEncrypt([]byte("short"), []byte("message"))
	require.Error(t, err)

	_, err = AesDecrypt(make([]byte, 16), []byte("short"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too short")

	encrypted, err := AesEncrypt(make([]byte, 16), []byte("message"))
	require.NoError(t, err)
	encrypted[len(encrypted)-1] ^= 1
	_, err = AesDecrypt(make([]byte, 16), encrypted)
	assert.Error(t, err)
}

func TestSharedKeyIsSymmetric(t *testing.T) {
	seedA := sha256.Sum256([]byte("alice"))
	seedB := sha256.Sum256([]byte("bob"))
	privateA := ed25519.NewKeyFromSeed(seedA[:])
	privateB := ed25519.NewKeyFromSeed(seedB[:])

	sharedA, err := SharedKey(privateB.Public().(ed25519.PublicKey), privateA)
	require.NoError(t, err)
	sharedB, err := SharedKey(privateA.Public().(ed25519.PublicKey), privateB)
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, sharedA)
	assert.Equal(t, sharedA, sharedB)
}

func TestHashAndConversationHelpersDoNotMutateInputs(t *testing.T) {
	members := []string{"z", "a", "m"}
	originalMembers := slices.Clone(members)
	first := HashMembers(members)
	second := HashMembers([]string{"m", "z", "a"})
	assert.Equal(t, first, second)
	assert.Equal(t, originalMembers, members)

	participants := []string{
		"f937ca18-d1ff-46f5-99e8-e23fbd6fd5f2",
		"0e0a20c8-31b8-4093-81b8-9cebd9bc8afc",
	}
	originalParticipants := slices.Clone(participants)
	GroupConversationId(
		"c8cb0ac7-d456-4341-be66-0b143aa09922",
		"group",
		participants,
		"01d21e2c-76f5-4940-8ea0-9b7f21728674",
	)
	assert.Equal(t, originalParticipants, participants)
}

func TestCollectionHelpers(t *testing.T) {
	input := []any{1, 2, 3, 4, 5}
	assert.Equal(t, [][]any{{1, 2}, {3, 4}, {5}}, Chunked(input, 2))
	assert.Nil(t, Chunked(input, 0))
	assert.Nil(t, Chunked(input, -1))
	assert.Nil(t, Chunked(nil, 2))

	assert.Equal(t, []string{"a", "b", "c"}, MakeUniqueStringSlice([]string{"a", "b", "a", "c", "b"}))
	assert.Empty(t, MakeUniqueStringSlice(nil))
}

func TestUniqueObjectID(t *testing.T) {
	id := UniqueObjectId("one", "two")
	assert.Equal(t, id, UniqueObjectId("one", "two"))
	assert.NotEqual(t, id, UniqueObjectId("one", "three"))
	parsed, err := uuid.FromString(id)
	require.NoError(t, err)
	assert.Equal(t, byte(3), parsed.Version())
	assert.Equal(t, uuid.VariantRFC9562, parsed.Variant())
}

func TestHMACHelpers(t *testing.T) {
	data := []byte("The quick brown fox jumps over the lazy dog")
	assert.Equal(t, "f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8", HmacSha256([]byte("key"), data))
	assert.Equal(t, "de7c9b85b8b78aa6bc8a7a36f70a90701c9db4d9", HmacSha1("key", data))
}

func TestTIPBodies(t *testing.T) {
	verify := TIPBodyForVerify(42)
	assert.Equal(t, []byte("TIP:VERIFY:00000000000000000000000000000042"), verify)

	amount := number.FromString("1.2300")
	want := TipBody(TIPTransferCreate + "asset" + "recipient" + amount.Persist() + "trace" + "memo")
	assert.Equal(t, want, TipBodyForTransfer("asset", "recipient", amount, "trace", "memo"))
	assert.Equal(t, TipBody(TIPPhoneNumberUpdate+"verification"+"1234"), TipBodyForPhoneNumberUpdate("verification", "1234"))
	assert.Equal(t, TipBody(TIPEmergencyContactCreate+"verification"+"1234"), TipBodyForEmergencyContactCreate("verification", "1234"))
	assert.Equal(t, TipBody(TIPAddressAdd+"asset"+"destination"+"tag"+"label"), TipBodyForAddressAdd("asset", "destination", "tag", "label"))
	assert.Equal(t, TipBody(TIPProvisioningUpdate+"device"+"secret"), TipBodyForProvisioningUpdate("device", "secret"))
	assert.Equal(t, TipBody(TIPOwnershipTransfer+"receiver"), TipBodyForOwnershipTransfer("receiver"))
	assert.Equal(t, TipBody(TIPSequencerRegister+"user"+"public"), TIPBodyForSequencerRegister("user", "public"))

	public := bytes.Repeat([]byte{1}, ed25519.PublicKeySize)
	migrated := TIPMigrateBody(public)
	decoded, err := hex.DecodeString(migrated)
	require.NoError(t, err)
	require.Len(t, decoded, ed25519.PublicKeySize+8)
	assert.Equal(t, byte(1), decoded[len(decoded)-1])
}

func TestURLSchemes(t *testing.T) {
	assert.Equal(t, "mixin://users/user-id", SchemeUsers("user-id"))
	assert.Equal(t, "mixin://codes/code-id", SchemeCodes("code-id"))
	assert.Equal(t, "mixin://snapshots/snapshot-id?trace=trace-id", SchemeSnapshots("snapshot-id", "trace-id"))
	assert.Equal(t, "mixin://conversations/conversation-id?user=user-id", SchemeConversations("conversation-id", "user-id"))

	pay, err := url.Parse(SchemePay("asset", "trace", "recipient", "hello world", number.FromString("1.2")))
	require.NoError(t, err)
	assert.Equal(t, "mixin", pay.Scheme)
	assert.Equal(t, "pay", pay.Host)
	assert.Equal(t, "asset", pay.Query().Get("asset"))
	assert.Equal(t, "hello world", pay.Query().Get("memo"))

	payload := []byte{0xfb, 0xff, 0x00}
	send, err := url.Parse(SchemeSend(SendSchemeCategoryImage, payload, "conversation"))
	require.NoError(t, err)
	decoded, err := base64.StdEncoding.DecodeString(send.Query().Get("data"))
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)
	assert.Equal(t, SendSchemeCategoryImage, send.Query().Get("category"))
	assert.Equal(t, "conversation", send.Query().Get("conversation"))
}

func TestMixAddressRejectsMalformedPayloads(t *testing.T) {
	_, err := NewMixAddressFromBytesUnchecked(nil)
	require.Error(t, err)

	payload := append([]byte{MixAddressVersion, 2, 1}, make([]byte, 16)...)
	_, err = NewMixAddressFromBytesUnchecked(payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "threshold")

	_, err = NewMixAddressFromString("invalid")
	assert.Error(t, err)
}
