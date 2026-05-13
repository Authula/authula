package ratelimit

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	plugintests "github.com/Authula/authula/plugins/rate-limit/tests"
	"github.com/Authula/authula/plugins/rate-limit/types"
)

func TestRateLimitPluginCheckEndpointRateLimitHook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		method              string
		requestURI          string
		clientIP            string
		config              types.RateLimitPluginConfig
		provider            *plugintests.FakeRateLimitProvider
		expectedHandled     bool
		expectedStatusCode  int
		expectedLimit       string
		expectedRemaining   string
		expectedProviderKey string
	}{
		{
			name:       "ignores unsupported methods",
			method:     http.MethodHead,
			requestURI: "/ignored",
			clientIP:   "127.0.0.1",
			config: types.RateLimitPluginConfig{
				Prefix:      "ratelimit:",
				Window:      time.Minute,
				Max:         10,
				CustomRules: map[string]types.RateLimitRule{},
			},
			provider:           plugintests.NewFakeRateLimitProvider(),
			expectedHandled:    false,
			expectedStatusCode: 0,
		},
		{
			name:       "allows request and sets headers",
			method:     http.MethodGet,
			requestURI: "/allowed",
			clientIP:   "127.0.0.1",
			config: types.RateLimitPluginConfig{
				Prefix:      "ratelimit:",
				Window:      time.Minute,
				Max:         10,
				CustomRules: map[string]types.RateLimitRule{},
			},
			provider:            plugintests.NewFakeRateLimitProvider().WithCheckResult(true, 3, time.Unix(2000, 0), nil),
			expectedHandled:     false,
			expectedStatusCode:  0,
			expectedLimit:       "10",
			expectedRemaining:   "7",
			expectedProviderKey: "ratelimit:127.0.0.1",
		},
		{
			name:       "blocks request when over limit",
			method:     http.MethodPost,
			requestURI: "/blocked",
			clientIP:   "10.0.0.2",
			config: types.RateLimitPluginConfig{
				Prefix:      "ratelimit:",
				Window:      time.Minute,
				Max:         5,
				CustomRules: map[string]types.RateLimitRule{},
			},
			provider:            plugintests.NewFakeRateLimitProvider().WithCheckResult(false, 6, time.Unix(3000, 0), nil),
			expectedHandled:     true,
			expectedStatusCode:  http.StatusTooManyRequests,
			expectedLimit:       "5",
			expectedRemaining:   "0",
			expectedProviderKey: "ratelimit:10.0.0.2",
		},
		{
			name:       "custom rule overrides default limits",
			method:     http.MethodGet,
			requestURI: "/custom",
			clientIP:   "10.0.0.3",
			config: types.RateLimitPluginConfig{
				Prefix: "ratelimit:",
				Window: time.Minute,
				Max:    10,
				CustomRules: map[string]types.RateLimitRule{
					"/custom": {Window: 2 * time.Minute, Max: 2},
				},
			},
			provider:            plugintests.NewFakeRateLimitProvider().WithCheckResult(true, 1, time.Unix(4000, 0), nil),
			expectedHandled:     false,
			expectedStatusCode:  0,
			expectedLimit:       "2",
			expectedRemaining:   "1",
			expectedProviderKey: "ratelimit:10.0.0.3",
		},
		{
			name:       "provider errors fail open",
			method:     http.MethodGet,
			requestURI: "/error",
			clientIP:   "10.0.0.4",
			config: types.RateLimitPluginConfig{
				Prefix:      "ratelimit:",
				Window:      time.Minute,
				Max:         10,
				CustomRules: map[string]types.RateLimitRule{},
			},
			provider:            plugintests.NewFakeRateLimitProvider().WithCheckError(errors.New("boom")),
			expectedHandled:     false,
			expectedStatusCode:  0,
			expectedProviderKey: "ratelimit:10.0.0.4",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plugin := &RateLimitPlugin{
				provider:     tc.provider,
				logger:       &internaltests.MockLogger{},
				pluginConfig: tc.config,
			}

			req := httptest.NewRequest(tc.method, tc.requestURI, nil)
			reqCtx := &models.RequestContext{
				Request:        req,
				ResponseWriter: httptest.NewRecorder(),
				ClientIP:       tc.clientIP,
				Values:         map[string]any{},
			}

			if err := plugin.checkEndpointRateLimitHook()(reqCtx); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if reqCtx.Handled != tc.expectedHandled {
				t.Fatalf("expected handled=%v, got %v", tc.expectedHandled, reqCtx.Handled)
			}

			if tc.expectedStatusCode != 0 {
				if got := reqCtx.ResponseStatus; got != tc.expectedStatusCode {
					t.Fatalf("expected status %d, got %d", tc.expectedStatusCode, got)
				}
				if !reqCtx.ResponseReady {
					t.Fatal("expected response to be marked ready")
				}
			}

			if tc.expectedLimit != "" {
				if got := reqCtx.ResponseWriter.Header().Get("X-RateLimit-Limit"); got != tc.expectedLimit {
					t.Fatalf("expected X-RateLimit-Limit=%q, got %q", tc.expectedLimit, got)
				}
			}
			if tc.expectedRemaining != "" {
				if got := reqCtx.ResponseWriter.Header().Get("X-RateLimit-Remaining"); got != tc.expectedRemaining {
					t.Fatalf("expected X-RateLimit-Remaining=%q, got %q", tc.expectedRemaining, got)
				}
			}
			if tc.expectedProviderKey != "" && tc.provider.LastCheckKey != tc.expectedProviderKey {
				t.Fatalf("expected provider key %q, got %q", tc.expectedProviderKey, tc.provider.LastCheckKey)
			}
		})
	}
}

