package services

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internalmocks "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/constants"
	"github.com/Authula/authula/plugins/jwt/events"
	"github.com/Authula/authula/plugins/jwt/tests"
	"github.com/Authula/authula/plugins/jwt/types"
)

const (
	testRefreshToken    = "refresh-token"
	testNewRefreshToken = "new-refresh-token"
	testSessionID       = "sess-1"
	testUserID          = "user-1"
)

var errRefreshTest = errors.New("refresh test error")

type refreshTokenTestFixture struct {
	logger           *internalmocks.LevelRecordingLogger
	eventBus         models.EventBus
	recorder         *internalmocks.RecordingEventBus
	sessionSvc       *tests.MockSessionService
	tokenSvc         *tests.MockTokenService
	repo             *tests.MockRefreshTokenRepository
	gracePeriod      time.Duration
	expiresIn        time.Duration
	refreshExpiresIn time.Duration
}

func newRefreshTokenTestFixture() *refreshTokenTestFixture {
	recorder := internalmocks.NewRecordingEventBus()

	return &refreshTokenTestFixture{
		logger:           &internalmocks.LevelRecordingLogger{},
		eventBus:         recorder,
		recorder:         recorder,
		sessionSvc:       &tests.MockSessionService{},
		tokenSvc:         &tests.MockTokenService{},
		repo:             &tests.MockRefreshTokenRepository{},
		gracePeriod:      10 * time.Second,
		expiresIn:        15 * time.Minute,
		refreshExpiresIn: 7 * 24 * time.Hour,
	}
}

func (f *refreshTokenTestFixture) newService() *refreshTokenService {
	service := NewRefreshTokenService(
		f.logger,
		f.eventBus,
		f.sessionSvc,
		f.tokenSvc,
		f.repo,
		f.gracePeriod,
		f.expiresIn,
		f.refreshExpiresIn,
	)

	return service.(*refreshTokenService)
}

func (f *refreshTokenTestFixture) assertExpectations(t *testing.T) {
	t.Helper()

	f.repo.AssertExpectations(t)
	f.sessionSvc.AssertExpectations(t)
	f.tokenSvc.AssertExpectations(t)
}

func (f *refreshTokenTestFixture) assertLogged(t *testing.T, message string) {
	t.Helper()

	require.Contains(t, f.logger.Messages(), message)
}

func activeRefreshRecord(expiresAt time.Time) *types.RefreshToken {
	return &types.RefreshToken{
		ID:        uuid.NewString(),
		SessionID: testSessionID,
		TokenHash: HashRefreshToken(testRefreshToken),
		ExpiresAt: expiresAt,
	}
}

func revokedRefreshRecord(revokedAt time.Time) *types.RefreshToken {
	record := activeRefreshRecord(time.Now().Add(time.Hour))
	record.IsRevoked = true
	record.RevokedAt = &revokedAt

	return record
}

func activeSession() *models.Session {
	return &models.Session{
		ID:        testSessionID,
		UserID:    testUserID,
		ExpiresAt: time.Now().Add(time.Hour),
	}
}

func validRefreshClaims() map[string]any {
	return map[string]any{
		"session_id":   testSessionID,
		jwt.SubjectKey: testUserID,
	}
}

func assertInvalidRefreshToken(t *testing.T, _ *refreshTokenTestFixture, response *types.RefreshTokenResponse, err error) {
	t.Helper()

	require.Nil(t, response)
	require.ErrorIs(t, err, constants.ErrInvalidRefreshToken)
}

func isTimeNear(target time.Time, tolerance time.Duration) func(time.Time) bool {
	return func(actual time.Time) bool {
		delta := actual.Sub(target)
		return delta >= -tolerance && delta <= tolerance
	}
}

