package repositories_test

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/chacha20poly1305"

	corerepos "github.com/Authula/authula/core/repositories"
	internaltests "github.com/Authula/authula/internal/tests"
)

var (
	keyOne   = internaltests.MustDecodeKey(internaltests.EncryptionKeyOne)
	keyTwo   = internaltests.MustDecodeKey(internaltests.EncryptionKeyTwo)
	keyThree = internaltests.MustDecodeKey(internaltests.EncryptionKeyThree)
)

func newTokenRepo(t *testing.T, currentKey []byte, keyring [][]byte) corerepos.TokenRepository {
	t.Helper()

	repo, err := corerepos.NewCryptoTokenRepository(currentKey, keyring)
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

// sealToken builds a ciphertext independently of the repository so Decrypt
// tests only exercise Decrypt.
func sealToken(t *testing.T, key, nonce []byte, token string) string {
	t.Helper()

	aead, err := chacha20poly1305.NewX(key)
	require.NoError(t, err)

	raw := make([]byte, 0, len(nonce)+len(token)+aead.Overhead())
	raw = append(raw, nonce...)
	raw = aead.Seal(raw, nonce, []byte(token), nil)

	return base64.RawURLEncoding.EncodeToString(raw)
}

// openToken decrypts independently of the repository so Encrypt tests only
// exercise Encrypt.
func openToken(t *testing.T, key []byte, encrypted string) (string, error) {
	t.Helper()

	raw, err := base64.RawURLEncoding.DecodeString(encrypted)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(raw), chacha20poly1305.NonceSizeX)

	aead, err := chacha20poly1305.NewX(key)
	require.NoError(t, err)

	plaintext, err := aead.Open(nil, raw[:chacha20poly1305.NonceSizeX], raw[chacha20poly1305.NonceSizeX:], nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
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

func TestCryptoTokenRepository_New(t *testing.T) {
	tests := []struct {
		name       string
		currentKey []byte
		decryptors [][]byte
		wantErr    error
	}{
		{
			name:       "accepts a valid current key",
			currentKey: keyOne,
		},
		{
			name:       "accepts a valid keyring",
			currentKey: keyTwo,
			decryptors: [][]byte{keyTwo, keyOne},
		},
		{
			name:    "rejects a nil current key",
			wantErr: corerepos.ErrInvalidSecret,
		},
		{
			name:       "rejects a 31-byte current key",
			currentKey: bytes.Repeat([]byte{0x01}, 31),
			wantErr:    corerepos.ErrInvalidSecret,
		},
		{
			name:       "rejects a 33-byte current key",
			currentKey: bytes.Repeat([]byte{0x01}, 33),
			wantErr:    corerepos.ErrInvalidSecret,
		},
		{
			name:       "rejects a nil decryption key",
			currentKey: keyOne,
			decryptors: [][]byte{nil},
			wantErr:    corerepos.ErrInvalidSecret,
		},
		{
			name:       "rejects a 31-byte decryption key",
			currentKey: keyOne,
			decryptors: [][]byte{bytes.Repeat([]byte{0x01}, 31)},
			wantErr:    corerepos.ErrInvalidSecret,
		},
		{
			name:       "rejects a 33-byte decryption key",
			currentKey: keyOne,
			decryptors: [][]byte{bytes.Repeat([]byte{0x01}, 33)},
			wantErr:    corerepos.ErrInvalidSecret,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := corerepos.NewCryptoTokenRepository(tt.currentKey, tt.decryptors)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, repo)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, repo)
		})
	}
}

