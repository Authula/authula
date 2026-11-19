package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
)

func hasServiceLogMessage(logger *internaltests.LevelRecordingLogger, substr string) bool {
	for _, message := range logger.Messages() {
		if strings.Contains(message, substr) {
			return true
		}
	}

	return false
}

func newMockSigningKeyService(config SigningKeyConfig) (SigningKeyService, *jwttests.MockSigningKeyRepository, *jwttests.MockEncryptor) {
	repo := new(jwttests.MockSigningKeyRepository)
	encryptor := new(jwttests.MockEncryptor)

	return NewSigningKeyService(repo, encryptor, &internaltests.MockLogger{}, config), repo, encryptor
}

func TestSigningKeyService_GetActiveKeyRepositoryError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, repo, _ := newMockSigningKeyService(SigningKeyConfig{})
	repo.On("GetActiveKey", ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256).Return(nil, assert.AnError).Once()

	key, err := service.GetActiveKey(ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256)

	require.ErrorIs(t, err, assert.AnError)
	assert.Nil(t, key)
	repo.AssertExpectations(t)
}

func TestSigningKeyService_GetPublicKeysErrorPaths(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	validKeys := []*models.SigningKey{
		{KeyID: "kid-1", PublicKey: "public", PrivateKey: "private"},
	}

	tests := []struct {
		name       string
		purpose    string
		setupMocks func(*jwttests.MockSigningKeyRepository)
		wantErr    error
		wantKeys   int
	}{
		{
			name:    "invalid purpose",
			purpose: "unknown",
			setupMocks: func(*jwttests.MockSigningKeyRepository) {
			},
			wantErr: models.ErrInvalidSigningKeyPurpose,
		},
		{
			name:    "repository error",
			purpose: models.SigningKeyPurposeJWTPlugin,
			setupMocks: func(repo *jwttests.MockSigningKeyRepository) {
				repo.On("GetPublicKeys", ctx, models.SigningKeyPurposeJWTPlugin, mock.Anything).
					Return(nil, assert.AnError).Once()
			},
			wantErr: assert.AnError,
		},
		{
			name:    "clears private key material",
			purpose: models.SigningKeyPurposeJWTPlugin,
			setupMocks: func(repo *jwttests.MockSigningKeyRepository) {
				repo.On("GetPublicKeys", ctx, models.SigningKeyPurposeJWTPlugin, mock.Anything).
					Return(validKeys, nil).Once()
			},
			wantKeys: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, repo, _ := newMockSigningKeyService(SigningKeyConfig{})
			tt.setupMocks(repo)

			keys, err := service.GetPublicKeys(ctx, tt.purpose)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, keys)
			} else {
				require.NoError(t, err)
				require.Len(t, keys, tt.wantKeys)
				assert.Empty(t, keys[0].PrivateKey)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestSigningKeyService_GenerateKeyErrorPaths(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	existing := &models.SigningKey{KeyID: "existing"}

	tests := []struct {
		name       string
		purpose    string
		algorithm  string
		setupMocks func(*jwttests.MockSigningKeyRepository, *jwttests.MockEncryptor)
		wantErr    error
		wantMsg    string
	}{
		{
			name:      "invalid purpose",
			purpose:   "unknown",
			algorithm: models.SigningKeyAlgorithmRS256,
			setupMocks: func(*jwttests.MockSigningKeyRepository, *jwttests.MockEncryptor) {
			},
			wantErr: models.ErrInvalidSigningKeyPurpose,
		},
		{
			name:      "invalid algorithm",
			purpose:   models.SigningKeyPurposeJWTPlugin,
			algorithm: "EdDSA",
			setupMocks: func(*jwttests.MockSigningKeyRepository, *jwttests.MockEncryptor) {
			},
			wantErr: models.ErrUnsupportedSigningAlgorithm,
		},
		{
			name:      "active key lookup failure",
			purpose:   models.SigningKeyPurposeJWTPlugin,
			algorithm: models.SigningKeyAlgorithmRS256,
			setupMocks: func(repo *jwttests.MockSigningKeyRepository, _ *jwttests.MockEncryptor) {
				repo.On("GetActiveKey", ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256).
					Return(nil, assert.AnError).Once()
			},
			wantErr: assert.AnError,
			wantMsg: "failed to inspect active signing key",
		},
		{
			name:      "encryption failure",
			purpose:   models.SigningKeyPurposeJWTPlugin,
			algorithm: models.SigningKeyAlgorithmRS256,
			setupMocks: func(repo *jwttests.MockSigningKeyRepository, encryptor *jwttests.MockEncryptor) {
				repo.On("GetActiveKey", ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256).
					Return(nil, models.ErrSigningKeyNotFound).Once()
				encryptor.On("Encrypt", mock.Anything).Return("", assert.AnError).Once()
			},
			wantErr: assert.AnError,
			wantMsg: "failed to encrypt private key",
		},
		{
			name:      "rotation failure without concurrent winner",
			purpose:   models.SigningKeyPurposeJWTPlugin,
			algorithm: models.SigningKeyAlgorithmRS256,
			setupMocks: func(repo *jwttests.MockSigningKeyRepository, encryptor *jwttests.MockEncryptor) {
				repo.On("GetActiveKey", ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256).
					Return(nil, models.ErrSigningKeyNotFound).Once()
				encryptor.On("Encrypt", mock.Anything).Return("encrypted", nil).Once()
				repo.On("Rotate", ctx, models.SigningKeyPurposeJWTPlugin, mock.Anything, mock.Anything).
					Return(assert.AnError).Once()
				repo.On("GetActiveKey", ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256).
					Return(nil, assert.AnError).Once()
			},
			wantErr: assert.AnError,
			wantMsg: "failed to generate signing key",
		},
		{
			name:      "rotation failure with concurrent winner succeeds",
			purpose:   models.SigningKeyPurposeJWTPlugin,
			algorithm: models.SigningKeyAlgorithmRS256,
			setupMocks: func(repo *jwttests.MockSigningKeyRepository, encryptor *jwttests.MockEncryptor) {
				repo.On("GetActiveKey", ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256).
					Return(nil, models.ErrSigningKeyNotFound).Once()
				encryptor.On("Encrypt", mock.Anything).Return("encrypted", nil).Once()
				repo.On("Rotate", ctx, models.SigningKeyPurposeJWTPlugin, mock.Anything, mock.Anything).
					Return(assert.AnError).Once()
				repo.On("GetActiveKey", ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256).
					Return(existing, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, repo, encryptor := newMockSigningKeyService(SigningKeyConfig{})
			tt.setupMocks(repo, encryptor)

			err := service.GenerateKey(ctx, tt.purpose, tt.algorithm)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				if tt.wantMsg != "" {
					require.ErrorContains(t, err, tt.wantMsg)
				}
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
			encryptor.AssertExpectations(t)
		})
	}
}

func TestSigningKeyService_RotateKeyWithGraceErrorPaths(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name       string
		purpose    string
		algorithm  string
		setupMocks func(*jwttests.MockSigningKeyRepository, *jwttests.MockEncryptor)
		wantErr    error
		wantMsg    string
	}{
		{
			name:      "invalid purpose",
			purpose:   "unknown",
			algorithm: models.SigningKeyAlgorithmRS256,
			setupMocks: func(*jwttests.MockSigningKeyRepository, *jwttests.MockEncryptor) {
			},
			wantErr: models.ErrInvalidSigningKeyPurpose,
		},
		{
			name:      "invalid algorithm",
			purpose:   models.SigningKeyPurposeJWTPlugin,
			algorithm: "HS256",
			setupMocks: func(*jwttests.MockSigningKeyRepository, *jwttests.MockEncryptor) {
			},
			wantErr: models.ErrUnsupportedSigningAlgorithm,
		},
		{
			name:      "encryption failure",
			purpose:   models.SigningKeyPurposeJWTPlugin,
			algorithm: models.SigningKeyAlgorithmRS256,
			setupMocks: func(_ *jwttests.MockSigningKeyRepository, encryptor *jwttests.MockEncryptor) {
				encryptor.On("Encrypt", mock.Anything).Return("", assert.AnError).Once()
			},
			wantErr: assert.AnError,
			wantMsg: "failed to encrypt private key",
		},
		{
			name:      "repository rotation failure",
			purpose:   models.SigningKeyPurposeJWTPlugin,
			algorithm: models.SigningKeyAlgorithmRS256,
			setupMocks: func(repo *jwttests.MockSigningKeyRepository, encryptor *jwttests.MockEncryptor) {
				encryptor.On("Encrypt", mock.Anything).Return("encrypted", nil).Once()
				repo.On("Rotate", ctx, models.SigningKeyPurposeJWTPlugin, mock.Anything, mock.Anything).
					Return(assert.AnError).Once()
			},
			wantErr: assert.AnError,
			wantMsg: "failed to rotate signing key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, repo, encryptor := newMockSigningKeyService(SigningKeyConfig{GracePeriod: time.Hour})
			tt.setupMocks(repo, encryptor)

			err := service.RotateKeyWithGrace(ctx, tt.purpose, tt.algorithm, time.Hour)

			require.ErrorIs(t, err, tt.wantErr)
			if tt.wantMsg != "" {
				require.ErrorContains(t, err, tt.wantMsg)
			}
			repo.AssertExpectations(t)
			encryptor.AssertExpectations(t)
		})
	}
}

func TestSigningKeyService_EffectiveGracePeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		config           SigningKeyConfig
		requested        time.Duration
		maxLifetime      func(context.Context) (time.Duration, error)
		wantGrace        time.Duration
		wantExtendedLog  bool
		wantLifetimeWarn bool
	}{
		{
			name:      "zero requested falls back to configured grace",
			config:    SigningKeyConfig{GracePeriod: 2 * time.Hour, ClockSkew: 10 * time.Minute},
			requested: 0,
			wantGrace: 2*time.Hour + 10*time.Minute,
		},
		{
			name:      "negative requested falls back to configured grace",
			config:    SigningKeyConfig{GracePeriod: 2 * time.Hour, ClockSkew: 10 * time.Minute},
			requested: -time.Minute,
			wantGrace: 2*time.Hour + 10*time.Minute,
		},
		{
			name:      "zero configured grace falls back to default",
			config:    SigningKeyConfig{GracePeriod: 0, ClockSkew: time.Minute},
			requested: 0,
			wantGrace: defaultGracePeriod + time.Minute,
		},
		{
			name:      "max token lifetime extends grace",
			config:    SigningKeyConfig{GracePeriod: time.Hour, ClockSkew: time.Minute},
			requested: time.Hour,
			maxLifetime: func(context.Context) (time.Duration, error) {
				return 3 * time.Hour, nil
			},
			wantGrace:       3*time.Hour + time.Minute,
			wantExtendedLog: true,
		},
		{
			name:      "max token lifetime error keeps requested grace",
			config:    SigningKeyConfig{GracePeriod: time.Hour, ClockSkew: time.Minute},
			requested: time.Hour,
			maxLifetime: func(context.Context) (time.Duration, error) {
				return 0, assert.AnError
			},
			wantGrace:        time.Hour + time.Minute,
			wantLifetimeWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := &internaltests.LevelRecordingLogger{}
			tt.config.MaxTokenLifetime = tt.maxLifetime
			service := &signingKeyService{logger: logger, config: tt.config}

			got := service.effectiveGracePeriod(context.Background(), tt.requested)

			assert.Equal(t, tt.wantGrace, got)
			if tt.wantExtendedLog {
				assert.True(t, hasServiceLogMessage(logger, "extending key rotation grace period"))
			}
			if tt.wantLifetimeWarn {
				assert.True(t, hasServiceLogMessage(logger, "failed to determine maximum token lifetime"))
			}
		})
	}
}

func TestGenerateKeyPairRejectsUnsupportedAlgorithm(t *testing.T) {
	t.Parallel()

	privatePEM, publicPEM, err := generateKeyPair("EdDSA")

	require.ErrorIs(t, err, models.ErrUnsupportedSigningAlgorithm)
	assert.Empty(t, privatePEM)
	assert.Empty(t, publicPEM)
}

func TestSigningKeyService_GetKeyByIDPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, repo, _ := newMockSigningKeyService(SigningKeyConfig{})
	repo.On("GetKeyByID", ctx, "kid-1").Return(nil, assert.AnError).Once()

	key, err := service.GetKeyByID(ctx, "kid-1")

	require.True(t, errors.Is(err, assert.AnError))
	assert.Nil(t, key)
	repo.AssertExpectations(t)
}