func TestNewRefreshTokenService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		check func(*testing.T, *refreshTokenTestFixture, *refreshTokenService)
	}{
		{
			name: "logger",
			check: func(t *testing.T, f *refreshTokenTestFixture, service *refreshTokenService) {
				require.Same(t, f.logger, service.logger)
			},
		},
		{
			name: "event_bus",
			check: func(t *testing.T, f *refreshTokenTestFixture, service *refreshTokenService) {
				require.Equal(t, f.eventBus, service.eventBus)
			},
		},
		{
			name: "session_service",
			check: func(t *testing.T, f *refreshTokenTestFixture, service *refreshTokenService) {
				require.Same(t, f.sessionSvc, service.sessionService)
			},
		},
		{
			name: "token_service",
			check: func(t *testing.T, f *refreshTokenTestFixture, service *refreshTokenService) {
				require.Same(t, f.tokenSvc, service.jwtService)
			},
		},
		{
			name: "refresh_token_repository",
			check: func(t *testing.T, f *refreshTokenTestFixture, service *refreshTokenService) {
				require.Same(t, f.repo, service.storage)
			},
		},
		{
			name: "grace_period",
			check: func(t *testing.T, f *refreshTokenTestFixture, service *refreshTokenService) {
				require.Equal(t, f.gracePeriod, service.gracePeriod)
			},
		},
		{
			name: "access_token_expiry",
			check: func(t *testing.T, f *refreshTokenTestFixture, service *refreshTokenService) {
				require.Equal(t, f.expiresIn, service.expiresIn)
			},
		},
		{
			name: "refresh_token_expiry",
			check: func(t *testing.T, f *refreshTokenTestFixture, service *refreshTokenService) {
				require.Equal(t, f.refreshExpiresIn, service.refreshExpiresIn)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newRefreshTokenTestFixture()
			service := fixture.newService()

			tt.check(t, fixture, service)
		})
	}
}

