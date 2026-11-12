package oauth2

import (
	"net/http"

	"github.com/Authula/authula/middleware"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/handlers"
)

func Routes(plugin *OAuth2Plugin) []models.Route {
	useCases := plugin.useCases

	// The binding cookie is always SameSite=Lax so the browser sends it on the
	// cross-site provider callback.
	cookieOptions := handlers.CookieOptions{
		Secure: plugin.globalConfig.Session.Secure,
		Domain: plugin.globalConfig.Session.Domain,
		MaxAge: plugin.pluginConfig.FlowStateTTL,
	}

	authorizeHandler := &handlers.AuthorizeHandler{
		UseCase: useCases.AuthorizeUseCase,
		Cookie:  cookieOptions,
		Logger:  plugin.logger,
	}

	callbackHandler := &handlers.CallbackHandler{
		UseCase: useCases.CallbackUseCase,
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
