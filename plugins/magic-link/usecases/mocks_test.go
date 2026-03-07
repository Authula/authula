package usecases

import (
	"time"

	inttests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
)

type mockUserService = inttests.MockUserService
type mockAccountService = inttests.MockAccountService
type mockSessionService = inttests.MockSessionService
type mockVerificationService = inttests.MockVerificationService
type mockTokenService = inttests.MockTokenService
type mockMailerService = inttests.MockMailerService
type mockLogger = inttests.MockLogger

func newMockUserService() *mockUserService {
	return &mockUserService{}
}

func newMockAccountService() *mockAccountService {
	return &mockAccountService{}
}

func newMockTokenService() *mockTokenService {
	return &mockTokenService{}
}

func newMockVerificationService() *mockVerificationService {
	return &mockVerificationService{}
}

func newMockMailerService() *mockMailerService {
	return &mockMailerService{}
}

func newMockSessionService() *mockSessionService {
	return &mockSessionService{}
}

func newSignInTestUseCase() (*SignInUseCaseImpl, *mockUserService, *mockAccountService, *mockTokenService, *mockVerificationService, *mockMailerService) {
	userSvc := newMockUserService()
	accountSvc := newMockAccountService()
	tokenSvc := newMockTokenService()
	verificationSvc := newMockVerificationService()
	mailerSvc := newMockMailerService()

	uc := &SignInUseCaseImpl{
		GlobalConfig: &models.Config{
			BaseURL:  "http://localhost",
			BasePath: "/auth",
			AppName:  "GoBetterAuth",
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

func newVerifyTestUseCase() (*VerifyUseCaseImpl, *mockUserService, *mockVerificationService, *mockTokenService) {
	userSvc := newMockUserService()
	verificationSvc := newMockVerificationService()
	tokenSvc := newMockTokenService()

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
