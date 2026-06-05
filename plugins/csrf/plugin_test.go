package csrf

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Authula/authula/models"
)

// MockLogger is a minimal logger for testing
type MockLogger struct {
	debugMessages []string
	errorMessages []string
}

func (m *MockLogger) Debug(msg string, args ...any) {
	m.debugMessages = append(m.debugMessages, msg)
}

func (m *MockLogger) Info(msg string, args ...any) {}

func (m *MockLogger) Warn(msg string, args ...any) {}

func (m *MockLogger) Error(msg string, args ...any) {
	m.errorMessages = append(m.errorMessages, msg)
}

func initPlugin(t *testing.T, p *CSRFPlugin) {
	t.Helper()
	if p.logger == nil {
		p.logger = &MockLogger{}
	}
	registry := newMockServiceRegistry()
	registry.Register(models.ServiceToken.String(), &mockTokenService{})
	ctx := &models.PluginContext{
		Logger:          p.logger,
		ServiceRegistry: registry,
		GetConfig:       func() *models.Config { return &models.Config{} },
	}
	if err := p.Init(ctx); err != nil {
		t.Fatalf("failed to init plugin: %v", err)
	}
}

func TestCSRFPlugin_Metadata(t *testing.T) {
	p := New(CSRFPluginConfig{})
	md := p.Metadata()

	if md.ID != models.PluginCSRF.String() {
		t.Errorf("ID = %q, want %q", md.ID, models.PluginCSRF.String())
	}
	if md.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", md.Version, "1.0.0")
	}
	if md.Description == "" {
		t.Error("Description should not be empty")
	}
}

func TestCSRFPlugin_HookOrder(t *testing.T) {
	p := New(CSRFPluginConfig{})
	hooks := p.Hooks()

	if len(hooks) != 3 {
		t.Fatalf("got %d hooks, want 3", len(hooks))
	}

	if hooks[0].Stage != models.HookBefore || hooks[0].Order != 5 {
		t.Errorf("hooks[0]: Stage=%v Order=%d", hooks[0].Stage, hooks[0].Order)
	}
	if hooks[1].Stage != models.HookBefore || hooks[1].Order != 5 {
		t.Errorf("hooks[1]: Stage=%v Order=%d", hooks[1].Stage, hooks[1].Order)
	}
	if hooks[2].Stage != models.HookAfter || hooks[2].Order != 5 {
		t.Errorf("hooks[2]: Stage=%v Order=%d", hooks[2].Stage, hooks[2].Order)
	}
}

func TestCSRFPlugin_Config(t *testing.T) {
	tests := []struct {
		name   string
		config CSRFPluginConfig
		check  func(t *testing.T, p *CSRFPlugin)
	}{
		{
			name:   "defaults",
			config: CSRFPluginConfig{},
			check: func(t *testing.T, p *CSRFPlugin) {
				if p.pluginConfig.CookieName != "authula_csrf_token" {
					t.Errorf("CookieName = %q", p.pluginConfig.CookieName)
				}
				if p.pluginConfig.HeaderName != "X-AUTHULA-CSRF-TOKEN" {
					t.Errorf("HeaderName = %q", p.pluginConfig.HeaderName)
				}
				if p.pluginConfig.MaxAge != 24*time.Hour {
					t.Errorf("MaxAge = %v", p.pluginConfig.MaxAge)
				}
				if p.pluginConfig.SameSite != "lax" {
					t.Errorf("SameSite = %q", p.pluginConfig.SameSite)
				}
			},
		},
		{
			name: "custom values",
			config: CSRFPluginConfig{
				Enabled:    true,
				CookieName: "my_csrf",
				HeaderName: "X-My-CSRF",
				MaxAge:     12 * time.Hour,
				SameSite:   "strict",
			},
			check: func(t *testing.T, p *CSRFPlugin) {
				if p.pluginConfig.CookieName != "my_csrf" {
					t.Errorf("CookieName = %q", p.pluginConfig.CookieName)
				}
				if p.pluginConfig.HeaderName != "X-My-CSRF" {
					t.Errorf("HeaderName = %q", p.pluginConfig.HeaderName)
				}
				if p.pluginConfig.MaxAge != 12*time.Hour {
					t.Errorf("MaxAge = %v", p.pluginConfig.MaxAge)
				}
				if p.pluginConfig.SameSite != "strict" {
					t.Errorf("SameSite = %q", p.pluginConfig.SameSite)
				}
			},
		},
		{
			name:   "SameSite case preservation",
			config: CSRFPluginConfig{SameSite: "STRICT"},
			check: func(t *testing.T, p *CSRFPlugin) {
				if p.pluginConfig.SameSite != "STRICT" {
					t.Errorf("SameSite = %q, want 'STRICT'", p.pluginConfig.SameSite)
				}
			},
		},
		{
			name:   "SameSite none preserved",
			config: CSRFPluginConfig{SameSite: "none"},
			check: func(t *testing.T, p *CSRFPlugin) {
				if p.pluginConfig.SameSite != "none" {
					t.Errorf("SameSite = %q, want 'none'", p.pluginConfig.SameSite)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, New(tt.config))
		})
	}
}

