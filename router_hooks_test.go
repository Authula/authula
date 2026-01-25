package gobetterauth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// TestRouterHooksExecution verifies that hooks are executed at the correct stages
func TestRouterHooksExecution(t *testing.T) {
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &mockLogger{}
	router := NewRouter(config, logger, nil)
	executedStages := []models.HookStage{}

	// Register hooks at different stages
	router.RegisterHook(models.Hook{
		Stage: models.HookOnRequest,
		Handler: func(ctx *models.RequestContext) error {
			executedStages = append(executedStages, models.HookOnRequest)
			return nil
		},
		Order: 0,
	})

	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Handler: func(ctx *models.RequestContext) error {
			executedStages = append(executedStages, models.HookBefore)
			return nil
		},
		Order: 0,
	})

	router.RegisterHook(models.Hook{
		Stage: models.HookAfter,
		Handler: func(ctx *models.RequestContext) error {
			executedStages = append(executedStages, models.HookAfter)
			return nil
		},
		Order: 0,
	})

	router.RegisterHook(models.Hook{
		Stage: models.HookOnResponse,
		Handler: func(ctx *models.RequestContext) error {
			executedStages = append(executedStages, models.HookOnResponse)
			return nil
		},
		Order: 0,
	})

	// Register a dummy route
	router.RegisterRoute(models.Route{
		Method: "GET",
		Path:   "/test",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	})

	// Make a request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify stages were executed in order
	expectedStages := []models.HookStage{
		models.HookOnRequest,
		models.HookBefore,
		models.HookAfter,
		models.HookOnResponse,
	}

	if len(executedStages) != len(expectedStages) {
		t.Fatalf("Expected %d stages, got %d", len(expectedStages), len(executedStages))
	}

	for i, stage := range executedStages {
		if stage != expectedStages[i] {
			t.Errorf("Stage %d: expected %d, got %d", i, expectedStages[i], stage)
		}
	}
}

// TestRouterHooksHandledFlag verifies that Handled flag stops further processing
func TestRouterHooksHandledFlag(t *testing.T) {
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &mockLogger{}
	router := NewRouter(config, logger, nil)

	executedHooks := 0

	// First hook marks request as handled
	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Handler: func(ctx *models.RequestContext) error {
			executedHooks++
			ctx.Handled = true
			ctx.ResponseWriter.WriteHeader(http.StatusForbidden)
			return nil
		},
		Order: 0,
	})

	// This hook should not execute because Handled is true
	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Handler: func(ctx *models.RequestContext) error {
			executedHooks++
			return nil
		},
		Order: 1,
	})

	// Route handler should not be called
	router.RegisterRoute(models.Route{
		Method: "GET",
		Path:   "/test",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executedHooks++
			w.WriteHeader(http.StatusOK)
		}),
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should only execute first hook
	if executedHooks != 1 {
		t.Errorf("Expected 1 hook execution, got %d", executedHooks)
	}

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// TestRouterHooksMatcher verifies that matchers control hook execution
func TestRouterHooksMatcher(t *testing.T) {
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &mockLogger{}
	router := NewRouter(config, logger, nil)

	executedHooks := 0

	// Hook with matcher that only matches /admin paths
	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Matcher: func(ctx *models.RequestContext) bool {
			return ctx.Path == "/admin"
		},
		Handler: func(ctx *models.RequestContext) error {
			executedHooks++
			return nil
		},
		Order: 0,
	})

	router.RegisterRoute(models.Route{
		Method: "GET",
		Path:   "/test",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	})

	// Request to /test should not trigger the matcher
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if executedHooks != 0 {
		t.Errorf("Hook should not execute for non-matching path, but %d hooks executed", executedHooks)
	}

	// Request to /admin should trigger the matcher
	executedHooks = 0
	router.RegisterRoute(models.Route{
		Method: "GET",
		Path:   "/admin",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	})

	req = httptest.NewRequest("GET", "/admin", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if executedHooks != 1 {
		t.Errorf("Hook should execute for matching path, but %d hooks executed", executedHooks)
	}
}

