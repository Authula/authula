package usecases

import (
	"context"
	"testing"
	"time"

	inttests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
)

type mockUserService = inttests.MockUserService
type mockAccountService = inttests.MockAccountService
type mockTokenService = inttests.MockTokenService
type mockVerificationService = inttests.MockVerificationService
type mockMailerService = inttests.MockMailerService
type mockSessionService = inttests.MockSessionService
type mockLogger = inttests.MockLogger

func newMockUserService(t *testing.T) *mockUserService {
	t.Helper()
	return inttests.NewMockUserService(t)
}

func newMockAccountService(t *testing.T) *mockAccountService {
	t.Helper()
	return inttests.NewMockAccountService(t)
}

func newMockTokenService(t *testing.T) *mockTokenService {
	t.Helper()
	return inttests.NewMockTokenService(t)
}

func newMockVerificationService(t *testing.T) *mockVerificationService {
	t.Helper()
	return inttests.NewMockVerificationService(t)
}

func newMockMailerService(t *testing.T) *mockMailerService {
	t.Helper()
	return inttests.NewMockMailerService(t)
}

func newMockSessionService(t *testing.T) *mockSessionService {
	t.Helper()
	return inttests.NewMockSessionService(t)
}

func newSignInTestUseCase(t *testing.T) (*SignInUseCaseImpl, *mockUserService, *mockAccountService, *mockTokenService, *mockVerificationService, *mockMailerService) {
	t.Helper()

	userSvc := newMockUserService(t)
	accountSvc := newMockAccountService(t)
	tokenSvc := newMockTokenService(t)
	verificationSvc := newMockVerificationService(t)
	mailerSvc := newMockMailerService(t)

	userSvc.GetByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
		return &models.User{ID: "user-1", Email: email}, nil
	}
	userSvc.CreateFn = func(ctx context.Context, name, email string, emailVerified bool, image *string) (*models.User, error) {
		return &models.User{ID: "user-1", Name: name, Email: email, EmailVerified: emailVerified, Image: image}, nil
	}

	accountSvc.CreateFn = func(ctx context.Context, userID, accountID, providerID string, password *string) (*models.Account, error) {
		return &models.Account{ID: "account-1", UserID: userID}, nil
	}

	tokenSvc.GenerateFn = func() (string, error) {
		return "test-token-123", nil
	}
	tokenSvc.HashFn = func(token string) string {
		return "hashed-" + token
	}

	verificationSvc.CreateFn = func(ctx context.Context, userID, hashedToken string, vType models.VerificationType, value string, expiry time.Duration) (*models.Verification, error) {
		return &models.Verification{ID: "verif-1", UserID: &userID, Type: vType}, nil
	}

	mailerSvc.SendEmailFn = func(ctx context.Context, to, subject, text, html string) error {
		return nil
	}

	uc := &SignInUseCaseImpl{
		GlobalConfig: &models.Config{
			BaseURL:  "http://localhost",
			BasePath: "/auth",
		},
		PluginConfig: &types.MagicLinkPluginConfig{
			ExpiresIn: 15 * time.Minute,
		},
		Logger:              &mockLogger{},
		UserService:         userSvc,
		AccountService:      accountSvc,
		TokenService:        tokenSvc,
		VerificationService: verificationSvc,
		MailerService:       mailerSvc,
	}

	return uc, userSvc, accountSvc, tokenSvc, verificationSvc, mailerSvc
}

func newVerifyTestUseCase(t *testing.T) (*VerifyUseCaseImpl, *mockUserService, *mockVerificationService, *mockTokenService) {
	t.Helper()

	userSvc := newMockUserService(t)
	verificationSvc := newMockVerificationService(t)
	tokenSvc := newMockTokenService(t)

	userSvc.GetByIDFn = func(ctx context.Context, id string) (*models.User, error) {
		return &models.User{ID: id, Email: "test@example.com"}, nil
	}
	userSvc.UpdateFieldsFn = func(ctx context.Context, id string, fields map[string]any) error {
		return nil
	}

	tokenSvc.GenerateFn = func() (string, error) {
		return "new-exchange-code", nil
	}
	tokenSvc.HashFn = func(token string) string {
		return "hashed-" + token
	}

	userID := "user-1"
	verificationSvc.GetByTokenFn = func(ctx context.Context, hashedToken string) (*models.Verification, error) {
		return &models.Verification{
			ID:         "verif-1",
			UserID:     &userID,
			Type:       models.TypeMagicLinkSignInRequest,
			Identifier: "test@example.com",
		}, nil
	}
	verificationSvc.IsExpiredFn = func(verif *models.Verification) bool {
		return false
	}
	verificationSvc.DeleteFn = func(ctx context.Context, id string) error {
		return nil
	}
	verificationSvc.CreateFn = func(ctx context.Context, userID, hashedToken string, vType models.VerificationType, value string, expiry time.Duration) (*models.Verification, error) {
		return &models.Verification{ID: "verif-2", UserID: &userID, Type: vType, Identifier: value}, nil
	}

	uc := &VerifyUseCaseImpl{
		GlobalConfig: &models.Config{
			Session: models.SessionConfig{ExpiresIn: 24 * time.Hour},
		},
		PluginConfig:        &types.MagicLinkPluginConfig{ExpiresIn: 15 * time.Minute},
		Logger:              &mockLogger{},
		UserService:         userSvc,
		VerificationService: verificationSvc,
		TokenService:        tokenSvc,
	}

	return uc, userSvc, verificationSvc, tokenSvc
}
