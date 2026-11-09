package security_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coresecurity "github.com/Authula/authula/core/security"
)

var _ coresecurity.TokenSigner = (*coresecurity.HMACSigner)(nil)

func TestNewHMACSigner(t *testing.T) {
	t.Parallel()

	signer := coresecurity.NewHMACSigner(rawKeyOne)

	require.NotNil(t, signer)
	assert.Implements(t, (*coresecurity.TokenSigner)(nil), signer)
}

func TestHMACSigner_Generate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		subject string
	}{
		{
			name:    "generates a token for a subject",
			subject: "user-123",
		},
		{
			name:    "generates a token for an empty subject",
			subject: "",
		},
		{
			name:    "generates a token for a unicode subject",
			subject: "user-✓",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			signer := coresecurity.NewHMACSigner(rawKeyOne)

			token, err := signer.Generate(context.Background(), tt.subject)

			require.NoError(t, err)
			assert.NotEmpty(t, token)

			signature, err := base64.RawURLEncoding.DecodeString(token)
			require.NoError(t, err)
			assert.Len(t, signature, sha256.Size)
		})
	}
}

func TestHMACSigner_GenerateUsesRandomNonce(t *testing.T) {
	t.Parallel()

	const tokenCount = 16

	signer := coresecurity.NewHMACSigner(rawKeyOne)
	tokens := make(map[string]struct{}, tokenCount)

	for range tokenCount {
		token, err := signer.Generate(context.Background(), "same-subject")
		require.NoError(t, err)

		_, exists := tokens[token]
		assert.False(t, exists, "Generate returned a duplicate token")

		tokens[token] = struct{}{}
	}

	assert.Len(t, tokens, tokenCount)
}

func TestHMACSigner_Verify(t *testing.T) {
	t.Parallel()

	payload := []byte("session-payload")
	otherPayload := []byte("other-session-payload")
	validToken := signHMACPayload(t, rawKeyOne, payload)
	tokenForOtherPayload := signHMACPayload(t, rawKeyOne, otherPayload)
	tokenSignedWithEncodedKey := signHMACPayload(t, []byte(validSecret), payload)

	tests := []struct {
		name    string
		token   string
		payload []byte
		wantOK  bool
		wantErr string
	}{
		{
			name:    "accepts a valid signature",
			token:   validToken,
			payload: payload,
			wantOK:  true,
		},
		{
			name:    "rejects a signature for a different payload",
			token:   tokenForOtherPayload,
			payload: payload,
			wantErr: "invalid token signature",
		},
		{
			name:    "rejects a token signed with the base64 text as the key",
			token:   tokenSignedWithEncodedKey,
			payload: payload,
			wantErr: "invalid token signature",
		},
		{
			name:    "rejects an empty token",
			token:   "",
			payload: payload,
			wantErr: "invalid token signature",
		},
		{
			name:    "rejects malformed base64",
			token:   "not-base64!!",
			payload: payload,
			wantErr: "illegal base64 data",
		},
		{
			name:    "rejects padded base64",
			token:   validToken + "=",
			payload: payload,
			wantErr: "illegal base64 data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			signer := coresecurity.NewHMACSigner(rawKeyOne)

			ok, err := signer.Verify(context.Background(), tt.token, tt.payload)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				assert.False(t, ok)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func signHMACPayload(t *testing.T, key, payload []byte) string {
	t.Helper()

	mac := hmac.New(sha256.New, key)
	_, err := mac.Write(payload)
	require.NoError(t, err)

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