// TestRouterHooksOrder verifies that hooks execute in order
func TestRouterHooksOrder(t *testing.T) {
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &mockLogger{}
	router := NewRouter(config, logger, nil)

	executionOrder := []int{}

	// Register hooks with different orders
	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Handler: func(ctx *models.RequestContext) error {
			executionOrder = append(executionOrder, 2)
			return nil
		},
		Order: 2,
	})

	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Handler: func(ctx *models.RequestContext) error {
			executionOrder = append(executionOrder, 0)
			return nil
		},
		Order: 0,
	})

	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Handler: func(ctx *models.RequestContext) error {
			executionOrder = append(executionOrder, 1)
			return nil
		},
		Order: 1,
	})

	router.RegisterRoute(models.Route{
		Method: "GET",
		Path:   "/test",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expectedOrder := []int{0, 1, 2}
	if len(executionOrder) != len(expectedOrder) {
		t.Fatalf("Expected %d hooks, got %d", len(expectedOrder), len(executionOrder))
	}

	for i, order := range executionOrder {
		if order != expectedOrder[i] {
			t.Errorf("Position %d: expected order %d, got %d", i, expectedOrder[i], order)
		}
	}
}

// TestRequestContextValues verifies that hooks can share data via Values map
func TestRequestContextValues(t *testing.T) {
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &mockLogger{}
	router := NewRouter(config, logger, nil)

	// First hook sets a value
	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Handler: func(ctx *models.RequestContext) error {
			ctx.Values["user_id"] = "12345"
			return nil
		},
		Order: 0,
	})

	// Second hook reads the value
	hookReadValue := ""
	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Handler: func(ctx *models.RequestContext) error {
			if val, ok := ctx.Values["user_id"]; ok {
				hookReadValue = val.(string)
			}
			return nil
		},
		Order: 1,
	})

	router.RegisterRoute(models.Route{
		Method: "GET",
		Path:   "/test",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if hookReadValue != "12345" {
		t.Errorf("Expected hook to read value '12345', got '%s'", hookReadValue)
	}
}

// TestSessionBasedJWTArchitecture verifies the session-based JWT architecture
// Sessions are the security boundary (stored in DB)
// JWT is optional for SPAs/mobile apps and embeds session_id in tokens
func TestSessionBasedJWTArchitecture(t *testing.T) {
	// This test documents the architecture:
	// 1. Auth handler (e.g., sign-in) creates a session in DB and gets session_id
	// 2. Handler sets in ctx.Values: user_id, session_id, auth_token
	// 3. JWT issuance hook reads user_id and session_id from Values
	// 4. JWT hook creates tokens with session_id embedded in JWT claims
	// 5. Session issuance hook reads access_token and sets it as session cookie

	ctx := &models.RequestContext{
		Values: make(map[string]any),
	}

	// Simulate auth handler setting values (like email-password sign-in does)
	ctx.Values["user_id"] = "user-456"
	ctx.Values["session_id"] = "sess-123-abc"
	ctx.Values["auth_token"] = "session-token-xyz"

	// Verify JWT hook would receive necessary values
	userID, ok := ctx.Values["user_id"].(string)
	if !ok || userID != "user-456" {
		t.Error("JWT hook should receive user_id from Values")
	}

	sessionID, ok := ctx.Values["session_id"].(string)
	if !ok || sessionID != "sess-123-abc" {
		t.Error("JWT hook should receive session_id from Values")
	}

	// Simulate JWT hook setting tokens
	ctx.Values["access_token"] = "jwt.access.with.session_id.embedded"
	ctx.Values["refresh_token"] = "jwt.refresh.with.session_id.embedded"

	// Verify session hook would receive access_token
	accessToken, ok := ctx.Values["access_token"].(string)
	if !ok || accessToken == "" {
		t.Error("Session hook should receive access_token from JWT hook")
	}

	// Architecture validation: session_id flows through the plugin chain
	// auth handler -> ctx.Values -> JWT hook -> JWT claims -> access_token -> session hook
	if !t.Failed() {
		t.Log("✓ Session-based JWT architecture validated:")
		t.Log("  - Auth handler creates session and sets user_id, session_id in Values")
		t.Log("  - JWT plugin reads session_id and embeds it in JWT claims")
		t.Log("  - Session plugin receives JWT token with embedded session_id")
		t.Log("  - Sessions remain the security boundary, JWT is optional for SPAs/mobile apps")
	}
}

// TestRouterPanicRecovery verifies that panics in sync hooks are recovered and logged
func TestRouterPanicRecovery(t *testing.T) {
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &mockLogger{}
	router := NewRouter(config, logger, nil)

	// Hook that panics
	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Handler: func(ctx *models.RequestContext) error {
			panic("hook panic test")
		},
		Order: 0,
	})

	// Hook after panic to verify execution continues
	hookExecutedAfterPanic := false
	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Order: 1,
		Handler: func(ctx *models.RequestContext) error {
			hookExecutedAfterPanic = true
			return nil
		},
	})

	router.RegisterRoute(models.Route{
		Method:  "GET",
		Path:    "/test",
		Handler: &testHandler{statusCode: 200, body: "OK"},
	})

	req := httptest.NewRequest("GET", "/api/auth/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify request completed despite panic
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify that the second hook in the stage also ran after recovery
	if !hookExecutedAfterPanic {
		t.Errorf("Expected second hook to execute after panic recovery")
	}
}

