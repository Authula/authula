package csrf

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/router"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

func TestCSRFCookiePreservedWithSetJSONResponse(t *testing.T) {
	// Create CSRF plugin
	plugin := New(CSRFPluginConfig{
		CookieName: "csrf_token",
		HeaderName: "X-CSRF-Token",
		MaxAge:     24 * time.Hour,
		SameSite:   "lax",
	})

	// Create mock plugin context
	ctx := &models.PluginContext{
		Logger: util.NewMockLogger(),
		GetConfig: func() *models.Config {
			return &models.Config{
				Security: models.SecurityConfig{
					TrustedOrigins: []string{"http://localhost:3000"},
				},
			}
		},
	}

	// Initialize plugin
	err := plugin.Init(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize CSRF plugin: %v", err)
	}

	// Create a mock request and response writer
	req := httptest.NewRequest("GET", "/test", nil)
	req.TLS = nil // Simulate HTTP for development
	w := httptest.NewRecorder()

	// Create request context
	reqCtx := &models.RequestContext{
		Request:         req,
		ResponseWriter:  w,
		Headers:         req.Header,
		Values:          make(map[string]any),
		ResponseHeaders: make(http.Header),
		Handled:         false,
	}

	// Set CSRF cookie (like the plugin does)
	token := "test-csrf-token-123"
	plugin.setCSRFCookie(reqCtx, token)

	// Now call SetJSONResponse (like validation does)
	reqCtx.SetJSONResponse(http.StatusForbidden, map[string]string{"message": "invalid csrf token"})

	// Simulate the router's finalizeResponse by creating a DeferredResponseWriter
	// and testing the override flow
	drw := &router.DeferredResponseWriter{
		Wrapped: w,
		Logger:  util.NewMockLogger(),
	}

	// Override with context (this is what router.finalizeResponse does)
	drw.OverrideWithContext(reqCtx)

	// Flush the response
	err = drw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Check that both the CSRF cookie and JSON response are preserved
	result := w.Result()

	// Verify CSRF cookie is preserved
	cookies := result.Cookies()
	if len(cookies) != 1 {
		t.Errorf("Expected 1 cookie, got %d", len(cookies))
	}
	if len(cookies) > 0 {
		if cookies[0].Name != "csrf_token" {
			t.Errorf("Expected cookie name 'csrf_token', got '%s'", cookies[0].Name)
		}
		if cookies[0].Value != token {
			t.Errorf("Expected cookie value '%s', got '%s'", token, cookies[0].Value)
		}
	}

	// Verify JSON response
	if result.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, result.StatusCode)
	}

	contentType := result.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Verify CSRF header is also set
	csrfHeader := result.Header.Get("X-CSRF-Token")
	if csrfHeader != token {
		t.Errorf("Expected CSRF header '%s', got '%s'", token, csrfHeader)
	}
}
