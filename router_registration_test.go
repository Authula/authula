package authula

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalrouter "github.com/Authula/authula/internal/router"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

// failingResponseWriter always fails body writes so flush-error handling can be
// exercised deterministically.
type failingResponseWriter struct {
	header http.Header
}

func (w *failingResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}

	return w.header
}

func (w *failingResponseWriter) WriteHeader(int) {}

func (w *failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRouterGetReturnsUnderlyingRouter(t *testing.T) {
	t.Parallel()

	router := newTestRouter("/api/auth", nil)
	registerTestRoute(router, http.MethodGet, "/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	require.NotNil(t, router.Get())
	assert.Same(t, router.router, router.Get())

	rec := httptest.NewRecorder()
	router.Get().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/ping", nil))
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestRouterRegistrationSkipsDisabledRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   *models.Config
		register func(*Router, http.Handler)
		method   string
		path     string
	}{
		{
			name: "RegisterRoute",
			config: &models.Config{
				BasePath:      "/api/auth",
				DisabledPaths: []string{"GET:/api/auth/skip"},
			},
			register: func(router *Router, handler http.Handler) {
				router.RegisterRoute(models.Route{Method: http.MethodGet, Path: "/skip", Handler: handler})
			},
			method: http.MethodGet,
			path:   "/api/auth/skip",
		},
		{
			name: "RegisterRoutes",
			config: &models.Config{
				BasePath:      "/api/auth",
				DisabledPaths: []string{"GET:/api/auth/skip"},
			},
			register: func(router *Router, handler http.Handler) {
				router.RegisterRoutes([]models.Route{
					{Method: http.MethodGet, Path: "/skip", Handler: handler},
				})
			},
			method: http.MethodGet,
			path:   "/api/auth/skip",
		},
		{
			name: "RegisterCustomRoute",
			config: &models.Config{
				BasePath:      "/api/auth",
				DisabledPaths: []string{"GET:/health"},
			},
			register: func(router *Router, handler http.Handler) {
				router.RegisterCustomRoute(models.Route{Method: http.MethodGet, Path: "/health", Handler: handler})
			},
			method: http.MethodGet,
			path:   "/health",
		},
		{
			name: "RegisterCustomRoutes",
			config: &models.Config{
				BasePath:      "/api/auth",
				DisabledPaths: []string{"GET:/health"},
			},
			register: func(router *Router, handler http.Handler) {
				router.RegisterCustomRoutes([]models.Route{
					{Method: http.MethodGet, Path: "/health", Handler: handler},
				})
			},
			method: http.MethodGet,
			path:   "/health",
		},
		{
			name: "RegisterCustomRouteGroup",
			config: &models.Config{
				BasePath:      "/api/auth",
				DisabledPaths: []string{"GET:/group/skip"},
			},
			register: func(router *Router, handler http.Handler) {
				router.RegisterCustomRouteGroup(models.RouteGroup{
					Path: "/group",
					Routes: []models.Route{
						{Method: http.MethodGet, Path: "/skip", Handler: handler},
					},
				})
			},
			method: http.MethodGet,
			path:   "/group/skip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := newTestRouterWithConfig(tt.config, nil)
			called := false

			tt.register(router, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))

			rec := performRequest(router, tt.method, tt.path)

			assert.Equal(t, http.StatusNotFound, rec.Code)
			assert.False(t, called, "disabled route handler must not run")
		})
	}
}

func TestRouterRegisterRouteSupportsAllMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
	}{
		{name: "PUT", method: http.MethodPut},
		{name: "PATCH", method: http.MethodPatch},
		{name: "DELETE", method: http.MethodDelete},
		{name: "custom method fallback", method: http.MethodTrace},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := newTestRouter("", nil)
			called := false

			router.RegisterRoute(models.Route{
				Method: tt.method,
				Path:   "/resource",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					called = true
					w.WriteHeader(http.StatusNoContent)
				}),
			})

			rec := performRequest(router, tt.method, "/resource")

			assert.Equal(t, http.StatusNoContent, rec.Code)
			assert.True(t, called)
		})
	}
}

func TestRouterSetRouteMetadataFromConfigBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		resetMaps    bool
		inputs       []map[string]map[string]any
		wantEntries  int
		wantMetadata map[string]map[string]any
	}{
		{
			name:      "initializes nil metadata maps",
			resetMaps: true,
			inputs: []map[string]map[string]any{
				{"GET:/thing": {"plugins": []string{"plugin.one"}}},
			},
			wantEntries: 1,
			wantMetadata: map[string]map[string]any{
				"GET:/api/auth/thing": {"plugins": []string{"plugin.one"}},
			},
		},
		{
			name: "skips keys without a method separator",
			inputs: []map[string]map[string]any{
				{
					"malformed":  {"plugins": []string{"ignored"}},
					"GET:/valid": {"plugins": []string{"plugin.one"}},
				},
			},
			wantEntries: 1,
			wantMetadata: map[string]map[string]any{
				"GET:/api/auth/valid": {"plugins": []string{"plugin.one"}},
			},
		},
		{
			name: "merges repeated keys",
			inputs: []map[string]map[string]any{
				{
					"GET:/thing": {
						"plugins":     []string{"plugin.one"},
						"permissions": []string{"read"},
					},
				},
				{
					"GET:/api/auth/thing": {
						"roles": []string{"admin"},
					},
				},
			},
			wantEntries: 1,
			wantMetadata: map[string]map[string]any{
				"GET:/api/auth/thing": {
					"plugins":     []string{"plugin.one"},
					"permissions": []string{"read"},
					"roles":       []string{"admin"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := newTestRouter("/api/auth", nil)
			if tt.resetMaps {
				router.routeMetadata = nil
				router.routeEntries = nil
			}

			for _, input := range tt.inputs {
				router.SetRouteMetadataFromConfig(input)
			}

			assert.Equal(t, tt.wantMetadata, router.routeMetadata)
			assert.Len(t, router.routeEntries, tt.wantEntries)
		})
	}
}

func TestRouterRouteMetadataLookupBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		basePath    string
		storedKey   string
		requestPath string
		wantPattern string
	}{
		{
			name:        "root path resolves base-prefixed key",
			basePath:    "/api/auth",
			storedKey:   "GET:/api/auth",
			requestPath: "/",
			wantPattern: "GET:/api/auth",
		},
		{
			name:        "dynamic pattern matches after base prefixing",
			basePath:    "/api/auth",
			storedKey:   "GET:/api/auth/items/{id}",
			requestPath: "/items/42",
			wantPattern: "GET:/api/auth/items/{id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := newTestRouter(tt.basePath, nil)
			router.routeMetadata[tt.storedKey] = map[string]any{
				"plugins": []string{"items.plugin"},
			}

			var captured *models.Route
			router.RegisterHook(models.Hook{
				Stage: models.HookOnRequest,
				Handler: func(ctx *models.RequestContext) error {
					captured = ctx.Route
					return nil
				},
			})

			rec := performRequest(router, http.MethodGet, tt.requestPath)

			assert.Equal(t, http.StatusNotFound, rec.Code)
			require.NotNil(t, captured)
			assert.Equal(t, tt.wantPattern, captured.Pattern)
			assert.Equal(t, []string{"items.plugin"}, captured.Metadata["plugins"])
		})
	}

	t.Run("skips malformed stored keys", func(t *testing.T) {
		t.Parallel()

		router := newTestRouter("", nil)
		router.routeMetadata["malformed"] = map[string]any{
			"plugins": []string{"ignored"},
		}

		metadata, pattern, ok := router.getRouteMetadata(http.MethodGet, "/anything")

		assert.False(t, ok)
		assert.Empty(t, pattern)
		assert.Nil(t, metadata)
	})
}

func routerTestCORSConfig(origins []string, credentials bool, exposed []string) *models.Config {
	return &models.Config{
		Security: models.SecurityConfig{
			CORS: models.CORSConfig{
				AllowCredentials: credentials,
				AllowedOrigins:   origins,
				AllowedMethods:   []string{http.MethodGet, http.MethodPost},
				AllowedHeaders:   []string{"Content-Type", "Authorization"},
				ExposedHeaders:   exposed,
				MaxAge:           90 * time.Second,
			},
		},
	}
}

func TestRouterCORSHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		config        *models.Config
		method        string
		origin        string
		wantStatus    int
		wantHeaders   map[string]string
		absentHeaders []string
	}{
		{
			name:       "wildcard origin echoes the request origin",
			config:     routerTestCORSConfig([]string{"*"}, false, nil),
			method:     http.MethodGet,
			origin:     "https://any.example",
			wantStatus: http.StatusNoContent,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://any.example",
			},
		},
		{
			name:       "suffix wildcard matches subdomain",
			config:     routerTestCORSConfig([]string{"*.example.com"}, false, nil),
			method:     http.MethodGet,
			origin:     "https://api.example.com",
			wantStatus: http.StatusNoContent,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://api.example.com",
			},
		},
		{
			name:       "allowed origin with credentials and exposed headers",
			config:     routerTestCORSConfig([]string{"https://app.example.com"}, true, []string{"X-Total-Count", "X-Request-Id"}),
			method:     http.MethodGet,
			origin:     "https://app.example.com",
			wantStatus: http.StatusNoContent,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "https://app.example.com",
				"Access-Control-Allow-Credentials": "true",
				"Access-Control-Expose-Headers":    "X-Total-Count, X-Request-Id",
			},
		},
		{
			name:          "credentials with wildcard origin is rejected",
			config:        routerTestCORSConfig([]string{"*"}, true, nil),
			method:        http.MethodGet,
			origin:        "https://any.example",
			wantStatus:    http.StatusNoContent,
			absentHeaders: []string{"Access-Control-Allow-Origin", "Access-Control-Allow-Credentials"},
		},
		{
			name:       "disallowed origin on preflight advertises methods",
			config:     routerTestCORSConfig([]string{"https://allowed.example"}, false, nil),
			method:     http.MethodOptions,
			origin:     "https://evil.example",
			wantStatus: http.StatusOK,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Methods": "GET, POST",
				"Access-Control-Allow-Headers": "Content-Type, Authorization",
				"Access-Control-Max-Age":       "90",
			},
			absentHeaders: []string{"Access-Control-Allow-Origin"},
		},
		{
			name:          "suffix wildcard does not match lookalike domain",
			config:        routerTestCORSConfig([]string{"*.example.com"}, false, nil),
			method:        http.MethodGet,
			origin:        "https://example.com.evil.test",
			wantStatus:    http.StatusNoContent,
			absentHeaders: []string{"Access-Control-Allow-Origin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := newTestRouterWithConfig(tt.config, nil)
			registerTestRoute(router, http.MethodGet, "/resource", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})

			rec := performRequestWithHeaders(router, tt.method, "/resource", map[string]string{"Origin": tt.origin})

			assert.Equal(t, tt.wantStatus, rec.Code)
			for key, want := range tt.wantHeaders {
				assert.Equal(t, want, rec.Header().Get(key), key)
			}
			for _, key := range tt.absentHeaders {
				assert.Empty(t, rec.Header().Get(key), key)
			}
			assert.Contains(t, rec.Header().Values("Vary"), "Origin")
		})
	}
}

func TestRouterRedirectDefaultsToFound(t *testing.T) {
	t.Parallel()

	router := newTestRouter("/api/auth", nil)
	const redirectURL = "https://example.com/target"

	registerTestRoute(router, http.MethodGet, "/redirect", func(_ http.ResponseWriter, r *http.Request) {
		ctx, ok := models.GetRequestContext(r.Context())
		require.True(t, ok)
		ctx.RedirectURL = redirectURL
	})

	rec := performRequest(router, http.MethodGet, "/api/auth/redirect")

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, redirectURL, rec.Header().Get("Location"))
}

