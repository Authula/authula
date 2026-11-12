package security_test

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coresecurity "github.com/Authula/authula/core/security"
)

const validSecret = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

var rawKeyOne = mustDecodeBase64(validSecret)

func mustDecodeBase64(value string) []byte {
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		panic(err)
	}

	return raw
}

func TestDecodeSecret(t *testing.T) {
	t.Parallel()

	unpadded := base64.RawStdEncoding.EncodeToString(rawKeyOne)
	urlSafe := base64.RawURLEncoding.EncodeToString(rawKeyOne)
	shortKey := base64.StdEncoding.EncodeToString(rawKeyOne[:31])
	longKey := base64.StdEncoding.EncodeToString(append(append([]byte{}, rawKeyOne...), 0x00))
	hexKey := hex.EncodeToString(rawKeyOne)
	doubleEncoded := base64.StdEncoding.EncodeToString([]byte(validSecret))

	tests := []struct {
		name    string
		value   string
		want    []byte
		wantErr string
	}{
		{
			name:  "accepts standard padded base64",
			value: validSecret,
			want:  rawKeyOne,
		},
		{
			name:  "trims surrounding whitespace",
			value: "  \t" + validSecret + "\n ",
			want:  rawKeyOne,
		},
		{
			name:    "rejects an internal space",
			value:   validSecret[:10] + " " + validSecret[10:],
			wantErr: "value must not contain whitespace",
		},
		{
			name:    "rejects an internal newline",
			value:   validSecret[:10] + "\n" + validSecret[10:],
			wantErr: "value must not contain whitespace",
		},
		{
			name:    "rejects unpadded base64",
			value:   unpadded,
			wantErr: "value must be standard padded Base64",
		},
		{
			name:    "rejects url-safe base64",
			value:   urlSafe,
			wantErr: "value must be standard padded Base64",
		},
		{
			name:    "rejects hex output",
			value:   hexKey,
			wantErr: "value must decode to 32 bytes, got 48",
		},
		{
			name:    "rejects double-encoded base64",
			value:   doubleEncoded,
			wantErr: "value must decode to 32 bytes, got 44",
		},
		{
			name:    "rejects an empty value",
			value:   "",
			wantErr: "value must not be empty",
		},
		{
			name:    "rejects a whitespace-only value",
			value:   "   ",
			wantErr: "value must not be empty",
		},
		{
			name:    "rejects invalid base64",
			value:   "not-base64!!",
			wantErr: "value must be standard padded Base64",
		},
		{
			name:    "rejects a 31-byte key",
			value:   shortKey,
			wantErr: "value must decode to 32 bytes, got 31",
		},
		{
			name:    "rejects a 33-byte key",
			value:   longKey,
			wantErr: "value must decode to 32 bytes, got 33",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := coresecurity.DecodeSecret(tt.value)

			if tt.wantErr != "" {
				require.ErrorIs(t, err, coresecurity.ErrInvalidSecret)
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Len(t, got, coresecurity.SecretKeySize)
		})
	}
}
