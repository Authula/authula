package services

import (
	"context"
	"testing"
	"time"
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
