package email_password

import (
	"net/http"

	"github.com/Authula/authula/middleware"
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
		TrustedOrigins:     plugin.globalConfig.Security.TrustedOrigins,
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
			Method: http.MethodPost,
			Path:   "/email-password/sign-up",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: signUpHandler.Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/email-password/sign-in",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: signInHandler.Handler(),
		},
		{
			Method: http.MethodGet,
			Path:   "/email-password/verify-email",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: verifyEmailHandler.Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/email-password/send-email-verification",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: sendEmailVerificationHandler.Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/email-password/request-password-reset",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: requestPasswordResetHandler.Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/email-password/change-password",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: changePasswordHandler.Handler(),
		},
		{
			Method: http.MethodPost,
			Path:   "/email-password/request-email-change",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequireActor(models.ActorUser),
			},
			Handler: requestEmailChangeHandler.Handler(),
		},
	}
}
