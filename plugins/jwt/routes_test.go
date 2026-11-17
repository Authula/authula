package jwt

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
	"github.com/Authula/authula/plugins/jwt/types"
)

func TestJWTPlugin_Routes(t *testing.T) {
	t.Parallel()

	plugin := &JWTPlugin{
		Logger:         &internaltests.MockLogger{},
		pluginConfig:   types.JWTPluginConfig{ExpiresIn: 15 * time.Minute},
		refreshService: new(jwttests.MockRefreshTokenService),
		cacheService:   new(jwttests.MockCacheService),
	}

	routes := plugin.Routes()

	tests := []struct {
		name           string
		path           string
		method         string
		wantMiddleware int
	}{
		{
			name:           "refresh_token_route",
			path:           "/token/refresh",
			method:         http.MethodPost,
			wantMiddleware: 1,
		},
		{
			name:           "well_known_jwks_route",
			path:           "/.well-known/jwks.json",
			method:         http.MethodGet,
			wantMiddleware: 0,
		},
	}

	require.Len(t, routes, len(tests))

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := routes[i]

			assert.Equal(t, tt.path, route.Path)
			assert.Equal(t, tt.method, route.Method)
			assert.Len(t, route.Middleware, tt.wantMiddleware)
			require.NotNil(t, route.Handler)

			for _, mw := range route.Middleware {
				assert.NotNil(t, mw)
			}
		})
	}
}

func TestJWTPlugin_Routes_MatchPackageRoutes(t *testing.T) {
	t.Parallel()

	plugin := &JWTPlugin{
		Logger:         &internaltests.MockLogger{},
		pluginConfig:   types.JWTPluginConfig{ExpiresIn: 15 * time.Minute},
		refreshService: new(jwttests.MockRefreshTokenService),
		cacheService:   new(jwttests.MockCacheService),
	}

	fromPlugin := plugin.Routes()
	fromFunction := Routes(plugin)

	require.Len(t, fromFunction, len(fromPlugin))
	for i := range fromFunction {
		assert.Equal(t, fromPlugin[i].Path, fromFunction[i].Path)
		assert.Equal(t, fromPlugin[i].Method, fromFunction[i].Method)
		assert.NotNil(t, fromFunction[i].Handler)
	}
}
