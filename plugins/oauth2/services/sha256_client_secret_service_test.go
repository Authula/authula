package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSHA256ClientSecretService_Generate(t *testing.T) {
	t.Parallel()

	service := NewSHA256ClientSecretService("authula_cs_")

	first, err := service.Generate()
	require.NoError(t, err)
	second, err := service.Generate()
	require.NoError(t, err)

	assert.NotEqual(t, first.Value, second.Value)
	assert.True(t, strings.HasPrefix(first.Value, "authula_cs_"))
	assert.Len(t, first.Value, len("authula_cs_")+clientSecretRandomLength)
	assert.Equal(t, "authula_cs_", first.Prefix)
	assert.Len(t, first.Start, clientSecretHintLength)
	assert.Len(t, first.Last, clientSecretHintLength)
	assert.Equal(t, first.Value[len("authula_cs_"):len("authula_cs_")+clientSecretHintLength], first.Start)
	assert.Equal(t, first.Value[len(first.Value)-clientSecretHintLength:], first.Last)
	assert.NotEmpty(t, first.Hash)
	assert.NotEmpty(t, first.Salt)
	assert.NotContains(t, first.Hash, first.Value)

	assert.True(t, service.Verify(first.Value, first.Hash, first.Salt))
	assert.False(t, service.Verify(second.Value, first.Hash, first.Salt))
}

func TestSHA256ClientSecretService_GenerateFailures(t *testing.T) {
	t.Parallel()

	randomErr := errors.New("random source failed")

	tests := []struct {
		name      string
		configure func(*sha256ClientSecretService)
		wantErr   string
	}{
		{
			name: "random string failure",
			configure: func(service *sha256ClientSecretService) {
				service.randomString = func(int) (string, error) { return "", randomErr }
			},
			wantErr: "failed to generate client secret",
		},
		{
			name: "salt failure",
			configure: func(service *sha256ClientSecretService) {
				service.readRandom = func([]byte) (int, error) { return 0, randomErr }
			},
			wantErr: "failed to generate client secret salt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewSHA256ClientSecretService("authula_cs_").(*sha256ClientSecretService)
			tt.configure(service)

			generated, err := service.Generate()

			require.ErrorContains(t, err, tt.wantErr)
			assert.ErrorIs(t, err, randomErr)
			assert.Nil(t, generated)
		})
	}
}

func TestSHA256ClientSecretService_HashAndVerify(t *testing.T) {
	t.Parallel()

	service := NewSHA256ClientSecretService("authula_cs_")
	secret, err := service.Generate()
	require.NoError(t, err)

	hash, salt, err := service.Hash(secret.Value)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEmpty(t, salt)
	assert.NotContains(t, hash, secret.Value)

	otherHash, otherSalt, err := service.Hash(secret.Value)
	require.NoError(t, err)
	assert.NotEqual(t, hash, otherHash)
	assert.NotEqual(t, salt, otherSalt)

	tests := []struct {
		name   string
		secret string
		hash   string
		salt   string
		want   bool
	}{
		{name: "correct secret", secret: secret.Value, hash: hash, salt: salt, want: true},
		{name: "generated hash", secret: secret.Value, hash: secret.Hash, salt: secret.Salt, want: true},
		{name: "wrong secret", secret: secret.Value + "x", hash: hash, salt: salt, want: false},
		{name: "empty secret", secret: "", hash: hash, salt: salt, want: false},
		{name: "empty hash", secret: secret.Value, hash: "", salt: salt, want: false},
		{name: "empty salt", secret: secret.Value, hash: hash, salt: "", want: false},
		{name: "malformed hash", secret: secret.Value, hash: "not base64!", salt: salt, want: false},
		{name: "malformed salt", secret: secret.Value, hash: hash, salt: "not base64!", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, service.Verify(tt.secret, tt.hash, tt.salt))
		})
	}
}
