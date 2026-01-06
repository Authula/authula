package secondarystorage

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// Helper function to create a memory storage with default config
func newTestMemorySecondaryStorage() *MemorySecondaryStorage {
	return NewMemorySecondaryStorage(MemoryStorageConfig{
		CleanupInterval: 1 * time.Minute,
	})
}

// Helper function to assert any value is a string
func assertString(t *testing.T, value any, expected string) {
	t.Helper()
	str, ok := value.(string)
	if !ok {
		t.Fatalf("expected string, got %T", value)
	}
	if str != expected {
		t.Fatalf("expected '%s', got '%s'", expected, str)
	}
}

func TestNewMemorySecondaryStorage(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	if storage == nil {
		t.Fatal("expected NewMemoryStorage to return a non-nil instance")
	}

	if storage.store == nil {
		t.Fatal("expected store to be initialized")
	}

	if len(storage.store) != 0 {
		t.Fatal("expected store to be empty on initialization")
	}
}

func TestSet_Success(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	err := storage.Set(ctx, key, value, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stored, err := storage.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	assertString(t, stored, "test_value")
}

func TestSet_InvalidType(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx := context.Background()

	err := storage.Set(ctx, "key", []byte("byte_value"), nil)
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
}

func TestSet_ContextCancelled(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := storage.Set(ctx, "key", "value", nil)
	if err == nil {
		t.Fatal("expected context cancelled error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestGet_Success(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	storage.Set(ctx, key, value, nil)

	retrieved, err := storage.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	assertString(t, retrieved, "test_value")
}

func TestGet_KeyNotFound(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx := context.Background()

	value, err := storage.Get(ctx, "nonexistent_key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if value != nil {
		t.Fatalf("expected nil value for non-existent key, got %v", value)
	}
}

func TestGet_ContextCancelled(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := storage.Get(ctx, "key")
	if err == nil {
		t.Fatal("expected context cancelled error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestGet_ExpiredEntry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		storage := newTestMemorySecondaryStorage()
		defer storage.Close()

		ctx := context.Background()
		key := "expiring_key"
		value := "expiring_value"
		ttl := 10 * time.Millisecond

		err := storage.Set(ctx, key, value, &ttl)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify entry exists
		retrieved, err := storage.Get(ctx, key)
		if err != nil {
			t.Fatalf("expected entry to exist, got error: %v", err)
		}
		if retrieved != value {
			t.Fatalf("expected value %s, got %v", value, retrieved)
		}

		// Wait for the entry to expire - the fake clock will automatically advance
		// when this goroutine is durably blocked on time.Sleep
		time.Sleep(11 * time.Millisecond)

		retrieved, err = storage.Get(ctx, key)
		if err != nil {
			t.Fatalf("expected no error for expired key, got %v", err)
		}
		if retrieved != nil {
			t.Fatalf("expected nil value for expired key, got %v", retrieved)
		}
	})
}

func TestDelete_Success(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	storage.Set(ctx, key, value, nil)

	err := storage.Delete(ctx, key)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	retrieved, err := storage.Get(ctx, key)
	if err != nil {
		t.Fatalf("expected no error for deleted key, got %v", err)
	}
	if retrieved != nil {
		t.Fatalf("expected nil value after deletion, got %v", retrieved)
	}
}

func TestDelete_KeyNotFound(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx := context.Background()

	err := storage.Delete(ctx, "nonexistent_key")
	if err != nil {
		t.Fatalf("expected no error for idempotent delete, got %v", err)
	}
}

func TestDelete_ContextCancelled(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := storage.Delete(ctx, "key")
	if err == nil {
		t.Fatal("expected context cancelled error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestValueMutation_Prevented(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx := context.Background()
	key := "test_key"
	originalValue := "original"

	storage.Set(ctx, key, originalValue, nil)

	retrieved, _ := storage.Get(ctx, key)
	assertString(t, retrieved, "original")

	retrieved2, _ := storage.Get(ctx, key)
	assertString(t, retrieved2, "original")
}

func TestSet_WithTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		storage := newTestMemorySecondaryStorage()
		defer storage.Close()

		ctx := context.Background()
		key := "ttl_key"
		value := "ttl_value"
		ttl := 20 * time.Millisecond

		err := storage.Set(ctx, key, value, &ttl)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Should exist immediately
		retrieved, err := storage.Get(ctx, "ttl_key")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		assertString(t, retrieved, "ttl_value")

		// Wait just before expiration using fake time
		time.Sleep(19 * time.Millisecond)
		synctest.Wait()

		// Should still exist
		_, err = storage.Get(ctx, key)
		if err != nil {
			t.Fatalf("expected entry to exist before expiration, got error: %v", err)
		}

		// Wait past expiration using fake time - 2ms more than TTL
		time.Sleep(2 * time.Millisecond)
		synctest.Wait()

		// Should now be expired and return nil
		retrieved, err = storage.Get(ctx, key)
		if err != nil {
			t.Fatalf("expected no error for expired key, got %v", err)
		}
		if retrieved != nil {
			t.Fatalf("expected nil for expired key, got %v", retrieved)
		}
	})
}

func TestSet_OverwriteExisting(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx := context.Background()
	key := "test_key"

	storage.Set(ctx, key, "value1", nil)
	storage.Set(ctx, key, "value2", nil)

	retrieved, _ := storage.Get(ctx, key)
	assertString(t, retrieved, "value2")
}

func TestConcurrentReads(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx := context.Background()
	key := "concurrent_key"
	value := "concurrent_value"

	storage.Set(ctx, key, value, nil)

	numGoroutines := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			retrieved, err := storage.Get(ctx, key)
			if err != nil {
				t.Errorf("expected no error in concurrent read, got %v", err)
			}
			assertString(t, retrieved, "concurrent_value")
		}()
	}

	wg.Wait()
}

