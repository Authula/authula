package jwt

import (
	"net/http"

	"github.com/Authula/authula/middleware"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/handlers"
	"github.com/Authula/authula/plugins/jwt/usecases"
)

func Routes(plugin *JWTPlugin) []models.Route {
	refreshUseCase := usecases.NewRefreshTokenUseCase(
		plugin.Logger,
		plugin.refreshService,
	)

	jwksUseCase := usecases.NewJWKSUseCase(
		plugin.Logger,
		plugin.cacheService,
	)

	refreshHandler := &handlers.RefreshTokenHandler{
		Logger:              plugin.Logger,
		RefreshTokenUseCase: refreshUseCase,
	}

	jwksHandler := &handlers.WellKnownJWKSHandler{
		Logger:      plugin.Logger,
		JWKSUseCase: jwksUseCase,
	}

	return []models.Route{
		{
			Path:   "/token/refresh",
			Method: http.MethodPost,
			Middleware: []func(http.Handler) http.Handler{
				middleware.RequirePublicOrUserActor(),
			},
			Handler: refreshHandler.Handle(),
		},
		{
			Path:    "/.well-known/jwks.json",
			Method:  http.MethodGet,
			Handler: jwksHandler.Handle(),
		},
	}
}
