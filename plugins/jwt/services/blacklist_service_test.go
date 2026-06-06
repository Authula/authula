package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	internaltests "github.com/Authula/authula/internal/tests"
)

func TestBlacklistService_BlacklistToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		jti       string
		expiresAt time.Time
		setupMock func(*internaltests.MockSecondaryStorage)
		wantErr   string
	}{
		{
			name:      "success",
			jti:       "test-jti",
			expiresAt: time.Now().Add(1 * time.Hour),
			setupMock: func(storage *internaltests.MockSecondaryStorage) {
				storage.On("Set", mock.Anything, "jwt:blacklist:token:test-jti", "1", mock.Anything).Return(nil)
			},
		},
		{
			name:      "empty jti",
			jti:       "",
			expiresAt: time.Now().Add(1 * time.Hour),
			setupMock: func(storage *internaltests.MockSecondaryStorage) {},
			wantErr:   "jti cannot be empty",
		},
		{
			name:      "expired token",
			jti:       "test-jti",
			expiresAt: time.Now().Add(-1 * time.Hour),
			setupMock: func(storage *internaltests.MockSecondaryStorage) {},
		},
		{
			name:      "storage error",
			jti:       "test-jti",
			expiresAt: time.Now().Add(1 * time.Hour),
			setupMock: func(storage *internaltests.MockSecondaryStorage) {
				storage.On("Set", mock.Anything, "jwt:blacklist:token:test-jti", "1", mock.Anything).Return(assert.AnError)
			},
			wantErr: "failed to blacklist token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storage := new(internaltests.MockSecondaryStorage)
			svc := NewBlacklistService(storage, &internaltests.MockLogger{})
			tt.setupMock(storage)

			err := svc.BlacklistToken(context.Background(), tt.jti, tt.expiresAt)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			storage.AssertExpectations(t)
		})
	}
}

func TestBlacklistService_IsBlacklisted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		jti             string
		setupMock       func(*internaltests.MockSecondaryStorage)
		wantBlacklisted bool
		wantErr         string
	}{
		{
			name: "blacklisted",
			jti:  "test-jti",
			setupMock: func(storage *internaltests.MockSecondaryStorage) {
				storage.On("Get", mock.Anything, "jwt:blacklist:token:test-jti").Return("1", nil)
			},
			wantBlacklisted: true,
		},
		{
			name: "not blacklisted",
			jti:  "test-jti",
			setupMock: func(storage *internaltests.MockSecondaryStorage) {
				storage.On("Get", mock.Anything, "jwt:blacklist:token:test-jti").Return(nil, nil)
			},
		},
		{
			name:      "empty jti",
			jti:       "",
			setupMock: func(storage *internaltests.MockSecondaryStorage) {},
		},
		{
			name: "storage error",
			jti:  "test-jti",
			setupMock: func(storage *internaltests.MockSecondaryStorage) {
				storage.On("Get", mock.Anything, "jwt:blacklist:token:test-jti").Return(nil, assert.AnError)
			},
			wantErr: "failed to check blacklist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storage := new(internaltests.MockSecondaryStorage)
			svc := NewBlacklistService(storage, &internaltests.MockLogger{})
			tt.setupMock(storage)

			blacklisted, err := svc.IsBlacklisted(context.Background(), tt.jti)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantBlacklisted, blacklisted)
			storage.AssertExpectations(t)
		})
	}
}

