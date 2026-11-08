package repositories_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/chacha20poly1305"

	corerepos "github.com/Authula/authula/core/repositories"
	"github.com/Authula/authula/models"
)

var (
	keyOneRaw   = []byte("0123456789abcdef0123456789abcdef")
	keyTwoRaw   = []byte("fedcba9876543210fedcba9876543210")
	keyThreeRaw = []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	keyOne   = base64.StdEncoding.EncodeToString(keyOneRaw)
	keyTwo   = base64.StdEncoding.EncodeToString(keyTwoRaw)
	keyThree = base64.StdEncoding.EncodeToString(keyThreeRaw)
)

func newTokenRepo(t *testing.T, secret string, secrets []models.SecretConfig) corerepos.TokenRepository {
	t.Helper()

	repo, err := corerepos.NewCryptoTokenRepository(secret, secrets)
	require.NoError(t, err)
	require.NotNil(t, repo)

	return repo
}

func mustEncryptToken(t *testing.T, repo corerepos.TokenRepository, token string) string {
	t.Helper()

	encrypted, err := repo.Encrypt(token)
	require.NoError(t, err)
	return encrypted
}

// tamperWithByte flips one byte of a base64url-encoded payload.
func tamperWithByte(t *testing.T, encoded string, index int) string {
	t.Helper()

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	require.Less(t, index, len(raw))

	raw[index] ^= 0xFF
	return base64.RawURLEncoding.EncodeToString(raw)
}

func TestCryptoTokenRepository_New_KeyParsing(t *testing.T) {
	unpadded := base64.RawStdEncoding.EncodeToString(keyOneRaw)
	urlSafe := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xFB}, 32))
	shortKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x01}, 31))
	longKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x01}, 33))

	tests := []struct {
		name    string
		secret  string
		secrets []models.SecretConfig
		wantErr bool
	}{
		{
			name:   "accepts padded standard base64",
			secret: keyOne,
		},
		{
			name:   "accepts unpadded standard base64",
			secret: unpadded,
		},
		{
			name:   "trims surrounding whitespace",
			secret: "  \t" + keyOne + "\n ",
		},
		{
			name:    "rejects an internal space",
			secret:  keyOne[:10] + " " + keyOne[10:],
			wantErr: true,
		},
		{
			name:    "rejects an internal newline",
			secret:  keyOne[:10] + "\n" + keyOne[10:],
			wantErr: true,
		},
		{
			name:    "rejects url-safe base64",
			secret:  urlSafe,
			wantErr: true,
		},
		{
			name:    "rejects an empty key",
			secret:  "",
			wantErr: true,
		},
		{
			name:    "rejects a whitespace-only key",
			secret:  "   ",
			wantErr: true,
		},
		{
			name:    "rejects invalid base64",
			secret:  "not-base64!!",
			wantErr: true,
		},
		{
			name:    "rejects a 31-byte key",
			secret:  shortKey,
			wantErr: true,
		},
		{
			name:    "rejects a 33-byte key",
			secret:  longKey,
			wantErr: true,
		},
		{
			name:    "rejects an invalid versioned key",
			secret:  keyOne,
			secrets: []models.SecretConfig{{Version: 1, Value: "not-base64!!"}},
			wantErr: true,
		},
		{
			name:    "rejects an invalid singular fallback",
			secret:  "not-base64!!",
			secrets: []models.SecretConfig{{Version: 1, Value: keyOne}},
			wantErr: true,
		},
		{
			name:    "accepts versioned keys with a singular fallback",
			secret:  keyThree,
			secrets: []models.SecretConfig{{Version: 2, Value: keyTwo}, {Version: 1, Value: keyOne}},
		},
		{
			name:    "accepts versioned keys without a singular fallback",
			secrets: []models.SecretConfig{{Version: 2, Value: keyTwo}, {Version: 1, Value: keyOne}},
		},
		{
			name:    "trims whitespace in versioned keys",
			secrets: []models.SecretConfig{{Version: 1, Value: "  " + keyOne + "  "}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := corerepos.NewCryptoTokenRepository(tt.secret, tt.secrets)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, corerepos.ErrInvalidSecret)
				assert.Nil(t, repo)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, repo)
		})
	}
}

