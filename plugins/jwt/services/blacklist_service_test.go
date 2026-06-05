package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockSecondaryStorage implements models.SecondaryStorage
type mockSecondaryStorage struct {
	mock.Mock
}

func (m *mockSecondaryStorage) Get(ctx context.Context, key string) (any, error) {
	args := m.Called(ctx, key)
	return args.Get(0), args.Error(1)
}

func (m *mockSecondaryStorage) Set(ctx context.Context, key string, value any, ttl *time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *mockSecondaryStorage) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockSecondaryStorage) Incr(ctx context.Context, key string, ttl *time.Duration) (int, error) {
	args := m.Called(ctx, key, ttl)
	return args.Int(0), args.Error(1)
}

func (m *mockSecondaryStorage) TTL(ctx context.Context, key string) (*time.Duration, error) {
	args := m.Called(ctx, key)
	if v := args.Get(0); v != nil {
		return v.(*time.Duration), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSecondaryStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

// noopLogger implements models.Logger
type noopLogger struct{}

func (l *noopLogger) Debug(msg string, args ...any) {}
func (l *noopLogger) Info(msg string, args ...any)  {}
func (l *noopLogger) Warn(msg string, args ...any)  {}
func (l *noopLogger) Error(msg string, args ...any) {}

func TestBlacklistService(t *testing.T) {
	t.Parallel()

	t.Run("BlacklistToken", func(t *testing.T) {
		storage := new(mockSecondaryStorage)
		svc := NewBlacklistService(storage, &noopLogger{})

		ctx := context.Background()
		futureTime := time.Now().Add(1 * time.Hour)

		storage.On("Set", ctx, "jwt:blacklist:token:test-jti", "1", mock.Anything).Return(nil)

		err := svc.BlacklistToken(ctx, "test-jti", futureTime)
		assert.NoError(t, err)
		storage.AssertExpectations(t)
	})

	t.Run("BlacklistToken_empty_jti", func(t *testing.T) {
		storage := new(mockSecondaryStorage)
		svc := NewBlacklistService(storage, &noopLogger{})

		ctx := context.Background()
		futureTime := time.Now().Add(1 * time.Hour)

		err := svc.BlacklistToken(ctx, "", futureTime)
		assert.EqualError(t, err, "jti cannot be empty")
	})

	t.Run("IsBlacklisted_true", func(t *testing.T) {
		storage := new(mockSecondaryStorage)
		svc := NewBlacklistService(storage, &noopLogger{})

		ctx := context.Background()

		storage.On("Get", ctx, "jwt:blacklist:token:test-jti").Return("1", nil)

		blacklisted, err := svc.IsBlacklisted(ctx, "test-jti")
		assert.NoError(t, err)
		assert.True(t, blacklisted)
		storage.AssertExpectations(t)
	})

	t.Run("IsBlacklisted_false", func(t *testing.T) {
		storage := new(mockSecondaryStorage)
		svc := NewBlacklistService(storage, &noopLogger{})

		ctx := context.Background()

		storage.On("Get", ctx, "jwt:blacklist:token:test-jti").Return(nil, nil)

		blacklisted, err := svc.IsBlacklisted(ctx, "test-jti")
		assert.NoError(t, err)
		assert.False(t, blacklisted)
		storage.AssertExpectations(t)
	})

	t.Run("BlacklistAllSessionTokens", func(t *testing.T) {
		storage := new(mockSecondaryStorage)
		svc := NewBlacklistService(storage, &noopLogger{})

		ctx := context.Background()
		futureTime := time.Now().Add(1 * time.Hour)

		storage.On("Set", ctx, "jwt:blacklist:session:sess-1", "1", mock.Anything).Return(nil)

		err := svc.BlacklistAllSessionTokens(ctx, "sess-1", futureTime)
		assert.NoError(t, err)
		storage.AssertExpectations(t)
	})

	t.Run("BlacklistAllSessionTokens_empty", func(t *testing.T) {
		storage := new(mockSecondaryStorage)
		svc := NewBlacklistService(storage, &noopLogger{})

		ctx := context.Background()
		futureTime := time.Now().Add(1 * time.Hour)

		err := svc.BlacklistAllSessionTokens(ctx, "", futureTime)
		assert.EqualError(t, err, "sessionID cannot be empty")
	})
}
