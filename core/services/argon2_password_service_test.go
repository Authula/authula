package services_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coreservices "github.com/Authula/authula/core/services"
	rootservices "github.com/Authula/authula/services"
)

const (
	argon2SaltLength = 16
	argon2HashLength = 32
)

func TestArgon2PasswordService_Hash(t *testing.T) {
	t.Parallel()

	svc := coreservices.NewArgon2PasswordService()
	require.Implements(t, (*rootservices.PasswordService)(nil), svc)

	first, err := svc.Hash("correct horse battery staple")
	require.NoError(t, err)
	second, err := svc.Hash("correct horse battery staple")
	require.NoError(t, err)

	assert.NotEmpty(t, first)
	assert.NotEqual(t, first, second, "each hash must use a fresh random salt")
	assert.NotContains(t, first, "=", "hash must be raw-std base64 without padding")

	decoded, err := base64.RawStdEncoding.DecodeString(first)
	require.NoError(t, err)
	assert.Len(t, decoded, argon2SaltLength+argon2HashLength, "salt followed by a 32-byte derived key")
}

func TestArgon2PasswordService_Verify(t *testing.T) {
	t.Parallel()

	svc := coreservices.NewArgon2PasswordService()
	encoded, err := svc.Hash("s3cret")
	require.NoError(t, err)

	decoded, err := base64.RawStdEncoding.DecodeString(encoded)
	require.NoError(t, err)

	// Same 16-byte salt but a hash portion of the wrong length.
	wrongHashLength := base64.RawStdEncoding.EncodeToString(
		append(decoded[:argon2SaltLength:argon2SaltLength], make([]byte, argon2SaltLength)...),
	)

	tests := []struct {
		name     string
		password string
		encoded  string
		want     bool
	}{
		{
			name:     "correct password",
			password: "s3cret",
			encoded:  encoded,
			want:     true,
		},
		{
			name:     "wrong password",
			password: "s3cret!",
			encoded:  encoded,
			want:     false,
		},
		{
			name:     "invalid base64",
			password: "s3cret",
			encoded:  "not*base64!",
			want:     false,
		},
		{
			name:     "decoded payload shorter than the salt",
			password: "s3cret",
			encoded:  base64.RawStdEncoding.EncodeToString([]byte("short")),
			want:     false,
		},
		{
			name:     "empty encoded hash",
			password: "s3cret",
			encoded:  "",
			want:     false,
		},
		{
			name:     "hash portion has the wrong length",
			password: "s3cret",
			encoded:  wrongHashLength,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Subtests stay sequential on purpose: each Verify allocates 64 MiB and
			// parallel cases would multiply that memory inside the race suite.
			assert.Equal(t, tt.want, svc.Verify(tt.password, tt.encoded))
		})
	}
}
