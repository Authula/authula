package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	internalmocks "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/jwt/tests"
	"github.com/Authula/authula/plugins/jwt/types"
)

type refreshTokenTestFixture struct {
	logger           *internalmocks.MockLogger
	eventBus         *internalmocks.MockEventBus
	sessionSvc       *tests.MockSessionService
	tokenSvc         *tests.MockTokenService
	repo             *tests.MockRefreshTokenRepository
	gracePeriod      time.Duration
	refreshExpiresIn time.Duration
}

func newRefreshTokenTestFixture() *refreshTokenTestFixture {
	return &refreshTokenTestFixture{
		logger:           &internalmocks.MockLogger{},
		eventBus:         &internalmocks.MockEventBus{},
		sessionSvc:       &tests.MockSessionService{},
		tokenSvc:         &tests.MockTokenService{},
		repo:             &tests.MockRefreshTokenRepository{},
		gracePeriod:      10 * time.Second,
		refreshExpiresIn: 7 * 24 * time.Hour,
	}
}

func (f *refreshTokenTestFixture) newService() RefreshTokenService {
	return &refreshTokenService{
		logger:           f.logger,
		eventBus:         f.eventBus,
		sessionService:   f.sessionSvc,
		jwtService:       f.tokenSvc,
		storage:          f.repo,
		gracePeriod:      f.gracePeriod,
		refreshExpiresIn: f.refreshExpiresIn,
	}
}

func TestRefreshTokenService_RefreshTokens(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(f *refreshTokenTestFixture)
		wantErr string
	}{
		{
			name: "successful_rotation",
			setup: func(f *refreshTokenTestFixture) {
				now := time.Now()
				record := &types.RefreshToken{
					ID:        uuid.New().String(),
					SessionID: "sess-1",
					TokenHash: "hash",
					ExpiresAt: now.Add(1 * time.Hour),
					IsRevoked: false,
				}
				session := &models.Session{
					ID:     "sess-1",
					UserID: "user-1",
				}
				tokenPair := &types.TokenPair{
					AccessToken:  "access-token-1",
					RefreshToken: "refresh-token-1",
				}

				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
				f.sessionSvc.On("GetByID", ctx, "sess-1").Return(session, nil)
				f.repo.On("RevokeRefreshToken", ctx, mock.Anything).Return(nil)
				f.tokenSvc.On("GenerateUserToken", ctx, "user-1", "sess-1").Return(tokenPair, nil)
				f.repo.On("StoreRefreshToken", ctx, mock.Anything).Return(nil)
			},
		},
		{
			name: "token_not_found",
			setup: func(f *refreshTokenTestFixture) {
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(nil, nil)
			},
			wantErr: "invalid refresh token",
		},
		{
			name: "token_expired",
			setup: func(f *refreshTokenTestFixture) {
				now := time.Now()
				record := &types.RefreshToken{
					ID:        uuid.New().String(),
					SessionID: "sess-1",
					TokenHash: "hash",
					ExpiresAt: now.Add(-1 * time.Hour),
					IsRevoked: false,
				}
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
			},
			wantErr: "refresh token expired",
		},
		{
			name: "session_not_found",
			setup: func(f *refreshTokenTestFixture) {
				now := time.Now()
				record := &types.RefreshToken{
					ID:        uuid.New().String(),
					SessionID: "sess-1",
					TokenHash: "hash",
					ExpiresAt: now.Add(1 * time.Hour),
					IsRevoked: false,
				}
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
				f.sessionSvc.On("GetByID", ctx, "sess-1").Return(nil, nil)
			},
			wantErr: "session expired or invalid",
		},
		{
			name: "reuse_tier1_recovery",
			setup: func(f *refreshTokenTestFixture) {
				now := time.Now()
				revokedAt := now.Add(-5 * time.Second)
				record := &types.RefreshToken{
					ID:               uuid.New().String(),
					SessionID:        "sess-1",
					TokenHash:        "hash",
					ExpiresAt:        now.Add(1 * time.Hour),
					IsRevoked:        true,
					RevokedAt:        &revokedAt,
					LastReuseAttempt: nil,
				}
				session := &models.Session{
					ID:     "sess-1",
					UserID: "user-1",
				}
				tokenPair := &types.TokenPair{
					AccessToken:  "access-token-1",
					RefreshToken: "refresh-token-1",
				}

				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
				f.repo.On("SetLastReuseAttempt", ctx, mock.Anything).Return(nil)
				f.sessionSvc.On("GetByID", ctx, "sess-1").Return(session, nil)
				f.repo.On("RevokeRefreshToken", ctx, mock.Anything).Return(nil)
				f.tokenSvc.On("GenerateUserToken", ctx, "user-1", "sess-1").Return(tokenPair, nil)
				f.repo.On("StoreRefreshToken", ctx, mock.Anything).Return(nil)
			},
		},
		{
			name: "reuse_tier2_throttle",
			setup: func(f *refreshTokenTestFixture) {
				now := time.Now()
				revokedAt := now.Add(-5 * time.Second)
				lastReuse := now.Add(-2 * time.Second)
				record := &types.RefreshToken{
					ID:               uuid.New().String(),
					SessionID:        "sess-1",
					TokenHash:        "hash",
					ExpiresAt:        now.Add(1 * time.Hour),
					IsRevoked:        true,
					RevokedAt:        &revokedAt,
					LastReuseAttempt: &lastReuse,
				}
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
			},
			wantErr: "invalid refresh token",
		},
		{
			name: "reuse_tier3_reject",
			setup: func(f *refreshTokenTestFixture) {
				now := time.Now()
				revokedAt := now.Add(-30 * time.Second)
				record := &types.RefreshToken{
					ID:               uuid.New().String(),
					SessionID:        "sess-1",
					TokenHash:        "hash",
					ExpiresAt:        now.Add(1 * time.Hour),
					IsRevoked:        true,
					RevokedAt:        &revokedAt,
					LastReuseAttempt: nil,
				}
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
			},
			wantErr: "invalid refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRefreshTokenTestFixture()
			f.eventBus.On("Publish", mock.Anything).Return(nil).Maybe()
			if tt.setup != nil {
				tt.setup(f)
			}
			svc := f.newService()

			resp, err := svc.RefreshTokens(ctx, "refresh-token")
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotEmpty(t, resp.AccessToken)
				require.NotEmpty(t, resp.RefreshToken)
			}

			f.repo.AssertExpectations(t)
			f.sessionSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}

func TestRefreshTokenService_StoreInitialRefreshToken(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		f := newRefreshTokenTestFixture()
		f.eventBus.On("Publish", mock.Anything).Return(nil).Maybe()
		f.repo.On("StoreRefreshToken", ctx, mock.Anything).Return(nil)

		svc := f.newService()
		future := time.Now().Add(24 * time.Hour)
		err := svc.StoreInitialRefreshToken(ctx, "refresh-token", "sess-1", future)

		require.NoError(t, err)
		f.repo.AssertExpectations(t)
	})
}