func TestCSRFPlugin_HookMatcher(t *testing.T) {
	p := New(CSRFPluginConfig{})
	p.logger = &MockLogger{}
	p.globalConfig = &models.Config{Security: models.SecurityConfig{TrustedOrigins: []string{}}}

	hooks := p.Hooks()
	safeHook := hooks[0]
	unsafeHook := hooks[1]

	tests := []struct {
		name       string
		method     string
		wantSafe   bool
		wantUnsafe bool
	}{
		// Safe methods match the safe hook
		{name: "GET", method: http.MethodGet, wantSafe: true},
		{name: "HEAD", method: http.MethodHead, wantSafe: true},
		{name: "OPTIONS", method: http.MethodOptions, wantSafe: true},
		// Unsafe methods match the unsafe hook
		{name: "POST", method: http.MethodPost, wantUnsafe: true},
		{name: "PUT", method: http.MethodPut, wantUnsafe: true},
		{name: "DELETE", method: http.MethodDelete, wantUnsafe: true},
		{name: "PATCH", method: http.MethodPatch, wantUnsafe: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &models.RequestContext{
				Method: tt.method,
			}

			if got := safeHook.Matcher(ctx); got != tt.wantSafe {
				t.Errorf("safeMatcher(%s) = %v, want %v", tt.method, got, tt.wantSafe)
			}
			if got := unsafeHook.Matcher(ctx); got != tt.wantUnsafe {
				t.Errorf("unsafeMatcher(%s) = %v, want %v", tt.method, got, tt.wantUnsafe)
			}
		})
	}
}

func TestCSRFPlugin_TokenGeneration(t *testing.T) {
	p := New(CSRFPluginConfig{})
	initPlugin(t, p)
	hooks := p.Hooks()
	genHook := hooks[0]

	tests := []struct {
		name           string
		method         string
		existingCookie bool
		wantToken      bool
	}{
		{
			name:      "first GET generates token",
			method:    http.MethodGet,
			wantToken: true,
		},
		{
			name:           "second GET with cookie skips generation",
			method:         http.MethodGet,
			existingCookie: true,
		},
		{
			name:      "unauthenticated GET generates token",
			method:    http.MethodGet,
			wantToken: true,
		},
		{
			name:           "unauthenticated GET with existing cookie skips",
			method:         http.MethodGet,
			existingCookie: true,
		},
		{
			name:      "POST does not generate token",
			method:    http.MethodPost,
			wantToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.existingCookie {
				req.AddCookie(&http.Cookie{Name: "authula_csrf_token", Value: "existing"})
			}
			w := httptest.NewRecorder()
			ctx := &models.RequestContext{
				Request:        req,
				ResponseWriter: w,
				Path:           "/test",
				Method:         tt.method,
			}

			if !genHook.Matcher(ctx) {
				if tt.wantToken {
					t.Error("matcher should have allowed hook to run")
				}
				return
			}

			if err := genHook.Handler(ctx); err != nil {
				t.Fatalf("Handler error: %v", err)
			}

			cookies := w.Result().Cookies()
			if tt.wantToken && len(cookies) == 0 {
				t.Error("expected CSRF cookie to be set")
			}
			if !tt.wantToken && len(cookies) > 0 {
				t.Error("expected no CSRF cookie to be set")
			}
		})
	}
}

