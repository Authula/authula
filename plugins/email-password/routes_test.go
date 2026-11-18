package email_password

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
)

func TestRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		method              string
		path                string
		wantMiddlewareCount int
	}{
		{name: "sign up route", method: http.MethodPost, path: "/email-password/sign-up", wantMiddlewareCount: 1},
		{name: "sign in route", method: http.MethodPost, path: "/email-password/sign-in", wantMiddlewareCount: 1},
		{name: "verify email route", method: http.MethodGet, path: "/email-password/verify-email", wantMiddlewareCount: 1},
		{name: "send email verification route", method: http.MethodPost, path: "/email-password/send-email-verification", wantMiddlewareCount: 1},
		{name: "request password reset route", method: http.MethodPost, path: "/email-password/request-password-reset", wantMiddlewareCount: 1},
		{name: "change password route", method: http.MethodPost, path: "/email-password/change-password", wantMiddlewareCount: 1},
		{name: "request email change route", method: http.MethodPost, path: "/email-password/request-email-change", wantMiddlewareCount: 1},
	}

	routes := Routes(newInitializedTestPlugin(t))
	require.Len(t, routes, len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			route := findRoute(t, routes, tt.method, tt.path)

			require.Len(t, route.Middleware, tt.wantMiddlewareCount)
			require.NotNil(t, route.Handler)
		})
	}
}

func TestEmailPasswordPlugin_Routes(t *testing.T) {
	t.Parallel()

	plugin := newInitializedTestPlugin(t)

	require.Equal(t, routeKeys(Routes(plugin)), routeKeys(plugin.Routes()))
}

func findRoute(t *testing.T, routes []models.Route, method string, path string) models.Route {
	t.Helper()

	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return route
		}
	}

	t.Fatalf("route %s %s is not registered", method, path)
	return models.Route{}
}

func routeKeys(routes []models.Route) []string {
	keys := make([]string, 0, len(routes))
	for _, route := range routes {
		keys = append(keys, route.Method+" "+route.Path)
	}
	return keys
}