func TestRefreshTokenService_RefreshTokens(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name  string
		setup func(*testing.T, *refreshTokenTestFixture)
		check func(*testing.T, *refreshTokenTestFixture, *types.RefreshTokenResponse, error)
	}{
		{
			name: "successful rotation",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				now := time.Now()

				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
				f.sessionSvc.On("GetByID", ctx, testSessionID).Return(activeSession(), nil)
				f.tokenSvc.On("GenerateUserToken", ctx, testUserID, testSessionID, mock.Anything).Return(&types.TokenPair{
					AccessToken:  "access-token-2",
					RefreshToken: testNewRefreshToken,
				}, nil)
				f.repo.On("RotateRefreshToken", ctx, HashRefreshToken(testRefreshToken), mock.MatchedBy(func(record *types.RefreshToken) bool {
					if record == nil || uuid.Validate(record.ID) != nil {
						return false
					}

					return record.SessionID == testSessionID &&
						record.TokenHash == HashRefreshToken(testNewRefreshToken) &&
						!record.IsRevoked &&
						record.RevokedAt == nil &&
						record.LastReuseAttempt == nil &&
						isTimeNear(now.Add(f.refreshExpiresIn), 5*time.Second)(record.ExpiresAt) &&
						isTimeNear(now, 5*time.Second)(record.CreatedAt)
				})).Return(true, nil)
			},
			check: func(t *testing.T, _ *refreshTokenTestFixture, response *types.RefreshTokenResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, &types.RefreshTokenResponse{
					AccessToken:  "access-token-2",
					RefreshToken: testNewRefreshToken,
					TokenType:    "Bearer",
					ExpiresIn:    900,
				}, response)
			},
		},
		{
			name: "validation error is rejected before lookup",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(nil, errRefreshTest)
			},
			check: func(t *testing.T, f *refreshTokenTestFixture, response *types.RefreshTokenResponse, err error) {
				assertInvalidRefreshToken(t, f, response, err)
				f.repo.AssertNotCalled(t, "GetRefreshToken", mock.Anything, mock.Anything)
			},
		},
		{
			name: "storage lookup error",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(nil, errRefreshTest)
			},
			check: assertInvalidRefreshToken,
		},
		{
			name: "record not found",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(nil, constants.ErrRefreshTokenNotFound)
			},
			check: assertInvalidRefreshToken,
		},
		{
			name: "missing session claim",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				claims := validRefreshClaims()
				delete(claims, "session_id")
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(claims, nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
			},
			check: assertInvalidRefreshToken,
		},
		{
			name: "non string session claim",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				claims := validRefreshClaims()
				claims["session_id"] = 123
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(claims, nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
			},
			check: assertInvalidRefreshToken,
		},
		{
			name: "session claim does not match stored record",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				claims := validRefreshClaims()
				claims["session_id"] = "sess-2"
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(claims, nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
			},
			check: func(t *testing.T, f *refreshTokenTestFixture, response *types.RefreshTokenResponse, err error) {
				assertInvalidRefreshToken(t, f, response, err)
				f.sessionSvc.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
			},
		},
		{
			name: "revoked token without revoked at is rejected",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				record := activeRefreshRecord(time.Now().Add(time.Hour))
				record.IsRevoked = true
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(record, nil)
			},
			check: func(t *testing.T, f *refreshTokenTestFixture, response *types.RefreshTokenResponse, err error) {
				assertInvalidRefreshToken(t, f, response, err)
				f.assertLogged(t, "token marked revoked but has no revoked_at timestamp")
				f.repo.AssertNotCalled(t, "RevokeAllSessionTokens", mock.Anything, mock.Anything)
			},
		},
		{
			name: "reuse within grace is throttled without side effects",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(revokedRefreshRecord(time.Now().Add(-time.Second)), nil)
			},
			check: func(t *testing.T, f *refreshTokenTestFixture, response *types.RefreshTokenResponse, err error) {
				assertInvalidRefreshToken(t, f, response, err)
				f.assertLogged(t, "[AUTH_REUSE_THROTTLED] Refresh token reuse within the grace period rejected")

				event := f.recorder.NextEvent(t)
				require.Equal(t, constants.EventTokenReuseThrottled, event.Type)
				require.Equal(t, constants.EventTokenReuseThrottled, event.Payload["type"])
				require.Equal(t, testSessionID, event.Payload["session_id"])
				require.Equal(t, HashRefreshToken(testRefreshToken), event.Payload["token_hash"])
				require.Equal(t, float64(2), event.Payload["attempt_count"])
				require.Equal(t, "10000ms", event.Payload["grace_period_config"])
				require.Positive(t, event.Payload["delta_ms"])

				f.repo.AssertNotCalled(t, "RevokeAllSessionTokens", mock.Anything, mock.Anything)
				f.sessionSvc.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
			},
		},
		{
			name: "reuse after grace invalidates the session",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(revokedRefreshRecord(time.Now().Add(-time.Minute)), nil)
				f.repo.On("RevokeAllSessionTokens", ctx, testSessionID).Return(nil)
				f.sessionSvc.On("Delete", ctx, testSessionID).Return(nil)
			},
			check: func(t *testing.T, f *refreshTokenTestFixture, response *types.RefreshTokenResponse, err error) {
				assertInvalidRefreshToken(t, f, response, err)
				f.assertLogged(t, "[AUTH_REUSE_MALICIOUS] Refresh token reuse after the grace period invalidates the session")

				event := f.recorder.NextEvent(t)
				require.Equal(t, constants.EventTokenReuseMalicious, event.Type)
				require.Equal(t, constants.EventTokenReuseMalicious, event.Payload["type"])
				require.Equal(t, testSessionID, event.Payload["session_id"])
				require.Equal(t, HashRefreshToken(testRefreshToken), event.Payload["token_hash"])
				require.Equal(t, "10000ms", event.Payload["grace_period_config"])
				require.Positive(t, event.Payload["delta_ms"])
			},
		},
		{
			name: "expired record is rejected",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(-time.Second)), nil)
			},
			check: func(t *testing.T, f *refreshTokenTestFixture, response *types.RefreshTokenResponse, err error) {
				assertInvalidRefreshToken(t, f, response, err)
				f.sessionSvc.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
			},
		},
		{
			name: "session not found",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
				f.sessionSvc.On("GetByID", ctx, testSessionID).Return(nil, coreerrors.ErrSessionNotFound)
			},
			check: func(t *testing.T, f *refreshTokenTestFixture, response *types.RefreshTokenResponse, err error) {
				assertInvalidRefreshToken(t, f, response, err)
				f.assertLogged(t, "refresh token belongs to a missing session")
			},
		},
		{
			name: "session lookup error",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
				f.sessionSvc.On("GetByID", ctx, testSessionID).Return(nil, errRefreshTest)
			},
			check: assertInvalidRefreshToken,
		},
		{
			name: "nil session",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
				f.sessionSvc.On("GetByID", ctx, testSessionID).Return(nil, nil)
			},
			check: func(t *testing.T, f *refreshTokenTestFixture, response *types.RefreshTokenResponse, err error) {
				assertInvalidRefreshToken(t, f, response, err)
				f.assertLogged(t, "refresh token belongs to a missing or expired session")
			},
		},
		{
			name: "expired session",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				session := activeSession()
				session.ExpiresAt = time.Now().Add(-time.Second)
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
				f.sessionSvc.On("GetByID", ctx, testSessionID).Return(session, nil)
			},
			check: func(t *testing.T, f *refreshTokenTestFixture, response *types.RefreshTokenResponse, err error) {
				assertInvalidRefreshToken(t, f, response, err)
				f.assertLogged(t, "refresh token belongs to a missing or expired session")
			},
		},
		{
			name: "missing subject claim",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				claims := validRefreshClaims()
				delete(claims, jwt.SubjectKey)
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(claims, nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
				f.sessionSvc.On("GetByID", ctx, testSessionID).Return(activeSession(), nil)
			},
			check: assertInvalidRefreshToken,
		},
		{
			name: "subject claim does not match session",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				claims := validRefreshClaims()
				claims[jwt.SubjectKey] = "user-2"
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(claims, nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
				f.sessionSvc.On("GetByID", ctx, testSessionID).Return(activeSession(), nil)
			},
			check: assertInvalidRefreshToken,
		},
		{
			name: "token generation error",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
				f.sessionSvc.On("GetByID", ctx, testSessionID).Return(activeSession(), nil)
				f.tokenSvc.On("GenerateUserToken", ctx, testUserID, testSessionID, mock.Anything).Return(nil, errRefreshTest)
			},
			check: func(t *testing.T, f *refreshTokenTestFixture, response *types.RefreshTokenResponse, err error) {
				require.Nil(t, response)
				require.EqualError(t, err, "failed to generate tokens")
				f.repo.AssertNotCalled(t, "RotateRefreshToken", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name: "rotation storage error",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
				f.sessionSvc.On("GetByID", ctx, testSessionID).Return(activeSession(), nil)
				f.tokenSvc.On("GenerateUserToken", ctx, testUserID, testSessionID, mock.Anything).Return(&types.TokenPair{
					AccessToken:  "access-token-2",
					RefreshToken: testNewRefreshToken,
				}, nil)
				f.repo.On("RotateRefreshToken", ctx, HashRefreshToken(testRefreshToken), mock.Anything).Return(false, errRefreshTest)
			},
			check: assertInvalidRefreshToken,
		},
		{
			name: "rotation conflict with revoked token is handled as reuse",
			setup: func(_ *testing.T, f *refreshTokenTestFixture) {
				f.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil).Once()
				f.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(revokedRefreshRecord(time.Now().Add(-time.Minute)), nil).Once()
				f.sessionSvc.On("GetByID", ctx, testSessionID).Return(activeSession(), nil)
				f.tokenSvc.On("GenerateUserToken", ctx, testUserID, testSessionID, mock.Anything).Return(&types.TokenPair{
					AccessToken:  "access-token-2",
					RefreshToken: testNewRefreshToken,
				}, nil)
				f.repo.On("RotateRefreshToken", ctx, HashRefreshToken(testRefreshToken), mock.Anything).Return(false, nil)
				f.repo.On("RevokeAllSessionTokens", ctx, testSessionID).Return(nil)
				f.sessionSvc.On("Delete", ctx, testSessionID).Return(nil)
			},
			check: func(t *testing.T, f *refreshTokenTestFixture, response *types.RefreshTokenResponse, err error) {
				assertInvalidRefreshToken(t, f, response, err)

				event := f.recorder.NextEvent(t)
				require.Equal(t, constants.EventTokenReuseMalicious, event.Type)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newRefreshTokenTestFixture()
			tt.setup(t, fixture)

			service := fixture.newService()
			response, err := service.RefreshTokens(ctx, testRefreshToken)

			tt.check(t, fixture, response, err)
			fixture.assertExpectations(t)
		})
	}
}

