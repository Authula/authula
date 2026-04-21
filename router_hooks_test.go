package authula

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

func newTestRouter(basePath string, opts *RouterOptions) *Router {
	return newTestRouterWithConfig(&models.Config{BasePath: basePath}, opts)
}

func newTestRouterWithConfig(config *models.Config, opts *RouterOptions) *Router {
	if config == nil {
		config = &models.Config{}
	}

	return NewRouter(config, &tests.MockLogger{}, opts)
}

func registerTestRoute(router *Router, method, path string, handler func(http.ResponseWriter, *http.Request)) {
	router.RegisterRoute(models.Route{
		Method:  method,
		Path:    path,
		Handler: http.HandlerFunc(handler),
	})
}

func performRequest(router *Router, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func assertHookStages(t *testing.T, got, want []models.HookStage) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d stages, got %d", len(want), len(got))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stage %d: expected %d, got %d", i, want[i], got[i])
		}
	}
}

func assertIntSlice(t *testing.T, got, want []int) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d values, got %d", len(want), len(got))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("value %d: expected %d, got %d", i, want[i], got[i])
		}
	}
}

func TestRouterHookLifecycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "executes stages in order",
			run: func(t *testing.T) {
				router := newTestRouter("/api/auth", nil)
				executedStages := make([]models.HookStage, 0, 4)

				for _, stage := range []models.HookStage{
					models.HookOnRequest,
					models.HookBefore,
					models.HookAfter,
					models.HookOnResponse,
				} {
					router.RegisterHook(models.Hook{
						Stage: stage,
						Handler: func(ctx *models.RequestContext) error {
							executedStages = append(executedStages, stage)
							return nil
						},
					})
				}

				registerTestRoute(router, http.MethodGet, "/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				w := performRequest(router, http.MethodGet, "/api/auth/test")
				if w.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
				}

				assertHookStages(t, executedStages, []models.HookStage{
					models.HookOnRequest,
					models.HookBefore,
					models.HookAfter,
					models.HookOnResponse,
				})
			},
		},
		{
			name: "handled flag stops further processing",
			run: func(t *testing.T) {
				router := newTestRouter("/api/auth", nil)
				executedHooks := 0

				router.RegisterHook(models.Hook{
					Stage: models.HookBefore,
					Handler: func(ctx *models.RequestContext) error {
						executedHooks++
						ctx.Handled = true
						ctx.ResponseWriter.WriteHeader(http.StatusForbidden)
						return nil
					},
				})

				router.RegisterHook(models.Hook{
					Stage: models.HookBefore,
					Order: 1,
					Handler: func(ctx *models.RequestContext) error {
						executedHooks++
						return nil
					},
				})

				registerTestRoute(router, http.MethodGet, "/test", func(w http.ResponseWriter, r *http.Request) {
					executedHooks++
					w.WriteHeader(http.StatusOK)
				})

				w := performRequest(router, http.MethodGet, "/api/auth/test")
				if executedHooks != 1 {
					t.Fatalf("expected 1 hook execution, got %d", executedHooks)
				}

				if w.Code != http.StatusForbidden {
					t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
				}
			},
		},
		{
			name: "matcher controls hook execution",
			run: func(t *testing.T) {
				router := newTestRouter("/api/auth", nil)
				executedHooks := 0

				router.RegisterHook(models.Hook{
					Stage: models.HookBefore,
					Matcher: func(ctx *models.RequestContext) bool {
						return ctx.Path == "/api/auth/admin"
					},
					Handler: func(ctx *models.RequestContext) error {
						executedHooks++
						return nil
					},
				})

				registerTestRoute(router, http.MethodGet, "/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				w := performRequest(router, http.MethodGet, "/api/auth/test")
				if w.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
				}

				if executedHooks != 0 {
					t.Fatalf("hook should not execute for non-matching path, got %d", executedHooks)
				}

				registerTestRoute(router, http.MethodGet, "/admin", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				w = performRequest(router, http.MethodGet, "/api/auth/admin")
				if w.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
				}

				if executedHooks != 1 {
					t.Fatalf("hook should execute for matching path, got %d", executedHooks)
				}
			},
		},
		{
			name: "hooks execute in order",
			run: func(t *testing.T) {
				router := newTestRouter("/api/auth", nil)
				executionOrder := make([]int, 0, 3)

				for _, order := range []int{2, 0, 1} {
					router.RegisterHook(models.Hook{
						Stage: models.HookBefore,
						Order: order,
						Handler: func(ctx *models.RequestContext) error {
							executionOrder = append(executionOrder, order)
							return nil
						},
					})
				}

				registerTestRoute(router, http.MethodGet, "/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				w := performRequest(router, http.MethodGet, "/api/auth/test")
				if w.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
				}

				assertIntSlice(t, executionOrder, []int{0, 1, 2})
			},
		},
		{
			name: "request context values flow through hooks",
			run: func(t *testing.T) {
				router := newTestRouter("/api/auth", nil)
				hookReadValue := ""

				router.RegisterHook(models.Hook{
					Stage: models.HookBefore,
					Handler: func(ctx *models.RequestContext) error {
						ctx.Values["user_id"] = "12345"
						return nil
					},
				})

				router.RegisterHook(models.Hook{
					Stage: models.HookBefore,
					Order: 1,
					Handler: func(ctx *models.RequestContext) error {
						if val, ok := ctx.Values["user_id"]; ok {
							hookReadValue = val.(string)
						}
						return nil
					},
				})

				registerTestRoute(router, http.MethodGet, "/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				w := performRequest(router, http.MethodGet, "/api/auth/test")
				if w.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
				}

				if hookReadValue != "12345" {
					t.Fatalf("expected hook to read value %q, got %q", "12345", hookReadValue)
				}
			},
		},
		{
			name: "panic recovery allows later hook to run",
			run: func(t *testing.T) {
				router := newTestRouter("/api/auth", nil)
				hookExecutedAfterPanic := false

				router.RegisterHook(models.Hook{
					Stage: models.HookBefore,
					Handler: func(ctx *models.RequestContext) error {
						panic("hook panic test")
					},
				})

				router.RegisterHook(models.Hook{
					Stage: models.HookBefore,
					Order: 1,
					Handler: func(ctx *models.RequestContext) error {
						hookExecutedAfterPanic = true
						return nil
					},
				})

				registerTestRoute(router, http.MethodGet, "/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				w := performRequest(router, http.MethodGet, "/api/auth/test")
				if w.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
				}

				if !hookExecutedAfterPanic {
					t.Fatal("expected second hook to execute after panic recovery")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestRouterDisabledPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "wildcard disables organizations route group across methods",
			run: func(t *testing.T) {
				router := newTestRouterWithConfig(&models.Config{
					BasePath: "/api/auth",
					DisabledPaths: []string{
						"/organizations/*",
					},
				}, nil)

				hooksCalled := 0
				handlerCalled := false

				router.RegisterHook(models.Hook{
					Stage: models.HookOnRequest,
					Handler: func(ctx *models.RequestContext) error {
						hooksCalled++
						return nil
					},
				})

				router.RegisterRoute(models.Route{
					Method: http.MethodGet,
					Path:   "/organizations",
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						handlerCalled = true
						w.WriteHeader(http.StatusOK)
					}),
				})

				router.RegisterRoute(models.Route{
					Method: http.MethodPost,
					Path:   "/organizations",
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						handlerCalled = true
						w.WriteHeader(http.StatusCreated)
					}),
				})

				for _, request := range []struct {
					method string
					path   string
				}{
					{method: http.MethodGet, path: "/api/auth/organizations"},
					{method: http.MethodPost, path: "/api/auth/organizations"},
				} {
					w := performRequest(router, request.method, request.path)
					if w.Code != http.StatusNotFound {
						t.Fatalf("expected status %d for %s, got %d", http.StatusNotFound, request.method, w.Code)
					}
				}

				if hooksCalled != 0 {
					t.Fatalf("expected wildcard disabled path to skip hooks, got %d hook executions", hooksCalled)
				}

				if handlerCalled {
					t.Fatal("expected wildcard disabled path to skip route handler execution")
				}
			},
		},
		{
			name: "exact method disable preserves other methods",
			run: func(t *testing.T) {
				router := newTestRouterWithConfig(&models.Config{
					BasePath: "/api/auth",
					DisabledPaths: []string{
						"GET:/organizations",
					},
				}, nil)

				hookCount := 0
				getCalled := false
				postCalled := false

				router.RegisterHook(models.Hook{
					Stage: models.HookOnRequest,
					Handler: func(ctx *models.RequestContext) error {
						hookCount++
						return nil
					},
				})

				router.RegisterRoute(models.Route{
					Method: http.MethodGet,
					Path:   "/organizations",
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						getCalled = true
						w.WriteHeader(http.StatusOK)
					}),
				})

				router.RegisterRoute(models.Route{
					Method: http.MethodPost,
					Path:   "/organizations",
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						postCalled = true
						w.WriteHeader(http.StatusCreated)
					}),
				})

				w := performRequest(router, http.MethodGet, "/api/auth/organizations")
				if w.Code != http.StatusNotFound {
					t.Fatalf("expected GET status %d, got %d", http.StatusNotFound, w.Code)
				}

				if hookCount != 0 {
					t.Fatalf("expected disabled GET path to skip hooks, got %d hook executions", hookCount)
				}

				if getCalled {
					t.Fatal("expected disabled GET route handler to remain uncalled")
				}

				w = performRequest(router, http.MethodPost, "/api/auth/organizations")
				if w.Code != http.StatusCreated {
					t.Fatalf("expected POST status %d, got %d", http.StatusCreated, w.Code)
				}

				if hookCount != 1 {
					t.Fatalf("expected POST request to reach hooks, got %d hook executions", hookCount)
				}

				if !postCalled {
					t.Fatal("expected enabled POST route handler to run")
				}
			},
		},
		{
			name: "nearby path remains enabled",
			run: func(t *testing.T) {
				router := newTestRouterWithConfig(&models.Config{
					BasePath: "/api/auth",
					DisabledPaths: []string{
						"/organizations/*",
					},
				}, nil)

				hookCount := 0
				handlerCalled := false

				router.RegisterHook(models.Hook{
					Stage: models.HookOnRequest,
					Handler: func(ctx *models.RequestContext) error {
						hookCount++
						return nil
					},
				})

				router.RegisterRoute(models.Route{
					Method: http.MethodGet,
					Path:   "/organization",
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						handlerCalled = true
						w.WriteHeader(http.StatusOK)
					}),
				})

				w := performRequest(router, http.MethodGet, "/api/auth/organization")
				if w.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
				}

				if hookCount != 1 {
					t.Fatalf("expected enabled path to reach hooks, got %d hook executions", hookCount)
				}

				if !handlerCalled {
					t.Fatal("expected nearby enabled route handler to run")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestRouterHookErrorModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "fail fast stops remaining hooks",
			run: func(t *testing.T) {
				router := newTestRouterWithConfig(&models.Config{BasePath: "/api/auth"}, &RouterOptions{
					HookErrorMode: HookErrorModeFailFast,
				})

				hookExecutionOrder := make([]int, 0, 1)

				router.RegisterHook(models.Hook{
					Stage: models.HookBefore,
					Order: 1,
					Handler: func(ctx *models.RequestContext) error {
						hookExecutionOrder = append(hookExecutionOrder, 1)
						return fmt.Errorf("hook error")
					},
				})

				router.RegisterHook(models.Hook{
					Stage: models.HookBefore,
					Order: 2,
					Handler: func(ctx *models.RequestContext) error {
						hookExecutionOrder = append(hookExecutionOrder, 2)
						return nil
					},
				})

				router.RegisterHook(models.Hook{
					Stage: models.HookAfter,
					Order: 1,
					Handler: func(ctx *models.RequestContext) error {
						hookExecutionOrder = append(hookExecutionOrder, 3)
						return nil
					},
				})

				registerTestRoute(router, http.MethodGet, "/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				w := performRequest(router, http.MethodGet, "/api/auth/test")
				if w.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
				}

				assertIntSlice(t, hookExecutionOrder, []int{1})
			},
		},
		{
			name: "silent mode ignores errors",
			run: func(t *testing.T) {
				router := newTestRouterWithConfig(&models.Config{BasePath: "/api/auth"}, &RouterOptions{
					HookErrorMode: HookErrorModeSilent,
				})

				hookExecutionOrder := make([]int, 0, 2)

				router.RegisterHook(models.Hook{
					Stage: models.HookBefore,
					Order: 1,
					Handler: func(ctx *models.RequestContext) error {
						hookExecutionOrder = append(hookExecutionOrder, 1)
						return fmt.Errorf("hook error - should be silent")
					},
				})

				router.RegisterHook(models.Hook{
					Stage: models.HookBefore,
					Order: 2,
					Handler: func(ctx *models.RequestContext) error {
						hookExecutionOrder = append(hookExecutionOrder, 2)
						return nil
					},
				})

				registerTestRoute(router, http.MethodGet, "/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				w := performRequest(router, http.MethodGet, "/api/auth/test")
				if w.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
				}

				assertIntSlice(t, hookExecutionOrder, []int{1, 2})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestRouterAsyncHookTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "times out asynchronously without failing the request",
			timeout: 100 * time.Millisecond,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				router := newTestRouterWithConfig(&models.Config{BasePath: "/api/auth"}, &RouterOptions{
					AsyncHookTimeout: tc.timeout,
				})

				router.RegisterHook(models.Hook{
					Stage: models.HookOnResponse,
					Async: true,
					Handler: func(ctx *models.RequestContext) error {
						select {
						case <-ctx.Request.Context().Done():
							return nil
						case <-time.After(500 * time.Millisecond):
							return nil
						}
					},
				})

				registerTestRoute(router, http.MethodGet, "/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				w := performRequest(router, http.MethodGet, "/api/auth/test")
				time.Sleep(600 * time.Millisecond)
				synctest.Wait()

				if w.Code != http.StatusOK {
					t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
				}
			})
		})
	}
}

func TestRouterRedirectHandledRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		redirectURL string
		statusCode  int
	}{
		{
			name:        "redirect url is honored even when handled",
			redirectURL: "https://example.com/callback",
			statusCode:  http.StatusFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter("/api/auth", nil)

			router.RegisterRoute(models.Route{
				Method: http.MethodGet,
				Path:   "/test",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					reqCtx, ok := models.GetRequestContext(r.Context())
					if !ok {
						t.Fatal("expected request context")
					}
					reqCtx.RedirectURL = tc.redirectURL
					reqCtx.ResponseStatus = tc.statusCode
					reqCtx.Handled = true
				}),
			})

			w := performRequest(router, http.MethodGet, "/api/auth/test")
			if w.Code != tc.statusCode {
				t.Fatalf("expected status %d, got %d", tc.statusCode, w.Code)
			}

			if location := w.Header().Get("Location"); location != tc.redirectURL {
				t.Fatalf("expected Location header %q, got %q", tc.redirectURL, location)
			}
		})
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
	if _, err := w.Write([]byte(h.body)); err != nil {
		fmt.Printf("failed to write response body: %v\n", err)
	}
}
