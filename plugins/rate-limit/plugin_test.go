package ratelimit

import (
	"context"
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
