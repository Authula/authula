package secondarystorage

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/models"
)

// DatabaseStorageOption customizes a DatabaseSecondaryStorage at construction.
type DatabaseStorageOption func(*DatabaseSecondaryStorage)

// WithDatabaseStorageLogger injects the logger used for non-fatal cleanup
// errors. A nil logger is ignored.
func WithDatabaseStorageLogger(logger models.Logger) DatabaseStorageOption {
	return func(storage *DatabaseSecondaryStorage) {
		if logger != nil {
			storage.logger = logger
		}
	}
}

// DatabaseSecondaryStorage implements the SecondaryStorage interface using Bun ORM.
type DatabaseSecondaryStorage struct {
	db bun.IDB
	// logger receives non-fatal errors (cleanup, expired-entry deletions).
	logger models.Logger
	// cleanupInterval controls how often expired entries are cleaned up.
	cleanupInterval time.Duration
	// stopCleanup is used to signal the cleanup goroutine to stop.
	stopCleanup chan struct{}
	// done signals that the cleanup goroutine has stopped.
	done chan struct{}
	// cleanupStarted tracks whether the cleanup goroutine has been started.
	cleanupStarted bool
	// closeOnce ensures Close() is idempotent.
	closeOnce sync.Once
}

func NewDatabaseSecondaryStorage(db bun.IDB, config DatabaseStorageConfig, opts ...DatabaseStorageOption) *DatabaseSecondaryStorage {
	cleanupInterval := time.Minute
	if config.CleanupInterval != 0 {
		cleanupInterval = config.CleanupInterval
	}

	storage := &DatabaseSecondaryStorage{
		db:              db,
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
		done:            make(chan struct{}),
		cleanupStarted:  false,
	}

	for _, opt := range opts {
		opt(storage)
	}

	return storage
}

// logError reports a non-fatal error through the injected logger. It is a
// no-op when no logger was supplied.
func (storage *DatabaseSecondaryStorage) logError(msg string, args ...any) {
	if storage.logger != nil {
		storage.logger.Error(msg, args...)
	}
}

// StartCleanup starts the background cleanup goroutine that removes expired entries.
// This should be called after database migrations have completed.
// It is safe to call this multiple times - subsequent calls will be no-ops.
func (storage *DatabaseSecondaryStorage) StartCleanup() {
	if storage.cleanupStarted {
		return
	}
	storage.cleanupStarted = true
	go storage.cleanupExpiredEntries()
}

// Get retrieves a value from the database by key.
// Returns nil if the key does not exist or has expired.
// Expired entries are deleted immediately to prevent database bloat.
func (storage *DatabaseSecondaryStorage) Get(ctx context.Context, key string) (any, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context canceled: %w", ctx.Err())
	default:
	}

	var entry KeyValueStore
	err := storage.db.NewSelect().Model(&entry).Where("key = ?", key).Scan(ctx)

	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("context error: %w", err)
		}
		// If no rows found, Scan returns nil for simple case
		if entry.Key == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if entry.Key == "" {
		return nil, nil
	}

	// Check if entry has expired
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		// Delete the expired entry
		if _, err := storage.db.NewDelete().Model(&KeyValueStore{}).Where("key = ?", key).Exec(ctx); err != nil {
			storage.logError("error deleting expired entry", "key", key, "error", err)
		}
		return nil, nil
	}

	return entry.Value, nil
}

// Set stores a value in the database with an optional TTL.
// The value must be a string. If ttl is nil, the entry will not expire.
func (storage *DatabaseSecondaryStorage) Set(ctx context.Context, key string, value any, ttl *time.Duration) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled: %w", ctx.Err())
	default:
	}

	valueStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be of type string, got %T", value)
	}

	entry := KeyValueStore{
		Key:   key,
		Value: valueStr,
	}

	if ttl != nil {
		expiresAt := time.Now().Add(*ttl)
		entry.ExpiresAt = &expiresAt
	}

	_, err := storage.db.NewInsert().Model(&entry).On("CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, expires_at = EXCLUDED.expires_at").Exec(ctx)

	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	return nil
}

// Delete removes a key from the database.
// It is idempotent: deleting a non-existent key does not return an error.
func (storage *DatabaseSecondaryStorage) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled: %w", ctx.Err())
	default:
	}

	_, err := storage.db.NewDelete().Model(&KeyValueStore{}).Where("key = ?", key).Exec(ctx)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	// Idempotent: do not return error if key does not exist
	return nil
}

