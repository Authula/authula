package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/rate-limit/services"
	"github.com/Authula/authula/plugins/rate-limit/types"
)

func TestInMemoryProvider(t *testing.T) {
	provider := services.NewInMemoryProvider()
	defer func() {
		if err := provider.Close(); err != nil {
			t.Fatalf("failed to close provider: %v", err)
		}
	}()

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
	config := types.RateLimitPluginConfig{
		Enabled:  true,
		Window:   1 * time.Minute,
		Max:      100,
		Prefix:   "ratelimit:",
		Provider: types.RateLimitProviderInMemory,
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
	provider := services.NewInMemoryProvider()
	defer func() {
		if err := provider.Close(); err != nil {
			t.Fatalf("failed to close provider: %v", err)
		}
	}()

	if name := provider.GetName(); name != string(types.RateLimitProviderInMemory) {
		t.Errorf("in-memory provider name should be 'memory', got %s", name)
	}
}

func TestRateLimitPlugin_Init_RegistersSharedService(t *testing.T) {
	t.Parallel()

	plugin := New(types.RateLimitPluginConfig{Provider: types.RateLimitProviderInMemory})
	registry := &internaltests.MockServiceRegistry{}
	registry.On("Register", models.ServiceRateLimit.String(), mock.Anything).Once()

	err := plugin.Init(&models.PluginContext{
		Logger:          &internaltests.MockLogger{},
		ServiceRegistry: registry,
		GetConfig: func() *models.Config {
			return &models.Config{PreParsedConfigs: map[string]any{models.PluginRateLimit.String(): types.RateLimitPluginConfig{Provider: types.RateLimitProviderInMemory}}}
		},
	})
	require.NoError(t, err)
	registry.AssertExpectations(t)
}

func TestRateLimiterService_DelegatesToProvider(t *testing.T) {
	t.Parallel()

	provider := services.NewInMemoryProvider()
	service := services.NewRateLimiterService(provider)
	allowed, count, _, err := service.CheckAndIncrement(context.Background(), "test:key", time.Minute, 1)
	require.NoError(t, err)
	if !allowed || count != 1 {
		t.Fatalf("expected allowed first request, got allowed=%v count=%d", allowed, count)
	}
}