func TestRateLimitPluginHandleStoreRateLimitRuleHook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		contextValue         any
		provider             *plugintests.FakeRateLimitProvider
		expectedGetCalls     int
		expectedSetCalls     int
		expectedStoredKey    string
		expectedStoredWindow time.Duration
		expectedStoredMax    int
	}{
		{
			name:             "missing context is a no-op",
			provider:         plugintests.NewFakeRateLimitProvider(),
			expectedGetCalls: 0,
			expectedSetCalls: 0,
		},
		{
			name:         "invalid context is ignored",
			contextValue: models.RateLimitRuleContext{Key: "", WindowSeconds: 60, MaxRequests: 10},
			provider:     plugintests.NewFakeRateLimitProvider(),
		},
		{
			name:             "existing rule is not stored again",
			contextValue:     models.RateLimitRuleContext{Key: "rule-1", WindowSeconds: 60, MaxRequests: 10},
			provider:         plugintests.NewFakeRateLimitProvider().WithExistingRule("rule-1", time.Minute, 10),
			expectedGetCalls: 1,
			expectedSetCalls: 0,
		},
		{
			name:                 "new rule is stored",
			contextValue:         models.RateLimitRuleContext{Key: "rule-2", WindowSeconds: 120, MaxRequests: 7},
			provider:             plugintests.NewFakeRateLimitProvider(),
			expectedGetCalls:     1,
			expectedSetCalls:     1,
			expectedStoredKey:    "rule-2",
			expectedStoredWindow: 120 * time.Second,
			expectedStoredMax:    7,
		},
		{
			name:             "get rule errors are fail open",
			contextValue:     models.RateLimitRuleContext{Key: "rule-3", WindowSeconds: 120, MaxRequests: 7},
			provider:         plugintests.NewFakeRateLimitProvider().WithGetError(errors.New("get failed")),
			expectedGetCalls: 1,
			expectedSetCalls: 0,
		},
		{
			name:                 "set rule errors are fail open",
			contextValue:         models.RateLimitRuleContext{Key: "rule-4", WindowSeconds: 120, MaxRequests: 7},
			provider:             plugintests.NewFakeRateLimitProvider().WithSetError(errors.New("set failed")),
			expectedGetCalls:     1,
			expectedSetCalls:     1,
			expectedStoredKey:    "rule-4",
			expectedStoredWindow: 120 * time.Second,
			expectedStoredMax:    7,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plugin := &RateLimitPlugin{provider: tc.provider, logger: &internaltests.MockLogger{}}
			reqCtx := newTestRequestContext(t, map[string]any{})
			if tc.contextValue != nil {
				reqCtx.Values[models.ContextRateLimitRule.String()] = tc.contextValue
			}

			if err := plugin.handleStoreRateLimitRuleHook()(reqCtx); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.provider.GetCalls != tc.expectedGetCalls {
				t.Fatalf("expected GetRule calls=%d, got %d", tc.expectedGetCalls, tc.provider.GetCalls)
			}
			if tc.provider.SetCalls != tc.expectedSetCalls {
				t.Fatalf("expected SetRule calls=%d, got %d", tc.expectedSetCalls, tc.provider.SetCalls)
			}
			if tc.expectedStoredKey != "" {
				if tc.provider.LastSetKey != tc.expectedStoredKey {
					t.Fatalf("expected stored key %q, got %q", tc.expectedStoredKey, tc.provider.LastSetKey)
				}
				if tc.provider.LastSetWindow != tc.expectedStoredWindow {
					t.Fatalf("expected stored window %v, got %v", tc.expectedStoredWindow, tc.provider.LastSetWindow)
				}
				if tc.provider.LastSetMax != tc.expectedStoredMax {
					t.Fatalf("expected stored max %d, got %d", tc.expectedStoredMax, tc.provider.LastSetMax)
				}
			}
		})
	}
}

