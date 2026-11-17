package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Authula/authula/plugins/rate-limit/types"
)

func TestInMemoryProviderRuleLifecycle(t *testing.T) {
	t.Parallel()

	provider := NewInMemoryProvider()
	ctx := context.Background()

	window := 2 * time.Minute
	maxRequests := 42

	if err := provider.SetRule(ctx, "api-key-1", window, maxRequests); err != nil {
		t.Fatalf("SetRule returned error: %v", err)
	}

	gotWindow, gotMaxRequests, found, err := provider.GetRule(ctx, "api-key-1")
	if err != nil {
		t.Fatalf("GetRule returned error: %v", err)
	}
	if !found {
		t.Fatal("expected rule to be found")
	}
	if gotWindow != window || gotMaxRequests != maxRequests {
		t.Fatalf("unexpected rule values: window=%v max=%d", gotWindow, gotMaxRequests)
	}

	if err := provider.DeleteRule(ctx, "api-key-1"); err != nil {
		t.Fatalf("DeleteRule returned error: %v", err)
	}

	_, _, found, err = provider.GetRule(ctx, "api-key-1")
	if err != nil {
		t.Fatalf("GetRule after delete returned error: %v", err)
	}
	if found {
		t.Fatal("expected rule to be deleted")
	}
}

func TestInMemoryProvider_MetadataAndValues(t *testing.T) {
	t.Parallel()

	provider := NewInMemoryProvider()
	t.Cleanup(func() { _ = provider.Close() })

	if provider.GetName() != "memory" {
		t.Fatalf("expected provider name memory, got %q", provider.GetName())
	}

	ctx := context.Background()
	value, err := provider.GetValue(ctx, "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != nil {
		t.Fatalf("expected nil value, got %#v", value)
	}

	allowed, count, resetAt, err := provider.CheckAndIncrement(ctx, "key-1", time.Minute, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed || count != 1 {
		t.Fatalf("expected allowed count=1, got allowed=%v count=%d", allowed, count)
	}
	if resetAt.Before(time.Now()) {
		t.Fatalf("expected reset time in the future, got %v", resetAt)
	}

	value, err = provider.GetValue(ctx, "key-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rateLimitValue, ok := value.(types.RateLimitValue)
	if !ok {
		t.Fatalf("expected RateLimitValue, got %T", value)
	}
	if rateLimitValue.Count != 1 {
		t.Fatalf("expected count 1, got %d", rateLimitValue.Count)
	}

	if err := provider.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestInMemoryProvider_CheckAndIncrementEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		cancelContext bool
		seed          *inMemoryEntry
		window        time.Duration
		max           int
		wantErr       bool
		wantAllowed   bool
		wantCount     int
		wantReset     bool
	}{
		{
			name:          "canceled_context_errors",
			cancelContext: true,
			window:        time.Minute,
			max:           5,
			wantErr:       true,
		},
		{
			name:        "expired_entry_is_reset",
			seed:        &inMemoryEntry{count: 9, expiresAt: time.Now().Add(-time.Minute)},
			window:      time.Minute,
			max:         5,
			wantAllowed: true,
			wantCount:   1,
			wantReset:   true,
		},
		{
			name:        "over_limit_is_denied",
			seed:        &inMemoryEntry{count: 5, expiresAt: time.Now().Add(time.Minute)},
			window:      time.Minute,
			max:         5,
			wantAllowed: false,
			wantCount:   6,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			provider := NewInMemoryProvider()
			t.Cleanup(func() { _ = provider.Close() })
			if tc.seed != nil {
				provider.store["key-1"] = tc.seed
			}

			ctx := context.Background()
			if tc.cancelContext {
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
			}

			allowed, count, resetAt, err := provider.CheckAndIncrement(ctx, "key-1", tc.window, tc.max)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tc.wantAllowed || count != tc.wantCount {
				t.Fatalf("expected allowed=%v count=%d, got allowed=%v count=%d", tc.wantAllowed, tc.wantCount, allowed, count)
			}
			if tc.wantReset && resetAt.Before(time.Now()) {
				t.Fatalf("expected reset time in the future, got %v", resetAt)
			}
		})
	}
}

func TestInMemoryProvider_CleanupExpired(t *testing.T) {
	t.Parallel()

	provider := NewInMemoryProviderWithConfig(types.MemoryStorageConfig{CleanupInterval: 5 * time.Millisecond})
	t.Cleanup(func() { _ = provider.Close() })

	provider.store["expired"] = &inMemoryEntry{count: 1, expiresAt: time.Now().Add(-time.Minute)}

	assert.Eventually(t, func() bool {
		provider.mu.RLock()
		defer provider.mu.RUnlock()
		_, exists := provider.store["expired"]
		return !exists
	}, time.Second, 5*time.Millisecond)
}