func TestRefreshTokenService_RefreshTokens_ExtraClaims(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	disallowedClaims := map[string]any{
		"iss":             "https://issuer",
		"aud":             []string{"authula"},
		"sub":             testUserID,
		"user_id":         testUserID,
		"session_id":      testSessionID,
		"iat":             time.Now(),
		"exp":             time.Now().Add(time.Hour),
		"nbf":             time.Now(),
		"jti":             "jti-1",
		"token_type":      "refresh",
		"actor_type":      "user",
		"organization_id": "org-1",
		"scope":           "read",
		"client_id":       "client-1",
		"tenant":          "acme",
	}

	tests := []struct {
		name      string
		claims    map[string]any
		wantExtra map[string]any
	}{
		{
			name: "allowlisted claims are forwarded",
			claims: func() map[string]any {
				claims := validRefreshClaims()
				claims["impersonator_id"] = "admin-1"
				claims["impersonator_scope"] = "read write"
				return claims
			}(),
			wantExtra: map[string]any{"impersonator_id": "admin-1", "impersonator_scope": "read write"},
		},
		{
			name: "disallowed claims are filtered",
			claims: func() map[string]any {
				claims := validRefreshClaims()
				maps.Copy(claims, disallowedClaims)
				claims["impersonator_id"] = "admin-1"
				return claims
			}(),
			wantExtra: map[string]any{"impersonator_id": "admin-1"},
		},
		{
			name:      "only_disallowed_claims_produce_nil",
			claims:    disallowedClaims,
			wantExtra: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newRefreshTokenTestFixture()
			fixture.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(tt.claims, nil)
			fixture.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
			fixture.sessionSvc.On("GetByID", ctx, testSessionID).Return(activeSession(), nil)
			fixture.tokenSvc.On("GenerateUserToken", ctx, testUserID, testSessionID, mock.MatchedBy(func(extra map[string]any) bool {
				return reflect.DeepEqual(tt.wantExtra, extra)
			})).Return(&types.TokenPair{
				AccessToken:  "access-token-2",
				RefreshToken: testNewRefreshToken,
			}, nil)
			fixture.repo.On("RotateRefreshToken", ctx, HashRefreshToken(testRefreshToken), mock.Anything).Return(true, nil)

			service := fixture.newService()
			response, err := service.RefreshTokens(ctx, testRefreshToken)

			require.NoError(t, err)
			require.NotNil(t, response)
			fixture.assertExpectations(t)
		})
	}
}

