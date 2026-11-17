package secondarystorage

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/env"
)

// newTestRedisSecondaryStorage starts an in-process Redis server and returns a
// storage instance connected to it. The Redis URL environment variable is
// cleared so constructor options are the only source of configuration.
func newTestRedisSecondaryStorage(t *testing.T) (*RedisSecondaryStorage, *miniredis.Miniredis) {
	t.Helper()

	t.Setenv(env.EnvRedisURL, "")

	server := miniredis.RunT(t)

	storage, err := NewRedisSecondaryStorage(RedisSecondaryStorageOptions{
		URL:        "redis://" + server.Addr(),
		MaxRetries: -1,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = storage.Close()
	})

	return storage, server
}

func TestNewRedisSecondaryStorageDefaults(t *testing.T) {
	t.Setenv(env.EnvRedisURL, "")
	server := miniredis.RunT(t)

	storage, err := NewRedisSecondaryStorage(RedisSecondaryStorageOptions{
		URL: "redis://" + server.Addr(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = storage.Close()
	})

	options := storage.client.Options()
	assert.Equal(t, 3, options.MaxRetries)
	assert.Equal(t, 10, options.PoolSize)
	assert.Equal(t, 30*time.Second, options.PoolTimeout)
	assert.Equal(t, 5*time.Second, options.ReadTimeout)
	assert.Equal(t, 5*time.Second, options.WriteTimeout)
}

func TestNewRedisSecondaryStorageCustomOptions(t *testing.T) {
	t.Setenv(env.EnvRedisURL, "")
	server := miniredis.RunT(t)

	tests := []struct {
		name        string
		opts        RedisSecondaryStorageOptions
		wantRetries int
		wantPool    int
		wantTimeout time.Duration
	}{
		{
			name:        "explicit values win",
			opts:        RedisSecondaryStorageOptions{MaxRetries: 1, PoolSize: 2, PoolTimeout: time.Second},
			wantRetries: 1,
			wantPool:    2,
			wantTimeout: time.Second,
		},
		{
			name:        "zero values fall back to defaults",
			opts:        RedisSecondaryStorageOptions{},
			wantRetries: 3,
			wantPool:    10,
			wantTimeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.opts
			opts.URL = "redis://" + server.Addr()

			storage, err := NewRedisSecondaryStorage(opts)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = storage.Close()
			})

			options := storage.client.Options()
			assert.Equal(t, tt.wantRetries, options.MaxRetries)
			assert.Equal(t, tt.wantPool, options.PoolSize)
			assert.Equal(t, tt.wantTimeout, options.PoolTimeout)
		})
	}
}

func TestNewRedisSecondaryStorageUsesEnvironmentURL(t *testing.T) {
	server := miniredis.RunT(t)
	t.Setenv(env.EnvRedisURL, "redis://"+server.Addr())

	storage, err := NewRedisSecondaryStorage(RedisSecondaryStorageOptions{
		URL: "://not-a-valid-url",
	})
	require.NoError(t, err, "the environment URL must take precedence over the option")
	t.Cleanup(func() {
		_ = storage.Close()
	})

	assert.NoError(t, storage.Set(context.Background(), "env-key", "env-value", nil))
}

func TestNewRedisSecondaryStorageInvalidURL(t *testing.T) {
	t.Setenv(env.EnvRedisURL, "")

	storage, err := NewRedisSecondaryStorage(RedisSecondaryStorageOptions{URL: "://missing-scheme"})

	assert.Nil(t, storage)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Redis URL")
}

func TestNewRedisSecondaryStorageConnectionFailure(t *testing.T) {
	t.Setenv(env.EnvRedisURL, "")
	server, err := miniredis.Run()
	require.NoError(t, err)
	addr := server.Addr()
	server.Close()

	storage, err := NewRedisSecondaryStorage(RedisSecondaryStorageOptions{
		URL:         "redis://" + addr,
		MaxRetries:  -1,
		PoolTimeout: 100 * time.Millisecond,
	})

	assert.Nil(t, storage)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to Redis at")
}

