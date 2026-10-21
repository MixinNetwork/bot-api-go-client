package bot

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptMessageDataRoundTrip(t *testing.T) {
	senderSeed := sha256.Sum256([]byte("sender"))
	recipientSeed := sha256.Sum256([]byte("recipient"))
	recipientPrivate := ed25519.NewKeyFromSeed(recipientSeed[:])
	recipientPublic, err := PublicKeyToCurve25519(recipientPrivate.Public().(ed25519.PublicKey))
	require.NoError(t, err)
	sessionID := "489cfe0b-08d8-47f4-a330-fff193cc8086"
	session := &Session{
		UserID:    "e95b1d4e-4d49-4ac3-9402-988804458adc",
		SessionID: sessionID,
		PublicKey: base64.RawURLEncoding.EncodeToString(recipientPublic),
	}
	want := base64.RawURLEncoding.EncodeToString([]byte("secret message"))

	encrypted, err := EncryptMessageData(want, []*Session{session}, hex.EncodeToString(senderSeed[:]))
	require.NoError(t, err)
	assert.NotEqual(t, want, encrypted)
	decrypted, err := DecryptMessageData(encrypted, sessionID, hex.EncodeToString(recipientSeed[:]))
	require.NoError(t, err)
	assert.Equal(t, want, decrypted)
}

func TestEncryptMessageDataValidation(t *testing.T) {
	seed := sha256.Sum256([]byte("sender"))
	privateKey := hex.EncodeToString(seed[:])
	validSession := &Session{
		SessionID: "489cfe0b-08d8-47f4-a330-fff193cc8086",
		PublicKey: base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
	}

	tests := []struct {
		name     string
		data     string
		sessions []*Session
		key      string
	}{
		{name: "bad data", data: "%%%", sessions: []*Session{validSession}, key: privateKey},
		{name: "no sessions", data: "", key: privateKey},
		{name: "nil session", data: "", sessions: []*Session{nil}, key: privateKey},
		{name: "bad private key", data: "", sessions: []*Session{validSession}, key: "00"},
		{name: "bad public key", data: "", sessions: []*Session{{SessionID: validSession.SessionID, PublicKey: "AA"}}, key: privateKey},
		{name: "bad session ID", data: "", sessions: []*Session{{SessionID: "bad", PublicKey: validSession.PublicKey}}, key: privateKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncryptMessageData(tt.data, tt.sessions, tt.key)
			assert.Error(t, err)
		})
	}
}

func TestDecryptMessageDataValidation(t *testing.T) {
	senderSeed := sha256.Sum256([]byte("sender"))
	recipientSeed := sha256.Sum256([]byte("recipient"))
	recipientPrivate := ed25519.NewKeyFromSeed(recipientSeed[:])
	recipientPublic, err := PublicKeyToCurve25519(recipientPrivate.Public().(ed25519.PublicKey))
	require.NoError(t, err)
	sessionID := "489cfe0b-08d8-47f4-a330-fff193cc8086"
	session := &Session{SessionID: sessionID, PublicKey: base64.RawURLEncoding.EncodeToString(recipientPublic)}
	encrypted, err := EncryptMessageData("cGF5bG9hZA", []*Session{session}, hex.EncodeToString(senderSeed[:]))
	require.NoError(t, err)
	validKey := hex.EncodeToString(recipientSeed[:])

	t.Run("bad base64", func(t *testing.T) {
		_, err := DecryptMessageData("%%%", sessionID, validKey)
		assert.Error(t, err)
	})
	t.Run("bad private key", func(t *testing.T) {
		_, err := DecryptMessageData(encrypted, sessionID, "00")
		assert.Error(t, err)
	})
	t.Run("short payload", func(t *testing.T) {
		_, err := DecryptMessageData(base64.RawURLEncoding.EncodeToString([]byte{1}), sessionID, validKey)
		assert.Error(t, err)
	})
	t.Run("unsupported version", func(t *testing.T) {
		raw := decodeRawBase64(t, encrypted)
		raw[0] = 2
		_, err := DecryptMessageData(base64.RawURLEncoding.EncodeToString(raw), sessionID, validKey)
		assert.Error(t, err)
	})
	t.Run("invalid session count", func(t *testing.T) {
		raw := decodeRawBase64(t, encrypted)
		binary.LittleEndian.PutUint16(raw[1:3], 100)
		_, err := DecryptMessageData(base64.RawURLEncoding.EncodeToString(raw), sessionID, validKey)
		assert.Error(t, err)
	})
	t.Run("missing session", func(t *testing.T) {
		_, err := DecryptMessageData(encrypted, "c76310d8-c563-499e-9866-c61ae2cbee11", validKey)
		assert.Error(t, err)
	})
	t.Run("tampered ciphertext", func(t *testing.T) {
		raw := decodeRawBase64(t, encrypted)
		raw[len(raw)-1] ^= 1
		_, err := DecryptMessageData(base64.RawURLEncoding.EncodeToString(raw), sessionID, validKey)
		assert.Error(t, err)
	})
}

