package usecases

import (
	"context"
	"time"

	inttests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
)

type emailPasswordTestFixture struct {
	globalConfig    *models.Config
	pluginConfig    types.EmailPasswordPluginConfig
	userSvc         *inttests.MockUserService
	accountSvc      *inttests.MockAccountService
	sessionSvc      *inttests.MockSessionService
	verificationSvc *inttests.MockVerificationService
	tokenSvc        *inttests.MockTokenService
	passwordSvc     *inttests.MockPasswordService
	mailerSvc       *inttests.MockMailerService
	logger          *inttests.MockLogger
	eventBus        *inttests.MockEventBus
}

func newEmailPasswordTestFixture() *emailPasswordTestFixture {
	return &emailPasswordTestFixture{
		globalConfig: &models.Config{
			BaseURL:  "http://localhost",
			BasePath: "/auth",
			AppName:  "TestApp",
			Session:  models.SessionConfig{ExpiresIn: time.Hour},
		},
		pluginConfig: types.EmailPasswordPluginConfig{
			MinPasswordLength:           8,
			MaxPasswordLength:           128,
			RequireEmailVerification:    true,
			AutoSignIn:                  true,
			EmailVerificationExpiresIn:  24 * time.Hour,
			PasswordResetExpiresIn:      time.Hour,
			RequestEmailChangeExpiresIn: time.Hour,
		},
		userSvc:         &inttests.MockUserService{},
		accountSvc:      &inttests.MockAccountService{},
		sessionSvc:      &inttests.MockSessionService{},
		verificationSvc: &inttests.MockVerificationService{},
		tokenSvc:        &inttests.MockTokenService{},
		passwordSvc:     &inttests.MockPasswordService{},
		mailerSvc:       &inttests.MockMailerService{},
		logger:          &inttests.MockLogger{},
		eventBus:        &inttests.MockEventBus{},
	}
}

func (f *emailPasswordTestFixture) signUpUseCase() SignUpUseCase {
	return NewSignUpUseCase(f.globalConfig, f.pluginConfig, f.logger, f.userSvc, f.accountSvc, f.sessionSvc, f.tokenSvc, f.passwordSvc, f.eventBus)
}

func (f *emailPasswordTestFixture) signInUseCase() SignInUseCase {
	return NewSignInUseCase(f.globalConfig, f.pluginConfig, f.logger, f.userSvc, f.accountSvc, f.sessionSvc, f.tokenSvc, f.passwordSvc, f.eventBus)
}

func (f *emailPasswordTestFixture) verifyEmailUseCase() VerifyEmailUseCase {
	return NewVerifyEmailUseCase(f.pluginConfig, f.logger, f.userSvc, f.accountSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc, f.eventBus)
}

func (f *emailPasswordTestFixture) sendEmailVerificationUseCase() SendEmailVerificationUseCase {
	return NewSendEmailVerificationUseCase(f.globalConfig, f.pluginConfig, f.logger, f.userSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc)
}

func (f *emailPasswordTestFixture) requestPasswordResetUseCase() RequestPasswordResetUseCase {
	return NewRequestPasswordResetUseCase(f.logger, f.globalConfig, f.pluginConfig, f.userSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc)
}

func (f *emailPasswordTestFixture) changePasswordUseCase() ChangePasswordUseCase {
	return NewChangePasswordUseCase(f.logger, f.pluginConfig, f.userSvc, f.accountSvc, f.verificationSvc, f.tokenSvc, f.passwordSvc, f.mailerSvc, f.eventBus)
}

func (f *emailPasswordTestFixture) requestEmailChangeUseCase() RequestEmailChangeUseCase {
	return NewRequestEmailChangeUseCase(f.logger, f.globalConfig, f.pluginConfig, f.userSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc)
}

func testRequestContext() context.Context {
	return models.SetRequestContext(context.Background(), &models.RequestContext{Values: map[string]any{}})
}
