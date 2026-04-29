package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/magic-link/types"
)

type exchangeUseCaseTestHarness struct {
	useCase         *ExchangeUseCaseImpl
	userSvc         *mockUserService
	sessionSvc      *mockSessionService
	verificationSvc *mockVerificationService
	tokenSvc        *mockTokenService
	verification    *models.Verification
	user            *models.User
}

func newExchangeUseCaseTestHarness() *exchangeUseCaseTestHarness {
	h := &exchangeUseCaseTestHarness{
		userSvc:         newMockUserService(),
		sessionSvc:      newMockSessionService(),
		verificationSvc: newMockVerificationService(),
		tokenSvc:        newMockTokenService(),
		user:            &models.User{ID: "user-123", Email: "user@example.com"},
	}

	userID := h.user.ID
	h.verification = &models.Verification{
		ID:     "verif-123",
		UserID: &userID,
		Type:   models.TypeMagicLinkExchangeCode,
	}

	h.useCase = &ExchangeUseCaseImpl{
		GlobalConfig: &models.Config{
			Session: models.SessionConfig{ExpiresIn: 30 * time.Minute},
		},
		PluginConfig:        &types.MagicLinkPluginConfig{},
		Logger:              &mockLogger{},
		UserService:         h.userSvc,
		SessionService:      h.sessionSvc,
		VerificationService: h.verificationSvc,
		TokenService:        h.tokenSvc,
	}

	return h
}

