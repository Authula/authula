package magiclink

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/handlers"
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
			Path:    "/magic-link/sign-in",
			Method:  http.MethodPost,
			Handler: signInHandler.Handler(),
		},
		{
			Path:    "/magic-link/verify",
			Method:  http.MethodGet,
			Handler: verifyHandler.Handler(),
		},
		{
			Path:    "/magic-link/exchange",
			Method:  http.MethodPost,
			Handler: exchangeHandler.Handler(),
		},
	}
}