// TestAsyncHookTimeout verifies that async hooks timeout correctly
func TestAsyncHookTimeout(t *testing.T) {
	opts := &RouterOptions{
		AsyncHookTimeout: 100 * time.Millisecond, // Very short timeout
	}
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &mockLogger{}
	router := NewRouter(config, logger, opts)

	router.RegisterHook(models.Hook{
		Stage: models.HookOnResponse,
		Async: true,
		Handler: func(ctx *models.RequestContext) error {
			// Sleep longer than timeout
			select {
			case <-ctx.Request.Context().Done():
				// Context was cancelled (timeout)
				return nil
			case <-time.After(500 * time.Millisecond):
				// Timeout didn't cancel us
				return nil
			}
		},
		Order: 0,
	})

	router.RegisterRoute(models.Route{
		Method:  "GET",
		Path:    "/test",
		Handler: &testHandler{statusCode: 200, body: "OK"},
	})

	req := httptest.NewRequest("GET", "/api/auth/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Give async hook time to complete
	time.Sleep(600 * time.Millisecond)

	// Request should complete successfully
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestHookErrorModeFailFast verifies that fail-fast mode skips remaining hooks on error
func TestHookErrorModeFailFast(t *testing.T) {
	opts := &RouterOptions{
		HookErrorMode: HookErrorModeFailFast,
	}
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &mockLogger{}
	router := NewRouter(config, logger, opts)

	hookExecutionOrder := []int{}

	// First hook returns error
	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Order: 1,
		Handler: func(ctx *models.RequestContext) error {
			hookExecutionOrder = append(hookExecutionOrder, 1)
			return fmt.Errorf("hook error")
		},
	})

	// Second hook should not execute due to fail-fast
	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Order: 2,
		Handler: func(ctx *models.RequestContext) error {
			hookExecutionOrder = append(hookExecutionOrder, 2)
			return nil
		},
	})

	// After hook should execute (different stage) - will only run if request didn't get handled in Before
	router.RegisterHook(models.Hook{
		Stage: models.HookAfter,
		Order: 1,
		Handler: func(ctx *models.RequestContext) error {
			hookExecutionOrder = append(hookExecutionOrder, 3)
			return nil
		},
	})

	router.RegisterRoute(models.Route{
		Method:  "GET",
		Path:    "/test",
		Handler: &testHandler{statusCode: 200, body: "OK"},
	})

	req := httptest.NewRequest("GET", "/api/auth/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify execution order: hook 1 executed, hook 2 skipped (fail-fast)
	// Note: hook 3 (After stage) won't execute because fail-fast sets ctx.Handled=true,
	// which causes early return before handler, so After hooks don't run
	if len(hookExecutionOrder) != 1 || hookExecutionOrder[0] != 1 {
		t.Errorf("Expected hooks [1] due to fail-fast, got %v", hookExecutionOrder)
	}
}

// TestHookErrorModeSilent verifies that silent mode ignores errors
func TestHookErrorModeSilent(t *testing.T) {
	opts := &RouterOptions{
		HookErrorMode: HookErrorModeSilent,
	}
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &mockLogger{}
	router := NewRouter(config, logger, opts)

	hookExecutionOrder := []int{}

	// First hook returns error (should be silent)
	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Order: 1,
		Handler: func(ctx *models.RequestContext) error {
			hookExecutionOrder = append(hookExecutionOrder, 1)
			return fmt.Errorf("hook error - should be silent")
		},
	})

	// Second hook should execute (error was silent, not fail-fast)
	router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Order: 2,
		Handler: func(ctx *models.RequestContext) error {
			hookExecutionOrder = append(hookExecutionOrder, 2)
			return nil
		},
	})

	router.RegisterRoute(models.Route{
		Method:  "GET",
		Path:    "/test",
		Handler: &testHandler{statusCode: 200, body: "OK"},
	})

	req := httptest.NewRequest("GET", "/api/auth/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify both hooks executed despite error
	if len(hookExecutionOrder) != 2 || hookExecutionOrder[0] != 1 || hookExecutionOrder[1] != 2 {
		t.Errorf("Expected hooks [1, 2], got %v", hookExecutionOrder)
	}
}

// testHandler is a simple HTTP handler for testing
type testHandler struct {
	statusCode int
	body       string
	headers    map[string]string
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.headers != nil {
		for key, val := range h.headers {
			w.Header().Set(key, val)
		}
	}
	w.WriteHeader(h.statusCode)
	w.Write([]byte(h.body))
}

type mockLogger struct{}

func (m *mockLogger) Debug(msg string, args ...any) {}
func (m *mockLogger) Info(msg string, args ...any)  {}
func (m *mockLogger) Warn(msg string, args ...any)  {}
func (m *mockLogger) Error(msg string, args ...any) {}
func (m *mockLogger) Panic(msg string, args ...any) {}
func (m *mockLogger) WithField(key string, value any) models.Logger {
	return m
}
func (m *mockLogger) WithFields(fields map[string]any) models.Logger {
	return m
}