func TestCryptoTokenRepository_EncryptFormat(t *testing.T) {
	repo := newTokenRepo(t, keyOne, nil)
	token := "plaintext-token-value"

	first := mustEncryptToken(t, repo, token)
	second := mustEncryptToken(t, repo, token)

	assert.NotEqual(t, first, second, "encryption must use a fresh nonce")
	assert.NotContains(t, first, token, "ciphertext must not contain the plaintext")
	assert.NotContains(t, first, "=", "output must use unpadded base64url")

	firstRaw, err := base64.RawURLEncoding.DecodeString(first)
	require.NoError(t, err)
	secondRaw, err := base64.RawURLEncoding.DecodeString(second)
	require.NoError(t, err)

	assert.Len(t, firstRaw, chacha20poly1305.NonceSizeX+len(token)+chacha20poly1305.Overhead)
	assert.Equal(t, first, base64.RawURLEncoding.EncodeToString(firstRaw), "ciphertext must have no prefix")
	assert.NotEqual(t, firstRaw[:chacha20poly1305.NonceSizeX], secondRaw[:chacha20poly1305.NonceSizeX], "nonces must differ")
}

func TestCryptoTokenRepository_EncryptDecrypt(t *testing.T) {
	repo := newTokenRepo(t, keyOne, nil)

	generated, err := repo.Generate()
	require.NoError(t, err)

	tests := []struct {
		name  string
		token string
	}{
		{name: "round-trips an empty token", token: ""},
		{name: "round-trips a short token", token: "token"},
		{name: "round-trips a generated token", token: generated},
		{name: "round-trips a long token", token: strings.Repeat("x", 1024)},
		{name: "round-trips a unicode token", token: "tøken-日本語-🔐"},
		{name: "round-trips a token with whitespace", token: "  token  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted := mustEncryptToken(t, repo, tt.token)

			decrypted, err := repo.Decrypt(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.token, decrypted)
			assert.Equal(t, repo.Hash(tt.token), repo.Hash(decrypted))
		})
	}
}

func TestCryptoTokenRepository_New_SameSecretInteroperates(t *testing.T) {
	first := newTokenRepo(t, keyOne, nil)
	second := newTokenRepo(t, keyOne, nil)

	encrypted := mustEncryptToken(t, first, "shared-token")

	decrypted, err := second.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "shared-token", decrypted)
}

func TestCryptoTokenRepository_New_DifferentSecretFails(t *testing.T) {
	first := newTokenRepo(t, keyOne, nil)
	second := newTokenRepo(t, keyTwo, nil)

	encrypted := mustEncryptToken(t, first, "shared-token")

	decrypted, err := second.Decrypt(encrypted)

	require.Error(t, err)
	assert.Empty(t, decrypted)
	assert.ErrorIs(t, err, corerepos.ErrTokenTampered)
}

