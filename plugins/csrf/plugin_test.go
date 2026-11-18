package csrf

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
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

// failingTokenService always fails token generation so error branches can be exercised.
type failingTokenService struct {
	err error
}

func (f *failingTokenService) Generate() (string, error) { return "", f.err }
func (f *failingTokenService) Hash(token string) string  { return "hashed-" + token }
func (f *failingTokenService) Encrypt(token string) (string, error) {
	return "encrypted-" + token, nil
}
func (f *failingTokenService) Decrypt(encrypted string) (string, error) {
	return encrypted, nil
}

// initPluginWithServices initializes the plugin with the given global config and an
// optional token service, leaving it ready for hook and middleware assertions.
func initPluginWithServices(t *testing.T, p *CSRFPlugin, config *models.Config, tokenService any) error {
	t.Helper()
	p.logger = &MockLogger{}
	registry := newMockServiceRegistry()
	if tokenService != nil {
		registry.Register(models.ServiceToken.String(), tokenService)
	}
	return p.Init(&models.PluginContext{
		Logger:          p.logger,
		ServiceRegistry: registry,
		GetConfig:       func() *models.Config { return config },
	})
}

func initPlugin(t *testing.T, p *CSRFPlugin) {
	t.Helper()
	if err := initPluginWithServices(t, p, &models.Config{}, &mockTokenService{}); err != nil {
		t.Fatalf("failed to init plugin: %v", err)
	}
}

