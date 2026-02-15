package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
)

func TestExchangeUseCase_Success(t *testing.T) {
	h := newExchangeUseCaseTestHarness(t)
	ip := "127.0.0.1"
	ua := "TestAgent/1.0"

	verificationDeleted := false
	h.verificationSvc.DeleteFn = func(ctx context.Context, id string) error {
		verificationDeleted = true
		if id != h.verification.ID {
			t.Fatalf("expected verification id %s, got %s", h.verification.ID, id)
		}
		return nil
	}

	var capturedHashed string
	h.sessionSvc.CreateFn = func(ctx context.Context, userID, hashedToken string, ipAddress, userAgent *string, maxAge time.Duration) (*models.Session, error) {
		if !verificationDeleted {
			t.Fatalf("expected verification to be deleted before session creation")
		}
		capturedHashed = hashedToken
		if ipAddress == nil || *ipAddress != ip {
			t.Fatalf("expected ip %s, got %v", ip, ipAddress)
		}
		if userAgent == nil || *userAgent != ua {
			t.Fatalf("expected user agent %s, got %v", ua, userAgent)
		}
		if maxAge != h.useCase.GlobalConfig.Session.ExpiresIn {
			t.Fatalf("expected maxAge %s, got %s", h.useCase.GlobalConfig.Session.ExpiresIn, maxAge)
		}
		return &models.Session{ID: "session-123", UserID: userID}, nil
	}

	result, err := h.useCase.Exchange(context.Background(), "raw-token", &ip, &ua)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Session == nil || result.Session.ID != "session-123" {
		t.Fatalf("expected session id session-123, got %v", result.Session)
	}
	if result.User == nil || result.User.ID != h.user.ID {
		t.Fatalf("expected user %s, got %v", h.user.ID, result.User)
	}
	if capturedHashed != "hashed-generated-session-token" {
		t.Fatalf("expected hashed token hashed-generated-session-token, got %s", capturedHashed)
	}
	if result.SessionToken != "generated-session-token" {
		t.Fatalf("expected generated session token, got %s", result.SessionToken)
	}
}

func TestExchangeUseCase_GetByTokenError(t *testing.T) {
	h := newExchangeUseCaseTestHarness(t)
	h.verificationSvc.GetByTokenFn = func(ctx context.Context, hashedToken string) (*models.Verification, error) {
		return nil, errors.New("lookup failed")
	}

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)
	if err == nil || err.Error() != "lookup failed" {
		t.Fatalf("expected lookup failure, got %v", err)
	}
}

func TestExchangeUseCase_InvalidOrExpiredToken(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		h := newExchangeUseCaseTestHarness(t)
		h.verificationSvc.GetByTokenFn = func(ctx context.Context, hashedToken string) (*models.Verification, error) {
			return nil, nil
		}

		_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)
		if err == nil || err.Error() != "invalid or expired token" {
			t.Fatalf("expected invalid token error, got %v", err)
		}
	})

	t.Run("expired", func(t *testing.T) {
		h := newExchangeUseCaseTestHarness(t)
		h.verificationSvc.IsExpiredFn = func(verif *models.Verification) bool {
			return true
		}

		_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)
		if err == nil || err.Error() != "invalid or expired token" {
			t.Fatalf("expected invalid token error, got %v", err)
		}
	})
}

func TestExchangeUseCase_InvalidTokenType(t *testing.T) {
	h := newExchangeUseCaseTestHarness(t)
	h.verification.Type = models.TypeEmailVerification

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)
	if err == nil || err.Error() != "invalid token type" {
		t.Fatalf("expected invalid token type error, got %v", err)
	}
}

func TestExchangeUseCase_RejectsOldSignInTokens(t *testing.T) {
	h := newExchangeUseCaseTestHarness(t)
	h.verification.Type = models.TypeMagicLinkSignInRequest

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)
	if err == nil || err.Error() != "invalid token type" {
		t.Fatalf("expected invalid token type error for old sign-in token, got %v", err)
	}
}

