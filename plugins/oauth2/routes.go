package oauth2

import (
	"net/http"

	"github.com/Authula/authula/middleware"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/handlers"
	"github.com/Authula/authula/util"
)

func Routes(plugin *OAuth2Plugin) []models.Route {
	useCases := plugin.useCases

	cookieOptions := handlers.CookieOptions{
		Secure:   plugin.globalConfig.Session.Secure,
		Domain:   plugin.globalConfig.Session.Domain,
		SameSite: util.ParseSameSite(plugin.globalConfig.Session.SameSite),
		MaxAge:   constants.FlowCookieTTL,
	}

	authorizeHandler := &handlers.AuthorizeHandler{
		UseCase: useCases.AuthorizeUseCase,
		Cookie:  cookieOptions,
		Logger:  plugin.logger,
	}

	callbackHandler := &handlers.CallbackHandler{
		UseCase: useCases.CallbackUseCase,
		Cookie:  cookieOptions,
		Logger:  plugin.logger,
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