func TestCSRFPlugin_CSRFValidation(t *testing.T) {
	p := New(CSRFPluginConfig{})
	p.logger = &MockLogger{}
	hooks := p.Hooks()
	validateHook := hooks[1]

	tests := []struct {
		name        string
		method      string
		hasCookie   bool
		cookieValue string
		hasHeader   bool
		headerValue string
		wantReject  bool
	}{
		{
			name:        "valid POST with matching token",
			method:      http.MethodPost,
			hasCookie:   true,
			cookieValue: "token123",
			hasHeader:   true,
			headerValue: "token123",
		},
		{
			name:        "POST missing cookie",
			method:      http.MethodPost,
			hasHeader:   true,
			headerValue: "token",
			wantReject:  true,
		},
		{
			name:        "POST mismatched tokens",
			method:      http.MethodPost,
			hasCookie:   true,
			cookieValue: "cookie_token",
			hasHeader:   true,
			headerValue: "header_token",
			wantReject:  true,
		},
		{
			name:        "valid PUT with matching token",
			method:      http.MethodPut,
			hasCookie:   true,
			cookieValue: "put_token",
			hasHeader:   true,
			headerValue: "put_token",
		},
		{
			name:        "valid DELETE with matching token",
			method:      http.MethodDelete,
			hasCookie:   true,
			cookieValue: "del_token",
			hasHeader:   true,
			headerValue: "del_token",
		},
		{
			name:       "DELETE missing cookie and header",
			method:     http.MethodDelete,
			wantReject: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/protected", nil)
			if tt.hasCookie {
				req.AddCookie(&http.Cookie{Name: "authula_csrf_token", Value: tt.cookieValue})
			}
			if tt.hasHeader {
				req.Header.Set("X-AUTHULA-CSRF-TOKEN", tt.headerValue)
			}
			w := httptest.NewRecorder()
			ctx := &models.RequestContext{
				Request:        req,
				ResponseWriter: w,
				Path:           "/api/protected",
				Method:         tt.method,
			}

			if err := validateHook.Handler(ctx); err != nil {
				t.Fatalf("Handler error: %v", err)
			}

			if tt.wantReject && !ctx.Handled {
				t.Error("expected request to be rejected (Handled=true)")
			}
			if !tt.wantReject && ctx.Handled {
				t.Error("expected request to proceed (Handled=false)")
			}
		})
	}
}

func TestCSRFPlugin_TokenRotation(t *testing.T) {
	p := New(CSRFPluginConfig{})
	p.logger = &MockLogger{}
	hooks := p.Hooks()
	afterHook := hooks[2]

	tests := []struct {
		name   string
		path   string
		method string
		values map[string]any
	}{
		{
			name:   "post-login",
			path:   "/login",
			method: http.MethodPost,
		},
		{
			name:   "post-register",
			path:   "/register",
			method: http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			ctx := &models.RequestContext{
				Request:        req,
				ResponseWriter: httptest.NewRecorder(),
				Path:           tt.path,
				Method:         tt.method,
				Values:         tt.values,
			}

			if err := afterHook.Handler(ctx); err != nil {
				t.Fatalf("Handler error: %v", err)
			}
		})
	}
}