func TestExchangeUseCase_UserLookupError(t *testing.T) {
	h := newExchangeUseCaseTestHarness(t)
	h.userSvc.GetByIDFn = func(ctx context.Context, id string) (*models.User, error) {
		return nil, errors.New("user lookup failed")
	}

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)
	if err == nil || err.Error() != "user lookup failed" {
		t.Fatalf("expected user lookup error, got %v", err)
	}
}

func TestExchangeUseCase_UserNotFound(t *testing.T) {
	h := newExchangeUseCaseTestHarness(t)
	h.userSvc.GetByIDFn = func(ctx context.Context, id string) (*models.User, error) {
		return nil, nil
	}

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)
	if err == nil || err.Error() != "user not found" {
		t.Fatalf("expected user not found error, got %v", err)
	}
}

func TestExchangeUseCase_DeleteVerificationFails(t *testing.T) {
	h := newExchangeUseCaseTestHarness(t)
	h.verificationSvc.DeleteFn = func(ctx context.Context, id string) error {
		return errors.New("delete failed")
	}

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)
	if err == nil || err.Error() != "delete failed" {
		t.Fatalf("expected delete failure, got %v", err)
	}
}

func TestExchangeUseCase_SessionCreationFails(t *testing.T) {
	h := newExchangeUseCaseTestHarness(t)
	deleteCalled := false
	h.verificationSvc.DeleteFn = func(ctx context.Context, id string) error {
		deleteCalled = true
		return nil
	}
	h.sessionSvc.CreateFn = func(ctx context.Context, userID, hashedToken string, ipAddress, userAgent *string, maxAge time.Duration) (*models.Session, error) {
		if !deleteCalled {
			t.Fatalf("expected verification deletion before session creation")
		}
		return nil, errors.New("session failed")
	}

	_, err := h.useCase.Exchange(context.Background(), "raw-token", nil, nil)
	if err == nil || err.Error() != "session failed" {
		t.Fatalf("expected session failure, got %v", err)
	}
}

type exchangeUseCaseTestHarness struct {
	useCase         *ExchangeUseCaseImpl
	userSvc         *mockUserService
	sessionSvc      *mockSessionService
	verificationSvc *mockVerificationService
	tokenSvc        *mockTokenService
	verification    *models.Verification
	user            *models.User
}

func newExchangeUseCaseTestHarness(t *testing.T) *exchangeUseCaseTestHarness {
	t.Helper()

	h := &exchangeUseCaseTestHarness{
		userSvc:         newMockUserService(t),
		sessionSvc:      newMockSessionService(t),
		verificationSvc: newMockVerificationService(t),
		tokenSvc:        newMockTokenService(t),
		user:            &models.User{ID: "user-123", Email: "user@example.com"},
	}

	userID := h.user.ID
	h.verification = &models.Verification{
		ID:     "verif-123",
		UserID: &userID,
		Type:   models.TypeMagicLinkExchangeCode,
	}

	h.tokenSvc.HashFn = func(token string) string {
		return "hashed-" + token
	}
	h.tokenSvc.GenerateFn = func() (string, error) {
		return "generated-session-token", nil
	}

	h.verificationSvc.GetByTokenFn = func(ctx context.Context, hashedToken string) (*models.Verification, error) {
		return h.verification, nil
	}
	h.verificationSvc.IsExpiredFn = func(verif *models.Verification) bool {
		return false
	}
	h.verificationSvc.DeleteFn = func(ctx context.Context, id string) error {
		return nil
	}

	h.userSvc.GetByIDFn = func(ctx context.Context, id string) (*models.User, error) {
		return h.user, nil
	}
	h.userSvc.UpdateFieldsFn = func(ctx context.Context, id string, fields map[string]any) error {
		if verified, ok := fields["email_verified"].(bool); ok {
			h.user.EmailVerified = verified
		}
		return nil
	}

	h.sessionSvc.CreateFn = func(ctx context.Context, userID, hashedToken string, ipAddress, userAgent *string, maxAge time.Duration) (*models.Session, error) {
		return &models.Session{ID: "session-default", UserID: userID}, nil
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
