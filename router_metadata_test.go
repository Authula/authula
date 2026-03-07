package gobetterauth

import (
	"net/http"
	"net/http/httptest"
	"slices"
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
		"GET:/resource/details": {
			"plugins": []string{"plugin.auth", "plugin.verification"},
		},
		"POST:/resource/create": {
			"plugins": []string{"plugin.process"},
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
		Path:    "/resource/details",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	router.RegisterRoute(route)

	// Make a test request
	req := httptest.NewRequest("GET", "/auth/resource/details", nil)
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
	expected := []string{"plugin.auth", "plugin.verification"}
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
		"GET:/resource/details": {
			"plugins": []string{"plugin.auth"},
		},
		"POST:/resource/create": {
			"plugins": []string{"plugin.process"},
		},
	}
	router.SetRouteMetadataFromConfig(routeMetadata)

	// Track hook execution
	sessionHookCalled := false
	bearerHookCalled := false
	emailPasswordHookCalled := false

	// Register hooks with PluginID
	plugin1Hook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "plugin.auth",
		Handler: func(ctx *models.RequestContext) error {
			sessionHookCalled = true
			return nil
		},
		Order: 10,
	}

	plugin2Hook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "plugin.verification",
		Handler: func(ctx *models.RequestContext) error {
			bearerHookCalled = true
			return nil
		},
		Order: 5,
	}

	plugin3Hook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "plugin.process",
		Handler: func(ctx *models.RequestContext) error {
			emailPasswordHookCalled = true
			return nil
		},
		Order: 10,
	}

	router.RegisterHook(plugin1Hook)
	router.RegisterHook(plugin2Hook)
	router.RegisterHook(plugin3Hook)

	// Register test routes
	router.RegisterRoute(models.Route{
		Method:  "GET",
		Path:    "/resource/details",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	})

	router.RegisterRoute(models.Route{
		Method:  "POST",
		Path:    "/resource/create",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	})

	// Test 1: GET /auth/resource/details should only run plugin.auth hook (not plugin.verification)
	sessionHookCalled = false
	bearerHookCalled = false
	emailPasswordHookCalled = false

	req := httptest.NewRequest("GET", "/auth/resource/details", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !sessionHookCalled {
		t.Error("plugin.auth hook should have been called for /auth/resource/details")
	}
	if bearerHookCalled {
		t.Error("plugin.verification hook should NOT have been called for /auth/resource/details")
	}
	if emailPasswordHookCalled {
		t.Error("plugin.process hook should NOT have been called for /auth/resource/details")
	}

	// Test 2: POST /auth/resource/create should only run plugin.process hook
	sessionHookCalled = false
	bearerHookCalled = false
	emailPasswordHookCalled = false

	req = httptest.NewRequest("POST", "/auth/resource/create", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if sessionHookCalled {
		t.Error("plugin.auth hook should NOT have been called for /auth/resource/create")
	}
	if bearerHookCalled {
		t.Error("plugin.verification hook should NOT have been called for /auth/resource/create")
	}
	if !emailPasswordHookCalled {
		t.Error("plugin.process hook should have been called for /auth/resource/create")
	}
}

// TestRouteMetadataConversion verifies ConvertRouteMetadata utility function
func TestRouteMetadataConversion(t *testing.T) {
	routes := []models.RouteMapping{
		{
			Path:    "/resource/list",
			Method:  "GET",
			Plugins: []string{"plugin.auth", "plugin.verification"},
		},
		{
			Path:    "/resource/create",
			Method:  "POST",
			Plugins: []string{"plugin.process"},
		},
		{
			Path:    "/resource/update",
			Method:  "POST",
			Plugins: []string{"plugin.auth", "plugin.validation"},
		},
	}

	metadata, err := util.ConvertRouteMetadata(routes)
	if err != nil {
		t.Fatalf("ConvertRouteMetadata failed: %v", err)
	}

	// Test 1: Verify GET:/resource/list
	key := "GET:/resource/list"
	if _, exists := metadata[key]; !exists {
		t.Fatalf("expected key %s in metadata", key)
	}

	pluginIDs, ok := metadata[key]["plugins"].([]string)
	if !ok {
		t.Fatalf("plugins not found or wrong type for %s", key)
	}

	if len(pluginIDs) != 2 || pluginIDs[0] != "plugin.auth" || pluginIDs[1] != "plugin.verification" {
		t.Errorf("unexpected plugins for %s: %v", key, pluginIDs)
	}

	// Test 2: Verify POST:/resource/create
	key = "POST:/resource/create"
	if _, exists := metadata[key]; !exists {
		t.Fatalf("expected key %s in metadata", key)
	}

	pluginIDs, ok = metadata[key]["plugins"].([]string)
	if !ok {
		t.Fatalf("plugins not found or wrong type for %s", key)
	}

	if len(pluginIDs) != 1 || pluginIDs[0] != "plugin.process" {
		t.Errorf("unexpected plugins for %s: %v", key, pluginIDs)
	}

	// Test 3: Verify POST:/resource/update
	key = "POST:/resource/update"
	if _, exists := metadata[key]; !exists {
		t.Fatalf("expected key %s in metadata", key)
	}

	pluginIDs, ok = metadata[key]["plugins"].([]string)
	if !ok {
		t.Fatalf("plugins not found or wrong type for %s", key)
	}

	if len(pluginIDs) != 2 || pluginIDs[0] != "plugin.auth" || pluginIDs[1] != "plugin.validation" {
		t.Errorf("unexpected plugins for %s: %v", key, pluginIDs)
	}
}

func TestRouteMetadataFromConfigMergesWithRouteMetadata(t *testing.T) {
	config := &models.Config{
		BasePath: "/auth",
	}
	logger := &testLogger{}
	router := NewRouter(config, logger, nil)

	router.SetRouteMetadataFromConfig(map[string]map[string]any{
		"GET:/resource/action": {
			"plugins": []string{"plugin.primary"},
		},
	})

	router.RegisterRoute(models.Route{
		Method: http.MethodGet,
		Path:   "/resource/action",
		Metadata: map[string]any{
			"plugins": []string{"plugin.primary", "plugin.secondary"},
		},
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	})

	var capturedCtx *models.RequestContext
	router.RegisterHook(models.Hook{
		Stage: models.HookOnRequest,
		Handler: func(ctx *models.RequestContext) error {
			capturedCtx = ctx
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/resource/action", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if capturedCtx == nil || capturedCtx.Route == nil {
		t.Fatalf("expected captured route metadata")
	}

	plugins, ok := capturedCtx.Route.Metadata["plugins"].([]string)
	if !ok || len(plugins) != 2 || plugins[0] != "plugin.primary" || plugins[1] != "plugin.secondary" {
		t.Fatalf("expected merged plugins metadata, got %v", capturedCtx.Route.Metadata["plugins"])
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
		"POST:/auth/resource/{id}/action": {
			"plugins": []string{"plugin.process", "plugin.context"},
		},
		"GET:/auth/verify/{code}": {
			"plugins": []string{"plugin.verify"},
		},
		"POST:/auth/resource/{id}/action/complete": {
			"plugins": []string{"plugin.finalize"},
		},
	}
	router.SetRouteMetadataFromConfig(routeMetadata)

	// Track which hooks are called
	var hooksExecuted []string

	// Register hooks with plugin IDs
	plugin1Hook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "plugin.process",
		Handler: func(ctx *models.RequestContext) error {
			hooksExecuted = append(hooksExecuted, "plugin.process")
			return nil
		},
		Order: 10,
	}

	plugin2Hook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "plugin.context",
		Handler: func(ctx *models.RequestContext) error {
			hooksExecuted = append(hooksExecuted, "plugin.context")
			return nil
		},
		Order: 5,
	}

	plugin3Hook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "plugin.verify",
		Handler: func(ctx *models.RequestContext) error {
			hooksExecuted = append(hooksExecuted, "plugin.verify")
			return nil
		},
		Order: 0,
	}

	plugin4Hook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "plugin.finalize",
		Handler: func(ctx *models.RequestContext) error {
			hooksExecuted = append(hooksExecuted, "plugin.finalize")
			return nil
		},
		Order: 0,
	}

	router.RegisterHook(plugin1Hook)
	router.RegisterHook(plugin2Hook)
	router.RegisterHook(plugin3Hook)
	router.RegisterHook(plugin4Hook)

	// Register test routes
	router.RegisterRoute(models.Route{
		Method:  "POST",
		Path:    "/resource/{id}/action",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	})

	router.RegisterRoute(models.Route{
		Method:  "GET",
		Path:    "/verify/{code}",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	})

	router.RegisterRoute(models.Route{
		Method:  "POST",
		Path:    "/resource/{id}/action/complete",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	})

	// Test 1: POST /auth/resource/123/action should match the dynamic route pattern
	hooksExecuted = nil
	req := httptest.NewRequest("POST", "/auth/resource/123/action", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if len(hooksExecuted) != 2 {
		t.Errorf("expected 2 hooks to execute for POST /auth/resource/123/action, got %d: %v", len(hooksExecuted), hooksExecuted)
	}
	if len(hooksExecuted) >= 1 && hooksExecuted[0] != "plugin.context" {
		t.Errorf("expected plugin.context hook first (order 5), got %s", hooksExecuted[0])
	}
	if len(hooksExecuted) >= 2 && hooksExecuted[1] != "plugin.process" {
		t.Errorf("expected plugin.process hook second (order 10), got %s", hooksExecuted[1])
	}

	// Test 2: POST /auth/resource/456/action should also match the same pattern
	hooksExecuted = nil
	req = httptest.NewRequest("POST", "/auth/resource/456/action", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if len(hooksExecuted) != 2 {
		t.Errorf("expected 2 hooks to execute for POST /auth/resource/456/action, got %d: %v", len(hooksExecuted), hooksExecuted)
	}

	// Test 3: GET /auth/verify/token123 should match the plugin.verify route
	hooksExecuted = nil
	req = httptest.NewRequest("GET", "/auth/verify/token123", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if len(hooksExecuted) != 1 {
		t.Errorf("expected 1 hook to execute for GET /auth/verify/token123, got %d: %v", len(hooksExecuted), hooksExecuted)
	}
	if len(hooksExecuted) >= 1 && hooksExecuted[0] != "plugin.verify" {
		t.Errorf("expected plugin.verify hook, got %s", hooksExecuted[0])
	}

	// Test 4: POST /auth/resource/789/action/complete should match the four-segment pattern
	hooksExecuted = nil
	req = httptest.NewRequest("POST", "/auth/resource/789/action/complete", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if len(hooksExecuted) != 1 {
		t.Errorf("expected 1 hook to execute for POST /auth/resource/789/action/complete, got %d: %v", len(hooksExecuted), hooksExecuted)
	}
	if len(hooksExecuted) >= 1 && hooksExecuted[0] != "plugin.finalize" {
		t.Errorf("expected plugin.finalize hook, got %s", hooksExecuted[0])
	}

	// Test 5: Wrong method should not match (GET instead of POST)
	hooksExecuted = nil
	req = httptest.NewRequest("GET", "/auth/resource/999/action", nil)
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
		{"/resource/action/item", "/resource/action/item", true, "exact path match"},
		{"/api/data/123", "/api/data/123", true, "exact path with numbers"},

		// Dynamic parameter matches
		{"/resource/action/item", "/resource/action/{param}", true, "single dynamic parameter"},
		{"/resource/action/other", "/resource/action/{param}", true, "single dynamic parameter different value"},
		{"/verify/code123", "/verify/{code}", true, "dynamic code parameter"},
		{"/api/entities/123/items/456", "/api/entities/{id}/items/{itemId}", true, "multiple dynamic parameters"},
		{"/api/entities/name/status/active", "/api/entities/{name}/status/{state}", true, "multiple params with non-numeric values"},

		// Non-matches
		{"/resource/action", "/resource/action/{param}", false, "missing dynamic parameter"},
		{"/resource/action/item/extra", "/resource/action/{param}", false, "extra path segment"},
		{"/other/path", "/resource/action/{param}", false, "completely different path"},
		{"/resource/action/item", "/resource/other", false, "different static segments"},
		{"/api/entities/123/items", "/api/entities/{id}/items/{itemId}", false, "missing second dynamic parameter"},
	}

	for _, tc := range testCases {
		result := matchRoutePath(tc.requestPath, tc.pattern)
		if result != tc.shouldMatch {
			t.Errorf("%s: matchRoutePath(%q, %q) = %v, want %v",
				tc.description, tc.requestPath, tc.pattern, result, tc.shouldMatch)
		}
	}
}

// TestDynamicPathWithBasePathDebug tests the exact scenario: /api/auth/resource/123/action
// with route mapping of /resource/{id}/action and basePath of /api/auth
func TestDynamicPathWithBasePathDebug(t *testing.T) {
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &testLogger{}
	router := NewRouter(config, logger, nil)

	// Simulating what happens in auth.go Handler()
	// The config has "/resource/{id}/action" which gets the basePath prefixed
	// So it becomes "/api/auth/resource/{id}/action"
	routeMetadata := map[string]map[string]any{
		"POST:/api/auth/resource/{id}/action": {
			"plugins": []string{"plugin.process"},
		},
	}
	router.SetRouteMetadataFromConfig(routeMetadata)

	// Track hook execution
	var hooksExecuted []string

	processHook := models.Hook{
		Stage:    models.HookBefore,
		PluginID: "plugin.process",
		Handler: func(ctx *models.RequestContext) error {
			hooksExecuted = append(hooksExecuted, "plugin.process")
			if ctx.Route != nil {
				t.Logf("Route metadata: %v", ctx.Route.Metadata)
			}
			return nil
		},
		Order: 0,
	}

	router.RegisterHook(processHook)

	// Register the route
	router.RegisterRoute(models.Route{
		Method:  "POST",
		Path:    "/resource/{id}/action",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	})

	// Test: Request to /api/auth/resource/123/action
	req := httptest.NewRequest("POST", "/api/auth/resource/123/action", nil)
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

func (l *testLogger) Debug(msg string, args ...any) {}
func (l *testLogger) Info(msg string, args ...any)  {}
func (l *testLogger) Warn(msg string, args ...any)  {}
func (l *testLogger) Error(msg string, args ...any) {}

// TestDoubleBasePathApplication tests that basePath is not applied twice
func TestDoubleBasePathApplication(t *testing.T) {
	config := &models.Config{
		BasePath: "/api/auth",
	}
	logger := &testLogger{}
	router := NewRouter(config, logger, nil)

	// This simulates what auth.go does:
	// 1. Route mapping: "/resource/{id}/action"
	// 2. ConvertRouteMetadata produces: "POST:/resource/{id}/action"
	// 3. ApplyBasePathToMetadataKey produces: "POST:/api/auth/resource/{id}/action"
	// 4. SetRouteMetadataFromConfig is called with the adjusted key

	// So the key already has basePath applied
	routeMetadata := map[string]map[string]any{
		"POST:/api/auth/resource/{id}/action": {
			"plugins": []string{"plugin.process"},
		},
	}
	router.SetRouteMetadataFromConfig(routeMetadata)

	// The router should store this correctly
	expectedKey := "POST:/api/auth/resource/{id}/action"
	if _, exists := router.routeMetadata[expectedKey]; !exists {
		t.Errorf("expected key %s in routeMetadata, but got keys: %v", expectedKey, keysFromMap(router.routeMetadata))
	}

	// The segments should be correct
	if len(router.routeEntries) != 1 {
		t.Fatalf("expected 1 routeEntry, got %d", len(router.routeEntries))
	}

	entry := router.routeEntries[0]
	expectedSegments := []string{"api", "auth", "resource", "{id}", "action"}
	if len(entry.Segments) != len(expectedSegments) {
		t.Errorf("expected %d segments, got %d: %v", len(expectedSegments), len(entry.Segments), entry.Segments)
	}

	for i, seg := range entry.Segments {
		if seg != expectedSegments[i] {
			t.Errorf("segment %d: expected %q, got %q", i, expectedSegments[i], seg)
		}
	}
}

func TestRouteGroupMetadata(t *testing.T) {
	config := &models.Config{
		BasePath: "/",
	}
	logger := &testLogger{}
	router := NewRouter(config, logger, nil)

	router.RegisterCustomRouteGroup(
		models.RouteGroup{
			Path: "/resource",
			Routes: []models.Route{
				{Method: "GET", Path: "", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
				{Method: "GET", Path: "/details", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
					Metadata: map[string]any{
						"plugins": []string{
							"plugin.verification",
						},
					},
				},
			},
			Metadata: map[string]any{
				"plugins": []string{
					"plugin.auth",
				},
			},
		},
	)

	testPluginsAny := router.routeMetadata["GET:/resource"]["plugins"]
	testPlugins := testPluginsAny.([]string)

	if !slices.Contains(testPlugins, "plugin.auth") {
		t.Errorf("expected plugin.auth in plugins, got %s", testPlugins)
	}

	detailsPluginsAny := router.routeMetadata["GET:/resource/details"]["plugins"]
	detailsPlugins := detailsPluginsAny.([]string)

	if !slices.Contains(detailsPlugins, "plugin.auth") || !slices.Contains(detailsPlugins, "plugin.verification") {
		t.Errorf("expected plugin.auth and plugin.verification in plugins, got %s", detailsPlugins)
	}
}

func TestRouteGroupMetadataDuplicatePlugins(t *testing.T) {
	config := &models.Config{
		BasePath: "/",
	}
	logger := &testLogger{}
	router := NewRouter(config, logger, nil)

	router.RegisterCustomRouteGroup(
		models.RouteGroup{
			Path: "/resource",
			Routes: []models.Route{
				{Method: "GET", Path: "", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
					Metadata: map[string]any{
						"plugins": []string{
							"plugin.auth",
						},
					},
				},
			},
			Metadata: map[string]any{
				"plugins": []string{
					"plugin.auth",
				},
			},
		},
	)

	resourcePluginsAny := router.routeMetadata["GET:/resource"]["plugins"]
	resourcePlugins := resourcePluginsAny.([]string)

	if len(resourcePlugins) != 1 {
		t.Errorf("expected only one plugin, got %d %s", len(resourcePlugins), resourcePlugins)
	}
}

func keysFromMap(m map[string]map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
