package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	internaltests "github.com/Authula/authula/internal/tests"
)

func TestBlacklistService(t *testing.T) {
	t.Parallel()

	t.Run("BlacklistToken", func(t *testing.T) {
		storage := new(internaltests.MockSecondaryStorage)
		svc := NewBlacklistService(storage, &internaltests.MockLogger{})

		ctx := context.Background()
		futureTime := time.Now().Add(1 * time.Hour)

		storage.On("Set", ctx, "jwt:blacklist:token:test-jti", "1", mock.Anything).Return(nil)

		err := svc.BlacklistToken(ctx, "test-jti", futureTime)
		assert.NoError(t, err)
		storage.AssertExpectations(t)
	})

	t.Run("BlacklistToken_empty_jti", func(t *testing.T) {
		storage := new(internaltests.MockSecondaryStorage)
		svc := NewBlacklistService(storage, &internaltests.MockLogger{})

		ctx := context.Background()
		futureTime := time.Now().Add(1 * time.Hour)

		err := svc.BlacklistToken(ctx, "", futureTime)
		assert.EqualError(t, err, "jti cannot be empty")
	})

	t.Run("IsBlacklisted_true", func(t *testing.T) {
		storage := new(internaltests.MockSecondaryStorage)
		svc := NewBlacklistService(storage, &internaltests.MockLogger{})

		ctx := context.Background()

		storage.On("Get", ctx, "jwt:blacklist:token:test-jti").Return("1", nil)

		blacklisted, err := svc.IsBlacklisted(ctx, "test-jti")
		assert.NoError(t, err)
		assert.True(t, blacklisted)
		storage.AssertExpectations(t)
	})

	t.Run("IsBlacklisted_false", func(t *testing.T) {
		storage := new(internaltests.MockSecondaryStorage)
		svc := NewBlacklistService(storage, &internaltests.MockLogger{})

		ctx := context.Background()

		storage.On("Get", ctx, "jwt:blacklist:token:test-jti").Return(nil, nil)

		blacklisted, err := svc.IsBlacklisted(ctx, "test-jti")
		assert.NoError(t, err)
		assert.False(t, blacklisted)
		storage.AssertExpectations(t)
	})

	t.Run("BlacklistAllSessionTokens", func(t *testing.T) {
		storage := new(internaltests.MockSecondaryStorage)
		svc := NewBlacklistService(storage, &internaltests.MockLogger{})

		ctx := context.Background()
		futureTime := time.Now().Add(1 * time.Hour)

		storage.On("Set", ctx, "jwt:blacklist:session:sess-1", "1", mock.Anything).Return(nil)

		err := svc.BlacklistAllSessionTokens(ctx, "sess-1", futureTime)
		assert.NoError(t, err)
		storage.AssertExpectations(t)
	})

	t.Run("BlacklistAllSessionTokens_empty", func(t *testing.T) {
		storage := new(internaltests.MockSecondaryStorage)
		svc := NewBlacklistService(storage, &internaltests.MockLogger{})

		ctx := context.Background()
		futureTime := time.Now().Add(1 * time.Hour)

		err := svc.BlacklistAllSessionTokens(ctx, "", futureTime)
		assert.EqualError(t, err, "sessionID cannot be empty")
	})
}