func TestRefreshTokenService_RefreshTokens_AuditMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	metadata := events.AuditMetadata{
		ClientIP:  "203.0.113.10",
		UserAgent: "test-agent",
		DeviceID:  "device-1",
	}

	tests := []struct {
		name         string
		metadata     events.AuditMetadata
		withMetadata bool
		wantMetadata map[string]any
	}{
		{
			name:         "RefreshTokens uses empty metadata",
			metadata:     events.AuditMetadata{},
			withMetadata: false,
			wantMetadata: map[string]any{},
		},
		{
			name:         "RefreshTokensWithMetadata forwards audit metadata",
			metadata:     metadata,
			withMetadata: true,
			wantMetadata: map[string]any{
				"client_ip":  "203.0.113.10",
				"user_agent": "test-agent",
				"device_id":  "device-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newRefreshTokenTestFixture()
			expectReuseAfterGrace(t, fixture, ctx)

			service := fixture.newService()

			var (
				response *types.RefreshTokenResponse
				err      error
			)
			if tt.withMetadata {
				response, err = service.RefreshTokensWithMetadata(ctx, testRefreshToken, tt.metadata)
			} else {
				response, err = service.RefreshTokens(ctx, testRefreshToken)
			}

			assertInvalidRefreshToken(t, fixture, response, err)
			event := fixture.recorder.NextEvent(t)
			require.Equal(t, tt.wantMetadata, event.Payload["metadata"])
			fixture.assertExpectations(t)
		})
	}
}

func TestRefreshTokenService_HandleRotationConflict(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tokenHash := HashRefreshToken(testRefreshToken)

	tests := []struct {
		name  string
		setup func(*refreshTokenTestFixture)
		check func(*testing.T, *refreshTokenTestFixture, error)
	}{
		{
			name: "re-read error is rejected",
			setup: func(f *refreshTokenTestFixture) {
				f.repo.On("GetRefreshToken", ctx, tokenHash).Return(nil, errRefreshTest)
			},
			check: func(t *testing.T, f *refreshTokenTestFixture, err error) {
				require.ErrorIs(t, err, constants.ErrInvalidRefreshToken)
				f.assertLogged(t, "refresh token re-read failed")
			},
		},
		{
			name: "missing re-read record is rejected",
			setup: func(f *refreshTokenTestFixture) {
				f.repo.On("GetRefreshToken", ctx, tokenHash).Return(nil, constants.ErrRefreshTokenNotFound)
			},
			check: func(t *testing.T, _ *refreshTokenTestFixture, err error) {
				require.ErrorIs(t, err, constants.ErrInvalidRefreshToken)
			},
		},
		{
			name: "still active record is rejected",
			setup: func(f *refreshTokenTestFixture) {
				f.repo.On("GetRefreshToken", ctx, tokenHash).Return(activeRefreshRecord(time.Now().Add(time.Hour)), nil)
			},
			check: func(t *testing.T, _ *refreshTokenTestFixture, err error) {
				require.ErrorIs(t, err, constants.ErrInvalidRefreshToken)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newRefreshTokenTestFixture()
			tt.setup(fixture)

			service := fixture.newService()
			err := service.handleRotationConflict(ctx, tokenHash, events.AuditMetadata{})

			tt.check(t, fixture, err)
			fixture.assertExpectations(t)
		})
	}
}

