package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalmocks "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/tests"
	"github.com/Authula/authula/plugins/jwt/types"
)

type refreshTokenTestFixture struct {
	logger           *internalmocks.MockLogger
	eventBus         *internalmocks.MockEventBus
	sessionSvc       *tests.MockSessionService
	tokenSvc         *tests.MockTokenService
	repo             *tests.MockRefreshTokenRepository
	blacklistSvc     *tests.MockBlacklistService
	gracePeriod      time.Duration
	expiresIn        time.Duration
	refreshExpiresIn time.Duration
}

func newRefreshTokenTestFixture() *refreshTokenTestFixture {
	return &refreshTokenTestFixture{
		logger:           &internalmocks.MockLogger{},
		eventBus:         &internalmocks.MockEventBus{},
		sessionSvc:       &tests.MockSessionService{},
		tokenSvc:         &tests.MockTokenService{},
		repo:             &tests.MockRefreshTokenRepository{},
		blacklistSvc:     &tests.MockBlacklistService{},
		gracePeriod:      10 * time.Second,
		expiresIn:        15 * time.Minute,
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
		blacklistService: f.blacklistSvc,
		gracePeriod:      f.gracePeriod,
		expiresIn:        f.expiresIn,
		refreshExpiresIn: f.refreshExpiresIn,
	}
}

func activeRefreshRecord(expiresAt time.Time) *types.RefreshToken {
	return &types.RefreshToken{
		ID:        uuid.New().String(),
		SessionID: "sess-1",
		TokenHash: "hash",
		ExpiresAt: expiresAt,
		IsRevoked: false,
	}
}

func revokedRefreshRecord(revokedAt time.Time, lastReuse *time.Time) *types.RefreshToken {
	return &types.RefreshToken{
		ID:               uuid.New().String(),
		SessionID:        "sess-1",
		TokenHash:        "hash",
		ExpiresAt:        time.Now().Add(time.Hour),
		IsRevoked:        true,
		RevokedAt:        &revokedAt,
		LastReuseAttempt: lastReuse,
	}
}

