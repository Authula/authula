package gobetterauth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// TestRouteMetadataPopulation verifies that route metadata is properly assigned to RequestContext
func TestRouteMetadataPopulation(t *testing.T) {
	// Create a router with test logger
	config := &models.Config{
		BasePath: "/auth",
	}
	logger := &testLogger{}
	router := NewRouter(config, logger, nil)

	// Set up route metadata from config
	routeMetadata := map[string]map[string]any{
		"GET:/me": {
			"plugins": []string{"session.auth", "bearer.auth"},
		},
		"POST:/sign-in": {
			"plugins": []string{"email_password.issuance"},
		},
	}
	router.SetRouteMetadataFromConfig(routeMetadata)

	// Create a test hook to capture the RequestContext
	var capturedCtx *models.RequestContext
	hook := models.Hook{
		Stage: models.HookOnRequest,
		Handler: func(ctx *models.RequestContext) error {
			capturedCtx = ctx
			return nil
		},
		Order: 0,
	}
	router.RegisterHook(hook)

	// Register a test route
	route := models.Route{
		Method:  "GET",
		Path:    "/me",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	router.RegisterRoute(route)

	// Make a test request
	req := httptest.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify that ctx.Route.Metadata was populated correctly
	if capturedCtx == nil {
		t.Fatal("capturedCtx is nil - hook was not called")
	}

	if capturedCtx.Route == nil {
		t.Fatal("capturedCtx.Route is nil")
	}

	pluginIDs, ok := capturedCtx.Route.Metadata["plugins"].([]string)
	if !ok {
		t.Fatal("plugins metadata not found or wrong type")
	}

	// Verify the plugins list
	expected := []string{"session.auth", "bearer.auth"}
	if len(pluginIDs) != len(expected) {
		t.Fatalf("expected %v plugins, got %v", expected, pluginIDs)
	}

	for i, pid := range pluginIDs {
		if pid != expected[i] {
			t.Fatalf("expected plugin %s at index %d, got %s", expected[i], i, pid)
		}
	}
}

// TestPluginIDBasedHookExecution verifies that hooks with PluginID only execute when their ID is in route metadata
func TestPluginIDBasedHookExecution(t *testing.T) {
	config := &models.Config{
		BasePath: "/auth",
	}
	logger := &testLogger{}
	router := NewRouter(config, logger, nil)

	// Set up route metadata
	routeMetadata := map[string]map[string]any{
		"GET:/me": {
			"plugins": []string{"session.auth"},
		},
		"POST:/sign-in": {
			"plugins": []string{"email_password.issuance"},
		},
	}
	router.SetRouteMetadataFromConfig(routeMetadata)

	// Track hook execution
	sessionHookCalled := false
	bearerHookCalled := false
	emailPasswordHookCalled := false

	// Register hooks with PluginID
	sessionHook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "session.auth",
		Handler: func(ctx *models.RequestContext) error {
			sessionHookCalled = true
			return nil
		},
		Order: 10,
	}

	bearerHook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "bearer.auth",
		Handler: func(ctx *models.RequestContext) error {
			bearerHookCalled = true
			return nil
		},
		Order: 5,
	}

	emailPasswordHook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "email_password.issuance",
		Handler: func(ctx *models.RequestContext) error {
			emailPasswordHookCalled = true
			return nil
		},
		Order: 10,
	}

	router.RegisterHook(sessionHook)
	router.RegisterHook(bearerHook)
	router.RegisterHook(emailPasswordHook)

	// Register test routes
	router.RegisterRoute(models.Route{
		Method:  "GET",
		Path:    "/me",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	})

	router.RegisterRoute(models.Route{
		Method:  "POST",
		Path:    "/sign-in",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	})

	// Test 1: GET /auth/me should only run session.auth hook (not bearer.auth)
	sessionHookCalled = false
	bearerHookCalled = false
	emailPasswordHookCalled = false

	req := httptest.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !sessionHookCalled {
		t.Error("session.auth hook should have been called for /auth/me")
	}
	if bearerHookCalled {
		t.Error("bearer.auth hook should NOT have been called for /auth/me")
	}
	if emailPasswordHookCalled {
		t.Error("email_password.issuance hook should NOT have been called for /auth/me")
	}

	// Test 2: POST /auth/sign-in should only run email_password.issuance hook
	sessionHookCalled = false
	bearerHookCalled = false
	emailPasswordHookCalled = false

	req = httptest.NewRequest("POST", "/auth/sign-in", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if sessionHookCalled {
		t.Error("session.auth hook should NOT have been called for /auth/sign-in")
	}
	if bearerHookCalled {
		t.Error("bearer.auth hook should NOT have been called for /auth/sign-in")
	}
	if !emailPasswordHookCalled {
		t.Error("email_password.issuance hook should have been called for /auth/sign-in")
	}
}