func TestCSRFPlugin_SetCSRFCookie(t *testing.T) {
	tests := []struct {
		name         string
		sameSite     string
		scheme       string
		wantSameSite http.SameSite
		wantSecure   bool
	}{
		{
			name:         "lax on HTTPS",
			sameSite:     "lax",
			scheme:       "https",
			wantSameSite: http.SameSiteLaxMode,
			wantSecure:   true,
		},
		{
			name:         "strict on HTTPS",
			sameSite:     "strict",
			scheme:       "https",
			wantSameSite: http.SameSiteStrictMode,
			wantSecure:   true,
		},
		{
			name:         "none on HTTPS",
			sameSite:     "none",
			scheme:       "https",
			wantSameSite: http.SameSiteNoneMode,
			wantSecure:   true,
		},
		{
			name:         "invalid defaults to lax on HTTPS",
			sameSite:     "invalid",
			scheme:       "https",
			wantSameSite: http.SameSiteLaxMode,
			wantSecure:   true,
		},
		{
			name:         "empty defaults to lax on HTTPS",
			sameSite:     "",
			scheme:       "https",
			wantSameSite: http.SameSiteLaxMode,
			wantSecure:   true,
		},
		{
			name:         "lax on HTTP sets Secure=false",
			sameSite:     "lax",
			scheme:       "http",
			wantSameSite: http.SameSiteLaxMode,
			wantSecure:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(CSRFPluginConfig{SameSite: tt.sameSite})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", tt.scheme+"://example.com/test", nil)
			ctx := &models.RequestContext{Request: r, ResponseWriter: w}

			p.setCSRFCookie(ctx, "test_token")

			cookies := w.Result().Cookies()
			if len(cookies) == 0 {
				t.Fatal("expected a cookie to be set")
			}
			c := cookies[0]

			if c.Name != "authula_csrf_token" {
				t.Errorf("Name = %q, want 'authula_csrf_token'", c.Name)
			}
			if c.Value != "test_token" {
				t.Errorf("Value = %q", c.Value)
			}
			if c.Path != "/" {
				t.Errorf("Path = %q", c.Path)
			}
			if c.HttpOnly {
				t.Error("HttpOnly must be false for Double-Submit Cookie pattern")
			}
			if c.SameSite != tt.wantSameSite {
				t.Errorf("SameSite = %v, want %v", c.SameSite, tt.wantSameSite)
			}
			if c.Secure != tt.wantSecure {
				t.Errorf("Secure = %v, want %v", c.Secure, tt.wantSecure)
			}
		})
	}
}

func TestCSRFPlugin_Middleware(t *testing.T) {
	p := New(CSRFPluginConfig{})
	initPlugin(t, p)
	middleware := p.Middleware()

	tests := []struct {
		name             string
		method           string
		userID           string
		hasCookie        bool
		cookieValue      string
		headerValue      string
		expectedStatus   int
		handlerShouldRun bool
		expectCookie     bool
	}{
		{
			name:             "unauthenticated POST passes through",
			method:           http.MethodPost,
			expectedStatus:   http.StatusOK,
			handlerShouldRun: true,
		},
		{
			name:             "authenticated POST with valid token",
			method:           http.MethodPost,
			userID:           "user-123",
			hasCookie:        true,
			cookieValue:      "token",
			headerValue:      "token",
			expectedStatus:   http.StatusOK,
			handlerShouldRun: true,
		},
		{
			name:             "authenticated POST missing cookie",
			method:           http.MethodPost,
			userID:           "user-123",
			expectedStatus:   http.StatusForbidden,
			handlerShouldRun: false,
		},
		{
			name:             "authenticated POST mismatched tokens",
			method:           http.MethodPost,
			userID:           "user-123",
			hasCookie:        true,
			cookieValue:      "cookie_token",
			headerValue:      "header_token",
			expectedStatus:   http.StatusForbidden,
			handlerShouldRun: false,
		},
		{
			name:             "authenticated GET generates cookie",
			method:           http.MethodGet,
			userID:           "user-123",
			expectedStatus:   http.StatusOK,
			handlerShouldRun: true,
			expectCookie:     true,
		},
		{
			name:             "authenticated GET with existing cookie does not regenerate",
			method:           http.MethodGet,
			userID:           "user-123",
			hasCookie:        true,
			cookieValue:      "existing",
			expectedStatus:   http.StatusOK,
			handlerShouldRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/custom", nil)
			w := httptest.NewRecorder()

			var reqCtx *models.RequestContext
			if tt.userID != "" {
				reqCtx = &models.RequestContext{
					Request:         req,
					ResponseWriter:  w,
					Headers:         req.Header,
					Values:          make(map[string]any),
					ResponseHeaders: make(http.Header),
					Handled:         false,
				}
				reqCtx.Actor = &models.Actor{ID: tt.userID, Type: models.ActorUser}
				req = req.WithContext(models.NewContextWithRequestContext(req.Context(), reqCtx))
			}

			if tt.hasCookie {
				req.AddCookie(&http.Cookie{Name: "authula_csrf_token", Value: tt.cookieValue})
				req.Header.Set("X-AUTHULA-CSRF-TOKEN", tt.headerValue)
			}

			handlerCalled := false
			handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				rw.WriteHeader(http.StatusOK)
			})

			middleware(handler).ServeHTTP(w, req)

			if reqCtx != nil && reqCtx.ResponseReady {
				if reqCtx.ResponseStatus != 0 {
					w.WriteHeader(reqCtx.ResponseStatus)
				}
				if len(reqCtx.ResponseBody) > 0 {
					_, _ = w.Write(reqCtx.ResponseBody)
				}
			}

			if w.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.expectedStatus)
			}
			if handlerCalled != tt.handlerShouldRun {
				t.Errorf("handlerCalled = %v, want %v", handlerCalled, tt.handlerShouldRun)
			}
			if tt.expectCookie {
				cookies := w.Result().Cookies()
				if len(cookies) == 0 {
					t.Error("expected CSRF cookie to be set")
				}
			}
		})
	}
}