func TestRefreshTokenService_InvalidateSessionFamily(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name      string
		revokeErr error
		deleteErr error
		wantLog   string
	}{
		{
			name: "success",
		},
		{
			name:      "revoke error is logged",
			revokeErr: errRefreshTest,
			wantLog:   "failed to revoke session refresh tokens",
		},
		{
			name:      "delete error is logged",
			deleteErr: errRefreshTest,
			wantLog:   "failed to delete invalidated session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newRefreshTokenTestFixture()
			fixture.repo.On("RevokeAllSessionTokens", ctx, testSessionID).Return(tt.revokeErr)
			fixture.sessionSvc.On("Delete", ctx, testSessionID).Return(tt.deleteErr)

			service := fixture.newService()
			service.invalidateSessionFamily(ctx, testSessionID)

			if tt.wantLog != "" {
				fixture.assertLogged(t, tt.wantLog)
			}
			fixture.assertExpectations(t)
		})
	}
}

func TestFilterAllowedExtraClaims(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		claims map[string]any
		want   map[string]any
	}{
		{
			name:   "nil claims",
			claims: nil,
			want:   nil,
		},
		{
			name: "only disallowed claims",
			claims: map[string]any{
				"sub":        testUserID,
				"user_id":    testUserID,
				"session_id": testSessionID,
				"iss":        "issuer",
				"tenant":     "acme",
			},
			want: nil,
		},
		{
			name:   "allowlisted claims",
			claims: map[string]any{"impersonator_id": "admin-1", "impersonator_scope": "read write"},
			want:   map[string]any{"impersonator_id": "admin-1", "impersonator_scope": "read write"},
		},
		{
			name:   "mixed claims",
			claims: map[string]any{"impersonator_id": "admin-1", "tenant": "acme"},
			want:   map[string]any{"impersonator_id": "admin-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, filterAllowedExtraClaims(tt.claims))
		})
	}
}

func TestRefreshTokenService_ReuseEvents_WithNilEventBus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		emit func(*refreshTokenService)
	}{
		{
			name: "throttled event",
			emit: func(service *refreshTokenService) {
				service.emitTokenReuseThrottledEvent(testSessionID, "hash", 10, 1000, events.AuditMetadata{})
			},
		},
		{
			name: "malicious event",
			emit: func(service *refreshTokenService) {
				service.emitTokenReuseMaliciousEvent(testSessionID, "hash", 10, 1000, events.AuditMetadata{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newRefreshTokenTestFixture()
			fixture.eventBus = nil
			service := fixture.newService()

			require.NotPanics(t, func() {
				tt.emit(service)
			})
		})
	}
}

func TestRefreshTokenService_StoreInitialRefreshToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	expiresAt := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name          string
		recordMatcher func(*types.RefreshToken) bool
		wantErr       error
	}{
		{
			name: "stores token record",
			recordMatcher: func(record *types.RefreshToken) bool {
				if record == nil || uuid.Validate(record.ID) != nil {
					return false
				}

				return record.SessionID == testSessionID &&
					record.TokenHash == HashRefreshToken(testRefreshToken) &&
					record.ExpiresAt.Equal(expiresAt) &&
					!record.IsRevoked &&
					record.RevokedAt == nil &&
					isTimeNear(time.Now(), 5*time.Second)(record.CreatedAt)
			},
		},
		{
			name: "storage error is returned",
			recordMatcher: func(record *types.RefreshToken) bool {
				return record != nil
			},
			wantErr: errRefreshTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newRefreshTokenTestFixture()
			fixture.repo.On("StoreRefreshToken", ctx, mock.MatchedBy(tt.recordMatcher)).Return(tt.wantErr)

			service := fixture.newService()
			err := service.StoreInitialRefreshToken(ctx, testRefreshToken, testSessionID, expiresAt)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			fixture.assertExpectations(t)
		})
	}
}

func TestHashRefreshToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		token         string
		want          string
		otherToken    string
		wantDifferent bool
	}{
		{
			name:  "known token",
			token: testRefreshToken,
			want:  "0eb17643d4e9261163783a420859c92c7d212fa9624106a12b510afbec266120",
		},
		{
			name:  "empty token",
			token: "",
			want:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:          "different_tokens_produce_different_hashes",
			token:         "token-a",
			otherToken:    "token-b",
			wantDifferent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.otherToken != "" {
				require.Equal(t, tt.wantDifferent, HashRefreshToken(tt.token) != HashRefreshToken(tt.otherToken))
				return
			}

			require.Equal(t, tt.want, HashRefreshToken(tt.token))
			require.Equal(t, HashRefreshToken(tt.token), HashRefreshToken(tt.token))
		})
	}
}

func TestRefreshTokenService_RevokeRefreshToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		token       string
		getRecord   *types.RefreshToken
		getErr      error
		revokeErr   error
		wantErr     error
		wantLog     string
		wantRevoked bool
	}{
		{
			name:        "revokes an existing token by its hash",
			token:       testRefreshToken,
			getRecord:   activeRefreshRecord(time.Now().Add(time.Hour)),
			wantRevoked: true,
		},
		{
			name:    "rejects an empty token",
			token:   "",
			wantErr: constants.ErrRefreshTokenIsRequired,
		},
		{
			name:    "returns not found when the token is unknown",
			token:   testRefreshToken,
			getErr:  constants.ErrRefreshTokenNotFound,
			wantErr: constants.ErrRefreshTokenNotFound,
		},
		{
			name:    "returns the lookup error",
			token:   testRefreshToken,
			getErr:  errRefreshTest,
			wantErr: errRefreshTest,
			wantLog: "refresh token lookup failed",
		},
		{
			name:        "returns the revoke error",
			token:       testRefreshToken,
			getRecord:   activeRefreshRecord(time.Now().Add(time.Hour)),
			revokeErr:   errRefreshTest,
			wantErr:     errRefreshTest,
			wantLog:     "failed to revoke refresh token",
			wantRevoked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			fixture := newRefreshTokenTestFixture()

			if tt.token == "" {
				err := fixture.newService().RevokeRefreshToken(ctx, tt.token)
				require.ErrorIs(t, err, tt.wantErr)
				fixture.repo.AssertNotCalled(t, "GetRefreshToken", mock.Anything, mock.Anything)
				fixture.repo.AssertNotCalled(t, "RevokeRefreshToken", mock.Anything, mock.Anything)
				fixture.assertExpectations(t)
				return
			}

			tokenHash := HashRefreshToken(tt.token)
			fixture.repo.On("GetRefreshToken", ctx, tokenHash).Return(tt.getRecord, tt.getErr)
			if tt.wantRevoked {
				fixture.repo.On("RevokeRefreshToken", ctx, tokenHash).Return(tt.revokeErr)
			}

			err := fixture.newService().RevokeRefreshToken(ctx, tt.token)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tt.wantRevoked {
				fixture.repo.AssertCalled(t, "RevokeRefreshToken", ctx, tokenHash)
			} else {
				fixture.repo.AssertNotCalled(t, "RevokeRefreshToken", mock.Anything, mock.Anything)
			}

			if tt.wantLog != "" {
				fixture.assertLogged(t, tt.wantLog)
			}

			fixture.assertExpectations(t)
		})
	}
}

func TestRefreshTokenService_CleanupExpiredTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantErr error
	}{
		{
			name: "delegates to the repository",
		},
		{
			name:    "returns the repository error",
			wantErr: errRefreshTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newRefreshTokenTestFixture()
			fixture.repo.On("CleanupExpiredTokens", mock.Anything).Return(tt.wantErr)

			err := fixture.newService().CleanupExpiredTokens(context.Background())

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			fixture.assertExpectations(t)
		})
	}
}

func expectReuseAfterGrace(t *testing.T, fixture *refreshTokenTestFixture, ctx context.Context) {
	t.Helper()

	fixture.tokenSvc.On("ValidateRefreshToken", ctx, testRefreshToken).Return(validRefreshClaims(), nil)
	fixture.repo.On("GetRefreshToken", ctx, HashRefreshToken(testRefreshToken)).Return(revokedRefreshRecord(time.Now().Add(-time.Minute)), nil)
	fixture.repo.On("RevokeAllSessionTokens", ctx, testSessionID).Return(nil)
	fixture.sessionSvc.On("Delete", ctx, testSessionID).Return(nil)
}
