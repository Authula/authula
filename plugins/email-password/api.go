package email_password

import (
	"context"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/handlers"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/usecases"
)

type API struct {
	useCases *usecases.UseCases
}

func (a *API) SignUp(
	ctx context.Context,
	name string,
	email string,
	password string,
	image *string,
	callbackURL *string,
	ipAddress *string,
	userAgent *string,
) (*types.SignUpResult, error) {
	return a.useCases.SignUpUseCase.SignUp(ctx, name, email, password, image, callbackURL, ipAddress, userAgent)
}

func (a *API) SignIn(
	ctx context.Context,
	email string,
	password string,
	callbackURL *string,
	ipAddress *string,
	userAgent *string,
) (*types.SignInResult, error) {
	return a.useCases.SignInUseCase.SignIn(ctx, email, password, callbackURL, ipAddress, userAgent)
}

func (a *API) VerifyEmail(ctx context.Context, tokenStr string) (models.VerificationType, error) {
	return a.useCases.VerifyEmailUseCase.VerifyEmail(ctx, tokenStr)
}

func (a *API) SendEmailVerification(ctx context.Context, email string, callbackURL *string) error {
	return a.useCases.SendEmailVerificationUseCase.Send(ctx, email, callbackURL)
}

func (a *API) RequestPasswordReset(ctx context.Context, email string, callbackURL *string) error {
	return a.useCases.RequestPasswordResetUseCase.RequestReset(ctx, email, callbackURL)
}

func (a *API) ChangePassword(ctx context.Context, tokenStr string, newPassword string) error {
	return a.useCases.ChangePasswordUseCase.ChangePassword(ctx, tokenStr, newPassword)
}

func (a *API) RequestEmailChange(ctx context.Context, userID string, newEmail string, callbackURL *string) error {
	return a.useCases.RequestEmailChangeUseCase.RequestChange(ctx, userID, newEmail, callbackURL)
}

func BuildAPI(plugin *EmailPasswordPlugin) *API {
	useCases := BuildUseCases(plugin)
	return &API{useCases: useCases}
}

// Routes returns all routes for the email/password plugin
func Routes(plugin *EmailPasswordPlugin) []models.Route {
	useCases := BuildUseCases(plugin)

	signUpHandler := &handlers.SignUpHandler{
		Logger:                       plugin.logger,
		PluginConfig:                 plugin.pluginConfig,
		SignUpUseCase:                useCases.SignUpUseCase,
		SendEmailVerificationUseCase: useCases.SendEmailVerificationUseCase,
	}

	signInHandler := &handlers.SignInHandler{
		Logger:                       plugin.logger,
		PluginConfig:                 plugin.pluginConfig,
		SignInUseCase:                useCases.SignInUseCase,
		SendEmailVerificationUseCase: useCases.SendEmailVerificationUseCase,
	}

	verifyEmailHandler := &handlers.VerifyEmailHandler{
		VerifyEmailUseCase: useCases.VerifyEmailUseCase,
	}

	sendEmailVerificationHandler := &handlers.SendEmailVerificationHandler{
		UseCase: useCases.SendEmailVerificationUseCase,
	}

	requestPasswordResetHandler := &handlers.RequestPasswordResetHandler{
		UseCase: useCases.RequestPasswordResetUseCase,
	}

	changePasswordHandler := &handlers.ChangePasswordHandler{
		UseCase: useCases.ChangePasswordUseCase,
	}

	requestEmailChangeHandler := &handlers.RequestEmailChangeHandler{
		UseCase: useCases.RequestEmailChangeUseCase,
	}

	return []models.Route{
		{
			Method:  http.MethodPost,
			Path:    "/sign-up",
			Handler: signUpHandler.Handler(),
		},
		{
			Method:  http.MethodPost,
			Path:    "/sign-in",
			Handler: signInHandler.Handler(),
		},
		{
			Method:  http.MethodGet,
			Path:    "/verify-email",
			Handler: verifyEmailHandler.Handler(),
		},
		{
			Method:  http.MethodPost,
			Path:    "/send-email-verification",
			Handler: sendEmailVerificationHandler.Handler(),
		},
		{
			Method:  http.MethodPost,
			Path:    "/request-password-reset",
			Handler: requestPasswordResetHandler.Handler(),
		},
		{
			Method:  http.MethodPost,
			Path:    "/change-password",
			Handler: changePasswordHandler.Handler(),
		},
		{
			Method:  http.MethodPost,
			Path:    "/request-email-change",
			Handler: requestEmailChangeHandler.Handler(),
		},
	}
}