func TestNewRedisSecondaryStoragePanicsWithoutURL(t *testing.T) {
	t.Setenv(env.EnvRedisURL, "")

	assert.PanicsWithValue(t, "redis URL must be provided", func() {
		_, _ = NewRedisSecondaryStorage(RedisSecondaryStorageOptions{})
	})
}

func TestRedisSecondaryStorageGet(t *testing.T) {
	storage, _ := newTestRedisSecondaryStorage(t)
	ctx := context.Background()

	value, err := storage.Get(ctx, "missing")
	require.NoError(t, err)
	assert.Nil(t, value)

	require.NoError(t, storage.Set(ctx, "present", "hello", nil))

	value, err = storage.Get(ctx, "present")
	require.NoError(t, err)
	assert.Equal(t, "hello", value)
}

func TestRedisSecondaryStorageSet(t *testing.T) {
	storage, _ := newTestRedisSecondaryStorage(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{name: "string value", value: "hello"},
		{name: "empty string", value: ""},
		{name: "bytes are rejected", value: []byte("hello"), wantErr: true},
		{name: "int is rejected", value: 42, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.Set(ctx, "key", tt.value, nil)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "value must be of type string")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRedisSecondaryStorageDeleteIsIdempotent(t *testing.T) {
	storage, _ := newTestRedisSecondaryStorage(t)
	ctx := context.Background()

	require.NoError(t, storage.Set(ctx, "key", "value", nil))
	require.NoError(t, storage.Delete(ctx, "key"))
	require.NoError(t, storage.Delete(ctx, "key"), "deleting a missing key must not fail")

	value, err := storage.Get(ctx, "key")
	require.NoError(t, err)
	assert.Nil(t, value)
}

func TestRedisSecondaryStorageExpiry(t *testing.T) {
	storage, server := newTestRedisSecondaryStorage(t)
	ctx := context.Background()
	ttl := 10 * time.Second

	require.NoError(t, storage.Set(ctx, "expiring", "value", &ttl))

	remaining, err := storage.TTL(ctx, "expiring")
	require.NoError(t, err)
	require.NotNil(t, remaining)
	assert.Positive(t, *remaining)
	assert.LessOrEqual(t, *remaining, ttl)

	server.FastForward(ttl)

	value, err := storage.Get(ctx, "expiring")
	require.NoError(t, err)
	assert.Nil(t, value)

	remaining, err = storage.TTL(ctx, "expiring")
	require.NoError(t, err)
	assert.Nil(t, remaining)
}

func TestRedisSecondaryStorageTTL(t *testing.T) {
	storage, _ := newTestRedisSecondaryStorage(t)
	ctx := context.Background()
	ttl := 10 * time.Second

	tests := []struct {
		name      string
		key       string
		prepare   func(t *testing.T)
		wantNil   bool
		wantAbove time.Duration
	}{
		{
			name:    "missing key",
			key:     "missing",
			prepare: func(t *testing.T) {},
			wantNil: true,
		},
		{
			name: "key without expiry",
			key:  "persistent",
			prepare: func(t *testing.T) {
				require.NoError(t, storage.Set(ctx, "persistent", "value", nil))
			},
			wantNil: true,
		},
		{
			name: "key with expiry",
			key:  "ttl-key",
			prepare: func(t *testing.T) {
				require.NoError(t, storage.Set(ctx, "ttl-key", "value", &ttl))
			},
			wantAbove: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t)

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

func TestRedisSecondaryStorageIncr(t *testing.T) {
	storage, _ := newTestRedisSecondaryStorage(t)
	ctx := context.Background()

	count, err := storage.Incr(ctx, "counter", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = storage.Incr(ctx, "counter", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	value, err := storage.Get(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, "2", value)
}

func TestRedisSecondaryStorageIncrAppliesTTLOnlyOnCreation(t *testing.T) {
	storage, server := newTestRedisSecondaryStorage(t)
	ctx := context.Background()
	ttl := 10 * time.Second

	count, err := storage.Incr(ctx, "counter", &ttl)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	remaining, err := storage.TTL(ctx, "counter")
	require.NoError(t, err)
	require.NotNil(t, remaining)

	// A later increment must not extend the existing expiry.
	server.FastForward(4 * time.Second)
	count, err = storage.Incr(ctx, "counter", &ttl)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	remaining, err = storage.TTL(ctx, "counter")
	require.NoError(t, err)
	require.NotNil(t, remaining)
	assert.LessOrEqual(t, *remaining, 6*time.Second)
}

func TestRedisSecondaryStorageIncrRejectsNonIntegerValues(t *testing.T) {
	storage, _ := newTestRedisSecondaryStorage(t)
	ctx := context.Background()

	require.NoError(t, storage.Set(ctx, "text", "not-a-number", nil))

	count, err := storage.Incr(ctx, "text", nil)

	assert.Zero(t, count)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis incr error")
}

func TestRedisSecondaryStorageScan(t *testing.T) {
	storage, _ := newTestRedisSecondaryStorage(t)
	ctx := context.Background()

	for _, key := range []string{"session:a", "session:b", "user:c"} {
		require.NoError(t, storage.Set(ctx, key, "value", nil))
	}

	tests := []struct {
		name    string
		prefix  string
		want    []string
		wantAny bool
	}{
		{name: "matching prefix", prefix: "session:", want: []string{"session:a", "session:b"}},
		{name: "no matches", prefix: "missing:"},
		{name: "empty prefix returns everything", prefix: "", want: []string{"session:a", "session:b", "user:c"}},
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

func TestRedisSecondaryStorageScanReturnsAllPages(t *testing.T) {
	storage, _ := newTestRedisSecondaryStorage(t)
	ctx := context.Background()

	const keyCount = 250
	for i := range keyCount {
		require.NoError(t, storage.Set(ctx, fmt.Sprintf("page:%03d", i), "value", nil))
	}

	keys, err := storage.Scan(ctx, "page:")
	require.NoError(t, err)
	assert.Len(t, keys, keyCount)
}

func TestRedisSecondaryStorageOperationErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		op      func(storage *RedisSecondaryStorage) error
		wantMsg string
	}{
		{
			name:    "get",
			op:      func(storage *RedisSecondaryStorage) error { _, err := storage.Get(ctx, "key"); return err },
			wantMsg: "redis get error",
		},
		{
			name:    "set",
			op:      func(storage *RedisSecondaryStorage) error { return storage.Set(ctx, "key", "value", nil) },
			wantMsg: "redis set error",
		},
		{
			name:    "delete",
			op:      func(storage *RedisSecondaryStorage) error { return storage.Delete(ctx, "key") },
			wantMsg: "redis delete error",
		},
		{
			name:    "incr",
			op:      func(storage *RedisSecondaryStorage) error { _, err := storage.Incr(ctx, "key", nil); return err },
			wantMsg: "redis exists check error",
		},
		{
			name:    "ttl",
			op:      func(storage *RedisSecondaryStorage) error { _, err := storage.TTL(ctx, "key"); return err },
			wantMsg: "redis ttl error",
		},
		{
			name:    "scan",
			op:      func(storage *RedisSecondaryStorage) error { _, err := storage.Scan(ctx, "prefix:"); return err },
			wantMsg: "redis scan error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, _ := newTestRedisSecondaryStorage(t)
			require.NoError(t, storage.client.Close())

			err := tt.op(storage)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantMsg)
			assert.True(t, errors.Is(err, redis.ErrClosed), "error should wrap redis.ErrClosed: %v", err)
		})
	}
}

func TestRedisSecondaryStorageContextCancellation(t *testing.T) {
	storage, _ := newTestRedisSecondaryStorage(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		op   func() error
	}{
		{name: "get", op: func() error { _, err := storage.Get(ctx, "key"); return err }},
		{name: "set", op: func() error { return storage.Set(ctx, "key", "value", nil) }},
		{name: "delete", op: func() error { return storage.Delete(ctx, "key") }},
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

func TestRedisSecondaryStorageClose(t *testing.T) {
	tests := []struct {
		name    string
		storage *RedisSecondaryStorage
	}{
		{name: "nil client is safe", storage: &RedisSecondaryStorage{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, tt.storage.Close())
		})
	}

	t.Run("connected client closes", func(t *testing.T) {
		storage, _ := newTestRedisSecondaryStorage(t)

		assert.NoError(t, storage.Close())
	})
}
