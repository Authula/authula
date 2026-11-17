package accesscontrol

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/access-control/usecases"
)

func TestRoutes(t *testing.T) {
	t.Parallel()

	plugin := &AccessControlPlugin{
		useCases: usecases.NewAccessControlUseCases(nil, nil, nil, nil, nil),
	}

	expected := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/access-control/roles"},
		{http.MethodGet, "/access-control/roles"},
		{http.MethodGet, "/access-control/roles/by-name/{role_name}"},
		{http.MethodGet, "/access-control/roles/{role_id}"},
		{http.MethodPatch, "/access-control/roles/{role_id}"},
		{http.MethodDelete, "/access-control/roles/{role_id}"},
		{http.MethodPost, "/access-control/permissions"},
		{http.MethodGet, "/access-control/permissions"},
		{http.MethodGet, "/access-control/permissions/by-key/{permission_key}"},
		{http.MethodGet, "/access-control/permissions/{permission_id}"},
		{http.MethodPatch, "/access-control/permissions/{permission_id}"},
		{http.MethodDelete, "/access-control/permissions/{permission_id}"},
		{http.MethodPost, "/access-control/roles/{role_id}/permissions"},
		{http.MethodGet, "/access-control/roles/{role_id}/permissions"},
		{http.MethodPut, "/access-control/roles/{role_id}/permissions"},
		{http.MethodDelete, "/access-control/roles/{role_id}/permissions/{permission_id}"},
		{http.MethodGet, "/access-control/users/{user_id}/roles"},
		{http.MethodPut, "/access-control/users/{user_id}/roles"},
		{http.MethodPost, "/access-control/users/{user_id}/roles"},
		{http.MethodDelete, "/access-control/users/{user_id}/roles/{role_id}"},
		{http.MethodGet, "/access-control/users/{user_id}/permissions"},
		{http.MethodPost, "/access-control/users/{user_id}/permissions/check"},
	}

	routes := plugin.Routes()
	require.Len(t, routes, len(expected))

	seen := make(map[string]struct{}, len(routes))
	for i, route := range routes {
		require.Equal(t, expected[i].method, route.Method)
		require.Equal(t, expected[i].path, route.Path)
		require.NotNil(t, route.Handler, "route %s %s must have a handler", route.Method, route.Path)
		require.Len(t, route.Middleware, 1, "route %s %s must require authentication", route.Method, route.Path)

		key := route.Method + " " + route.Path
		_, duplicate := seen[key]
		require.False(t, duplicate, "duplicate route %s", key)
		seen[key] = struct{}{}
	}
}
