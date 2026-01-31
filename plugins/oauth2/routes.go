package oauth2

import (
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/oauth2/handlers"
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
	}
}
