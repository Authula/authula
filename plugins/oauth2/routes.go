package oauth2

import (
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/handlers"
)

func Routes(plugin *OAuth2Plugin) []models.Route {
	useCases := BuildUseCases(plugin)

	authorizeHandler := &handlers.AuthorizeHandler{
		UseCase: useCases.AuthorizeUseCase,
	}

	callbackHandler := &handlers.CallbackHandler{
		UseCase: useCases.CallbackUseCase,
		HMACKey: plugin.hmacKey,
	}

	refreshHandler := &handlers.RefreshHandler{
		UseCase: useCases.RefreshUseCase,
	}

	linkHandler := &handlers.LinkHandler{
		UseCase: useCases.LinkAccountUseCase,
	}

	return []models.Route{
		{
			Method:  "GET",
			Path:    "/oauth2/authorize/{provider}",
			Handler: authorizeHandler.Handler(),
		},
		{
			Method:  "GET",
			Path:    "/oauth2/callback/{provider}",
			Handler: callbackHandler.Handler(),
		},
		{
			Method:  "POST",
			Path:    "/oauth2/refresh/{provider}",
			Handler: refreshHandler.Handler(),
		},
		{
			Method:  "POST",
			Path:    "/oauth2/link/{provider}",
			Handler: linkHandler.Handler(),
		},
	}
}