// newHeaderProtectedPlugin returns a plugin with header protection enabled and
// https://app.example.com as a trusted origin.
func newHeaderProtectedPlugin(t *testing.T) *CSRFPlugin {
	t.Helper()
	p := New(CSRFPluginConfig{EnableHeaderProtection: true})
	require.NoError(t, initPluginWithServices(t, p, &models.Config{
		Security: models.SecurityConfig{TrustedOrigins: []string{"https://app.example.com"}},
	}, &mockTokenService{}))
	require.NotNil(t, p.cop)
	return p
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
		wantHeader     string
	}{
		{
			name:      "first GET generates token",
			method:    http.MethodGet,
			wantToken: true,
		},
		{
			name:           "second GET with cookie skips generation but echoes token header",
			method:         http.MethodGet,
			existingCookie: true,
			wantHeader:     "existing",
		},
		{
			name:      "unauthenticated GET generates token",
			method:    http.MethodGet,
			wantToken: true,
		},
		{
			name:           "unauthenticated GET with existing cookie skips but echoes token header",
			method:         http.MethodGet,
			existingCookie: true,
			wantHeader:     "existing",
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

			// Cross-origin clients cannot read the cookie, so the token must always be
			// discoverable from the response header on safe requests.
			gotHeader := w.Header().Get("X-AUTHULA-CSRF-TOKEN")
			if tt.wantToken && gotHeader == "" {
				t.Error("expected CSRF token header on generation")
			}
			if tt.wantHeader != "" && gotHeader != tt.wantHeader {
				t.Errorf("CSRF token header = %q, want %q", gotHeader, tt.wantHeader)
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
	p.globalConfig = &models.Config{}
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
			p.globalConfig = &models.Config{}
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

func TestCSRFPlugin_CookieDomain(t *testing.T) {
	tests := []struct {
		name       string
		domain     string
		wantDomain string
	}{
		{name: "host-only when no domain is configured", domain: "", wantDomain: ""},
		{name: "follows the session cookie domain", domain: "example.com", wantDomain: "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(CSRFPluginConfig{})
			p.globalConfig = &models.Config{Session: models.SessionConfig{Domain: tt.domain}}

			setRecorder := httptest.NewRecorder()
			p.setCSRFCookie(&models.RequestContext{
				Request:        httptest.NewRequest("GET", "https://api.example.com/test", nil),
				ResponseWriter: setRecorder,
			}, "test_token")
			setCookies := setRecorder.Result().Cookies()
			if len(setCookies) != 1 {
				t.Fatalf("expected 1 cookie on set, got %d", len(setCookies))
			}
			if setCookies[0].Domain != tt.wantDomain {
				t.Errorf("set Domain = %q, want %q", setCookies[0].Domain, tt.wantDomain)
			}

			clearRecorder := httptest.NewRecorder()
			p.clearCSRFCookie(&models.RequestContext{
				Request:        httptest.NewRequest("POST", "https://api.example.com/sign-out", nil),
				ResponseWriter: clearRecorder,
			})
			clearCookies := clearRecorder.Result().Cookies()
			if len(clearCookies) != 1 {
				t.Fatalf("expected 1 cookie on clear, got %d", len(clearCookies))
			}
			if clearCookies[0].Domain != tt.wantDomain {
				t.Errorf("clear Domain = %q, want %q", clearCookies[0].Domain, tt.wantDomain)
			}
		})
	}
}

// The browser only deletes a cookie when the clear carries the same scoping attributes as the set.
func TestCSRFPlugin_ClearCookieMatchesSetAttributes(t *testing.T) {
	tests := []struct {
		name     string
		sameSite string
		secure   bool
		scheme   string
	}{
		{name: "lax over http", sameSite: "lax", scheme: "http"},
		{name: "strict with secure flag over http", sameSite: "strict", secure: true, scheme: "http"},
		{name: "none over https", sameSite: "none", scheme: "https"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(CSRFPluginConfig{SameSite: tt.sameSite, Secure: tt.secure})
			p.globalConfig = &models.Config{Session: models.SessionConfig{Domain: "example.com"}}

			setRecorder := httptest.NewRecorder()
			p.setCSRFCookie(&models.RequestContext{
				Request:        httptest.NewRequest("GET", tt.scheme+"://api.example.com/test", nil),
				ResponseWriter: setRecorder,
			}, "test_token")
			set := setRecorder.Result().Cookies()[0]

			clearRecorder := httptest.NewRecorder()
			p.clearCSRFCookie(&models.RequestContext{
				Request:        httptest.NewRequest("POST", tt.scheme+"://api.example.com/sign-out", nil),
				ResponseWriter: clearRecorder,
			})
			clear := clearRecorder.Result().Cookies()[0]

			if clear.Value != "" || clear.MaxAge != -1 {
				t.Errorf("clear must expire the cookie, got Value=%q MaxAge=%d", clear.Value, clear.MaxAge)
			}
			if clear.Name != set.Name || clear.Path != set.Path || clear.Domain != set.Domain {
				t.Errorf("clear scope (%s %s %s) != set scope (%s %s %s)", clear.Name, clear.Path, clear.Domain, set.Name, set.Path, set.Domain)
			}
			if clear.Secure != set.Secure {
				t.Errorf("clear Secure = %v, set Secure = %v", clear.Secure, set.Secure)
			}
			if clear.SameSite != set.SameSite {
				t.Errorf("clear SameSite = %v, set SameSite = %v", clear.SameSite, set.SameSite)
			}
			if clear.HttpOnly != set.HttpOnly {
				t.Errorf("clear HttpOnly = %v, set HttpOnly = %v", clear.HttpOnly, set.HttpOnly)
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
		expectHeader     string
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
			name:             "authenticated GET with existing cookie does not regenerate but echoes token header",
			method:           http.MethodGet,
			userID:           "user-123",
			hasCookie:        true,
			cookieValue:      "existing",
			expectedStatus:   http.StatusOK,
			handlerShouldRun: true,
			expectHeader:     "existing",
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
			if tt.expectHeader != "" {
				if got := w.Header().Get("X-AUTHULA-CSRF-TOKEN"); got != tt.expectHeader {
					t.Errorf("CSRF token header = %q, want %q", got, tt.expectHeader)
				}
			}
		})
	}
}

func TestCSRFPlugin_HeaderProtection(t *testing.T) {
	tests := []struct {
		name       string
		enable     bool
		method     string
		target     string
		headers    map[string]string
		wantReject bool
	}{
		{
			name:    "disabled plugin ignores cross-site requests",
			method:  http.MethodPost,
			target:  "https://api.example.com/auth/sign-in",
			headers: map[string]string{"Origin": "https://evil.example.com", "Sec-Fetch-Site": "cross-site"},
		},
		{
			name:    "GET with cross-site metadata is allowed",
			enable:  true,
			method:  http.MethodGet,
			target:  "https://api.example.com/auth/me",
			headers: map[string]string{"Origin": "https://evil.example.com", "Sec-Fetch-Site": "cross-site"},
		},
		{
			name:    "HEAD with cross-site metadata is allowed",
			enable:  true,
			method:  http.MethodHead,
			target:  "https://api.example.com/auth/me",
			headers: map[string]string{"Origin": "https://evil.example.com", "Sec-Fetch-Site": "cross-site"},
		},
		{
			name:    "OPTIONS with cross-site metadata is allowed",
			enable:  true,
			method:  http.MethodOptions,
			target:  "https://api.example.com/auth/me",
			headers: map[string]string{"Origin": "https://evil.example.com", "Sec-Fetch-Site": "cross-site"},
		},
		{
			name:    "same-origin metadata is allowed",
			enable:  true,
			method:  http.MethodPost,
			target:  "https://api.example.com/auth/sign-in",
			headers: map[string]string{"Sec-Fetch-Site": "same-origin"},
		},
		{
			name:    "none metadata is allowed",
			enable:  true,
			method:  http.MethodPost,
			target:  "https://api.example.com/auth/sign-in",
			headers: map[string]string{"Sec-Fetch-Site": "none"},
		},
		{
			name:       "same-site metadata from an untrusted origin is rejected",
			enable:     true,
			method:     http.MethodPost,
			target:     "https://api.example.com/auth/sign-in",
			headers:    map[string]string{"Sec-Fetch-Site": "same-site", "Origin": "https://evil.example.com"},
			wantReject: true,
		},
		{
			name:    "same-site metadata from a trusted origin is allowed",
			enable:  true,
			method:  http.MethodPost,
			target:  "https://api.example.com/auth/sign-in",
			headers: map[string]string{"Sec-Fetch-Site": "same-site", "Origin": "https://app.example.com"},
		},
		{
			name:       "cross-site metadata is rejected",
			enable:     true,
			method:     http.MethodPost,
			target:     "https://api.example.com/auth/sign-in",
			headers:    map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example.com"},
			wantReject: true,
		},
		{
			name:    "cross-site metadata from a trusted origin is allowed",
			enable:  true,
			method:  http.MethodPost,
			target:  "https://api.example.com/auth/sign-in",
			headers: map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://app.example.com"},
		},
		{
			name:    "origin matching the host is allowed without fetch metadata",
			enable:  true,
			method:  http.MethodPost,
			target:  "https://api.example.com/auth/sign-in",
			headers: map[string]string{"Origin": "https://api.example.com"},
		},
		{
			name:       "origin not matching the host is rejected without fetch metadata",
			enable:     true,
			method:     http.MethodPost,
			target:     "https://api.example.com/auth/sign-in",
			headers:    map[string]string{"Origin": "https://evil.example.com"},
			wantReject: true,
		},
		{
			name:   "missing fetch metadata headers are allowed",
			enable: true,
			method: http.MethodPost,
			target: "https://api.example.com/auth/sign-in",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(CSRFPluginConfig{})
			if tt.enable {
				p = newHeaderProtectedPlugin(t)
			}

			req := httptest.NewRequest(tt.method, tt.target, nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			err := p.validateHeaderProtection(req)
			if !tt.wantReject {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			logger, ok := p.logger.(*MockLogger)
			require.True(t, ok)
			assert.Contains(t, logger.debugMessages, "header-based csrf check failed")
		})
	}
}

func TestCSRFPlugin_ConfigReturnsCurrentConfig(t *testing.T) {
	p := New(CSRFPluginConfig{Enabled: true, CookieName: "custom_csrf"})

	got, ok := p.Config().(CSRFPluginConfig)
	require.True(t, ok, "Config() should return a CSRFPluginConfig value")

	assert.True(t, got.Enabled)
	assert.Equal(t, "custom_csrf", got.CookieName)
	assert.Equal(t, "X-AUTHULA-CSRF-TOKEN", got.HeaderName)
	assert.Equal(t, 24*time.Hour, got.MaxAge)
	assert.Equal(t, "lax", got.SameSite)
}

func TestCSRFPlugin_Close(t *testing.T) {
	p := New(CSRFPluginConfig{})
	initPlugin(t, p)

	assert.NoError(t, p.Close())
}

func TestCSRFPlugin_Init(t *testing.T) {
	tests := []struct {
		name         string
		config       *models.Config
		tokenService any
		wantErr      string
		check        func(t *testing.T, p *CSRFPlugin)
	}{
		{
			name:         "applies defaults when plugin config is absent",
			config:       &models.Config{},
			tokenService: &mockTokenService{},
			check: func(t *testing.T, p *CSRFPlugin) {
				assert.Equal(t, "authula_csrf_token", p.pluginConfig.CookieName)
				assert.Equal(t, "X-AUTHULA-CSRF-TOKEN", p.pluginConfig.HeaderName)
				assert.Equal(t, 24*time.Hour, p.pluginConfig.MaxAge)
				assert.Equal(t, "lax", p.pluginConfig.SameSite)
				assert.NotNil(t, p.tokenService)
			},
		},
		{
			name: "loads values from the plugins map",
			config: &models.Config{Plugins: models.PluginsConfig{
				models.PluginCSRF.String(): map[string]any{
					"enabled":     true,
					"cookie_name": "custom_csrf",
					"header_name": "X-Custom-CSRF",
					"same_site":   "strict",
					"max_age":     "90m",
				},
			}},
			tokenService: &mockTokenService{},
			check: func(t *testing.T, p *CSRFPlugin) {
				assert.True(t, p.pluginConfig.Enabled)
				assert.Equal(t, "custom_csrf", p.pluginConfig.CookieName)
				assert.Equal(t, "X-Custom-CSRF", p.pluginConfig.HeaderName)
				assert.Equal(t, "strict", p.pluginConfig.SameSite)
				assert.Equal(t, 90*time.Minute, p.pluginConfig.MaxAge)
			},
		},
		{
			name: "pre-parsed config takes precedence over the plugins map",
			config: &models.Config{
				Plugins: models.PluginsConfig{
					models.PluginCSRF.String(): map[string]any{"cookie_name": "from_map"},
				},
				PreParsedConfigs: map[string]any{
					models.PluginCSRF.String(): CSRFPluginConfig{CookieName: "from_preparsed"},
				},
			},
			tokenService: &mockTokenService{},
			check: func(t *testing.T, p *CSRFPlugin) {
				assert.Equal(t, "from_preparsed", p.pluginConfig.CookieName)
			},
		},
		{
			name: "returns an error for invalid plugin config",
			config: &models.Config{Plugins: models.PluginsConfig{
				models.PluginCSRF.String(): map[string]any{"max_age": "not-a-duration"},
			}},
			tokenService: &mockTokenService{},
			wantErr:      "max_age",
		},
		{
			name: "enables header protection with trusted origins",
			config: &models.Config{
				Plugins: models.PluginsConfig{
					models.PluginCSRF.String(): map[string]any{"enable_header_protection": true},
				},
				Security: models.SecurityConfig{TrustedOrigins: []string{"https://app.example.com"}},
			},
			tokenService: &mockTokenService{},
			check: func(t *testing.T, p *CSRFPlugin) {
				assert.True(t, p.pluginConfig.EnableHeaderProtection)
				assert.NotNil(t, p.cop)
			},
		},
		{
			name: "rejects trusted origins without a scheme",
			config: &models.Config{
				Plugins: models.PluginsConfig{
					models.PluginCSRF.String(): map[string]any{"enable_header_protection": true},
				},
				Security: models.SecurityConfig{TrustedOrigins: []string{"app.example.com"}},
			},
			tokenService: &mockTokenService{},
			wantErr:      "must include scheme",
		},
		{
			name: "rejects trusted origins with a path",
			config: &models.Config{
				Plugins: models.PluginsConfig{
					models.PluginCSRF.String(): map[string]any{"enable_header_protection": true},
				},
				Security: models.SecurityConfig{TrustedOrigins: []string{"https://app.example.com/callback"}},
			},
			tokenService: &mockTokenService{},
			wantErr:      "path",
		},
		{
			name:    "returns an error when the token service is missing",
			config:  &models.Config{},
			wantErr: "token service not available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(CSRFPluginConfig{})

			err := initPluginWithServices(t, p, tt.config, tt.tokenService)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			tt.check(t, p)
		})
	}
}

func TestCSRFPlugin_SignedOutMatcher(t *testing.T) {
	p := New(CSRFPluginConfig{})
	matcher := p.Hooks()[2].Matcher
	require.NotNil(t, matcher)

	tests := []struct {
		name   string
		values map[string]any
		want   bool
	}{
		{name: "nil values", values: nil, want: false},
		{name: "signed out flag set to true", values: map[string]any{models.ContextAuthSignOut.String(): true}, want: true},
		{name: "signed out flag set to false", values: map[string]any{models.ContextAuthSignOut.String(): false}, want: false},
		{name: "signed in request without the flag", values: map[string]any{models.ContextAuthSuccess.String(): true}, want: false},
		{name: "non-boolean flag value", values: map[string]any{models.ContextAuthSignOut.String(): "true"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &models.RequestContext{Values: tt.values}
			assert.Equal(t, tt.want, matcher(ctx))
		})
	}
}

func TestCSRFPlugin_GenerateCSRFTokenHook(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		tokenService any
		wantStatus   int
		wantHandled  bool
		wantCookie   bool
		wantMessage  string
	}{
		{
			name:         "unsafe method is ignored",
			method:       http.MethodPost,
			tokenService: &mockTokenService{},
		},
		{
			name:         "token generation failure returns 500",
			method:       http.MethodGet,
			tokenService: &failingTokenService{err: errors.New("generate failed")},
			wantStatus:   http.StatusInternalServerError,
			wantHandled:  true,
			wantMessage:  "failed to generate csrf token",
		},
		{
			name:         "HEAD generates a token",
			method:       http.MethodHead,
			tokenService: &mockTokenService{},
			wantCookie:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(CSRFPluginConfig{})
			require.NoError(t, initPluginWithServices(t, p, &models.Config{}, tt.tokenService))

			genHook := p.Hooks()[0]
			reqCtx := internaltests.NewRequestContext(t, tt.method, "/auth/me", nil)

			require.NoError(t, genHook.Handler(reqCtx))

			assert.Equal(t, tt.wantHandled, reqCtx.Handled)
			assert.Equal(t, tt.wantCookie, len(reqCtx.ResponseWriter.(*httptest.ResponseRecorder).Result().Cookies()) > 0)

			if tt.wantMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tt.wantStatus, tt.wantMessage)
			}
		})
	}
}