func TestCryptoTokenRepository_Encrypt(t *testing.T) {
	repo := newTokenRepo(t, keyOne, nil)

	generated, err := repo.Generate()
	require.NoError(t, err)

	tests := []struct {
		name           string
		currentKey     []byte
		keyring        [][]byte
		token          string
		checkPlaintext bool
		wrongKey       []byte
	}{
		{
			name:       "encrypts an empty token",
			currentKey: keyOne,
			token:      "",
		},
		{
			name:       "encrypts a short token",
			currentKey: keyOne,
			token:      "token",
		},
		{
			name:       "encrypts a generated token",
			currentKey: keyOne,
			token:      generated,
		},
		{
			name:       "encrypts a long token",
			currentKey: keyOne,
			token:      strings.Repeat("x", 1024),
		},
		{
			name:       "encrypts a unicode token",
			currentKey: keyOne,
			token:      "tøken-日本語-🔐",
		},
		{
			name:       "encrypts a token with whitespace",
			currentKey: keyOne,
			token:      "  token  ",
		},
		{
			name:           "hides a plaintext token",
			currentKey:     keyOne,
			token:          "plaintext-token-value",
			checkPlaintext: true,
		},
		{
			name:       "uses the current key instead of a decryption-only key",
			currentKey: keyTwo,
			keyring:    [][]byte{keyTwo, keyOne},
			token:      "token",
			wrongKey:   keyOne,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newTokenRepo(t, tt.currentKey, tt.keyring)

			first := mustEncryptToken(t, repo, tt.token)
			second := mustEncryptToken(t, repo, tt.token)

			assert.NotEqual(t, first, second, "encryption must use a fresh nonce")
			if tt.checkPlaintext {
				assert.NotContains(t, first, tt.token, "ciphertext must not contain the plaintext")
			}
			assert.NotContains(t, first, "=", "output must use unpadded base64url")

			firstRaw, err := base64.RawURLEncoding.DecodeString(first)
			require.NoError(t, err)
			secondRaw, err := base64.RawURLEncoding.DecodeString(second)
			require.NoError(t, err)

			assert.Len(t, firstRaw, chacha20poly1305.NonceSizeX+len(tt.token)+chacha20poly1305.Overhead)
			assert.Equal(t, first, base64.RawURLEncoding.EncodeToString(firstRaw), "ciphertext must have no prefix")
			assert.NotEqual(t, firstRaw[:chacha20poly1305.NonceSizeX], secondRaw[:chacha20poly1305.NonceSizeX], "nonces must differ")

			plaintext, err := openToken(t, tt.currentKey, first)
			require.NoError(t, err)
			assert.Equal(t, tt.token, plaintext)

			if tt.wrongKey != nil {
				_, err := openToken(t, tt.wrongKey, first)
				assert.Error(t, err)
			}
		})
	}
}