func TestCryptoTokenRepository_Keyring(t *testing.T) {
	t.Run("encrypts with the first configured key", func(t *testing.T) {
		repo := newTokenRepo(t, "", []models.SecretConfig{
			{Version: 2, Value: keyTwo},
			{Version: 1, Value: keyOne},
		})

		encrypted := mustEncryptToken(t, repo, "token")

		currentOnly := newTokenRepo(t, keyTwo, nil)
		decrypted, err := currentOnly.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, "token", decrypted)

		oldOnly := newTokenRepo(t, keyOne, nil)
		_, err = oldOnly.Decrypt(encrypted)
		assert.ErrorIs(t, err, corerepos.ErrTokenTampered)
	})

	t.Run("decrypts a value encrypted with an older key", func(t *testing.T) {
		oldRepo := newTokenRepo(t, keyOne, nil)
		encrypted := mustEncryptToken(t, oldRepo, "token")

		repo := newTokenRepo(t, "", []models.SecretConfig{
			{Version: 2, Value: keyTwo},
			{Version: 1, Value: keyOne},
		})

		decrypted, err := repo.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, "token", decrypted)
	})

	t.Run("uses the singular secret as the final fallback", func(t *testing.T) {
		singularRepo := newTokenRepo(t, keyThree, nil)
		encrypted := mustEncryptToken(t, singularRepo, "token")

		repo := newTokenRepo(t, keyThree, []models.SecretConfig{
			{Version: 2, Value: keyTwo},
			{Version: 1, Value: keyOne},
		})

		decrypted, err := repo.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, "token", decrypted)
	})

	t.Run("fails when the required key version was removed", func(t *testing.T) {
		oldRepo := newTokenRepo(t, keyOne, nil)
		encrypted := mustEncryptToken(t, oldRepo, "token")

		repo := newTokenRepo(t, keyThree, []models.SecretConfig{
			{Version: 2, Value: keyTwo},
		})

		_, err := repo.Decrypt(encrypted)
		assert.ErrorIs(t, err, corerepos.ErrTokenTampered)
	})

	t.Run("allows gaps between versions", func(t *testing.T) {
		repo := newTokenRepo(t, "", []models.SecretConfig{
			{Version: 1, Value: keyOne},
			{Version: 3, Value: keyTwo},
		})

		encrypted := mustEncryptToken(t, repo, "token")
		decrypted, err := repo.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, "token", decrypted)
	})

	t.Run("never encrypts with a decryption-only key", func(t *testing.T) {
		repo := newTokenRepo(t, "", []models.SecretConfig{
			{Version: 2, Value: keyTwo},
			{Version: 1, Value: keyOne},
		})

		encrypted := mustEncryptToken(t, repo, "token")

		decryptionOnly := newTokenRepo(t, keyOne, nil)
		_, err := decryptionOnly.Decrypt(encrypted)
		assert.ErrorIs(t, err, corerepos.ErrTokenTampered)
	})

	t.Run("decryption does not mutate the stored value", func(t *testing.T) {
		repo := newTokenRepo(t, keyOne, nil)
		encrypted := mustEncryptToken(t, repo, "token")
		stored := encrypted

		for range 2 {
			decrypted, err := repo.Decrypt(encrypted)
			require.NoError(t, err)
			assert.Equal(t, "token", decrypted)
		}

		assert.Equal(t, stored, encrypted)
	})

	t.Run("rejects pre-v1.51.0 sha256 key derivation data", func(t *testing.T) {
		legacyKey := sha256.Sum256([]byte("legacy-secret"))
		aead, err := chacha20poly1305.NewX(legacyKey[:])
		require.NoError(t, err)

		nonce := make([]byte, chacha20poly1305.NonceSizeX)
		ciphertext := aead.Seal(nil, nonce, []byte("legacy-token"), nil)
		legacyValue := base64.RawURLEncoding.EncodeToString(append(nonce, ciphertext...))

		repo := newTokenRepo(t, keyOne, nil)
		_, err = repo.Decrypt(legacyValue)
		assert.ErrorIs(t, err, corerepos.ErrTokenTampered)
	})
}

func TestCryptoTokenRepository_DefaultSecretFallback(t *testing.T) {
	const devDefault = "YXV0aHVsYS1kZXYta2V5LTAxMjM0NTY3ODlhYmNkZWY="

	defaultRepo := newTokenRepo(t, devDefault, nil)
	encrypted := mustEncryptToken(t, defaultRepo, "token")

	repo := newTokenRepo(t, devDefault, []models.SecretConfig{
		{Version: 2, Value: keyTwo},
		{Version: 1, Value: keyOne},
	})

	decrypted, err := repo.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "token", decrypted)
}