func TestConcurrentWritesAndReads(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx := context.Background()
	numGoroutines := 50
	var wg sync.WaitGroup

	// Spawn writers
	for i := range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := "key_" + string(rune(i))
			value := "value_" + string(rune(i))
			storage.Set(ctx, key, value, nil)
		}()
	}

	// Spawn readers
	for i := range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := "key_" + string(rune(i))
			_, _ = storage.Get(ctx, key)
		}()
	}

	wg.Wait()
}

func TestConcurrentDeletes(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx := context.Background()
	numKeys := 50

	// Setup: add keys
	for i := range numKeys {
		key := "delete_key_" + string(rune(i))
		value := []byte("delete_value_" + string(rune(i)))
		storage.Set(ctx, key, value, nil)
	}

	// Concurrent deletes
	var wg sync.WaitGroup
	for i := range numKeys {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := "delete_key_" + string(rune(i))
			storage.Delete(ctx, key)
		}()
	}

	wg.Wait()

	// Verify all keys are deleted
	for i := range numKeys {
		key := "delete_key_" + string(rune(i))
		retrieved, err := storage.Get(ctx, key)
		if err != nil {
			t.Fatalf("expected no error for deleted key %s, got %v", key, err)
		}
		if retrieved != nil {
			t.Fatalf("expected nil for deleted key %s, got %v", key, retrieved)
		}
	}
}

func TestCleanupExpiredEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Create storage with custom initialization for fast cleanup
		storage := &MemorySecondaryStorage{
			store:           make(map[string]*storageEntry),
			cleanupInterval: 5 * time.Millisecond, // Very fast cleanup for testing
			stopCleanup:     make(chan struct{}),
			done:            make(chan struct{}),
		}
		// Start the cleanup goroutine with the custom interval
		go storage.cleanupExpiredEntries()
		defer storage.Close()

		ctx := context.Background()

		// Add multiple entries with TTL
		ttl := 2 * time.Millisecond
		for i := range 10 {
			key := "cleanup_key_" + string(rune(i))
			value := "cleanup_value_" + string(rune(i))
			storage.Set(ctx, key, value, &ttl)
		}

		// Verify entries exist
		storage.mu.RLock()
		count := len(storage.store)
		storage.mu.RUnlock()
		if count != 10 {
			t.Fatalf("expected 10 entries, got %d", count)
		}

		// Wait for expiration and cleanup to complete
		// The fake clock will advance when the cleanup ticker and this goroutine
		// are both durably blocked on time operations
		time.Sleep(7 * time.Millisecond)

		// Verify entries are cleaned up
		storage.mu.RLock()
		count = len(storage.store)
		storage.mu.RUnlock()
		if count != 0 {
			t.Fatalf("expected 0 entries after cleanup, got %d", count)
		}
	})
}

func TestClose_StopsCleanup(t *testing.T) {
	storage := newTestMemorySecondaryStorage()

	// Verify cleanup goroutine is running
	select {
	case <-storage.done:
		t.Fatal("cleanup goroutine stopped before Close was called")
	default:
	}

	err := storage.Close()
	if err != nil {
		t.Fatalf("expected no error on Close, got %v", err)
	}

	// Verify cleanup goroutine has stopped
	select {
	case <-storage.done:
		// Successfully closed
	case <-time.After(1 * time.Second):
		t.Fatal("cleanup goroutine did not stop after Close")
	}
}

func TestMultipleKeys(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx := context.Background()

	keys := []string{"key1", "key2", "key3"}
	values := []string{"value1", "value2", "value3"}

	for i := range keys {
		storage.Set(ctx, keys[i], values[i], nil)
	}

	for i := range keys {
		retrieved, err := storage.Get(ctx, keys[i])
		if err != nil {
			t.Fatalf("expected no error for key %s, got %v", keys[i], err)
		}
		assertString(t, retrieved, values[i])
	}
}

func TestContextDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		storage := newTestMemorySecondaryStorage()
		defer storage.Close()

		// Create a context with a timeout using fake time
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancel()

		// Advance time just before the deadline
		time.Sleep(10*time.Millisecond - time.Nanosecond)
		synctest.Wait()

		// Should not be cancelled yet
		if ctx.Err() != nil {
			t.Fatalf("before timeout, ctx.Err() = %v; want nil", ctx.Err())
		}

		// Advance past the deadline
		time.Sleep(time.Nanosecond)
		synctest.Wait()

		// Should now be cancelled
		if ctx.Err() != context.DeadlineExceeded {
			t.Fatalf("after timeout, ctx.Err() = %v; want DeadlineExceeded", ctx.Err())
		}

		// Try to set with expired context
		err := storage.Set(ctx, "key", "value", nil)
		if err == nil {
			t.Fatal("expected context deadline exceeded error, got nil")
		}

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context.DeadlineExceeded, got %v", err)
		}
	})
}

func TestRaceConditions(t *testing.T) {
	storage := newTestMemorySecondaryStorage()
	defer storage.Close()

	ctx := context.Background()
	key := "race_key"

	var counter int64
	numGoroutines := 50

	// Run concurrent operations
	var wg sync.WaitGroup
	for i := range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Interleave operations
			switch i % 3 {
			case 0:
				storage.Set(ctx, key, "value", nil)
				atomic.AddInt64(&counter, 1)
			case 1:
				storage.Get(ctx, key)
				atomic.AddInt64(&counter, 1)
			default:
				storage.Delete(ctx, key)
				atomic.AddInt64(&counter, 1)
			}
		}()
	}

	wg.Wait()

	// Verify all operations completed
	if atomic.LoadInt64(&counter) != int64(numGoroutines) {
		t.Fatalf("expected %d operations, got %d", numGoroutines, atomic.LoadInt64(&counter))
	}
}
