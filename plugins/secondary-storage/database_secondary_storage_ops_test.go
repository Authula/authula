package secondarystorage

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	internaltests "github.com/Authula/authula/internal/tests"
)

func insertTestEntry(t *testing.T, db bun.IDB, key, value string, expiresAt *time.Time) {
	t.Helper()

	entry := KeyValueStore{Key: key, Value: value, ExpiresAt: expiresAt}
	_, err := db.NewInsert().Model(&entry).Exec(context.Background())
	require.NoError(t, err)
}

func dropKeyValueStoreTable(t *testing.T, db bun.IDB) {
	t.Helper()

	_, err := db.NewDropTable().Model(&KeyValueStore{}).Exec(context.Background())
	require.NoError(t, err)
}

func TestNewDatabaseSecondaryStorageOptions(t *testing.T) {
	t.Parallel()

	logger := new(internaltests.RecordingLogger)

	tests := []struct {
		name            string
		config          DatabaseStorageConfig
		opts            []DatabaseStorageOption
		wantInterval    time.Duration
		wantSameLogger  bool
		wantNilLoggerOK bool
	}{
		{
			name:            "zero cleanup interval uses one minute",
			config:          DatabaseStorageConfig{},
			wantInterval:    time.Minute,
			wantNilLoggerOK: true,
		},
		{
			name:            "custom cleanup interval is preserved",
			config:          DatabaseStorageConfig{CleanupInterval: 30 * time.Second},
			wantInterval:    30 * time.Second,
			wantNilLoggerOK: true,
		},
		{
			name:           "logger option is applied",
			config:         DatabaseStorageConfig{},
			opts:           []DatabaseStorageOption{WithDatabaseStorageLogger(logger)},
			wantInterval:   time.Minute,
			wantSameLogger: true,
		},
		{
			name:            "nil logger option keeps the default",
			config:          DatabaseStorageConfig{},
			opts:            []DatabaseStorageOption{WithDatabaseStorageLogger(nil)},
			wantInterval:    time.Minute,
			wantNilLoggerOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storage := NewDatabaseSecondaryStorage(newTestDB(t), tt.config, tt.opts...)

			assert.Equal(t, tt.wantInterval, storage.cleanupInterval)
			assert.False(t, storage.cleanupStarted)
			if tt.wantSameLogger {
				assert.Same(t, logger, storage.logger)
			}
			if tt.wantNilLoggerOK {
				assert.Nil(t, storage.logger)
			}
		})
	}
}

func TestDatabaseStorageIncr(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	ctx := context.Background()

	first, err := storage.Incr(ctx, "counter", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, first)

	second, err := storage.Incr(ctx, "counter", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, second)

	value, err := storage.Get(ctx, "counter")
	require.NoError(t, err)
	assertStringDB(t, value, "2")
}

