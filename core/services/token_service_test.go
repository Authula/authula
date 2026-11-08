package services_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coreservices "github.com/Authula/authula/core/services"
	internaltests "github.com/Authula/authula/internal/tests"
	rootservices "github.com/Authula/authula/services"
)

func newTokenServiceForTest() (rootservices.TokenService, *internaltests.MockTokenRepository) {
	repo := new(internaltests.MockTokenRepository)
	return coreservices.NewTokenService(repo), repo
}

func TestTokenService_Generate(t *testing.T) {
	t.Parallel()

	errGenerate := errors.New("failed to generate token")

	tests := []struct {
		name      string
		repoToken string
		repoErr   error
		wantToken string
		wantErr   error
	}{
		{
			name:      "returns the generated token",
			repoToken: "generated-token",
			wantToken: "generated-token",
		},
		{
			name:    "propagates repository error",
			repoErr: errGenerate,
			wantErr: errGenerate,
		},
		{
			name:      "returns an empty token unchanged",
			repoToken: "",
			wantToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newTokenServiceForTest()
			repo.On("Generate").Return(tt.repoToken, tt.repoErr)

			got, err := svc.Generate()

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantToken, got)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestTokenService_Hash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		token    string
		repoHash string
		wantHash string
	}{
		{
			name:     "returns the repository hash",
			token:    "plain-token",
			repoHash: "hashed-token",
			wantHash: "hashed-token",
		},
		{
			name:     "forwards empty token unchanged",
			token:    "",
			repoHash: "hash-of-empty-token",
			wantHash: "hash-of-empty-token",
		},
		{
			name:     "forwards unicode and special characters unchanged",
			token:    "tøken-🔐/+==",
			repoHash: "hash-of-unicode-token",
			wantHash: "hash-of-unicode-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newTokenServiceForTest()
			repo.On("Hash", tt.token).Return(tt.repoHash)

			got := svc.Hash(tt.token)

			assert.Equal(t, tt.wantHash, got)
			repo.AssertExpectations(t)
		})
	}
}

func TestTokenService_Encrypt(t *testing.T) {
	t.Parallel()

	errEncrypt := errors.New("failed to encrypt token")

	tests := []struct {
		name          string
		token         string
		repoEncrypted string
		repoErr       error
		wantEncrypted string
		wantErr       error
	}{
		{
			name:          "returns the encrypted token",
			token:         "plain-token",
			repoEncrypted: "encrypted-token",
			wantEncrypted: "encrypted-token",
		},
		{
			name:    "propagates repository error",
			token:   "plain-token",
			repoErr: errEncrypt,
			wantErr: errEncrypt,
		},
		{
			name:          "forwards empty token unchanged",
			token:         "",
			repoEncrypted: "encrypted-empty-token",
			wantEncrypted: "encrypted-empty-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newTokenServiceForTest()
			repo.On("Encrypt", tt.token).Return(tt.repoEncrypted, tt.repoErr)

			got, err := svc.Encrypt(tt.token)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantEncrypted, got)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestTokenService_Decrypt(t *testing.T) {
	t.Parallel()

	errDecrypt := errors.New("invalid or tampered token")

	tests := []struct {
		name          string
		encrypted     string
		repoPlaintext string
		repoErr       error
		wantPlaintext string
		wantErr       error
	}{
		{
			name:          "returns the decrypted token",
			encrypted:     "encrypted-token",
			repoPlaintext: "plain-token",
			wantPlaintext: "plain-token",
		},
		{
			name:      "propagates repository error",
			encrypted: "tampered-token",
			repoErr:   errDecrypt,
			wantErr:   errDecrypt,
		},
		{
			name:          "forwards empty encrypted value unchanged",
			encrypted:     "",
			repoPlaintext: "",
			wantPlaintext: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newTokenServiceForTest()
			repo.On("Decrypt", tt.encrypted).Return(tt.repoPlaintext, tt.repoErr)

			got, err := svc.Decrypt(tt.encrypted)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantPlaintext, got)
			}

			repo.AssertExpectations(t)
		})
	}
}