func TestExchangeUseCase(t *testing.T) {
	ip := "127.0.0.1"
	ua := "TestAgent/1.0"

	testCases := []struct {
		name         string
		token        string
		ipAddress    *string
		userAgent    *string
		setup        func(t *testing.T, h *exchangeUseCaseTestHarness)
		assertResult func(t *testing.T, result *types.ExchangeResult, err error)
	}{
		{
			name:      "success returns a session and token",
			token:     "raw-token",
			ipAddress: &ip,
			userAgent: &ua,
			setup: func(t *testing.T, h *exchangeUseCaseTestHarness) {
				h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
				h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
				h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()
				h.userSvc.On("GetByID", mock.Anything, h.user.ID).Return(h.user, nil).Once()
				h.verificationSvc.On("Delete", mock.Anything, h.verification.ID).Return(nil).Once()
				h.tokenSvc.On("Generate").Return("generated-session-token", nil).Once()
				h.tokenSvc.On("Hash", "generated-session-token").Return("hashed-generated-session-token").Once()
				h.sessionSvc.On("Create", mock.Anything, h.user.ID, "hashed-generated-session-token", &ip, &ua, h.useCase.GlobalConfig.Session.ExpiresIn).
					Return(&models.Session{ID: "session-123", UserID: h.user.ID}, nil).Once()

				t.Cleanup(func() {
					h.userSvc.AssertExpectations(t)
					h.verificationSvc.AssertExpectations(t)
					h.tokenSvc.AssertExpectations(t)
					h.sessionSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.ExchangeResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "session-123", result.Session.ID)
				assert.Equal(t, "user-123", result.User.ID)
				assert.Equal(t, "generated-session-token", result.SessionToken)
			},
		},
		{
			name:  "verification lookup error returns the error",
			token: "raw-token",
			setup: func(t *testing.T, h *exchangeUseCaseTestHarness) {
				h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
				h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(nil, errors.New("lookup failed")).Once()

				t.Cleanup(func() {
					h.verificationSvc.AssertExpectations(t)
					h.tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.ExchangeResult, err error) {
				assert.EqualError(t, err, "lookup failed")
				assert.Nil(t, result)
			},
		},
		{
			name:  "missing verification is rejected",
			token: "raw-token",
			setup: func(t *testing.T, h *exchangeUseCaseTestHarness) {
				h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
				h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(nil, nil).Once()

				t.Cleanup(func() {
					h.verificationSvc.AssertExpectations(t)
					h.tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.ExchangeResult, err error) {
				assert.EqualError(t, err, "invalid or expired token")
				assert.Nil(t, result)
			},
		},
		{
			name:  "expired verification is rejected",
			token: "raw-token",
			setup: func(t *testing.T, h *exchangeUseCaseTestHarness) {
				h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
				h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
				h.verificationSvc.On("IsExpired", h.verification).Return(true).Once()

				t.Cleanup(func() {
					h.verificationSvc.AssertExpectations(t)
					h.tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.ExchangeResult, err error) {
				assert.EqualError(t, err, "invalid or expired token")
				assert.Nil(t, result)
			},
		},
		{
			name:  "invalid token type is rejected",
			token: "raw-token",
			setup: func(t *testing.T, h *exchangeUseCaseTestHarness) {
				h.verification.Type = models.TypeEmailVerification
				h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
				h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
				h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()

				t.Cleanup(func() {
					h.verificationSvc.AssertExpectations(t)
					h.tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.ExchangeResult, err error) {
				assert.EqualError(t, err, "invalid token type")
				assert.Nil(t, result)
			},
		},
		{
			name:  "old sign in tokens are rejected",
			token: "raw-token",
			setup: func(t *testing.T, h *exchangeUseCaseTestHarness) {
				h.verification.Type = models.TypeMagicLinkSignInRequest
				h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
				h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
				h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()

				t.Cleanup(func() {
					h.verificationSvc.AssertExpectations(t)
					h.tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.ExchangeResult, err error) {
				assert.EqualError(t, err, "invalid token type")
				assert.Nil(t, result)
			},
		},
		{
			name:  "user lookup error returns the error",
			token: "raw-token",
			setup: func(t *testing.T, h *exchangeUseCaseTestHarness) {
				h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
				h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
				h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()
				h.userSvc.On("GetByID", mock.Anything, h.user.ID).Return(nil, errors.New("user lookup failed")).Once()

				t.Cleanup(func() {
					h.userSvc.AssertExpectations(t)
					h.verificationSvc.AssertExpectations(t)
					h.tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.ExchangeResult, err error) {
				assert.EqualError(t, err, "user lookup failed")
				assert.Nil(t, result)
			},
		},
		{
			name:  "missing user is rejected",
			token: "raw-token",
			setup: func(t *testing.T, h *exchangeUseCaseTestHarness) {
				h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
				h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
				h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()
				h.userSvc.On("GetByID", mock.Anything, h.user.ID).Return(nil, nil).Once()

				t.Cleanup(func() {
					h.userSvc.AssertExpectations(t)
					h.verificationSvc.AssertExpectations(t)
					h.tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.ExchangeResult, err error) {
				assert.EqualError(t, err, "user not found")
				assert.Nil(t, result)
			},
		},
		{
			name:  "verification deletion failure is returned",
			token: "raw-token",
			setup: func(t *testing.T, h *exchangeUseCaseTestHarness) {
				h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
				h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
				h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()
				h.userSvc.On("GetByID", mock.Anything, h.user.ID).Return(h.user, nil).Once()
				h.verificationSvc.On("Delete", mock.Anything, h.verification.ID).Return(errors.New("delete failed")).Once()

				t.Cleanup(func() {
					h.userSvc.AssertExpectations(t)
					h.verificationSvc.AssertExpectations(t)
					h.tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.ExchangeResult, err error) {
				assert.EqualError(t, err, "delete failed")
				assert.Nil(t, result)
			},
		},
		{
			name:  "session creation failure is returned",
			token: "raw-token",
			setup: func(t *testing.T, h *exchangeUseCaseTestHarness) {
				h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
				h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
				h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()
				h.userSvc.On("GetByID", mock.Anything, h.user.ID).Return(h.user, nil).Once()
				h.verificationSvc.On("Delete", mock.Anything, h.verification.ID).Return(nil).Once()
				h.tokenSvc.On("Generate").Return("generated-session-token", nil).Once()
				h.tokenSvc.On("Hash", "generated-session-token").Return("hashed-generated-session-token").Once()
				h.sessionSvc.On("Create", mock.Anything, h.user.ID, "hashed-generated-session-token", (*string)(nil), (*string)(nil), h.useCase.GlobalConfig.Session.ExpiresIn).
					Return(nil, errors.New("session failed")).Once()

				t.Cleanup(func() {
					h.userSvc.AssertExpectations(t)
					h.verificationSvc.AssertExpectations(t)
					h.tokenSvc.AssertExpectations(t)
					h.sessionSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.ExchangeResult, err error) {
				assert.EqualError(t, err, "session failed")
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			h := newExchangeUseCaseTestHarness()
			tt.setup(t, h)

			result, err := h.useCase.Exchange(context.Background(), tt.token, tt.ipAddress, tt.userAgent)
			tt.assertResult(t, result, err)
		})
	}
}