func TestCSRFPlugin_ValidateCSRFTokenHook(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		headers     map[string]string
		cookieValue string
		headerValue string
		wantMessage string
	}{
		{
			name:   "safe method bypasses token validation",
			method: http.MethodGet,
		},
		{
			name:        "header protection rejects cross-site request",
			method:      http.MethodPost,
			headers:     map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example.com"},
			wantMessage: "csrf validation failed",
		},
		{
			name:        "missing cookie is rejected",
			method:      http.MethodPost,
			wantMessage: "missing csrf cookie",
		},
		{
			name:        "missing header is rejected",
			method:      http.MethodPost,
			cookieValue: "cookie_token",
			wantMessage: "missing csrf token in header",
		},
		{
			name:        "mismatched tokens are rejected",
			method:      http.MethodPost,
			cookieValue: "cookie_token",
			headerValue: "header_token",
			wantMessage: "invalid csrf token",
		},
		{
			name:        "matching tokens are allowed",
			method:      http.MethodPost,
			cookieValue: "shared_token",
			headerValue: "shared_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newHeaderProtectedPlugin(t)
			hook := p.Hooks()[1]

			reqCtx := internaltests.NewRequestContext(t, tt.method, "/auth/sign-in", tt.headers)
			if tt.cookieValue != "" {
				reqCtx.Request.AddCookie(&http.Cookie{Name: "authula_csrf_token", Value: tt.cookieValue})
			}
			if tt.headerValue != "" {
				reqCtx.Request.Header.Set("X-AUTHULA-CSRF-TOKEN", tt.headerValue)
			}

			require.NoError(t, hook.Handler(reqCtx))

			if tt.wantMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, http.StatusForbidden, tt.wantMessage)
				return
			}

			assert.False(t, reqCtx.Handled)
			assert.Equal(t, 0, reqCtx.ResponseStatus)
		})
	}
}

