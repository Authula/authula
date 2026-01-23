package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

func TestFlushPreservesCookies(t *testing.T) {
	// Create a mock ResponseWriter
	originalWriter := httptest.NewRecorder()

	// Create DeferredResponseWriter
	drw := &DeferredResponseWriter{
		Wrapped: originalWriter,
		Logger:  util.NewMockLogger(),
	}

	// Set a cookie directly on the original ResponseWriter (like CSRF plugin does)
	http.SetCookie(drw.Wrapped, &http.Cookie{
		Name:  "csrf_token",
		Value: "test-token-123",
		Path:  "/",
	})

	// Create a RequestContext with JSON response (like CSRF validation does)
	reqCtx := &models.RequestContext{
		ResponseStatus:  http.StatusForbidden,
		ResponseHeaders: make(http.Header),
		ResponseBody:    []byte(`{"message": "invalid csrf token"}`),
		ResponseReady:   true,
	}
	reqCtx.ResponseHeaders.Set("Content-Type", "application/json")

	// Override with context (simulating SetJSONResponse flow)
	drw.OverrideWithContext(reqCtx)

	// Flush the response
	err := drw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Check that both the CSRF cookie and JSON response headers are present
	result := originalWriter.Result()

	// Verify CSRF cookie is preserved
	cookies := result.Cookies()
	if len(cookies) != 1 {
		t.Errorf("Expected 1 cookie, got %d", len(cookies))
	}
	if len(cookies) > 0 && cookies[0].Name != "csrf_token" {
		t.Errorf("Expected cookie name 'csrf_token', got '%s'", cookies[0].Name)
	}
	if len(cookies) > 0 && cookies[0].Value != "test-token-123" {
		t.Errorf("Expected cookie value 'test-token-123', got '%s'", cookies[0].Value)
	}

	// Verify JSON content type header is set
	contentType := result.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Verify status code
	if result.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, result.StatusCode)
	}
}

func TestFlushPreservesMultipleCookies(t *testing.T) {
	// Create a mock ResponseWriter
	originalWriter := httptest.NewRecorder()

	// Create DeferredResponseWriter
	drw := &DeferredResponseWriter{
		Wrapped: originalWriter,
		Logger:  util.NewMockLogger(),
	}

	// Set multiple cookies directly on the original ResponseWriter
	http.SetCookie(drw.Wrapped, &http.Cookie{
		Name:  "csrf_token",
		Value: "test-token-123",
		Path:  "/",
	})
	http.SetCookie(drw.Wrapped, &http.Cookie{
		Name:  "session_id",
		Value: "session-456",
		Path:  "/",
	})

	// Create a RequestContext with JSON response that also sets a cookie
	reqCtx := &models.RequestContext{
		ResponseStatus:  http.StatusOK,
		ResponseHeaders: make(http.Header),
		ResponseBody:    []byte(`{"success": true}`),
		ResponseReady:   true,
	}
	reqCtx.ResponseHeaders.Set("Content-Type", "application/json")
	reqCtx.ResponseHeaders["Set-Cookie"] = []string{"new_cookie=new_value; Path=/"}

	// Override with context
	drw.OverrideWithContext(reqCtx)

	// Flush the response
	err := drw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Check that all cookies are present
	result := originalWriter.Result()
	cookies := result.Cookies()

	// Should have 3 cookies total (2 original + 1 override)
	if len(cookies) != 3 {
		t.Errorf("Expected 3 cookies, got %d", len(cookies))
	}

	// Verify specific cookies exist
	cookieMap := make(map[string]string)
	for _, cookie := range cookies {
		cookieMap[cookie.Name] = cookie.Value
	}

	if cookieMap["csrf_token"] != "test-token-123" {
		t.Errorf("CSRF token not preserved correctly")
	}
	if cookieMap["session_id"] != "session-456" {
		t.Errorf("Session ID not preserved correctly")
	}
	if cookieMap["new_cookie"] != "new_value" {
		t.Errorf("New cookie not set correctly")
	}
}
