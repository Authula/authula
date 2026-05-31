package oauth2

import (
	"net/http"

	"github.com/Authula/authula/middleware"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/handlers"
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
			Method: "GET",
			Path:   "/oauth2/authorize/{provider}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: authorizeHandler.Handle(),
		},
		{
			Method: "GET",
			Path:   "/oauth2/callback/{provider}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: callbackHandler.Handle(),
		},
	}
}