// Incr increments the integer value stored at key by 1.
// If the key does not exist, it is initialized to 0 and then incremented to 1.
// If ttl is provided, it will be set or updated on the key.
// Expired entries are deleted immediately before incrementing.
func (storage *DatabaseSecondaryStorage) Incr(ctx context.Context, key string, ttl *time.Duration) (int, error) {
	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("context canceled: %w", ctx.Err())
	default:
	}

	var count int

	var entry KeyValueStore
	err := storage.db.NewSelect().Model(&entry).Where("key = ?", key).Scan(ctx)

	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		if entry.Key != "" {
			// Entry exists, check if it's expired
			if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
				// Delete the expired entry
				if _, err := storage.db.NewDelete().Model(&KeyValueStore{}).Where("key = ?", key).Exec(ctx); err != nil {
					storage.logError("error deleting expired entry during incr", "key", key, "error", err)
				}
			} else {
				// Entry exists and is not expired
				if num, err := strconv.Atoi(entry.Value); err == nil {
					count = num
				} else {
					return 0, fmt.Errorf("value at key %s is not a valid integer: %w", key, err)
				}
			}
		}
	} else if err != nil {
		return 0, fmt.Errorf("database error: %w", err)
	} else if entry.Key != "" {
		// Entry exists, check if it's expired
		if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
			// Delete the expired entry
			if _, err := storage.db.NewDelete().Model(&KeyValueStore{}).Where("key = ?", key).Exec(ctx); err != nil {
				storage.logError("error deleting expired entry during incr", "key", key, "error", err)
			}
		} else {
			// Entry exists and is not expired
			if num, err := strconv.Atoi(entry.Value); err == nil {
				count = num
			} else {
				return 0, fmt.Errorf("value at key %s is not a valid integer: %w", key, err)
			}
		}
	}

	count++

	newEntry := KeyValueStore{
		Key:   key,
		Value: strconv.Itoa(count),
	}

	if ttl != nil {
		expiresAt := time.Now().Add(*ttl)
		newEntry.ExpiresAt = &expiresAt
	}

	_, err = storage.db.NewInsert().Model(&newEntry).On("CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, expires_at = EXCLUDED.expires_at").Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("database error: %w", err)
	}

	return count, nil
}

// cleanupExpiredEntries runs periodically to remove expired entries from the database.
// This prevents database bloat from entries with TTL that are never accessed.
func (storage *DatabaseSecondaryStorage) cleanupExpiredEntries() {
	ticker := time.NewTicker(storage.cleanupInterval)
	defer ticker.Stop()
	defer close(storage.done)

	for {
		select {
		case <-storage.stopCleanup:
			return
		case <-ticker.C:
			storage.removeExpiredEntries()
		}
	}
}

// removeExpiredEntries removes all expired entries from the database.
func (storage *DatabaseSecondaryStorage) removeExpiredEntries() {
	now := time.Now()
	ctx := context.Background()

	_, err := storage.db.NewDelete().Model(&KeyValueStore{}).
		Where("expires_at IS NOT NULL AND expires_at < ?", now).
		Exec(ctx)

	if err != nil {
		storage.logError("error cleaning up expired entries from key_value_store", "error", err)
	}
}

// TTL returns the remaining time to live for a key
// Returns nil if the key does not exist or has no expiration
func (storage *DatabaseSecondaryStorage) TTL(ctx context.Context, key string) (*time.Duration, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context canceled: %w", ctx.Err())
	default:
	}

	var entry KeyValueStore
	err := storage.db.NewSelect().Model(&entry).Where("key = ?", key).Scan(ctx)

	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("context error: %w", err)
		}
		if entry.Key == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if entry.Key == "" {
		return nil, nil
	}

	if entry.ExpiresAt == nil {
		return nil, nil
	}

	now := time.Now()
	if now.After(*entry.ExpiresAt) {
		// Entry has expired
		return nil, nil
	}

	ttl := entry.ExpiresAt.Sub(now)
	return &ttl, nil
}

// Scan returns all non-expired keys matching the given prefix.
func (storage *DatabaseSecondaryStorage) Scan(ctx context.Context, prefix string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context canceled: %w", ctx.Err())
	default:
	}

	var entries []KeyValueStore
	err := storage.db.NewSelect().Model(&entries).
		Where("key LIKE ?", prefix+"%").
		Where("expires_at IS NULL OR expires_at > ?", time.Now()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("database scan error: %w", err)
	}

	keys := make([]string, len(entries))
	for i, entry := range entries {
		keys[i] = entry.Key
	}
	return keys, nil
}

// Close gracefully shuts down the storage by stopping the cleanup goroutine.
// This method is idempotent and safe to call multiple times.
func (storage *DatabaseSecondaryStorage) Close() error {
	storage.closeOnce.Do(func() {
		if storage.cleanupStarted {
			close(storage.stopCleanup)
			<-storage.done
		}
	})
	return nil
}
