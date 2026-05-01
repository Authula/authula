package ratelimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/rate-limit/services"
	"github.com/Authula/authula/plugins/rate-limit/types"
)

func TestInMemoryProvider(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		key         string
		window      time.Duration
		maxRequests int
		calls       []struct {
			allowed bool
			count   int
		}
	}{
		{
			name:        "request sequence with overflow",
			key:         "test:key",
			window:      1 * time.Minute,
			maxRequests: 5,
			calls: []struct {
				allowed bool
				count   int
			}{
				{allowed: true, count: 1},
				{allowed: true, count: 2},
				{allowed: true, count: 3},
				{allowed: true, count: 4},
				{allowed: true, count: 5},
				{allowed: false, count: 6},
			},
		},
		{
			name:        "different key starts at one",
			key:         "different:key",
			window:      1 * time.Minute,
			maxRequests: 5,
			calls: []struct {
				allowed bool
				count   int
			}{
				{allowed: true, count: 1},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			provider := services.NewInMemoryProvider()
			t.Cleanup(func() {
				if err := provider.Close(); err != nil {
					t.Fatalf("failed to close provider: %v", err)
				}
			})

			ctx := context.Background()
			for _, call := range tc.calls {
				allowed, count, _, err := provider.CheckAndIncrement(ctx, tc.key, tc.window, tc.maxRequests)
				if err != nil {
					t.Fatalf("unexpected error for key %q: %v", tc.key, err)
				}
				if allowed != call.allowed || count != call.count {
					t.Fatalf("expected allowed=%v count=%d, got allowed=%v count=%d", call.allowed, call.count, allowed, count)
				}
			}
		})
	}
}

func TestRateLimitPluginConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		config     types.RateLimitPluginConfig
		expectedID string
	}{
		{
			name: "default in-memory config",
			config: types.RateLimitPluginConfig{
				Enabled:  true,
				Window:   1 * time.Minute,
				Max:      100,
				Prefix:   "ratelimit:",
				Provider: types.RateLimitProviderInMemory,
			},
			expectedID: models.PluginRateLimit.String(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(tc.config)
			metadata := plugin.Metadata()

			if metadata.ID != tc.expectedID {
				t.Fatalf("expected plugin ID %q, got %q", tc.expectedID, metadata.ID)
			}
			if plugin.Config() == nil {
				t.Fatal("plugin config should not be nil")
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

			if err := plugin.checkStoredRateLimitRuleHook()(reqCtx); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if reqCtx.Handled != tc.expectedHandledFirst {
				t.Fatalf("expected first handled=%v, got %v", tc.expectedHandledFirst, reqCtx.Handled)
			}
			if got := reqCtx.ResponseWriter.Header().Get("X-RateLimit-Limit"); got != tc.expectedLimitHeader {
				t.Fatalf("expected X-RateLimit-Limit=%q, got %q", tc.expectedLimitHeader, got)
			}

			if tc.secondCall {
				if err := plugin.checkStoredRateLimitRuleHook()(reqCtx); err != nil {
					t.Fatalf("unexpected error on second call: %v", err)
				}
				if reqCtx.Handled != tc.expectedHandledSecond {
					t.Fatalf("expected second handled=%v, got %v", tc.expectedHandledSecond, reqCtx.Handled)
				}
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

func newTestRateLimitPlugin(t *testing.T) (*RateLimitPlugin, types.RateLimitProvider) {
	t.Helper()

	provider := services.NewInMemoryProvider()
	t.Cleanup(func() {
		if err := provider.Close(); err != nil {
			t.Fatalf("failed to close provider: %v", err)
		}
	})

	return &RateLimitPlugin{provider: provider, logger: &internaltests.MockLogger{}}, provider
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