// TestRouteMetadataConversion verifies ConvertRouteMetadata utility function
func TestRouteMetadataConversion(t *testing.T) {
	routes := []models.RouteMapping{
		{
			Path:    "/auth/me",
			Method:  "GET",
			Plugins: []string{"session.auth", "bearer.auth"},
		},
		{
			Path:    "/auth/sign-in",
			Method:  "POST",
			Plugins: []string{"email_password.issuance"},
		},
		{
			Path:    "/auth/change-password",
			Method:  "POST",
			Plugins: []string{"session.auth", "csrf.protect"},
		},
	}

	metadata, err := util.ConvertRouteMetadata(routes)
	if err != nil {
		t.Fatalf("ConvertRouteMetadata failed: %v", err)
	}

	// Test 1: Verify GET:/auth/me
	key := "GET:/auth/me"
	if _, exists := metadata[key]; !exists {
		t.Fatalf("expected key %s in metadata", key)
	}

	pluginIDs, ok := metadata[key]["plugins"].([]string)
	if !ok {
		t.Fatalf("plugins not found or wrong type for %s", key)
	}

	if len(pluginIDs) != 2 || pluginIDs[0] != "session.auth" || pluginIDs[1] != "bearer.auth" {
		t.Errorf("unexpected plugins for %s: %v", key, pluginIDs)
	}

	// Test 2: Verify POST:/auth/sign-in
	key = "POST:/auth/sign-in"
	if _, exists := metadata[key]; !exists {
		t.Fatalf("expected key %s in metadata", key)
	}

	pluginIDs, ok = metadata[key]["plugins"].([]string)
	if !ok {
		t.Fatalf("plugins not found or wrong type for %s", key)
	}

	if len(pluginIDs) != 1 || pluginIDs[0] != "email_password.issuance" {
		t.Errorf("unexpected plugins for %s: %v", key, pluginIDs)
	}

	// Test 3: Verify POST:/auth/change-password
	key = "POST:/auth/change-password"
	if _, exists := metadata[key]; !exists {
		t.Fatalf("expected key %s in metadata", key)
	}

	pluginIDs, ok = metadata[key]["plugins"].([]string)
	if !ok {
		t.Fatalf("plugins not found or wrong type for %s", key)
	}

	if len(pluginIDs) != 2 || pluginIDs[0] != "session.auth" || pluginIDs[1] != "csrf.protect" {
		t.Errorf("unexpected plugins for %s: %v", key, pluginIDs)
	}
}

