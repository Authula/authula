package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// TestInMemoryProvider tests the in-memory rate limit provider
func TestInMemoryProvider(t *testing.T) {
	provider := NewInMemoryProvider()
	defer provider.Close()

	ctx := context.Background()
	window := 1 * time.Minute
	maxRequests := 5

	// Test initial request is allowed
	allowed, count, _, err := provider.CheckAndIncrement(ctx, "test:key", window, maxRequests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed || count != 1 {
		t.Errorf("first request should be allowed, got allowed=%v count=%d", allowed, count)
	}

	// Test requests up to limit
	for i := 2; i <= maxRequests; i++ {
		allowedN, countN, _, err := provider.CheckAndIncrement(ctx, "test:key", window, maxRequests)
		if err != nil {
			t.Fatalf("unexpected error at iteration %d: %v", i, err)
		}
		if !allowedN || countN != i {
			t.Errorf("request %d should be allowed, got allowed=%v count=%d", i, allowedN, countN)
		}
	}

	// Test request beyond limit is denied
	allowedN, countN, _, err := provider.CheckAndIncrement(ctx, "test:key", window, maxRequests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowedN || countN != maxRequests+1 {
		t.Errorf("request beyond limit should not be allowed, got allowed=%v count=%d", allowedN, countN)
	}

	// Test different key is independent
	allowedD, countD, _, err := provider.CheckAndIncrement(ctx, "different:key", window, maxRequests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowedD || countD != 1 {
		t.Errorf("different key should start at 1, got allowed=%v count=%d", allowedD, countD)
	}
}

// TestRateLimitPluginConfig tests the rate limit plugin config
func TestRateLimitPluginConfig(t *testing.T) {
	config := RateLimitPluginConfig{
		Enabled:  true,
		Window:   1 * time.Minute,
		Max:      100,
		Prefix:   "ratelimit:",
		Provider: RateLimitProviderInMemory,
	}

	plugin := New(config)
	metadata := plugin.Metadata()

	if metadata.ID != models.PluginRateLimit.String() {
		t.Errorf("plugin ID should be 'ratelimit', got %s", metadata.ID)
	}

	if plugin.Config() == nil {
		t.Error("plugin config should not be nil")
	}
}

// TestProviderNames ensures the provider is initialized with correct name
func TestProviderNames(t *testing.T) {
	provider := NewInMemoryProvider()
	defer provider.Close()

	if name := provider.GetName(); name != string(RateLimitProviderInMemory) {
		t.Errorf("in-memory provider name should be 'memory', got %s", name)
	}
}
