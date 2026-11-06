package oauth2

import (
	"net/http"

	"github.com/Authula/authula/middleware"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/handlers"
)

func Routes(plugin *OAuth2Plugin) []models.Route {
	useCases := plugin.useCases

	// Session SameSite is intentionally not used here: flow cookies follow
	// CookieOptions.flowCookieSameSite, which is None when Secure so that a
	// cross-site form POST (Sign in with Apple) still carries the state cookie.
	cookieOptions := handlers.CookieOptions{
		Secure:   plugin.globalConfig.Session.Secure,
		Domain:   plugin.globalConfig.Session.Domain,
		SameSite: http.SameSiteLaxMode,
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
		{
			Method: "POST",
			Path:   "/oauth2/callback/{provider}",
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: callbackHandler.Handle(),
		},
	}
}
