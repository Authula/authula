package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
)

func newMockKeyService(t *testing.T, logger models.Logger) (KeyService, *jwttests.MockSigningKeyService) {
	t.Helper()

	shared := new(jwttests.MockSigningKeyService)
	return NewKeyService(shared, logger, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256), shared
}

func TestKeyService_GenerateKeysIfMissingError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, shared := newMockKeyService(t, &internaltests.MockLogger{})
	shared.On("GenerateKey", ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256).
		Return(assert.AnError).Once()

	err := service.GenerateKeysIfMissing(ctx)

	require.ErrorIs(t, err, assert.AnError)
	require.ErrorContains(t, err, "failed to generate keys")
	shared.AssertExpectations(t)
}

func TestKeyService_GetActiveKeyPreservesSentinel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name    string
		shared  error
		wantMsg string
	}{
		{
			name:    "sentinel is preserved",
			shared:  models.ErrSigningKeyNotFound,
			wantMsg: "no active key found",
		},
		{
			name:    "other errors are wrapped",
			shared:  assert.AnError,
			wantMsg: "failed to get active key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, shared := newMockKeyService(t, &internaltests.MockLogger{})
			shared.On("GetActiveKey", ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256).
				Return(nil, tt.shared).Once()

			key, err := service.GetActiveKey(ctx)

			require.ErrorIs(t, err, tt.shared)
			require.ErrorContains(t, err, tt.wantMsg)
			assert.Nil(t, key)
			shared.AssertExpectations(t)
		})
	}
}

func TestKeyService_GetPublicKeysError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, shared := newMockKeyService(t, &internaltests.MockLogger{})
	shared.On("GetPublicKeys", ctx, models.SigningKeyPurposeJWTPlugin).Return(nil, assert.AnError).Once()

	keys, err := service.GetPublicKeys(ctx)

	require.ErrorIs(t, err, assert.AnError)
	require.ErrorContains(t, err, "failed to get public keys")
	assert.Nil(t, keys)
	shared.AssertExpectations(t)
}

func TestKeyService_IsKeyRotationDue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name    string
		key     *models.SigningKey
		err     error
		wantDue bool
	}{
		{
			name:    "lookup failure reports not due",
			err:     assert.AnError,
			wantDue: false,
		},
		{
			name:    "fresh key is not due",
			key:     &models.SigningKey{CreatedAt: time.Now().Add(-time.Minute)},
			wantDue: false,
		},
		{
			name:    "old key is due",
			key:     &models.SigningKey{CreatedAt: time.Now().Add(-2 * time.Hour)},
			wantDue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, shared := newMockKeyService(t, &internaltests.MockLogger{})
			shared.On("GetActiveKey", ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256).
				Return(tt.key, tt.err).Once()

			assert.Equal(t, tt.wantDue, service.IsKeyRotationDue(ctx, time.Hour))
			shared.AssertExpectations(t)
		})
	}
}

func TestKeyService_RotateKeysIfNeeded(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dueKey := &models.SigningKey{CreatedAt: time.Now().Add(-2 * time.Hour)}
	freshKey := &models.SigningKey{CreatedAt: time.Now()}

	tests := []struct {
		name            string
		key             *models.SigningKey
		rotateErr       error
		invalidate      func(context.Context) error
		wantRotated     bool
		wantErr         error
		wantInvalidated bool
		wantWarn        bool
	}{
		{
			name:        "not due does not rotate",
			key:         freshKey,
			wantRotated: false,
		},
		{
			name:        "rotation failure is returned",
			key:         dueKey,
			rotateErr:   assert.AnError,
			wantRotated: false,
			wantErr:     assert.AnError,
		},
		{
			name:        "rotation succeeds without invalidator",
			key:         dueKey,
			wantRotated: true,
		},
		{
			name:            "rotation succeeds and invalidates cache",
			key:             dueKey,
			wantRotated:     true,
			wantInvalidated: true,
			invalidate: func(context.Context) error {
				return nil
			},
		},
		{
			name:            "cache invalidation failure is only logged",
			key:             dueKey,
			wantRotated:     true,
			wantInvalidated: true,
			wantWarn:        true,
			invalidate: func(context.Context) error {
				return assert.AnError
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := &internaltests.LevelRecordingLogger{}
			service, shared := newMockKeyService(t, logger)
			shared.On("GetActiveKey", ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256).
				Return(tt.key, nil).Once()
			if tt.wantRotated || tt.rotateErr != nil {
				shared.On("RotateKeyWithGrace", ctx, models.SigningKeyPurposeJWTPlugin, models.SigningKeyAlgorithmRS256, time.Hour).
					Return(tt.rotateErr).Once()
			}

			invalidated := false
			var invalidate func(context.Context) error
			if tt.invalidate != nil {
				invalidate = func(ctx context.Context) error {
					invalidated = true
					return tt.invalidate(ctx)
				}
			}

			rotated, err := service.RotateKeysIfNeeded(ctx, time.Hour, time.Hour, invalidate)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantRotated, rotated)
			assert.Equal(t, tt.wantInvalidated, invalidated)
			if tt.wantWarn {
				assert.True(t, hasServiceLogMessage(logger, "failed to invalidate cache after key rotation"))
			}
			shared.AssertExpectations(t)
		})
	}
}

var _ models.Logger = (*internaltests.LevelRecordingLogger)(nil)