func TestDecryptMessageDataDoesNotExposeKeyPadding(t *testing.T) {
	senderSeed := sha256.Sum256([]byte("padding test sender"))
	recipientSeed := sha256.Sum256([]byte("padding test recipient"))
	recipientPrivate := ed25519.NewKeyFromSeed(recipientSeed[:])
	recipientPublic, err := PublicKeyToCurve25519(recipientPrivate.Public().(ed25519.PublicKey))
	require.NoError(t, err)
	const sessionID = "489cfe0b-08d8-47f4-a330-fff193cc8086"
	session := &Session{SessionID: sessionID, PublicKey: base64.RawURLEncoding.EncodeToString(recipientPublic)}
	encrypted, err := EncryptMessageData("cGF5bG9hZA", []*Session{session}, hex.EncodeToString(senderSeed[:]))
	require.NoError(t, err)
	privateKey := hex.EncodeToString(recipientSeed[:])

	// Both messages fail authentication. Altering the first wrapped-key block
	// also changes the last padding byte from 16 to 17 in the second block.
	badTag := decodeRawBase64(t, encrypted)
	badTag[len(badTag)-1] ^= 1
	_, tagErr := DecryptMessageData(base64.RawURLEncoding.EncodeToString(badTag), sessionID, privateKey)
	require.Error(t, tagErr)
	badPadding := decodeRawBase64(t, encrypted)
	const firstKeyBlockEnd = 35 + 16 + 16 + 16
	badPadding[firstKeyBlockEnd-1] ^= 1
	_, paddingErr := DecryptMessageData(base64.RawURLEncoding.EncodeToString(badPadding), sessionID, privateKey)
	require.Error(t, paddingErr)
	assert.Equal(t, tagErr.Error(), paddingErr.Error(), "decryption errors must not reveal CBC padding validity")

	// Padding lengths 1 through 15 are not valid for the fixed 16-byte key.
	// They must not produce a separate missing-session error either.
	for padding := byte(1); padding < 16; padding++ {
		malformed := decodeRawBase64(t, encrypted)
		for i := byte(0); i < padding; i++ {
			malformed[firstKeyBlockEnd-1-int(i)] ^= 16 ^ padding
		}
		_, err := DecryptMessageData(base64.RawURLEncoding.EncodeToString(malformed), sessionID, privateKey)
		require.Error(t, err)
		assert.Equal(t, tagErr.Error(), err.Error(), "padding length %d must not be observable", padding)
	}

	// Corrupt only the padding block, keeping the message key and GCM tag
	// intact. Successful payload authentication must not bypass padding checks.
	senderPrivate := ed25519.NewKeyFromSeed(senderSeed[:])
	shared, err := SharedKey(senderPrivate.Public().(ed25519.PublicKey), recipientPrivate)
	require.NoError(t, err)
	block, err := aes.NewCipher(shared[:])
	require.NoError(t, err)
	paddingOnly := decodeRawBase64(t, encrypted)
	invalidPadding := make([]byte, aes.BlockSize)
	cipher.NewCBCEncrypter(block, paddingOnly[firstKeyBlockEnd-aes.BlockSize:firstKeyBlockEnd]).CryptBlocks(
		paddingOnly[firstKeyBlockEnd:firstKeyBlockEnd+aes.BlockSize], invalidPadding,
	)
	plaintext, err := DecryptMessageData(base64.RawURLEncoding.EncodeToString(paddingOnly), sessionID, privateKey)
	require.Error(t, err)
	assert.Empty(t, plaintext)
	assert.Equal(t, tagErr.Error(), err.Error())
}

func decodeRawBase64(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	require.NoError(t, err)
	return decoded
}
