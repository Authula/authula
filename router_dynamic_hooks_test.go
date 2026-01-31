package gobetterauth

import (
	"net/http/httptest"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/stretchr/testify/require"
)

func TestDynamicRouteHooks(t *testing.T) {
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &TestLogger{}
	router := NewRouter(config, logger, nil)

	// Register a dynamic route: /api/auth/oauth2/callback/{provider}
	router.SetRouteMetadataFromConfig(map[string]map[string]any{
		"POST:/api/auth/oauth2/callback/{provider}": {
			"plugins": []string{"test.plugin"},
		},
	})

	// Track if hook was executed
	hookCalled := false
	testHook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "test.plugin",
		Handler: func(ctx *models.RequestContext) error {
			hookCalled = true
			// Store provider for verification
			provider := ctx.Path[len(config.BasePath+"/oauth2/callback/"):]
			ctx.Values["provider"] = provider
			ctx.ResponseWriter.Header().Set("X-Provider", provider)
			return nil
		},
		Order: 10,
	}

	router.RegisterHook(testHook)

	// Simulate request for provider = "google"
	req := httptest.NewRequest("POST", "/api/auth/oauth2/callback/google", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.True(t, hookCalled, "Hook should have been called for dynamic route")
	require.Equal(t, "google", w.Result().Header.Get("X-Provider"), "Provider should be 'google'")

	// Reset for provider = "discord"
	hookCalled = false
	req2 := httptest.NewRequest("POST", "/api/auth/oauth2/callback/discord", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	require.True(t, hookCalled, "Hook should have been called for dynamic route")
	require.Equal(t, "discord", w2.Result().Header.Get("X-Provider"), "Provider should be 'discord'")
}

// TestLogger is a minimal logger for testing
type TestLogger struct{}

func (l *TestLogger) Debug(msg string, keysAndValues ...any) {}
func (l *TestLogger) Warn(msg string, keysAndValues ...any)  {}
func (l *TestLogger) Info(msg string, keysAndValues ...any)  {}
func (l *TestLogger) Error(msg string, keysAndValues ...any) {}
