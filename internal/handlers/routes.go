package handlers

import (
	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/models"
)

func GetRoutes(config *models.Config, authService *auth.Service, basePath string, middleware *models.ApiMiddleware) []models.CustomRoute {
	useCases := auth.NewUseCases(config, authService)

	signIn := &SignInHandler{
		Config:  config,
		UseCase: useCases.SignInUseCase,
	}
	signUp := &SignUpHandler{
		Config:  config,
		UseCase: useCases.SignUpUseCase,
	}
	signOut := &SignOutHandler{
		Config:  config,
		UseCase: useCases.SignOutUseCase,
	}
	sendEmailVerification := &SendEmailVerificationHandler{
		Config:  config,
		UseCase: useCases.SendEmailVerificationUseCase,
	}
	verifyEmail := &VerifyEmailHandler{
		Config:  config,
		UseCase: useCases.VerifyEmailUseCase,
	}
	resetPassword := &ResetPasswordHandler{
		Config:  config,
		UseCase: useCases.ResetPasswordUseCase,
	}
	changePassword := &ChangePasswordHandler{
		Config:  config,
		UseCase: useCases.ChangePasswordUseCase,
	}
	changeEmailRequest := &EmailChangeHandler{
		Config:  config,
		UseCase: useCases.EmailChangeUseCase,
	}
	me := &MeHandler{
		Config:  config,
		UseCase: useCases.MeUseCase,
	}
	oauth2Login := &OAuth2LoginHandler{
		Config:  config,
		UseCase: useCases.OAuth2UseCase,
	}
	oauth2Callback := &OAuth2CallbackHandler{
		Config:  config,
		UseCase: useCases.OAuth2UseCase,
	}

	return []models.CustomRoute{
		{
			Method:  "POST",
			Path:    "/sign-in",
			Handler: signIn.Handler(),
		},
		{
			Method:  "POST",
			Path:    "/sign-up",
			Handler: signUp.Handler(),
		},
		{
			Method: "POST",
			Path:   "/email-verification",
			Middleware: []models.CustomRouteMiddleware{
				middleware.Auth(),
			},
			Handler: sendEmailVerification.Handler(),
		},
		{
			Method:  "GET",
			Path:    "/verify-email",
			Handler: verifyEmail.Handler(),
		},
		{
			Method: "POST",
			Path:   "/sign-out",
			Middleware: []models.CustomRouteMiddleware{
				middleware.Auth(),
				middleware.CSRF(),
			},
			Handler: signOut.Handler(),
		},
		{
			Method:  "POST",
			Path:    "/reset-password",
			Handler: resetPassword.Handler(),
		},
		{
			Method:  "POST",
			Path:    "/change-password",
			Handler: changePassword.Handler(),
		},
		{
			Method:  "POST",
			Path:    "/email-change",
			Handler: changeEmailRequest.Handler(),
		},
		{
			Method: "GET",
			Path:   "/me",
			Middleware: []models.CustomRouteMiddleware{
				middleware.Auth(),
			},
			Handler: me.Handler(),
		},
		{
			Method:  "GET",
			Path:    "/oauth2/{provider}/login",
			Handler: oauth2Login.Handler(),
		},
		{
			Method:  "GET",
			Path:    "/oauth2/{provider}/callback",
			Handler: oauth2Callback.Handler(),
		},
	}
}