// TestHookExecutionOrder verifies that hooks execute in correct stage and order
func TestHookExecutionOrder(t *testing.T) {
	config := &models.Config{
		BasePath: "/auth",
	}
	logger := &testLogger{}
	router := NewRouter(config, logger, nil)

	var executionLog []string

	// Register hooks with different stages and orders
	onRequestHook := models.Hook{
		Stage: models.HookOnRequest,
		Handler: func(ctx *models.RequestContext) error {
			executionLog = append(executionLog, "onrequest")
			return nil
		},
		Order: 0,
	}

	beforeHook1 := models.Hook{
		Stage: models.HookBefore,
		Handler: func(ctx *models.RequestContext) error {
			executionLog = append(executionLog, "before1")
			return nil
		},
		Order: 10,
	}

	beforeHook2 := models.Hook{
		Stage: models.HookBefore,
		Handler: func(ctx *models.RequestContext) error {
			executionLog = append(executionLog, "before2")
			return nil
		},
		Order: 5, // Should execute before beforeHook1
	}

	afterHook := models.Hook{
		Stage: models.HookAfter,
		Handler: func(ctx *models.RequestContext) error {
			executionLog = append(executionLog, "after")
			return nil
		},
		Order: 0,
	}

	router.RegisterHook(afterHook)
	router.RegisterHook(beforeHook1)
	router.RegisterHook(onRequestHook)
	router.RegisterHook(beforeHook2)

	// Register a test route
	router.RegisterRoute(models.Route{
		Method: "GET",
		Path:   "/test",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionLog = append(executionLog, "handler")
		}),
	})

	// Make a test request
	req := httptest.NewRequest("GET", "/auth/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify execution order: OnRequest → Before (5, 10) → Handler → After
	expected := []string{"onrequest", "before2", "before1", "handler", "after"}
	if len(executionLog) != len(expected) {
		t.Fatalf("expected %d executions, got %d", len(expected), len(executionLog))
	}

	for i, step := range expected {
		if executionLog[i] != step {
			t.Errorf("execution step %d: expected %s, got %s", i, step, executionLog[i])
		}
	}
}