func TestRateLimitPluginExecuteRateLimit(t *testing.T) {
	t.Parallel()

	now := time.Unix(5000, 0)
	tests := []struct {
		name               string
		maxRequests        int
		count              int
		resetTime          time.Time
		allowed            bool
		expectedHandled    bool
		expectedStatusCode int
		expectedLimit      string
		expectedRemaining  string
	}{
		{
			name:               "allowed request sets headers only",
			maxRequests:        10,
			count:              3,
			resetTime:          now,
			allowed:            true,
			expectedHandled:    false,
			expectedStatusCode: 0,
			expectedLimit:      "10",
			expectedRemaining:  "7",
		},
		{
			name:               "blocked request returns too many requests",
			maxRequests:        2,
			count:              3,
			resetTime:          now.Add(1 * time.Minute),
			allowed:            false,
			expectedHandled:    true,
			expectedStatusCode: http.StatusTooManyRequests,
			expectedLimit:      "2",
			expectedRemaining:  "0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reqCtx := newTestRequestContext(t, map[string]any{})
			plugin := &RateLimitPlugin{}

			if err := plugin.executeRateLimit(reqCtx, tc.maxRequests, tc.count, tc.resetTime, tc.allowed); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if reqCtx.Handled != tc.expectedHandled {
				t.Fatalf("expected handled=%v, got %v", tc.expectedHandled, reqCtx.Handled)
			}
			if got := reqCtx.ResponseWriter.Header().Get("X-RateLimit-Limit"); got != tc.expectedLimit {
				t.Fatalf("expected X-RateLimit-Limit=%q, got %q", tc.expectedLimit, got)
			}
			if got := reqCtx.ResponseWriter.Header().Get("X-RateLimit-Remaining"); got != tc.expectedRemaining {
				t.Fatalf("expected X-RateLimit-Remaining=%q, got %q", tc.expectedRemaining, got)
			}
			if got := reqCtx.ResponseWriter.Header().Get("X-RateLimit-Reset"); got != strconv.FormatInt(tc.resetTime.Unix(), 10) {
				t.Fatalf("expected X-RateLimit-Reset=%q, got %q", strconv.FormatInt(tc.resetTime.Unix(), 10), got)
			}
			if tc.expectedStatusCode != 0 {
				if got := reqCtx.ResponseStatus; got != tc.expectedStatusCode {
					t.Fatalf("expected status %d, got %d", tc.expectedStatusCode, got)
				}
				if !reqCtx.ResponseReady {
					t.Fatal("expected response to be marked ready")
				}
			}
		})
	}
}

func TestRateLimitPluginStoredRuleHook(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                  string
		values                map[string]any
		expectedHandledFirst  bool
		expectedLimitHeader   string
		secondCall            bool
		expectedHandledSecond bool
	}{
		{
			name:                 "no payload",
			values:               map[string]any{},
			expectedHandledFirst: false,
			expectedLimitHeader:  "",
		},
		{
			name: "valid stored rule allows first request",
			values: map[string]any{
				models.ContextRateLimitRule.String(): models.RateLimitRuleContext{Key: "api-key-1", WindowSeconds: 60, MaxRequests: 10},
			},
			expectedHandledFirst: false,
			expectedLimitHeader:  "10",
		},
		{
			name: "valid stored rule blocks second request",
			values: map[string]any{
				models.ContextRateLimitRule.String(): models.RateLimitRuleContext{Key: "api-key-2", WindowSeconds: 60, MaxRequests: 1},
			},
			expectedHandledFirst:  false,
			expectedLimitHeader:   "1",
			secondCall:            true,
			expectedHandledSecond: true,
		},
		{
			name: "invalid stored rule payload is ignored",
			values: map[string]any{
				models.ContextRateLimitRule.String(): models.RateLimitRuleContext{Key: "", WindowSeconds: 60, MaxRequests: 10},
			},
			expectedHandledFirst: false,
			expectedLimitHeader:  "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plugin, _ := newTestRateLimitPlugin(t)
			reqCtx := newTestRequestContext(t, tc.values)

			if err := plugin.checkRateLimitRuleHook()(reqCtx); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if reqCtx.Handled != tc.expectedHandledFirst {
				t.Fatalf("expected first handled=%v, got %v", tc.expectedHandledFirst, reqCtx.Handled)
			}
			if got := reqCtx.ResponseWriter.Header().Get("X-RateLimit-Limit"); got != tc.expectedLimitHeader {
				t.Fatalf("expected X-RateLimit-Limit=%q, got %q", tc.expectedLimitHeader, got)
			}

			if tc.secondCall {
				if err := plugin.checkRateLimitRuleHook()(reqCtx); err != nil {
					t.Fatalf("unexpected error on second call: %v", err)
				}
				if reqCtx.Handled != tc.expectedHandledSecond {
					t.Fatalf("expected second handled=%v, got %v", tc.expectedHandledSecond, reqCtx.Handled)
				}
			}
		})
	}
}

func TestGetRateLimitRuleContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		value      any
		expected   models.RateLimitRuleContext
		expectedOK bool
	}{
		{
			name:       "direct value",
			value:      models.RateLimitRuleContext{Key: "rule-1", WindowSeconds: 30, MaxRequests: 3},
			expected:   models.RateLimitRuleContext{Key: "rule-1", WindowSeconds: 30, MaxRequests: 3},
			expectedOK: true,
		},
		{
			name: "pointer value",
			value: func() *models.RateLimitRuleContext {
				v := models.RateLimitRuleContext{Key: "rule-2", WindowSeconds: 45, MaxRequests: 4}
				return &v
			}(),
			expected:   models.RateLimitRuleContext{Key: "rule-2", WindowSeconds: 45, MaxRequests: 4},
			expectedOK: true,
		},
		{
			name:       "nil pointer",
			value:      (*models.RateLimitRuleContext)(nil),
			expected:   models.RateLimitRuleContext{},
			expectedOK: false,
		},
		{
			name:       "invalid type",
			value:      "rule",
			expected:   models.RateLimitRuleContext{},
			expectedOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := getRateLimitRuleContext(tc.value)
			if ok != tc.expectedOK {
				t.Fatalf("expected ok=%v, got %v", tc.expectedOK, ok)
			}
			if got != tc.expected {
				t.Fatalf("expected %+v, got %+v", tc.expected, got)
			}
		})
	}
}

func TestRateLimitContextSetters(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		setter   func(*models.RequestContext)
		expected models.RateLimitRuleContext
	}{
		{
			name: "set rule context directly",
			setter: func(reqCtx *models.RequestContext) {
				SetRuleContext(reqCtx, models.RateLimitRuleContext{Key: "api-key-1", WindowSeconds: 60, MaxRequests: 100})
			},
			expected: models.RateLimitRuleContext{Key: "api-key-1", WindowSeconds: 60, MaxRequests: 100},
		},
		{
			name: "set rule context from record",
			setter: func(reqCtx *models.RequestContext) {
				SetRuleContextFromRecord(reqCtx, &types.RateLimitRuleRecord{Key: "api-key-2", WindowSeconds: 120, MaxRequests: 7})
			},
			expected: models.RateLimitRuleContext{Key: "api-key-2", WindowSeconds: 120, MaxRequests: 7},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reqCtx := &models.RequestContext{Values: make(map[string]any)}
			tc.setter(reqCtx)
			assertRateLimitRuleContext(t, reqCtx, tc.expected)
		})
	}
}

func newTestRequestContext(t *testing.T, values map[string]any) *models.RequestContext {
	t.Helper()

	if values == nil {
		values = make(map[string]any)
	}

	return &models.RequestContext{
		Request:        httptest.NewRequest(http.MethodGet, "/test", nil),
		ResponseWriter: httptest.NewRecorder(),
		Values:         values,
	}
}

func assertRateLimitRuleContext(t *testing.T, reqCtx *models.RequestContext, expected models.RateLimitRuleContext) {
	t.Helper()

	rawValue, ok := reqCtx.Values[models.ContextRateLimitRule.String()]
	if !ok {
		t.Fatal("expected rule context to be stored")
	}
	stored, ok := rawValue.(models.RateLimitRuleContext)
	if !ok {
		t.Fatalf("expected typed rule context, got %T", rawValue)
	}
	if stored != expected {
		t.Fatalf("unexpected stored rule context: got %+v, expected %+v", stored, expected)
	}
}
