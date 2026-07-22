package usecases

import (
	"context"
	"time"

	emailtmpl "github.com/Authula/authula/core/email/template"
	inttests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/services"
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
	tmplMgr         *emailtmpl.Manager
	hooksExecutor   *services.ServiceHookExecutor
}

func newEmailPasswordTestFixture() *emailPasswordTestFixture {
	tmplMgr := emailtmpl.NewManager()
	_ = tmplMgr.Register(emailtmpl.Definition{
		Name:    "verify_email",
		Subject: "Verify your email",
		Text:    "Verify your email by clicking the following link: {{.VerificationLink}}.",
		HTML:    "<p>Hello {{.UserEmail}}, verify: {{.VerificationLink}}</p>",
	})
	_ = tmplMgr.Register(emailtmpl.Definition{
		Name:    "password_reset",
		Subject: "Reset Your Password",
		Text:    "Reset link: {{.ResetLink}}",
		HTML:    "<p>Reset: {{.ResetLink}}</p>",
	})
	_ = tmplMgr.Register(emailtmpl.Definition{
		Name:    "password_changed",
		Subject: "Your password has been changed",
		Text:    "Password changed",
		HTML:    "<p>Password changed for {{.UserEmail}}</p>",
	})
	_ = tmplMgr.Register(emailtmpl.Definition{
		Name:    "email_change_request",
		Subject: "Confirm Your Email Change",
		Text:    "Change to {{.NewEmail}}: {{.ChangeLink}}",
		HTML:    "<p>Change to {{.NewEmail}}: {{.ChangeLink}}</p>",
	})
	_ = tmplMgr.Register(emailtmpl.Definition{
		Name:    "email_changed_notification",
		Subject: "Your email has been changed",
		Text:    "Changed from {{.OldEmail}} to {{.NewEmail}}",
		HTML:    "<p>Changed from {{.OldEmail}} to {{.NewEmail}}</p>",
	})

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
		tmplMgr:         tmplMgr,
	}
}

func (f *emailPasswordTestFixture) signUpUseCase() SignUpUseCase {
	return NewSignUpUseCase(f.globalConfig, f.pluginConfig, f.logger, f.userSvc, f.accountSvc, f.sessionSvc, f.tokenSvc, f.passwordSvc, f.eventBus, f.hooksExecutor)
}

func (f *emailPasswordTestFixture) signInUseCase() SignInUseCase {
	return NewSignInUseCase(f.globalConfig, f.pluginConfig, f.logger, f.userSvc, f.accountSvc, f.sessionSvc, f.tokenSvc, f.passwordSvc, f.eventBus, f.hooksExecutor)
}

func (f *emailPasswordTestFixture) verifyEmailUseCase() VerifyEmailUseCase {
	return NewVerifyEmailUseCase(f.globalConfig, f.pluginConfig, f.logger, f.userSvc, f.accountSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc, f.eventBus, f.tmplMgr, f.hooksExecutor)
}

func (f *emailPasswordTestFixture) sendEmailVerificationUseCase() SendEmailVerificationUseCase {
	return NewSendEmailVerificationUseCase(f.globalConfig, f.pluginConfig, f.logger, f.userSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc, f.tmplMgr)
}

func (f *emailPasswordTestFixture) requestPasswordResetUseCase() RequestPasswordResetUseCase {
	return NewRequestPasswordResetUseCase(f.logger, f.globalConfig, f.pluginConfig, f.userSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc, f.tmplMgr, f.hooksExecutor)
}

func (f *emailPasswordTestFixture) changePasswordUseCase() ChangePasswordUseCase {
	return NewChangePasswordUseCase(f.globalConfig, f.logger, f.pluginConfig, f.userSvc, f.accountSvc, f.verificationSvc, f.tokenSvc, f.passwordSvc, f.mailerSvc, f.eventBus, f.tmplMgr, f.hooksExecutor)
}

func (f *emailPasswordTestFixture) requestEmailChangeUseCase() RequestEmailChangeUseCase {
	return NewRequestEmailChangeUseCase(f.logger, f.globalConfig, f.pluginConfig, f.userSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc, f.tmplMgr, f.hooksExecutor)
}

func testRequestContext() context.Context {
	return models.SetRequestContext(context.Background(), &models.RequestContext{Values: map[string]any{}})
}
