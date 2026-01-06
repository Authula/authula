package jwt

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/handlers"
)

// Routes returns all HTTP routes for the JWT plugin
func Routes(plugin *JWTPlugin) []models.Route {
	// Create refresh token handler
	refreshHandler := &handlers.RefreshTokenHandler{
		Service: plugin.refreshService,
		Logger:  plugin.Logger,
	}

	// Create JWKS handler
	jwksHandler := &handlers.WellKnownJWKSHandler{
		CacheService: plugin.cacheService,
		Logger:       plugin.Logger,
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
