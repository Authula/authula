package magiclink

import (
	"net/http"

	"github.com/Authula/authula/middleware"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/magic-link/handlers"
)

func Routes(p *MagicLinkPlugin) []models.Route {
	useCases := BuildUseCases(p)

	signInHandler := &handlers.SignInHandler{
		UseCase: useCases.SignInUseCase,
	}

	verifyHandler := &handlers.VerifyHandler{
		UseCase:        useCases.VerifyUseCase,
		TrustedOrigins: p.globalConfig.Security.TrustedOrigins,
	}

	exchangeHandler := &handlers.ExchangeHandler{
		UseCase: useCases.ExchangeUseCase,
	}

	return []models.Route{
		{
			Path:   "/magic-link/sign-in",
			Method: http.MethodPost,
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: signInHandler.Handler(),
		},
		{
			Path:   "/magic-link/verify",
			Method: http.MethodGet,
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: verifyHandler.Handler(),
		},
		{
			Path:   "/magic-link/exchange",
			Method: http.MethodPost,
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: exchangeHandler.Handler(),
		},
	}
}