func TestRouterFlushErrorIsLogged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		setup  func(*testing.T, *Router)
	}{
		{
			name:   "finalized route response",
			method: http.MethodGet,
			path:   "/api/auth/fail",
			setup: func(t *testing.T, router *Router) {
				registerTestRoute(router, http.MethodGet, "/fail", func(_ http.ResponseWriter, r *http.Request) {
					ctx, ok := models.GetRequestContext(r.Context())
					require.True(t, ok)
					ctx.SetResponse(http.StatusOK, nil, []byte("body"))
				})
			},
		},
		{
			name:   "preflight completion response",
			method: http.MethodOptions,
			path:   "/api/auth/probe",
			setup: func(_ *testing.T, router *Router) {
				router.RegisterHook(models.Hook{
					Stage: models.HookOnComplete,
					Handler: func(ctx *models.RequestContext) error {
						ctx.SetResponse(http.StatusOK, nil, []byte("preflight"))
						if writer, ok := ctx.ResponseWriter.(*internalrouter.DeferredResponseWriter); ok {
							writer.OverrideWithContext(ctx)
						}
						return nil
					},
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := &internaltests.LevelRecordingLogger{}
			router := NewRouter(&models.Config{BasePath: "/api/auth"}, logger, nil)
			tt.setup(t, router)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			router.ServeHTTP(&failingResponseWriter{}, req)

			assert.Contains(t, logger.Messages(), "Failed to flush response")
		})
	}
}

func TestRouterFallsBackWhenClientIPIsInvalid(t *testing.T) {
	t.Parallel()

	logger := &internaltests.LevelRecordingLogger{}
	router := NewRouter(&models.Config{BasePath: "/api/auth"}, logger, nil)

	var clientIP string
	router.RegisterHook(models.Hook{
		Stage: models.HookOnRequest,
		Handler: func(ctx *models.RequestContext) error {
			clientIP = ctx.ClientIP
			return nil
		},
	})
	registerTestRoute(router, http.MethodGet, "/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/auth/ping", nil)
	req.RemoteAddr = "not-an-ip"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "0.0.0.0", clientIP)
	assert.Contains(t, logger.Messages(), "Failed to extract client IP")
}

func TestRouterHooksSkipWhenRouteIsMissing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(*Router, *models.RequestContext) bool
	}{
		{
			name: "on request hook",
			run: func(router *Router, ctx *models.RequestContext) bool {
				ran := false
				router.RegisterHook(models.Hook{
					Stage:    models.HookOnRequest,
					PluginID: "plugin.one",
					Handler: func(*models.RequestContext) error {
						ran = true
						return nil
					},
				})
				router.runHooks(models.HookOnRequest, ctx)
				return ran
			},
		},
		{
			name: "completion hook",
			run: func(router *Router, ctx *models.RequestContext) bool {
				ran := false
				router.RegisterHook(models.Hook{
					Stage:    models.HookOnComplete,
					PluginID: "plugin.one",
					Handler: func(*models.RequestContext) error {
						ran = true
						return nil
					},
				})
				router.runCompletionHooks(ctx)
				return ran
			},
		},
		{
			name: "bypass hook",
			run: func(router *Router, ctx *models.RequestContext) bool {
				ran := false
				router.RegisterHook(models.Hook{
					Stage:       models.HookOnRequest,
					PluginID:    "plugin.one",
					RunOnBypass: true,
					Handler: func(*models.RequestContext) error {
						ran = true
						return nil
					},
				})
				router.runBypassHooks(models.HookOnRequest, ctx)
				return ran
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := newTestRouter("/api/auth", nil)
			ctx := &models.RequestContext{
				Request: httptest.NewRequest(http.MethodGet, "/", nil),
				Values:  make(map[string]any),
			}

			assert.False(t, tt.run(router, ctx))
		})
	}
}

func TestRouterCompletionHookMatcherAndError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		matcher    models.HookMatcher
		handlerErr error
		wantRan    bool
		wantLog    string
	}{
		{
			name:    "matcher false skips the hook",
			matcher: func(*models.RequestContext) bool { return false },
			wantRan: false,
		},
		{
			name:       "handler error is logged",
			handlerErr: errors.New("completion failed"),
			wantRan:    true,
			wantLog:    "Hook handler error (sync)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := &internaltests.LevelRecordingLogger{}
			router := NewRouter(&models.Config{BasePath: "/api/auth"}, logger, nil)
			ran := false

			router.RegisterHook(models.Hook{
				Stage:   models.HookOnComplete,
				Matcher: tt.matcher,
				Handler: func(*models.RequestContext) error {
					ran = true
					return tt.handlerErr
				},
			})
			registerTestRoute(router, http.MethodGet, "/test", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			rec := performRequest(router, http.MethodGet, "/api/auth/test")

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.wantRan, ran)
			if tt.wantLog != "" {
				assert.Contains(t, logger.Messages(), tt.wantLog)
			}
		})
	}
}

