package secondarystorage

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemorySecondaryStorageIncr(t *testing.T) {
	t.Parallel()

	storage := newTestMemorySecondaryStorage()
	t.Cleanup(func() {
		assert.NoError(t, storage.Close())
	})
	ctx := context.Background()

	first, err := storage.Incr(ctx, "counter", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, first)

	second, err := storage.Incr(ctx, "counter", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, second)

	value, err := storage.Get(ctx, "counter")
	require.NoError(t, err)
	assertString(t, value, "2")
}

func TestMemorySecondaryStorageIncrWithTTL(t *testing.T) {
	t.Parallel()

	storage := newTestMemorySecondaryStorage()
	t.Cleanup(func() {
		assert.NoError(t, storage.Close())
	})
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

	count, err = storage.Incr(ctx, "counter", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	remaining, err = storage.TTL(ctx, "counter")
	require.NoError(t, err)
	assert.Nil(t, remaining, "incrementing without a ttl replaces the entry without an expiry")
}

func TestMemorySecondaryStorageIncrRestartsExpiredCounter(t *testing.T) {
	t.Parallel()

	storage := newTestMemorySecondaryStorage()
	t.Cleanup(func() {
		assert.NoError(t, storage.Close())
	})
	ctx := context.Background()

	expiredAt := time.Now().Add(-time.Hour)
	storage.mu.Lock()
	storage.store["counter"] = &storageEntry{value: "5", expiresAt: &expiredAt}
	storage.mu.Unlock()

	count, err := storage.Incr(ctx, "counter", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	value, err := storage.Get(ctx, "counter")
	require.NoError(t, err)
	assertString(t, value, "1")
}

func TestMemorySecondaryStorageIncrRejectsNonIntegerValues(t *testing.T) {
	t.Parallel()

	storage := newTestMemorySecondaryStorage()
	t.Cleanup(func() {
		assert.NoError(t, storage.Close())
	})
	ctx := context.Background()

	require.NoError(t, storage.Set(ctx, "text", "not-a-number", nil))

	count, err := storage.Incr(ctx, "text", nil)

	assert.Zero(t, count)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a valid integer")

	value, err := storage.Get(ctx, "text")
	require.NoError(t, err)
	assertString(t, value, "not-a-number")
}

func TestMemorySecondaryStorageIncrContextCancelled(t *testing.T) {
	t.Parallel()

	storage := newTestMemorySecondaryStorage()
	t.Cleanup(func() {
		assert.NoError(t, storage.Close())
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	count, err := storage.Incr(ctx, "counter", nil)

	assert.Zero(t, count)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestMemorySecondaryStorageTTL(t *testing.T) {
	t.Parallel()

	storage := newTestMemorySecondaryStorage()
	t.Cleanup(func() {
		assert.NoError(t, storage.Close())
	})
	ctx := context.Background()
	ttl := 10 * time.Second
	expiredAt := time.Now().Add(-time.Hour)

	require.NoError(t, storage.Set(ctx, "persistent", "value", nil))
	require.NoError(t, storage.Set(ctx, "expiring", "value", &ttl))

	storage.mu.Lock()
	storage.store["expired"] = &storageEntry{value: "value", expiresAt: &expiredAt}
	storage.mu.Unlock()

	tests := []struct {
		name      string
		key       string
		wantNil   bool
		wantAbove time.Duration
	}{
		{name: "missing key", key: "missing", wantNil: true},
		{name: "key without expiry", key: "persistent", wantNil: true},
		{name: "key with expiry", key: "expiring", wantAbove: 0},
		{name: "expired key", key: "expired", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remaining, err := storage.TTL(ctx, tt.key)

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

func TestMemorySecondaryStorageTTLContextCancelled(t *testing.T) {
	t.Parallel()

	storage := newTestMemorySecondaryStorage()
	t.Cleanup(func() {
		assert.NoError(t, storage.Close())
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	remaining, err := storage.TTL(ctx, "key")

	assert.Nil(t, remaining)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestMemorySecondaryStorageScan(t *testing.T) {
	t.Parallel()

	storage := newTestMemorySecondaryStorage()
	t.Cleanup(func() {
		assert.NoError(t, storage.Close())
	})
	ctx := context.Background()
	expiredAt := time.Now().Add(-time.Hour)

	require.NoError(t, storage.Set(ctx, "scan:a", "value", nil))
	require.NoError(t, storage.Set(ctx, "scan:b", "value", nil))
	require.NoError(t, storage.Set(ctx, "other:c", "value", nil))

	storage.mu.Lock()
	storage.store["scan:expired"] = &storageEntry{value: "value", expiresAt: &expiredAt}
	storage.mu.Unlock()

	tests := []struct {
		name   string
		prefix string
		want   []string
	}{
		{name: "matching prefix skips expired entries", prefix: "scan:", want: []string{"scan:a", "scan:b"}},
		{name: "no matches", prefix: "missing:"},
		{name: "empty prefix includes every unexpired key", prefix: "", want: []string{"other:c", "scan:a", "scan:b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, err := storage.Scan(ctx, tt.prefix)
			require.NoError(t, err)

			sort.Strings(keys)
			assert.Equal(t, tt.want, keys)
		})
	}
}

func TestMemorySecondaryStorageScanContextCancelled(t *testing.T) {
	t.Parallel()

	storage := newTestMemorySecondaryStorage()
	t.Cleanup(func() {
		assert.NoError(t, storage.Close())
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	keys, err := storage.Scan(ctx, "prefix:")

	assert.Nil(t, keys)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}
