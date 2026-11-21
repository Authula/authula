package services

import (
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

	assert.True(t, strings.HasPrefix(first, "authula_cs_"))
	assert.Len(t, first, len("authula_cs_")+clientSecretRandomLength)
	assert.NotEqual(t, first, second)
}

func TestSHA256ClientSecretService_HashAndVerify(t *testing.T) {
	t.Parallel()

	service := NewSHA256ClientSecretService("authula_cs_")
	secret, err := service.Generate()
	require.NoError(t, err)

	hash, salt, err := service.Hash(secret)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEmpty(t, salt)
	assert.NotContains(t, hash, secret)

	otherHash, otherSalt, err := service.Hash(secret)
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
		{name: "correct secret", secret: secret, hash: hash, salt: salt, want: true},
		{name: "wrong secret", secret: secret + "x", hash: hash, salt: salt, want: false},
		{name: "empty secret", secret: "", hash: hash, salt: salt, want: false},
		{name: "empty hash", secret: secret, hash: "", salt: salt, want: false},
		{name: "empty salt", secret: secret, hash: hash, salt: "", want: false},
		{name: "malformed hash", secret: secret, hash: "not base64!", salt: salt, want: false},
		{name: "malformed salt", secret: secret, hash: hash, salt: "not base64!", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, service.Verify(tt.secret, tt.hash, tt.salt))
		})
	}
}

func TestSHA256ClientSecretService_Prefix(t *testing.T) {
	t.Parallel()

	service := NewSHA256ClientSecretService("authula_cs_")

	assert.Equal(t, "authula_cs_a", service.Prefix("authula_cs_abcdefghijklmnop"))
	assert.Equal(t, "short", service.Prefix("short"))
	assert.Equal(t, "", service.Prefix(""))
}