func validRefreshClaims() map[string]any {
	return map[string]any{
		"session_id": "sess-1",
		"user_id":    "user-1",
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
				record := activeRefreshRecord(time.Now().Add(time.Hour))
				session := &models.Session{ID: "sess-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}
				tokenPair := &types.TokenPair{AccessToken: "access-token-1", RefreshToken: "refresh-token-1"}

				f.tokenSvc.On("ValidateRefreshToken", ctx, "refresh-token").Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
				f.sessionSvc.On("GetByID", ctx, "sess-1").Return(session, nil)
				f.tokenSvc.On("GenerateUserToken", ctx, "user-1", "sess-1", mock.Anything).Return(tokenPair, nil)
				f.repo.On("RotateRefreshToken", ctx, mock.Anything, mock.Anything).Return(true, nil)
			},
		},
		{
			name: "token_not_found",
			setup: func(f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, "refresh-token").Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(nil, nil)
			},
			wantErr: "invalid refresh token",
		},
		{
			name: "token_expired",
			setup: func(f *refreshTokenTestFixture) {
				record := activeRefreshRecord(time.Now().Add(-time.Hour))
				f.tokenSvc.On("ValidateRefreshToken", ctx, "refresh-token").Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
			},
			wantErr: "invalid refresh token",
		},
		{
			name: "session_not_found",
			setup: func(f *refreshTokenTestFixture) {
				record := activeRefreshRecord(time.Now().Add(time.Hour))
				f.tokenSvc.On("ValidateRefreshToken", ctx, "refresh-token").Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
				f.sessionSvc.On("GetByID", ctx, "sess-1").Return(nil, nil)
			},
			wantErr: "invalid refresh token",
		},
		{
			name: "session_expired",
			setup: func(f *refreshTokenTestFixture) {
				record := activeRefreshRecord(time.Now().Add(time.Hour))
				session := &models.Session{ID: "sess-1", UserID: "user-1", ExpiresAt: time.Now().Add(-time.Minute)}
				f.tokenSvc.On("ValidateRefreshToken", ctx, "refresh-token").Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
				f.sessionSvc.On("GetByID", ctx, "sess-1").Return(session, nil)
			},
			wantErr: "invalid refresh token",
		},
		{
			name: "reuse_within_grace_is_rejected_without_side_effects",
			setup: func(f *refreshTokenTestFixture) {
				record := revokedRefreshRecord(time.Now().Add(-5*time.Second), nil)
				f.tokenSvc.On("ValidateRefreshToken", ctx, "refresh-token").Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
			},
			wantErr: "invalid refresh token",
		},
		{
			name: "reuse_after_grace_invalidates_the_session",
			setup: func(f *refreshTokenTestFixture) {
				record := revokedRefreshRecord(time.Now().Add(-30*time.Second), nil)
				f.tokenSvc.On("ValidateRefreshToken", ctx, "refresh-token").Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
				f.repo.On("RevokeAllSessionTokens", ctx, "sess-1").Return(nil)
				f.blacklistSvc.On("BlacklistAllSessionTokens", ctx, "sess-1", mock.Anything).Return(nil)
				f.sessionSvc.On("Delete", ctx, "sess-1").Return(nil)
			},
			wantErr: "invalid refresh token",
		},
		{
			name: "rotation_conflict_re_reads_and_treats_as_reuse",
			setup: func(f *refreshTokenTestFixture) {
				record := activeRefreshRecord(time.Now().Add(time.Hour))
				revoked := revokedRefreshRecord(time.Now().Add(-30*time.Second), nil)
				session := &models.Session{ID: "sess-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}
				tokenPair := &types.TokenPair{AccessToken: "access-token-1", RefreshToken: "refresh-token-1"}

				f.tokenSvc.On("ValidateRefreshToken", ctx, "refresh-token").Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil).Once()
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(revoked, nil).Once()
				f.sessionSvc.On("GetByID", ctx, "sess-1").Return(session, nil)
				f.tokenSvc.On("GenerateUserToken", ctx, "user-1", "sess-1", mock.Anything).Return(tokenPair, nil)
				f.repo.On("RotateRefreshToken", ctx, mock.Anything, mock.Anything).Return(false, nil)
				f.repo.On("RevokeAllSessionTokens", ctx, "sess-1").Return(nil)
				f.blacklistSvc.On("BlacklistAllSessionTokens", ctx, "sess-1", mock.Anything).Return(nil)
				f.sessionSvc.On("Delete", ctx, "sess-1").Return(nil)
			},
			wantErr: "invalid refresh token",
		},
		{
			name: "invalid_signature_is_rejected_before_lookup",
			setup: func(f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, "refresh-token").Return(nil, assert.AnError)
			},
			wantErr: "invalid refresh token",
		},
		{
			name: "access_token_cannot_be_used_as_refresh",
			setup: func(f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, "refresh-token").Return(nil, assert.AnError)
			},
			wantErr: "invalid refresh token",
		},
		{
			name: "session_claim_does_not_match_stored_record",
			setup: func(f *refreshTokenTestFixture) {
				record := activeRefreshRecord(time.Now().Add(time.Hour))
				claims := validRefreshClaims()
				claims["session_id"] = "sess-2"
				f.tokenSvc.On("ValidateRefreshToken", ctx, "refresh-token").Return(claims, nil)
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
			},
			wantErr: "invalid refresh token",
		},
		{
			name: "user_claim_does_not_match_session",
			setup: func(f *refreshTokenTestFixture) {
				record := activeRefreshRecord(time.Now().Add(time.Hour))
				session := &models.Session{ID: "sess-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}
				claims := validRefreshClaims()
				claims["user_id"] = "user-2"
				f.tokenSvc.On("ValidateRefreshToken", ctx, "refresh-token").Return(claims, nil)
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
				f.sessionSvc.On("GetByID", ctx, "sess-1").Return(session, nil)
			},
			wantErr: "invalid refresh token",
		},
		{
			name: "reserved_claims_are_not_copied_to_new_tokens",
			setup: func(f *refreshTokenTestFixture) {
				record := activeRefreshRecord(time.Now().Add(time.Hour))
				session := &models.Session{ID: "sess-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}
				claims := validRefreshClaims()
				claims["iss"] = "https://issuer"
				claims["aud"] = []string{"authula"}
				claims["client_id"] = "client-1"
				claims["nbf"] = time.Now()
				claims["organization_id"] = "org-1"
				tokenPair := &types.TokenPair{AccessToken: "access-token-1", RefreshToken: "refresh-token-1"}

				f.tokenSvc.On("ValidateRefreshToken", ctx, "refresh-token").Return(claims, nil)
				f.repo.On("GetRefreshToken", ctx, mock.Anything).Return(record, nil)
				f.sessionSvc.On("GetByID", ctx, "sess-1").Return(session, nil)
				f.tokenSvc.On("GenerateUserToken", ctx, "user-1", "sess-1", mock.MatchedBy(func(extra map[string]any) bool {
					return len(extra) == 0
				})).Return(tokenPair, nil)
				f.repo.On("RotateRefreshToken", ctx, mock.Anything, mock.Anything).Return(true, nil)
			},
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
				require.Equal(t, "Bearer", resp.TokenType)
				require.Equal(t, int64(900), resp.ExpiresIn)
			}

			f.repo.AssertExpectations(t)
			f.sessionSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
			f.blacklistSvc.AssertExpectations(t)
		})
	}
}

func TestRefreshTokenService_StoreInitialRefreshToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		refreshToken string
		sessionID    string
		setupMock    func(f *refreshTokenTestFixture)
		wantErr      string
	}{
		{
			name:         "success",
			refreshToken: "refresh-token",
			sessionID:    "sess-1",
			setupMock: func(f *refreshTokenTestFixture) {
				f.repo.On("StoreRefreshToken", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:         "storage error",
			refreshToken: "refresh-token",
			sessionID:    "sess-1",
			setupMock: func(f *refreshTokenTestFixture) {
				f.repo.On("StoreRefreshToken", mock.Anything, mock.Anything).Return(assert.AnError)
			},
			wantErr: assert.AnError.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newRefreshTokenTestFixture()
			f.eventBus.On("Publish", mock.Anything).Return(nil).Maybe()
			tt.setupMock(f)
			svc := f.newService()

			future := time.Now().Add(24 * time.Hour)
			err := svc.StoreInitialRefreshToken(context.Background(), tt.refreshToken, tt.sessionID, future)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			f.repo.AssertExpectations(t)
		})
	}
}
