package magiclink

import (
	"net/http"
	"testing"

	"github.com/Authula/authula/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoutes_ReturnsExpectedRoutes(t *testing.T) {
	cfg := &models.Config{
		Security: models.SecurityConfig{
			TrustedOrigins: []string{"https://example.com"},
		},
	}
	p := &MagicLinkPlugin{globalConfig: cfg}

	routes := Routes(p)

	expected := []struct {
		path   string
		method string
	}{
		{path: "/magic-link/sign-in", method: http.MethodPost},
		{path: "/magic-link/verify", method: http.MethodGet},
		{path: "/magic-link/exchange", method: http.MethodPost},
	}

	require.Len(t, routes, len(expected))
	for i, want := range expected {
		t.Run(want.method+" "+want.path, func(t *testing.T) {
			assert.Equal(t, want.path, routes[i].Path)
			assert.Equal(t, want.method, routes[i].Method)
			assert.NotNil(t, routes[i].Handler)
			assert.Len(t, routes[i].Middleware, 1)
		})
	}
}

func TestMagicLinkPlugin_Routes(t *testing.T) {
	p := newInitializedPlugin(t)

	routes := p.Routes()

	require.Len(t, routes, 3)
	assert.Equal(t, "/magic-link/sign-in", routes[0].Path)
	assert.Equal(t, http.MethodPost, routes[0].Method)
	assert.Equal(t, "/magic-link/verify", routes[1].Path)
	assert.Equal(t, http.MethodGet, routes[1].Method)
	assert.Equal(t, "/magic-link/exchange", routes[2].Path)
	assert.Equal(t, http.MethodPost, routes[2].Method)
}