func TestCryptoTokenRepository_Decrypt_InvalidInput(t *testing.T) {
	repo := newTokenRepo(t, keyOne, nil)

	valid := mustEncryptToken(t, repo, "token")
	validRaw, err := base64.RawURLEncoding.DecodeString(valid)
	require.NoError(t, err)

	emptyRaw, err := base64.RawURLEncoding.DecodeString(mustEncryptToken(t, repo, ""))
	require.NoError(t, err)

	other := mustEncryptToken(t, repo, "another-token")
	otherRaw, err := base64.RawURLEncoding.DecodeString(other)
	require.NoError(t, err)

	mixedNonceAndBody := append(append([]byte{}, validRaw[:chacha20poly1305.NonceSizeX]...), otherRaw[chacha20poly1305.NonceSizeX:]...)

	tests := []struct {
		name      string
		encrypted string
		wantErr   error
	}{
		{
			name:      "rejects empty input",
			encrypted: "",
			wantErr:   corerepos.ErrInvalidEncryptedToken,
		},
		{
			name:      "rejects invalid base64",
			encrypted: "!!!",
			wantErr:   corerepos.ErrInvalidEncryptedToken,
		},
		{
			name:      "rejects payload shorter than the nonce",
			encrypted: base64.RawURLEncoding.EncodeToString(make([]byte, chacha20poly1305.NonceSizeX-1)),
			wantErr:   corerepos.ErrInvalidEncryptedToken,
		},
		{
			name:      "rejects payload containing only a nonce",
			encrypted: base64.RawURLEncoding.EncodeToString(make([]byte, chacha20poly1305.NonceSizeX)),
			wantErr:   corerepos.ErrInvalidEncryptedToken,
		},
		{
			name:      "rejects a short payload without a full tag",
			encrypted: base64.RawURLEncoding.EncodeToString(make([]byte, chacha20poly1305.NonceSizeX+chacha20poly1305.Overhead-1)),
			wantErr:   corerepos.ErrInvalidEncryptedToken,
		},
		{
			name:      "rejects a truncated tag on an empty token",
			encrypted: base64.RawURLEncoding.EncodeToString(emptyRaw[:len(emptyRaw)-1]),
			wantErr:   corerepos.ErrInvalidEncryptedToken,
		},
		{
			name:      "rejects tampered ciphertext",
			encrypted: tamperWithByte(t, valid, chacha20poly1305.NonceSizeX),
			wantErr:   corerepos.ErrTokenTampered,
		},
		{
			name:      "rejects tampered nonce",
			encrypted: tamperWithByte(t, valid, 0),
			wantErr:   corerepos.ErrTokenTampered,
		},
		{
			name:      "rejects tampered tag",
			encrypted: tamperWithByte(t, valid, len(validRaw)-1),
			wantErr:   corerepos.ErrTokenTampered,
		},
		{
			name:      "rejects a nonce from another ciphertext",
			encrypted: base64.RawURLEncoding.EncodeToString(mixedNonceAndBody),
			wantErr:   corerepos.ErrTokenTampered,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decrypted, err := repo.Decrypt(tt.encrypted)

			require.Error(t, err)
			assert.Empty(t, decrypted)
			assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
		})
	}
}

func TestCryptoTokenRepository_Generate(t *testing.T) {
	repo := newTokenRepo(t, keyOne, nil)
	seen := make(map[string]struct{}, 10)

	for range 10 {
		token, err := repo.Generate()
		require.NoError(t, err)

		assert.Len(t, token, 64)
		assert.Equal(t, strings.ToLower(token), token)

		raw, err := hex.DecodeString(token)
		require.NoError(t, err)
		assert.Len(t, raw, 32)

		_, exists := seen[token]
		assert.False(t, exists, "generated duplicate token")
		seen[token] = struct{}{}
	}

	assert.Len(t, seen, 10)
}

func TestCryptoTokenRepository_Hash(t *testing.T) {
	repo := newTokenRepo(t, keyOne, nil)

	tests := []struct {
		name  string
		token string
		want  string
	}{
		{
			name:  "hashes a known value",
			token: "abc",
			want:  "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		},
		{
			name:  "hashes an empty token",
			token: "",
			want:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:  "hashes a generated token",
			token: strings.Repeat("a", 64),
			want:  repo.Hash(strings.Repeat("a", 64)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repo.Hash(tt.token)

			assert.Equal(t, tt.want, got)
			assert.Len(t, got, 64)
			assert.Equal(t, got, repo.Hash(tt.token), "hashing must be deterministic")
		})
	}

	assert.NotEqual(t, repo.Hash("token-a"), repo.Hash("token-b"))
}

func FuzzCryptoTokenRepository_Decrypt(f *testing.F) {
	repo, err := corerepos.NewCryptoTokenRepository(keyOne, nil)
	if err != nil {
		f.Fatalf("failed to create repository: %v", err)
	}

	seed, err := repo.Encrypt("seed-token")
	if err != nil {
		f.Fatalf("failed to encrypt fuzz seed: %v", err)
	}

	f.Add("")
	f.Add("not-base64")
	f.Add(base64.RawURLEncoding.EncodeToString(make([]byte, chacha20poly1305.NonceSizeX)))
	f.Add(seed)

	f.Fuzz(func(t *testing.T, encrypted string) {
		// Decrypt must handle arbitrary input without panicking.
		_, _ = repo.Decrypt(encrypted)
	})
}
