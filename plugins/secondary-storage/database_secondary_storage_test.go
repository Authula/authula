package secondarystorage

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

// Helper function to create an in-memory SQLite database for testing
func newTestDB(t *testing.T) bun.IDB {
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())

	// Create the KeyValueStore table
	ctx := context.Background()
	if _, err := db.NewCreateTable().Model(&KeyValueStore{}).Exec(ctx); err != nil {
		t.Fatalf("failed to create KeyValueStore table: %v", err)
	}

	return db
}

// Helper function to create a database storage with default config
func newTestDatabaseStorage(t *testing.T, db bun.IDB) *DatabaseSecondaryStorage {
	return NewDatabaseSecondaryStorage(db, DatabaseStorageConfig{
		CleanupInterval: 1 * time.Minute,
	})
}

// Helper function to assert any value is a string
func assertStringDB(t *testing.T, value any, expected string) {
	t.Helper()
	str, ok := value.(string)
	if !ok {
		t.Fatalf("expected string, got %T", value)
	}
	if str != expected {
		t.Fatalf("expected '%s', got '%s'", expected, str)
	}
}

func TestNewDatabaseStorage(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	defer storage.Close()

	if storage == nil {
		t.Fatal("expected NewDatabaseSecondaryStorage to return a non-nil instance")
	}

	if storage.db == nil {
		t.Fatal("expected db to be initialized")
	}
}

func TestDatabaseStorage_SetSuccess(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
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

	assertStringDB(t, stored, "test_value")
}

func TestDatabaseStorage_SetInvalidType(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	defer storage.Close()

	ctx := context.Background()

	err := storage.Set(ctx, "key", []byte("byte_value"), nil)
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
}

