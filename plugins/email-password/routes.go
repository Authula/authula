package email_password

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/handlers"
)

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
			Path:    "/email-password/sign-up",
			Handler: signUpHandler.Handler(),
		},
		{
			Method:  http.MethodPost,
			Path:    "/email-password/sign-in",
			Handler: signInHandler.Handler(),
		},
		{
			Method:  http.MethodGet,
			Path:    "/email-password/verify-email",
			Handler: verifyEmailHandler.Handler(),
		},
		{
			Method:  http.MethodPost,
			Path:    "/email-password/send-email-verification",
			Handler: sendEmailVerificationHandler.Handler(),
		},
		{
			Method:  http.MethodPost,
			Path:    "/email-password/request-password-reset",
			Handler: requestPasswordResetHandler.Handler(),
		},
		{
			Method:  http.MethodPost,
			Path:    "/email-password/change-password",
			Handler: changePasswordHandler.Handler(),
		},
		{
			Method:  http.MethodPost,
			Path:    "/email-password/request-email-change",
			Handler: requestEmailChangeHandler.Handler(),
		},
	}
}
