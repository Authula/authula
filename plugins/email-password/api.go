package email_password

import (
	"context"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/handlers"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/types"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/usecases"
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
) (*types.SignUpResult, error) {
	return a.useCases.SignUpUseCase.SignUp(ctx, name, email, password, image, callbackURL)
}

func (a *API) SignIn(
	ctx context.Context,
	email string,
	password string,
	callbackURL *string,
) (*types.SignInResult, error) {
	return a.useCases.SignInUseCase.SignIn(ctx, email, password, callbackURL)
}

func (a *API) VerifyEmail(ctx context.Context, tokenStr string) error {
	return a.useCases.VerifyEmailUseCase.VerifyEmail(ctx, tokenStr)
}

func (a *API) SendVerificationEmail(ctx context.Context, email string, callbackURL *string) error {
	return a.useCases.SendVerificationEmailUseCase.Send(ctx, email, callbackURL)
}

func (a *API) RequestPasswordReset(ctx context.Context, email string, callbackURL *string) error {
	return a.useCases.RequestPasswordResetUseCase.RequestReset(ctx, email, callbackURL)
}

func (a *API) ChangePassword(ctx context.Context, tokenStr string, newPassword string) error {
	return a.useCases.ChangePasswordUseCase.ChangePassword(ctx, tokenStr, newPassword)
}

func BuildAPI(plugin *EmailPasswordPlugin) *API {
	useCases := BuildUseCases(plugin)
	return &API{useCases: useCases}
}

// Routes returns all routes for the email/password plugin
func Routes(plugin *EmailPasswordPlugin) []models.Route {
	useCases := BuildUseCases(plugin)

	signUpHandler := &handlers.SignUpHandler{
		UseCase: useCases.SignUpUseCase,
	}

	signInHandler := &handlers.SignInHandler{
		UseCase: useCases.SignInUseCase,
	}

	verifyEmailHandler := &handlers.VerifyEmailHandler{
		UseCase: useCases.VerifyEmailUseCase,
	}

	sendVerificationEmailHandler := &handlers.SendVerificationEmailHandler{
		UseCase: useCases.SendVerificationEmailUseCase,
	}

	requestPasswordResetHandler := &handlers.RequestPasswordResetHandler{
		UseCase: useCases.RequestPasswordResetUseCase,
	}

	changePasswordHandler := &handlers.ChangePasswordHandler{
		UseCase: useCases.ChangePasswordUseCase,
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
			Path:    "/send-verification-email",
			Handler: sendVerificationEmailHandler.Handler(),
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
	}
}

func BuildUseCases(plugin *EmailPasswordPlugin) *usecases.UseCases {
	sendVerificationEmailUseCase := &usecases.SendVerificationEmailUseCase{
		GlobalConfig:        plugin.globalConfig,
		PluginConfig:        plugin.pluginConfig,
		Logger:              plugin.logger,
		UserService:         plugin.userService,
		VerificationService: plugin.verificationService,
		TokenService:        plugin.tokenService,
		MailerService:       plugin.mailerService,
	}

	signUpUseCase := &usecases.SignUpUseCase{
		GlobalConfig:                 plugin.globalConfig,
		PluginConfig:                 plugin.pluginConfig,
		Logger:                       plugin.logger,
		UserService:                  plugin.userService,
		AccountService:               plugin.accountService,
		PasswordService:              plugin.passwordService,
		EventBus:                     plugin.ctx.EventBus,
		SendVerificationEmailUseCase: sendVerificationEmailUseCase,
	}

	signInUseCase := &usecases.SignInUseCase{
		GlobalConfig:                 plugin.globalConfig,
		PluginConfig:                 plugin.pluginConfig,
		Logger:                       plugin.logger,
		UserService:                  plugin.userService,
		AccountService:               plugin.accountService,
		PasswordService:              plugin.passwordService,
		EventBus:                     plugin.ctx.EventBus,
		SendVerificationEmailUseCase: sendVerificationEmailUseCase,
	}

	verifyEmailUseCase := &usecases.VerifyEmailUseCase{
		Logger:              plugin.logger,
		EventBus:            plugin.ctx.EventBus,
		UserService:         plugin.userService,
		VerificationService: plugin.verificationService,
		TokenService:        plugin.tokenService,
	}

	requestPasswordResetUseCase := &usecases.RequestPasswordResetUseCase{
		GlobalConfig:        plugin.globalConfig,
		PluginConfig:        plugin.pluginConfig,
		UserService:         plugin.userService,
		VerificationService: plugin.verificationService,
		MailerService:       plugin.mailerService,
	}

	changePasswordUseCase := &usecases.ChangePasswordUseCase{
		PluginConfig:        plugin.pluginConfig,
		AccountService:      plugin.accountService,
		VerificationService: plugin.verificationService,
		PasswordService:     plugin.passwordService,
	}

	return &usecases.UseCases{
		SignUpUseCase:                signUpUseCase,
		SignInUseCase:                signInUseCase,
		VerifyEmailUseCase:           verifyEmailUseCase,
		SendVerificationEmailUseCase: sendVerificationEmailUseCase,
		RequestPasswordResetUseCase:  requestPasswordResetUseCase,
		ChangePasswordUseCase:        changePasswordUseCase,
	}
}