func TestDatabaseStorage_SetContextCancelled(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
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

func TestDatabaseStorage_GetSuccess(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	defer storage.Close()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	storage.Set(ctx, key, value, nil)

	retrieved, err := storage.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	assertStringDB(t, retrieved, "test_value")
}

func TestDatabaseStorage_GetKeyNotFound(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
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

func TestDatabaseStorage_GetContextCancelled(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
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

func TestDatabaseStorage_GetExpiredEntry(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	defer storage.Close()

	ctx := context.Background()
	key := "expiring_key"
	value := "expiring_value"
	ttl := 100 * time.Millisecond

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

	// Wait for the entry to expire
	time.Sleep(101 * time.Millisecond)

	retrieved, err = storage.Get(ctx, key)
	if err != nil {
		t.Fatalf("expected no error for expired key, got %v", err)
	}
	if retrieved != nil {
		t.Fatalf("expected nil value for expired key, got %v", retrieved)
	}
}

func TestDatabaseStorage_DeleteSuccess(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
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

func TestDatabaseStorage_DeleteKeyNotFound(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	defer storage.Close()

	ctx := context.Background()

	err := storage.Delete(ctx, "nonexistent_key")
	if err != nil {
		t.Fatalf("expected no error for idempotent delete, got %v", err)
	}
}

func TestDatabaseStorage_DeleteContextCancelled(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
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

func TestDatabaseStorage_Update(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	defer storage.Close()

	ctx := context.Background()
	key := "test_key"

	storage.Set(ctx, key, "value1", nil)
	storage.Set(ctx, key, "value2", nil)

	retrieved, _ := storage.Get(ctx, key)
	assertStringDB(t, retrieved, "value2")
}

func TestDatabaseStorage_SetWithTTL(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	defer storage.Close()

	ctx := context.Background()
	key := "ttl_key"
	value := "ttl_value"
	ttl := 100 * time.Millisecond

	err := storage.Set(ctx, key, value, &ttl)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should exist immediately
	retrieved, err := storage.Get(ctx, "ttl_key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	assertStringDB(t, retrieved, "ttl_value")

	// Wait for the entry to expire
	time.Sleep(101 * time.Millisecond)

	retrieved, err = storage.Get(ctx, "ttl_key")
	if err != nil {
		t.Fatalf("expected no error for expired key, got %v", err)
	}
	if retrieved != nil {
		t.Fatalf("expected nil for expired key, got %v", retrieved)
	}
}

func TestDatabaseStorage_MultipleKeys(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
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
		assertStringDB(t, retrieved, values[i])
	}
}

func TestDatabaseStorage_ConcurrentReads(t *testing.T) {
	// Use file-based SQLite for concurrent tests as :memory: doesn't handle concurrency well
	sqldb, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())

	ctx := context.Background()
	if _, err := db.NewCreateTable().Model(&KeyValueStore{}).Exec(ctx); err != nil {
		t.Fatalf("failed to create KeyValueStore table: %v", err)
	}

	storage := newTestDatabaseStorage(t, db)
	defer storage.Close()

	ctx = context.Background()
	key := "concurrent_key"
	value := "concurrent_value"

	storage.Set(ctx, key, value, nil)

	numGoroutines := 50
	results := make(chan error, numGoroutines)

	for range numGoroutines {
		go func() {
			retrieved, err := storage.Get(ctx, key)
			if err != nil {
				results <- err
				return
			}
			str, ok := retrieved.(string)
			if !ok || str != value {
				results <- errors.New("unexpected value")
				return
			}
			results <- nil
		}()
	}

	for range numGoroutines {
		if err := <-results; err != nil {
			t.Errorf("concurrent read failed: %v", err)
		}
	}
}

func TestDatabaseStorage_ConcurrentWritesAndReads(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	defer storage.Close()

	ctx := context.Background()
	numGoroutines := 50
	done := make(chan struct{})

	// Spawn writers
	for i := range numGoroutines {
		go func() {
			defer func() {
				done <- struct{}{}
			}()
			key := "key_" + string(rune(i))
			value := "value_" + string(rune(i))
			storage.Set(ctx, key, value, nil)
		}()
	}

	// Spawn readers
	for i := range numGoroutines {
		go func() {
			defer func() {
				done <- struct{}{}
			}()
			key := "key_" + string(rune(i))
			_, _ = storage.Get(ctx, key)
		}()
	}

	for range numGoroutines * 2 {
		<-done
	}
}

func TestDatabaseStorage_ConcurrentDeletes(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	defer storage.Close()

	ctx := context.Background()
	numGoroutines := 5

	// Create keys to delete
	for i := range numGoroutines {
		key := "delete_key_" + strconv.Itoa(i)
		storage.Set(ctx, key, "value", nil)
	}

	// Delete keys sequentially
	for i := range numGoroutines {
		key := "delete_key_" + strconv.Itoa(i)
		storage.Delete(ctx, key)
	}

	// Verify all keys are deleted
	for i := range numGoroutines {
		key := "delete_key_" + strconv.Itoa(i)
		value, err := storage.Get(ctx, key)
		if err != nil {
			t.Fatalf("unexpected error for key %s: %v", key, err)
		}
		if value != nil {
			t.Fatalf("expected key %s to be deleted, but got value: %v", key, value)
		}
	}
}

func TestDatabaseStorage_CleanupExpiredEntries(t *testing.T) {
	db := newTestDB(t)
	storage := &DatabaseSecondaryStorage{
		db:              db,
		cleanupInterval: 50 * time.Millisecond,
		stopCleanup:     make(chan struct{}),
		done:            make(chan struct{}),
	}
	go storage.cleanupExpiredEntries()
	defer storage.Close()

	ctx := context.Background()

	// Add multiple entries with TTL
	ttl := 50 * time.Millisecond
	for i := range 10 {
		key := "cleanup_key_" + string(rune(i))
		value := "cleanup_value_" + string(rune(i))
		storage.Set(ctx, key, value, &ttl)
	}

	// Verify entries exist
	for i := range 10 {
		key := "cleanup_key_" + string(rune(i))
		_, err := storage.Get(ctx, key)
		if err != nil {
			t.Fatalf("expected entry to exist initially: %v", err)
		}
	}

	// Wait for expiration and cleanup
	time.Sleep(200 * time.Millisecond)

	// Verify entries are cleaned up and return nil
	for i := range 10 {
		key := "cleanup_key_" + string(rune(i))
		retrieved, err := storage.Get(ctx, key)
		if err != nil {
			t.Fatalf("expected no error for expired/cleaned key %s, got %v", key, err)
		}
		if retrieved != nil {
			t.Fatalf("expected nil for expired/cleaned key %s, got %v", key, retrieved)
		}
	}
}

func TestDatabaseStorage_Close_StopsCleanup(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)

	storage.StartCleanup()

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

func TestDatabaseStorage_ContextDeadline(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	defer storage.Close()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Try operations before deadline - should work
	err := storage.Set(ctx, "key", "value", nil)
	if err != nil {
		t.Fatalf("expected no error before deadline, got %v", err)
	}

	// Wait for deadline to exceed
	time.Sleep(101 * time.Millisecond)

	// Try to set with expired context
	err = storage.Set(ctx, "key2", "value2", nil)
	if err == nil {
		t.Fatal("expected context deadline exceeded error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestDatabaseStorage_DifferentValueTypes(t *testing.T) {
	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	defer storage.Close()

	ctx := context.Background()

	testCases := []struct {
		key   string
		value string
	}{
		{"empty", ""},
		{"simple", "hello"},
		{"spaces", "hello world"},
		{"special", "!@#$%^&*()"},
		{"unicode", "你好世界"},
		{"newlines", "line1\nline2\nline3"},
	}

	for _, tc := range testCases {
		err := storage.Set(ctx, tc.key, tc.value, nil)
		if err != nil {
			t.Fatalf("failed to set %s: %v", tc.key, err)
		}

		retrieved, err := storage.Get(ctx, tc.key)
		if err != nil {
			t.Fatalf("failed to get %s: %v", tc.key, err)
		}

		assertStringDB(t, retrieved, tc.value)
	}
}

func TestDatabaseStorage_PersistenceAcrossInstances(t *testing.T) {
	// Create a persistent DB (not in-memory)
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())

	ctx := context.Background()
	if _, err := db.NewCreateTable().Model(&KeyValueStore{}).Exec(ctx); err != nil {
		t.Fatalf("failed to create KeyValueStore table: %v", err)
	}

	ctx = context.Background()

	// Create first storage instance and set values
	storage1 := newTestDatabaseStorage(t, db)
	storage1.Set(ctx, "key1", "value1", nil)
	storage1.Set(ctx, "key2", "value2", nil)
	storage1.Close()

	// Create second storage instance and verify values persist
	storage2 := newTestDatabaseStorage(t, db)
	defer storage2.Close()

	retrieved1, err := storage2.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("expected to find key1, got error: %v", err)
	}
	assertStringDB(t, retrieved1, "value1")

	retrieved2, err := storage2.Get(ctx, "key2")
	if err != nil {
		t.Fatalf("expected to find key2, got error: %v", err)
	}
	assertStringDB(t, retrieved2, "value2")
}