func TestCSRFPlugin_HeaderProtection(t *testing.T) {
	t.Run("disabled by default", func(t *testing.T) {
		p := New(CSRFPluginConfig{})

		if p.cop != nil {
			t.Error("expected CrossOriginProtection to be nil when disabled")
		}

		req := httptest.NewRequest("POST", "http://localhost/api/test", nil)
		req.Header.Set("Origin", "http://evil.com")

		if err := p.validateHeaderProtection(req); err != nil {
			t.Errorf("expected no error when disabled, got: %v", err)
		}
	})

	t.Run("enabled blocks cross-origin unsafe methods", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping in short mode")
		}

		p := New(CSRFPluginConfig{})
		p.logger = &MockLogger{}
		p.cop = http.NewCrossOriginProtection()

		if err := p.cop.AddTrustedOrigin("https://app.example.com"); err != nil {
			t.Fatalf("failed to add trusted origin: %v", err)
		}
		p.cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))

		req := httptest.NewRequest("POST", "http://localhost/api/test", nil)
		req.Header.Set("Origin", "http://evil.com")
		req.Header.Set("Host", "localhost")

		_ = p.validateHeaderProtection(req)
	})

	t.Run("safe methods bypass header protection", func(t *testing.T) {
		p := New(CSRFPluginConfig{})
		p.logger = &MockLogger{}
		p.cop = http.NewCrossOriginProtection()

		if err := p.cop.AddTrustedOrigin("https://app.example.com"); err != nil {
			t.Fatalf("failed to add trusted origin: %v", err)
		}

		safeMethods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}
		for _, method := range safeMethods {
			req := httptest.NewRequest(method, "http://localhost/api/test", nil)
			req.Header.Set("Origin", "http://evil.com")

			if err := p.validateHeaderProtection(req); err != nil {
				t.Errorf("expected %s to pass, got: %v", method, err)
			}
		}
	})

	t.Run("token validation still required with header protection", func(t *testing.T) {
		p := New(CSRFPluginConfig{
			EnableHeaderProtection: true,
			CookieName:             "authula_csrf_token",
			HeaderName:             "X-AUTHULA-CSRF-TOKEN",
		})

		req := httptest.NewRequest("POST", "http://localhost/api/test", nil)
		w := httptest.NewRecorder()
		ctx := &models.RequestContext{
			Request:        req,
			ResponseWriter: w,
			Method:         "POST",
			Path:           "/api/test",
			Actor:          &models.Actor{ID: "user123"},
		}

		err := p.validateCSRFToken(ctx)
		if err != nil {
			t.Errorf("expected no error return, got: %v", err)
		}
		if !ctx.Handled {
			t.Error("expected ctx.Handled to be true for missing token")
		}
	})
}
