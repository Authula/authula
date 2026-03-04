package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
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

func TestExchangeUseCase_Success(t *testing.T) {
	h := newExchangeUseCaseTestHarness()
	ip := "127.0.0.1"
	ua := "TestAgent/1.0"

	h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
	h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
	h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()
	h.userSvc.On("GetByID", mock.Anything, h.user.ID).Return(h.user, nil).Once()
	h.verificationSvc.On("Delete", mock.Anything, h.verification.ID).Return(nil).Once()
	h.tokenSvc.On("Generate").Return("generated-session-token", nil).Once()
	h.tokenSvc.On("Hash", "generated-session-token").Return("hashed-generated-session-token").Once()
	h.sessionSvc.On("Create", mock.Anything, h.user.ID, "hashed-generated-session-token", &ip, &ua, h.useCase.GlobalConfig.Session.ExpiresIn).
		Return(&models.Session{ID: "session-123", UserID: h.user.ID}, nil).Once()

	result, err := h.useCase.Exchange(context.Background(), "raw-token", &ip, &ua)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "session-123", result.Session.ID)
	assert.Equal(t, h.user.ID, result.User.ID)
	assert.Equal(t, "generated-session-token", result.SessionToken)
}

func TestExchangeUseCase_GetByTokenError(t *testing.T) {
	h := newExchangeUseCaseTestHarness()

	h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
	h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(nil, errors.New("lookup failed")).Once()

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)

	assert.EqualError(t, err, "lookup failed")
}

func TestExchangeUseCase_InvalidOrExpiredToken(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		h := newExchangeUseCaseTestHarness()
		h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
		h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(nil, nil).Once()

		_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)
		assert.EqualError(t, err, "invalid or expired token")
	})

	t.Run("expired", func(t *testing.T) {
		h := newExchangeUseCaseTestHarness()
		h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
		h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
		h.verificationSvc.On("IsExpired", h.verification).Return(true).Once()

		_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)
		assert.EqualError(t, err, "invalid or expired token")
	})
}

func TestExchangeUseCase_InvalidTokenType(t *testing.T) {
	h := newExchangeUseCaseTestHarness()
	h.verification.Type = models.TypeEmailVerification

	h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
	h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
	h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)

	assert.EqualError(t, err, "invalid token type")
}

func TestExchangeUseCase_RejectsOldSignInTokens(t *testing.T) {
	h := newExchangeUseCaseTestHarness()
	h.verification.Type = models.TypeMagicLinkSignInRequest

	h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
	h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
	h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)

	assert.EqualError(t, err, "invalid token type")
}

func TestExchangeUseCase_UserLookupError(t *testing.T) {
	h := newExchangeUseCaseTestHarness()

	h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
	h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
	h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()
	h.userSvc.On("GetByID", mock.Anything, h.user.ID).Return(nil, errors.New("user lookup failed")).Once()

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)

	assert.EqualError(t, err, "user lookup failed")
}

func TestExchangeUseCase_UserNotFound(t *testing.T) {
	h := newExchangeUseCaseTestHarness()

	h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
	h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
	h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()
	h.userSvc.On("GetByID", mock.Anything, h.user.ID).Return(nil, nil).Once()

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)

	assert.EqualError(t, err, "user not found")
}

func TestExchangeUseCase_DeleteVerificationFails(t *testing.T) {
	h := newExchangeUseCaseTestHarness()

	h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
	h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
	h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()
	h.userSvc.On("GetByID", mock.Anything, h.user.ID).Return(h.user, nil).Once()
	h.verificationSvc.On("Delete", mock.Anything, h.verification.ID).Return(errors.New("delete failed")).Once()

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)

	assert.EqualError(t, err, "delete failed")
}

func TestExchangeUseCase_SessionCreationFails(t *testing.T) {
	h := newExchangeUseCaseTestHarness()

	h.tokenSvc.On("Hash", "raw-token").Return("hashed-raw-token").Once()
	h.verificationSvc.On("GetByToken", mock.Anything, "hashed-raw-token").Return(h.verification, nil).Once()
	h.verificationSvc.On("IsExpired", h.verification).Return(false).Once()
	h.userSvc.On("GetByID", mock.Anything, h.user.ID).Return(h.user, nil).Once()
	h.verificationSvc.On("Delete", mock.Anything, h.verification.ID).Return(nil).Once()
	h.tokenSvc.On("Generate").Return("generated-session-token", nil).Once()
	h.tokenSvc.On("Hash", "generated-session-token").Return("hashed-generated-session-token").Once()
	h.sessionSvc.On("Create", mock.Anything, h.user.ID, "hashed-generated-session-token", (*string)(nil), (*string)(nil), h.useCase.GlobalConfig.Session.ExpiresIn).
		Return(nil, errors.New("session failed")).Once()

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)

	assert.EqualError(t, err, "session failed")
}