// TestDynamicRouteMatching verifies that hooks run against dynamic routes with path parameters
func TestDynamicRouteMatching(t *testing.T) {
	config := &models.Config{
		BasePath: "/auth",
	}
	logger := &testLogger{}
	router := NewRouter(config, logger, nil)

	// Set up route metadata with dynamic routes (using {param} syntax)
	routeMetadata := map[string]map[string]any{
		"POST:/auth/oauth2/callback/{provider}": {
			"plugins": []string{"session.issuance", "session.context"},
		},
		"GET:/auth/verify/{token}": {
			"plugins": []string{"email.verify"},
		},
		"POST:/auth/oauth2/callback/{provider}/extra": {
			"plugins": []string{"oauth2.process"},
		},
	}
	router.SetRouteMetadataFromConfig(routeMetadata)

	// Track which hooks are called
	var hooksExecuted []string

	// Register hooks with plugin IDs
	sessionIssuanceHook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "session.issuance",
		Handler: func(ctx *models.RequestContext) error {
			hooksExecuted = append(hooksExecuted, "session.issuance")
			return nil
		},
		Order: 10,
	}

	sessionContextHook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "session.context",
		Handler: func(ctx *models.RequestContext) error {
			hooksExecuted = append(hooksExecuted, "session.context")
			return nil
		},
		Order: 5,
	}

	emailVerifyHook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "email.verify",
		Handler: func(ctx *models.RequestContext) error {
			hooksExecuted = append(hooksExecuted, "email.verify")
			return nil
		},
		Order: 0,
	}

	oauth2ProcessHook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "oauth2.process",
		Handler: func(ctx *models.RequestContext) error {
			hooksExecuted = append(hooksExecuted, "oauth2.process")
			return nil
		},
		Order: 0,
	}

	router.RegisterHook(sessionIssuanceHook)
	router.RegisterHook(sessionContextHook)
	router.RegisterHook(emailVerifyHook)
	router.RegisterHook(oauth2ProcessHook)

	// Register test routes
	router.RegisterRoute(models.Route{
		Method:  "POST",
		Path:    "/oauth2/callback/{provider}",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	})

	router.RegisterRoute(models.Route{
		Method:  "GET",
		Path:    "/verify/{token}",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	})

	router.RegisterRoute(models.Route{
		Method:  "POST",
		Path:    "/oauth2/callback/{provider}/extra",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	})

	// Test 1: POST /auth/oauth2/callback/discord should match the dynamic route pattern
	hooksExecuted = nil
	req := httptest.NewRequest("POST", "/auth/oauth2/callback/discord", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if len(hooksExecuted) != 2 {
		t.Errorf("expected 2 hooks to execute for POST /auth/oauth2/callback/discord, got %d: %v", len(hooksExecuted), hooksExecuted)
	}
	if len(hooksExecuted) >= 1 && hooksExecuted[0] != "session.context" {
		t.Errorf("expected session.context hook first (order 5), got %s", hooksExecuted[0])
	}
	if len(hooksExecuted) >= 2 && hooksExecuted[1] != "session.issuance" {
		t.Errorf("expected session.issuance hook second (order 10), got %s", hooksExecuted[1])
	}

	// Test 2: POST /auth/oauth2/callback/github should also match the same pattern
	hooksExecuted = nil
	req = httptest.NewRequest("POST", "/auth/oauth2/callback/github", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if len(hooksExecuted) != 2 {
		t.Errorf("expected 2 hooks to execute for POST /auth/oauth2/callback/github, got %d: %v", len(hooksExecuted), hooksExecuted)
	}

	// Test 3: GET /auth/verify/abc123def456 should match the email.verify route
	hooksExecuted = nil
	req = httptest.NewRequest("GET", "/auth/verify/abc123def456", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if len(hooksExecuted) != 1 {
		t.Errorf("expected 1 hook to execute for GET /auth/verify/abc123def456, got %d: %v", len(hooksExecuted), hooksExecuted)
	}
	if len(hooksExecuted) >= 1 && hooksExecuted[0] != "email.verify" {
		t.Errorf("expected email.verify hook, got %s", hooksExecuted[0])
	}

	// Test 4: POST /auth/oauth2/callback/google/extra should match the three-segment pattern
	hooksExecuted = nil
	req = httptest.NewRequest("POST", "/auth/oauth2/callback/google/extra", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if len(hooksExecuted) != 1 {
		t.Errorf("expected 1 hook to execute for POST /auth/oauth2/callback/google/extra, got %d: %v", len(hooksExecuted), hooksExecuted)
	}
	if len(hooksExecuted) >= 1 && hooksExecuted[0] != "oauth2.process" {
		t.Errorf("expected oauth2.process hook, got %s", hooksExecuted[0])
	}

	// Test 5: Wrong method should not match (POST should not match GET pattern)
	hooksExecuted = nil
	req = httptest.NewRequest("GET", "/auth/oauth2/callback/discord", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if len(hooksExecuted) > 0 {
		t.Errorf("expected no hooks for wrong method (GET instead of POST), got: %v", hooksExecuted)
	}
}

// TestRoutePathMatching verifies the matchRoutePath function with various patterns
func TestRoutePathMatching(t *testing.T) {
	testCases := []struct {
		requestPath string
		pattern     string
		shouldMatch bool
		description string
	}{
		// Exact matches
		{"/oauth2/callback/discord", "/oauth2/callback/discord", true, "exact path match"},
		{"/api/users/123", "/api/users/123", true, "exact path with numbers"},

		// Dynamic parameter matches
		{"/oauth2/callback/discord", "/oauth2/callback/{provider}", true, "single dynamic parameter"},
		{"/oauth2/callback/github", "/oauth2/callback/{provider}", true, "single dynamic parameter different value"},
		{"/verify/abc123def456", "/verify/{token}", true, "dynamic token parameter"},
		{"/api/users/123/posts/456", "/api/users/{id}/posts/{postId}", true, "multiple dynamic parameters"},
		{"/api/users/john/settings/profile", "/api/users/{username}/settings/{section}", true, "multiple params with non-numeric values"},

		// Non-matches
		{"/oauth2/callback", "/oauth2/callback/{provider}", false, "missing dynamic parameter"},
		{"/oauth2/callback/discord/extra", "/oauth2/callback/{provider}", false, "extra path segment"},
		{"/different/path", "/oauth2/callback/{provider}", false, "completely different path"},
		{"/oauth2/callback/discord", "/oauth2/authorize", false, "different static segments"},
		{"/api/users/123/posts", "/api/users/{id}/posts/{postId}", false, "missing second dynamic parameter"},
	}

	for _, tc := range testCases {
		result := matchRoutePath(tc.requestPath, tc.pattern)
		if result != tc.shouldMatch {
			t.Errorf("%s: matchRoutePath(%q, %q) = %v, want %v",
				tc.description, tc.requestPath, tc.pattern, result, tc.shouldMatch)
		}
	}
}

// TestDynamicPathWithBasePathDebug tests the exact scenario: /api/auth/oauth2/callback/google
// with route mapping of /oauth2/callback/{provider} and basePath of /api/auth
func TestDynamicPathWithBasePathDebug(t *testing.T) {
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &testLogger{}
	router := NewRouter(config, logger, nil)

	// Simulating what happens in auth.go Handler()
	// The config has "/oauth2/callback/{provider}" which gets the basePath prefixed
	// So it becomes "/api/auth/oauth2/callback/{provider}"
	routeMetadata := map[string]map[string]any{
		"POST:/api/auth/oauth2/callback/{provider}": {
			"plugins": []string{"oauth2.callback"},
		},
	}
	router.SetRouteMetadataFromConfig(routeMetadata)

	// Track hook execution
	var hooksExecuted []string

	oAuth2Hook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "oauth2.callback",
		Handler: func(ctx *models.RequestContext) error {
			hooksExecuted = append(hooksExecuted, "oauth2.callback")
			if ctx.Route != nil {
				t.Logf("Route metadata: %v", ctx.Route.Metadata)
			}
			return nil
		},
		Order: 0,
	}

	router.RegisterHook(oAuth2Hook)

	// Register the route
	router.RegisterRoute(models.Route{
		Method:  "POST",
		Path:    "/oauth2/callback/{provider}",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	})

	// Test: Request to /api/auth/oauth2/callback/google
	req := httptest.NewRequest("POST", "/api/auth/oauth2/callback/google", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	t.Logf("Hooks executed: %v", hooksExecuted)
	t.Logf("routeMetadata: %v", router.routeMetadata)
	t.Logf("routeEntries: %v", router.routeEntries)

	if len(hooksExecuted) != 1 {
		t.Errorf("expected 1 hook to execute, got %d: %v", len(hooksExecuted), hooksExecuted)
	}
}

// testLogger is a simple logger implementation for tests
type testLogger struct{}

func (l *testLogger) Debug(msg string, args ...interface{}) {}
func (l *testLogger) Info(msg string, args ...interface{})  {}
func (l *testLogger) Warn(msg string, args ...interface{})  {}
func (l *testLogger) Error(msg string, args ...interface{}) {}

// TestDoubleBasePathApplication tests that basePath is not applied twice
func TestDoubleBasePathApplication(t *testing.T) {
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &testLogger{}
	router := NewRouter(config, logger, nil)

	// This simulates what auth.go does:
	// 1. Route mapping: "/oauth2/callback/{provider}"
	// 2. ConvertRouteMetadata produces: "POST:/oauth2/callback/{provider}"
	// 3. ApplyBasePathToMetadataKey produces: "POST:/api/auth/oauth2/callback/{provider}"
	// 4. SetRouteMetadataFromConfig is called with the adjusted key

	// So the key already has basePath applied
	routeMetadata := map[string]map[string]any{
		"POST:/api/auth/oauth2/callback/{provider}": {
			"plugins": []string{"oauth2.callback"},
		},
	}
	router.SetRouteMetadataFromConfig(routeMetadata)

	// The router should store this correctly
	expectedKey := "POST:/api/auth/oauth2/callback/{provider}"
	if _, exists := router.routeMetadata[expectedKey]; !exists {
		t.Errorf("expected key %s in routeMetadata, but got keys: %v", expectedKey, keysFromMap(router.routeMetadata))
	}

	// The segments should be correct
	if len(router.routeEntries) != 1 {
		t.Fatalf("expected 1 routeEntry, got %d", len(router.routeEntries))
	}

	entry := router.routeEntries[0]
	expectedSegments := []string{"api", "auth", "oauth2", "callback", "{provider}"}
	if len(entry.Segments) != len(expectedSegments) {
		t.Errorf("expected %d segments, got %d: %v", len(expectedSegments), len(entry.Segments), entry.Segments)
	}

	for i, seg := range entry.Segments {
		if seg != expectedSegments[i] {
			t.Errorf("segment %d: expected %q, got %q", i, expectedSegments[i], seg)
		}
	}
}

func keysFromMap(m map[string]map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