func TestBlacklistService_BlacklistAllSessionTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sessionID string
		expiresAt time.Time
		setupMock func(*internaltests.MockSecondaryStorage)
		wantErr   string
	}{
		{
			name:      "success",
			sessionID: "sess-1",
			expiresAt: time.Now().Add(1 * time.Hour),
			setupMock: func(storage *internaltests.MockSecondaryStorage) {
				storage.On("Set", mock.Anything, "jwt:blacklist:session:sess-1", "1", mock.Anything).Return(nil)
			},
		},
		{
			name:      "empty sessionID",
			sessionID: "",
			expiresAt: time.Now().Add(1 * time.Hour),
			setupMock: func(storage *internaltests.MockSecondaryStorage) {},
			wantErr:   "sessionID cannot be empty",
		},
		{
			name:      "expired",
			sessionID: "sess-1",
			expiresAt: time.Now().Add(-1 * time.Hour),
			setupMock: func(storage *internaltests.MockSecondaryStorage) {},
		},
		{
			name:      "storage error",
			sessionID: "sess-1",
			expiresAt: time.Now().Add(1 * time.Hour),
			setupMock: func(storage *internaltests.MockSecondaryStorage) {
				storage.On("Set", mock.Anything, "jwt:blacklist:session:sess-1", "1", mock.Anything).Return(assert.AnError)
			},
			wantErr: "failed to blacklist session tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storage := new(internaltests.MockSecondaryStorage)
			svc := NewBlacklistService(storage, &internaltests.MockLogger{})
			tt.setupMock(storage)

			err := svc.BlacklistAllSessionTokens(context.Background(), tt.sessionID, tt.expiresAt)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			storage.AssertExpectations(t)
		})
	}
}

func TestBlacklistService_CleanupExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*internaltests.MockSecondaryStorage)
		wantErr   string
	}{
		{
			name: "no keys to clean",
			setupMock: func(storage *internaltests.MockSecondaryStorage) {
				storage.On("Scan", mock.Anything, "jwt:blacklist:token:").Return([]string{}, nil)
				storage.On("Scan", mock.Anything, "jwt:blacklist:session:").Return([]string{}, nil)
			},
		},
		{
			name: "scan error on first prefix",
			setupMock: func(storage *internaltests.MockSecondaryStorage) {
				storage.On("Scan", mock.Anything, "jwt:blacklist:token:").Return([]string{}, assert.AnError)
			},
			wantErr: "failed to scan blacklist keys",
		},
		{
			name: "expired keys deleted",
			setupMock: func(storage *internaltests.MockSecondaryStorage) {
				storage.On("Scan", mock.Anything, "jwt:blacklist:token:").Return([]string{"jwt:blacklist:token:key1"}, nil)
				storage.On("TTL", mock.Anything, "jwt:blacklist:token:key1").Return(nil, nil)
				storage.On("Delete", mock.Anything, "jwt:blacklist:token:key1").Return(nil)
				storage.On("Scan", mock.Anything, "jwt:blacklist:session:").Return([]string{}, nil)
			},
		},
		{
			name: "non-expired keys skipped",
			setupMock: func(storage *internaltests.MockSecondaryStorage) {
				dur := time.Hour
				storage.On("Scan", mock.Anything, "jwt:blacklist:token:").Return([]string{"jwt:blacklist:token:key1"}, nil)
				storage.On("TTL", mock.Anything, "jwt:blacklist:token:key1").Return(&dur, nil)
				storage.On("Scan", mock.Anything, "jwt:blacklist:session:").Return([]string{}, nil)
			},
		},
		{
			name: "TTL error continues",
			setupMock: func(storage *internaltests.MockSecondaryStorage) {
				storage.On("Scan", mock.Anything, "jwt:blacklist:token:").Return([]string{"jwt:blacklist:token:key1"}, nil)
				storage.On("TTL", mock.Anything, "jwt:blacklist:token:key1").Return(nil, assert.AnError)
				storage.On("Scan", mock.Anything, "jwt:blacklist:session:").Return([]string{}, nil)
			},
		},
		{
			name: "Delete error continues",
			setupMock: func(storage *internaltests.MockSecondaryStorage) {
				storage.On("Scan", mock.Anything, "jwt:blacklist:token:").Return([]string{"jwt:blacklist:token:key1"}, nil)
				storage.On("TTL", mock.Anything, "jwt:blacklist:token:key1").Return(nil, nil)
				storage.On("Delete", mock.Anything, "jwt:blacklist:token:key1").Return(assert.AnError)
				storage.On("Scan", mock.Anything, "jwt:blacklist:session:").Return([]string{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storage := new(internaltests.MockSecondaryStorage)
			svc := NewBlacklistService(storage, &internaltests.MockLogger{})
			tt.setupMock(storage)

			err := svc.CleanupExpired(context.Background())

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			storage.AssertExpectations(t)
		})
	}
}