func TestRouterBypassHookFiltering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pluginID   string
		matcher    models.HookMatcher
		handlerErr error
		wantRuns   int
		wantLog    string
	}{
		{
			name:     "hook for another plugin does not run",
			pluginID: "plugin.other",
			wantRuns: 0,
		},
		{
			name:     "hook rejected by its matcher does not run",
			pluginID: "plugin.allowed",
			matcher:  func(*models.RequestContext) bool { return false },
			wantRuns: 0,
		},
		{
			name:       "matching bypass hook runs and its error is logged",
			pluginID:   "plugin.allowed",
			handlerErr: errors.New("bypass failed"),
			wantRuns:   1,
			wantLog:    "Hook handler error (sync)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := &internaltests.LevelRecordingLogger{}
			router := NewRouter(&models.Config{BasePath: "/api/auth"}, logger, nil)
			router.SetRouteMetadataFromConfig(map[string]map[string]any{
				"OPTIONS:/api/auth/probe": {"plugins": []string{"plugin.allowed"}},
			})

			runs := 0
			router.RegisterHook(models.Hook{
				Stage:       models.HookOnRequest,
				RunOnBypass: true,
				PluginID:    tt.pluginID,
				Matcher:     tt.matcher,
				Handler: func(*models.RequestContext) error {
					runs++
					return tt.handlerErr
				},
			})

			rec := performRequest(router, http.MethodOptions, "/api/auth/probe")

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.wantRuns, runs)
			if tt.wantLog != "" {
				assert.Contains(t, logger.Messages(), tt.wantLog)
			}
		})
	}
}

func TestRouterAsyncHookErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		timeout     time.Duration
		handler     models.HookHandler
		wantMessage string
	}{
		{
			name: "timeout is logged",
			// A zero timeout expires the async context immediately, so the
			// handler observes DeadlineExceeded without any wall-clock wait.
			timeout: 0,
			handler: func(ctx *models.RequestContext) error {
				<-ctx.Request.Context().Done()
				return errors.New("timed out")
			},
			wantMessage: "Async hook timeout",
		},
		{
			name:    "immediate error is logged",
			timeout: time.Second,
			handler: func(*models.RequestContext) error {
				return errors.New("async failed")
			},
			wantMessage: "Hook handler error (async)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				logger := &internaltests.LevelRecordingLogger{}
				router := NewRouter(&models.Config{BasePath: "/api/auth"}, logger, &RouterOptions{
					AsyncHookTimeout: tt.timeout,
					HookErrorMode:    HookErrorModeContinue,
				})

				started := make(chan struct{})
				router.RegisterHook(models.Hook{
					Stage: models.HookOnResponse,
					Async: true,
					Handler: func(ctx *models.RequestContext) error {
						close(started)
						return tt.handler(ctx)
					},
				})
				registerTestRoute(router, http.MethodGet, "/async", func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				rec := performRequest(router, http.MethodGet, "/api/auth/async")
				<-started
				synctest.Wait()

				assert.Equal(t, http.StatusOK, rec.Code)
				assert.Contains(t, logger.Messages(), tt.wantMessage)
			})
		})
	}
}

func TestMatchRoutePathWildcardAndRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request string
		pattern string
		want    bool
	}{
		{name: "root matches root", request: "/", pattern: "/", want: true},
		{name: "wildcard matches empty remainder", request: "/foo", pattern: "/foo/*", want: true},
		{name: "wildcard matches one trailing segment", request: "/foo/bar", pattern: "/foo/*", want: true},
		{name: "wildcard matches many trailing segments", request: "/foo/bar/baz", pattern: "/foo/*", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, matchRoutePath(tt.request, tt.pattern))
		})
	}
}