func TestDatabaseStorageIncrWithTTL(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	ctx := context.Background()
	ttl := 10 * time.Second

	count, err := storage.Incr(ctx, "counter", &ttl)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	remaining, err := storage.TTL(ctx, "counter")
	require.NoError(t, err)
	require.NotNil(t, remaining)
	assert.Positive(t, *remaining)
	assert.LessOrEqual(t, *remaining, ttl)

	// A nil TTL on a later increment clears the stored expiry.
	count, err = storage.Incr(ctx, "counter", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	remaining, err = storage.TTL(ctx, "counter")
	require.NoError(t, err)
	assert.Nil(t, remaining)
}

func TestDatabaseStorageIncrRestartsExpiredCounter(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	ctx := context.Background()
	expiredAt := time.Now().Add(-time.Hour)

	insertTestEntry(t, db, "counter", "5", &expiredAt)

	count, err := storage.Incr(ctx, "counter", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	value, err := storage.Get(ctx, "counter")
	require.NoError(t, err)
	assertStringDB(t, value, "1")
}

func TestDatabaseStorageIncrRejectsNonIntegerValues(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	ctx := context.Background()

	require.NoError(t, storage.Set(ctx, "text", "not-a-number", nil))

	count, err := storage.Incr(ctx, "text", nil)

	assert.Zero(t, count)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a valid integer")

	value, err := storage.Get(ctx, "text")
	require.NoError(t, err)
	assertStringDB(t, value, "not-a-number")
}

func TestDatabaseStorageIncrDatabaseError(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	dropKeyValueStoreTable(t, db)

	count, err := storage.Incr(context.Background(), "counter", nil)

	assert.Zero(t, count)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestDatabaseStorageTTL(t *testing.T) {
	t.Parallel()

	ttl := 10 * time.Second
	expiredAt := time.Now().Add(-time.Hour)

	tests := []struct {
		name      string
		key       string
		prepare   func(t *testing.T, db bun.IDB, storage *DatabaseSecondaryStorage)
		wantNil   bool
		wantAbove time.Duration
	}{
		{
			name:    "missing key",
			key:     "missing",
			prepare: func(t *testing.T, db bun.IDB, storage *DatabaseSecondaryStorage) {},
			wantNil: true,
		},
		{
			name: "key without expiry",
			key:  "persistent",
			prepare: func(t *testing.T, db bun.IDB, storage *DatabaseSecondaryStorage) {
				require.NoError(t, storage.Set(context.Background(), "persistent", "value", nil))
			},
			wantNil: true,
		},
		{
			name: "key with expiry",
			key:  "expiring",
			prepare: func(t *testing.T, db bun.IDB, storage *DatabaseSecondaryStorage) {
				require.NoError(t, storage.Set(context.Background(), "expiring", "value", &ttl))
			},
			wantAbove: 0,
		},
		{
			name: "expired key",
			key:  "expired",
			prepare: func(t *testing.T, db bun.IDB, storage *DatabaseSecondaryStorage) {
				insertTestEntry(t, db, "expired", "value", &expiredAt)
			},
			wantNil: true,
		},
		{
			name: "query error reports no ttl",
			key:  "errored",
			prepare: func(t *testing.T, db bun.IDB, storage *DatabaseSecondaryStorage) {
				dropKeyValueStoreTable(t, db)
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDB(t)
			storage := newTestDatabaseStorage(t, db)
			tt.prepare(t, db, storage)

			remaining, err := storage.TTL(context.Background(), tt.key)

			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, remaining)
				return
			}
			require.NotNil(t, remaining)
			assert.Greater(t, *remaining, tt.wantAbove)
			assert.LessOrEqual(t, *remaining, ttl)
		})
	}
}

func TestDatabaseStorageScan(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	ctx := context.Background()
	expiredAt := time.Now().Add(-time.Hour)

	insertTestEntry(t, db, "scan:a", "value", nil)
	insertTestEntry(t, db, "scan:b", "value", nil)
	insertTestEntry(t, db, "other:c", "value", nil)
	insertTestEntry(t, db, "scan:expired", "value", &expiredAt)

	keys, err := storage.Scan(ctx, "scan:")
	require.NoError(t, err)

	sort.Strings(keys)
	assert.Equal(t, []string{"scan:a", "scan:b"}, keys)

	keys, err = storage.Scan(ctx, "missing:")
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestDatabaseStorageScanEverythingIncludesUnexpiredKeys(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	ctx := context.Background()

	insertTestEntry(t, db, "one", "value", nil)
	insertTestEntry(t, db, "two", "value", nil)

	keys, err := storage.Scan(ctx, "")
	require.NoError(t, err)

	sort.Strings(keys)
	assert.Equal(t, []string{"one", "two"}, keys)
}

func TestDatabaseStorageScanDatabaseError(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	dropKeyValueStoreTable(t, db)

	keys, err := storage.Scan(context.Background(), "prefix:")

	assert.Nil(t, keys)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database scan error")
}

func TestDatabaseStorageDeleteDatabaseError(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	dropKeyValueStoreTable(t, db)

	err := storage.Delete(context.Background(), "key")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestDatabaseStorageSetDatabaseError(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	dropKeyValueStoreTable(t, db)

	err := storage.Set(context.Background(), "key", "value", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

// A failed lookup is reported as a cache miss rather than an error, which keeps
// the storage contract identical to the other providers.
func TestDatabaseStorageGetQueryErrorIsMiss(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)
	dropKeyValueStoreTable(t, db)

	value, err := storage.Get(context.Background(), "key")

	require.NoError(t, err)
	assert.Nil(t, value)
}

func TestDatabaseStorageContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)

	tests := []struct {
		name string
		op   func() error
	}{
		{name: "incr", op: func() error { _, err := storage.Incr(ctx, "key", nil); return err }},
		{name: "ttl", op: func() error { _, err := storage.TTL(ctx, "key"); return err }},
		{name: "scan", op: func() error { _, err := storage.Scan(ctx, "prefix:"); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op()

			require.Error(t, err)
			assert.True(t, errors.Is(err, context.Canceled), "error should wrap context.Canceled: %v", err)
		})
	}
}

func TestDatabaseStorageStartCleanupIsIdempotent(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	storage := newTestDatabaseStorage(t, db)

	storage.StartCleanup()
	storage.StartCleanup()

	assert.True(t, storage.cleanupStarted)

	select {
	case <-storage.done:
		t.Fatal("cleanup goroutine stopped before Close was called")
	default:
	}

	assert.NoError(t, storage.Close())
	assert.NoError(t, storage.Close(), "Close must be idempotent")

	select {
	case <-storage.done:
	case <-time.After(time.Second):
		t.Fatal("cleanup goroutine did not stop after Close")
	}
}

func TestDatabaseStorageCloseWithoutCleanup(t *testing.T) {
	t.Parallel()

	storage := newTestDatabaseStorage(t, newTestDB(t))

	assert.NoError(t, storage.Close())
	assert.NoError(t, storage.Close())
}