func BuildUseCases(p *EmailPasswordPlugin) *usecases.UseCases {
	signUpUseCase := &usecases.SignUpUseCase{
		GlobalConfig:    p.globalConfig,
		PluginConfig:    p.pluginConfig,
		Logger:          p.logger,
		UserService:     p.userService,
		AccountService:  p.accountService,
		SessionService:  p.sessionService,
		TokenService:    p.tokenService,
		PasswordService: p.passwordService,
		EventBus:        p.ctx.EventBus,
	}

	signInUseCase := &usecases.SignInUseCase{
		GlobalConfig:    p.globalConfig,
		PluginConfig:    p.pluginConfig,
		Logger:          p.logger,
		UserService:     p.userService,
		AccountService:  p.accountService,
		SessionService:  p.sessionService,
		TokenService:    p.tokenService,
		PasswordService: p.passwordService,
		EventBus:        p.ctx.EventBus,
	}

	verifyEmailUseCase := &usecases.VerifyEmailUseCase{
		Logger:              p.logger,
		PluginConfig:        p.pluginConfig,
		UserService:         p.userService,
		AccountService:      p.accountService,
		VerificationService: p.verificationService,
		TokenService:        p.tokenService,
		MailerService:       p.mailerService,
		EventBus:            p.ctx.EventBus,
	}

	sendEmailVerificationUseCase := &usecases.SendEmailVerificationUseCase{
		GlobalConfig:        p.globalConfig,
		PluginConfig:        p.pluginConfig,
		Logger:              p.logger,
		UserService:         p.userService,
		VerificationService: p.verificationService,
		TokenService:        p.tokenService,
		MailerService:       p.mailerService,
	}

	requestPasswordResetUseCase := &usecases.RequestPasswordResetUseCase{
		Logger:              p.logger,
		GlobalConfig:        p.globalConfig,
		PluginConfig:        p.pluginConfig,
		UserService:         p.userService,
		VerificationService: p.verificationService,
		TokenService:        p.tokenService,
		MailerService:       p.mailerService,
	}

	changePasswordUseCase := &usecases.ChangePasswordUseCase{
		Logger:              p.logger,
		PluginConfig:        p.pluginConfig,
		UserService:         p.userService,
		AccountService:      p.accountService,
		VerificationService: p.verificationService,
		PasswordService:     p.passwordService,
		TokenService:        p.tokenService,
		MailerService:       p.mailerService,
		EventBus:            p.ctx.EventBus,
	}

	requestEmailChangeUseCase := &usecases.RequestEmailChangeUseCase{
		Logger:              p.logger,
		GlobalConfig:        p.globalConfig,
		PluginConfig:        p.pluginConfig,
		UserService:         p.userService,
		VerificationService: p.verificationService,
		TokenService:        p.tokenService,
		MailerService:       p.mailerService,
	}

	return &usecases.UseCases{
		SignUpUseCase:                signUpUseCase,
		SignInUseCase:                signInUseCase,
		VerifyEmailUseCase:           verifyEmailUseCase,
		SendEmailVerificationUseCase: sendEmailVerificationUseCase,
		RequestPasswordResetUseCase:  requestPasswordResetUseCase,
		ChangePasswordUseCase:        changePasswordUseCase,
		RequestEmailChangeUseCase:    requestEmailChangeUseCase,
	}
}