func TestCSRFPlugin_ValidateCSRFTokenMissingHeader(t *testing.T) {
	p := New(CSRFPluginConfig{})
	initPlugin(t, p)

	reqCtx := internaltests.NewRequestContext(t, http.MethodPost, "/api/protected", nil)
	reqCtx.Request.AddCookie(&http.Cookie{Name: "authula_csrf_token", Value: "cookie_token"})

	require.NoError(t, p.validateCSRFToken(reqCtx))
	internaltests.AssertErrorMessage(t, reqCtx, http.StatusForbidden, "missing csrf token in header")
}

func TestCSRFPlugin_MiddlewareEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		tokenService any
		actorID      string
		cookieValue  string
		wantStatus   int
		wantHandler  bool
		wantMessage  string
	}{
		{
			name:         "authenticated POST with cookie but no header",
			method:       http.MethodPost,
			tokenService: &mockTokenService{},
			actorID:      "user-1",
			cookieValue:  "cookie_token",
			wantStatus:   http.StatusForbidden,
			wantMessage:  "missing csrf token in header",
		},
		{
			name:         "request context without an actor passes through",
			method:       http.MethodPost,
			tokenService: &mockTokenService{},
			wantStatus:   http.StatusOK,
			wantHandler:  true,
		},
		{
			name:         "token generation failure returns 500",
			method:       http.MethodGet,
			tokenService: &failingTokenService{err: errors.New("generate failed")},
			actorID:      "user-1",
			wantStatus:   http.StatusInternalServerError,
			wantMessage:  "failed to generate csrf token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(CSRFPluginConfig{})
			require.NoError(t, initPluginWithServices(t, p, &models.Config{}, tt.tokenService))

			req := httptest.NewRequest(tt.method, "/api/custom", nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: "authula_csrf_token", Value: tt.cookieValue})
			}
			w := httptest.NewRecorder()

			reqCtx := &models.RequestContext{
				Request:         req,
				ResponseWriter:  w,
				Headers:         req.Header,
				Values:          make(map[string]any),
				ResponseHeaders: make(http.Header),
			}
			if tt.actorID != "" {
				reqCtx.Actor = &models.Actor{ID: tt.actorID, Type: models.ActorUser}
			}
			req = req.WithContext(models.NewContextWithRequestContext(req.Context(), reqCtx))

			handlerCalled := false
			next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				rw.WriteHeader(http.StatusOK)
			})

			p.Middleware()(next).ServeHTTP(w, req)

			if reqCtx.ResponseReady {
				if reqCtx.ResponseStatus != 0 {
					w.WriteHeader(reqCtx.ResponseStatus)
				}
				if len(reqCtx.ResponseBody) > 0 {
					_, _ = w.Write(reqCtx.ResponseBody)
				}
			}

			assert.Equal(t, tt.wantHandler, handlerCalled)
			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantMessage != "" {
				payload := internaltests.DecodeResponseJSON[struct {
					Message string `json:"message"`
				}](t, reqCtx)
				assert.Equal(t, tt.wantMessage, payload.Message)
			}
		})
	}
}
