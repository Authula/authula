package jwt

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/handlers"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/usecases"
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
			Path:    "/token/refresh",
			Method:  http.MethodPost,
			Handler: refreshHandler.Handler(),
		},
		{
			Path:    "/.well-known/jwks.json",
			Method:  http.MethodGet,
			Handler: jwksHandler.Handler(),
		},
	}
}