func TestCryptoTokenRepository_Decrypt(t *testing.T) {
	devDefault := internaltests.MustDecodeKey(internaltests.EncryptionKeyDevDefault)

	nonceOne := bytes.Repeat([]byte{0x01}, chacha20poly1305.NonceSizeX)
	nonceTwo := bytes.Repeat([]byte{0x02}, chacha20poly1305.NonceSizeX)
	nonceThree := bytes.Repeat([]byte{0x03}, chacha20poly1305.NonceSizeX)
	nonceFour := bytes.Repeat([]byte{0x04}, chacha20poly1305.NonceSizeX)

	valid := sealToken(t, keyOne, nonceOne, "token")
	empty := sealToken(t, keyOne, nonceTwo, "")
	older := sealToken(t, keyOne, nonceThree, "older-token")
	singular := sealToken(t, keyThree, nonceFour, "singular-token")
	devDefaultToken := sealToken(t, devDefault, nonceOne, "default-token")

	validRaw, err := base64.RawURLEncoding.DecodeString(valid)
	require.NoError(t, err)

	emptyRaw, err := base64.RawURLEncoding.DecodeString(empty)
	require.NoError(t, err)

	other := sealToken(t, keyOne, nonceFour, "another-token")
	otherRaw, err := base64.RawURLEncoding.DecodeString(other)
	require.NoError(t, err)

	mixedNonceAndBody := append(append([]byte{}, validRaw[:chacha20poly1305.NonceSizeX]...), otherRaw[chacha20poly1305.NonceSizeX:]...)

	tests := []struct {
		name         string
		currentKey   []byte
		keyring      [][]byte
		encrypted    string
		want         string
		wantErr      error
		decryptTwice bool
	}{
		{
			name:       "decrypts a token with the current key",
			currentKey: keyOne,
			encrypted:  valid,
			want:       "token",
		},
		{
			name:       "decrypts an empty token",
			currentKey: keyOne,
			encrypted:  empty,
			want:       "",
		},
		{
			name:       "decrypts a value encrypted with an older key",
			currentKey: keyTwo,
			keyring:    [][]byte{keyTwo, keyOne},
			encrypted:  older,
			want:       "older-token",
		},
		{
			name:       "uses the singular secret as the final fallback",
			currentKey: keyTwo,
			keyring:    [][]byte{keyTwo, keyOne, keyThree},
			encrypted:  singular,
			want:       "singular-token",
		},
		{
			name:       "falls back to the development default secret",
			currentKey: keyTwo,
			keyring:    [][]byte{keyTwo, keyOne, devDefault},
			encrypted:  devDefaultToken,
			want:       "default-token",
		},
		{
			name:       "allows gaps between versions",
			currentKey: keyOne,
			keyring:    [][]byte{keyOne, keyTwo},
			encrypted:  valid,
			want:       "token",
		},
		{
			name:         "decrypts repeatedly without mutating the stored value",
			currentKey:   keyOne,
			encrypted:    valid,
			want:         "token",
			decryptTwice: true,
		},
		{
			name:       "rejects a token encrypted with a different secret",
			currentKey: keyTwo,
			encrypted:  valid,
			wantErr:    corerepos.ErrTokenTampered,
		},
		{
			name:       "rejects a token whose required key was removed",
			currentKey: keyTwo,
			keyring:    [][]byte{keyTwo, keyThree},
			encrypted:  valid,
			wantErr:    corerepos.ErrTokenTampered,
		},
		{
			name:       "rejects empty input",
			currentKey: keyOne,
			encrypted:  "",
			wantErr:    corerepos.ErrInvalidEncryptedToken,
		},
		{
			name:       "rejects invalid base64",
			currentKey: keyOne,
			encrypted:  "!!!",
			wantErr:    corerepos.ErrInvalidEncryptedToken,
		},
		{
			name:       "rejects payload shorter than the nonce",
			currentKey: keyOne,
			encrypted:  base64.RawURLEncoding.EncodeToString(make([]byte, chacha20poly1305.NonceSizeX-1)),
			wantErr:    corerepos.ErrInvalidEncryptedToken,
		},
		{
			name:       "rejects payload containing only a nonce",
			currentKey: keyOne,
			encrypted:  base64.RawURLEncoding.EncodeToString(make([]byte, chacha20poly1305.NonceSizeX)),
			wantErr:    corerepos.ErrInvalidEncryptedToken,
		},
		{
			name:       "rejects a short payload without a full tag",
			currentKey: keyOne,
			encrypted:  base64.RawURLEncoding.EncodeToString(make([]byte, chacha20poly1305.NonceSizeX+chacha20poly1305.Overhead-1)),
			wantErr:    corerepos.ErrInvalidEncryptedToken,
		},
		{
			name:       "rejects a truncated tag on an empty token",
			currentKey: keyOne,
			encrypted:  base64.RawURLEncoding.EncodeToString(emptyRaw[:len(emptyRaw)-1]),
			wantErr:    corerepos.ErrInvalidEncryptedToken,
		},
		{
			name:       "rejects tampered ciphertext",
			currentKey: keyOne,
			encrypted:  tamperWithByte(t, valid, chacha20poly1305.NonceSizeX),
			wantErr:    corerepos.ErrTokenTampered,
		},
		{
			name:       "rejects tampered nonce",
			currentKey: keyOne,
			encrypted:  tamperWithByte(t, valid, 0),
			wantErr:    corerepos.ErrTokenTampered,
		},
		{
			name:       "rejects tampered tag",
			currentKey: keyOne,
			encrypted:  tamperWithByte(t, valid, len(validRaw)-1),
			wantErr:    corerepos.ErrTokenTampered,
		},
		{
			name:       "rejects a nonce from another ciphertext",
			currentKey: keyOne,
			encrypted:  base64.RawURLEncoding.EncodeToString(mixedNonceAndBody),
			wantErr:    corerepos.ErrTokenTampered,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newTokenRepo(t, tt.currentKey, tt.keyring)

			decrypted, err := repo.Decrypt(tt.encrypted)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, decrypted)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, decrypted)

			if tt.decryptTwice {
				stored := tt.encrypted
				for range 2 {
					decrypted, err = repo.Decrypt(tt.encrypted)
					require.NoError(t, err)
					assert.Equal(t, tt.want, decrypted)
				}
				assert.Equal(t, stored, tt.encrypted)
			}
		})
	}
}

func TestCryptoTokenRepository_Generate(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{
			name:  "generates a single token",
			count: 1,
		},
		{
			name:  "generates unique tokens",
			count: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newTokenRepo(t, keyOne, nil)
			seen := make(map[string]struct{}, tt.count)

			for range tt.count {
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

			assert.Len(t, seen, tt.count)
		})
	}
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
			name:  "hashes a long token",
			token: strings.Repeat("a", 64),
			want:  "ffe054fe7ae0cb6dc65c3af9b61d5209f439851db43d0ba5997337df154668eb",
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
