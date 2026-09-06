package bot

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"

	"filippo.io/edwards25519"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTIPSigningWithSeedAndCanonicalSpendKeys(t *testing.T) {
	seed := sha256.Sum256([]byte(t.Name()))
	private := ed25519.NewKeyFromSeed(seed[:])
	public := private.Public().(ed25519.PublicKey)
	digest := sha512.Sum512(seed[:])
	scalar, err := edwards25519.NewScalar().SetBytesWithClamping(digest[:32])
	require.NoError(t, err)
	var canonical crypto.Key
	copy(canonical[:], scalar.Bytes())

	for name, body := range map[string][]byte{
		"verify":  TIPBodyForVerify(123),
		"address": TipBodyForAddressAdd("asset", "destination", "tag", "label"),
	} {
		t.Run(name, func(t *testing.T) {
			seedSignature, err := signTipBody(body, hex.EncodeToString(seed[:]), false)
			require.NoError(t, err)
			assert.Equal(t, hex.EncodeToString(ed25519.Sign(private, body)), seedSignature, "seed signatures must remain compatible")

			signature, err := signTipBody(body, canonical.String(), true)
			require.NoError(t, err)
			raw, err := hex.DecodeString(signature)
			require.NoError(t, err)
			require.True(t, ed25519.Verify(public, body, raw), "canonical keys must sign for the account's existing public key")
			changedBody := bytes.Clone(body)
			changedBody[0] ^= 1
			assert.False(t, ed25519.Verify(public, changedBody, raw))

			if len(body) == len(crypto.Hash{}) {
				var message crypto.Hash
				copy(message[:], body)
				assert.Equal(t, canonical.Sign(message).String(), signature, "use the existing Mixin scalar signing construction")
			}
		})
	}
}

func TestTIPSigningRejectsInvalidCanonicalSpendKey(t *testing.T) {
	_, err := signTipBody(TIPBodyForVerify(123), hex.EncodeToString(bytes.Repeat([]byte{0xff}, 32)), true)
	require.Error(t, err)
}

func TestVerifyPINTipWithCanonicalSpendKey(t *testing.T) {
	user, sessionPrivate := newTestSafeUser(t.Name())
	spendSeed := sha256.Sum256([]byte("canonical TIP spend key"))
	spendPrivate := ed25519.NewKeyFromSeed(spendSeed[:])
	digest := sha512.Sum512(spendSeed[:])
	scalar, err := edwards25519.NewScalar().SetBytesWithClamping(digest[:32])
	require.NoError(t, err)
	user.SpendPrivateKey = hex.EncodeToString(scalar.Bytes())
	user.IsSpendPrivateSum = true
	serverSeed := sha256.Sum256([]byte("TIP verification server"))
	serverPrivate := ed25519.NewKeyFromSeed(serverSeed[:])
	user.ServerPublicKey = hex.EncodeToString(serverPrivate.Public().(ed25519.PublicKey))
	shared, err := SharedKey(sessionPrivate.Public().(ed25519.PublicKey), serverPrivate)
	require.NoError(t, err)
	block, err := aes.NewCipher(shared[:])
	require.NoError(t, err)

	useTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pin/verify", r.URL.Path)
		var request struct {
			PIN       string `json:"pin_base64"`
			Timestamp int64  `json:"timestamp"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		encrypted, err := base64.RawURLEncoding.DecodeString(request.PIN)
		require.NoError(t, err)
		require.Len(t, encrypted, 112)
		plain := make([]byte, len(encrypted)-aes.BlockSize)
		cipher.NewCBCDecrypter(block, encrypted[:aes.BlockSize]).CryptBlocks(plain, encrypted[aes.BlockSize:])
		assert.Equal(t, bytes.Repeat([]byte{16}, 16), plain[80:])
		assert.Equal(t, uint64(request.Timestamp), binary.LittleEndian.Uint64(plain[72:80]))
		if !ed25519.Verify(spendPrivate.Public().(ed25519.PublicKey), TIPBodyForVerify(request.Timestamp), plain[:ed25519.SignatureSize]) {
			t.Error("PIN request contains an invalid TIP signature")
			_, _ = w.Write([]byte(`{"error":{"status":202,"code":403,"description":"invalid TIP signature"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"user_id":"verified"}}`))
	}))
	verified, err := VerifyPINTip(context.Background(), user)
	require.NoError(t, err)
	require.NotNil(t, verified)
	assert.Equal(t, "verified", verified.UserId)
}
