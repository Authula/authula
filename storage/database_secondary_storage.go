package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// DatabaseSecondaryStorage implements the SecondaryStorage interface using GORM.
type DatabaseSecondaryStorage struct {
	db *gorm.DB
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

func NewDatabaseSecondaryStorage(db *gorm.DB, config models.SecondaryStorageDatabaseOptions) *DatabaseSecondaryStorage {
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

	return storage
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

	var entry models.KeyValueStore
	result := storage.db.WithContext(ctx).Where("key = ?", key).First(&entry)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, fmt.Errorf("database error: %w", result.Error)
	}

	// Check if entry has expired
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		// Delete the expired entry
		if err := storage.db.WithContext(ctx).Delete(&models.KeyValueStore{}, "key = ?", key).Error; err != nil {
			slog.Error("error deleting expired entry", slog.String("key", key), slog.Any("error", err))
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

	entry := models.KeyValueStore{
		Key:   key,
		Value: valueStr,
	}

	if ttl != nil {
		expiresAt := time.Now().Add(*ttl)
		entry.ExpiresAt = &expiresAt
	}

	result := storage.db.WithContext(ctx).Save(&entry)

	if result.Error != nil {
		return fmt.Errorf("database error: %w", result.Error)
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

	result := storage.db.WithContext(ctx).Where("key = ?", key).Delete(&models.KeyValueStore{})
	if result.Error != nil {
		return fmt.Errorf("database error: %w", result.Error)
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

	var entry models.KeyValueStore
	result := storage.db.WithContext(ctx).Where("key = ?", key).First(&entry)

	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return 0, fmt.Errorf("database error: %w", result.Error)
	}

	if result.Error == nil {
		// Entry exists, check if it's expired
		if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
			// Delete the expired entry
			if err := storage.db.WithContext(ctx).Delete(&models.KeyValueStore{}, "key = ?", key).Error; err != nil {
				slog.Error("error deleting expired entry during incr", slog.String("key", key), slog.Any("error", err))
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

	newEntry := models.KeyValueStore{
		Key:   key,
		Value: strconv.Itoa(count),
	}

	if ttl != nil {
		expiresAt := time.Now().Add(*ttl)
		newEntry.ExpiresAt = &expiresAt
	}

	if result := storage.db.WithContext(ctx).Save(&newEntry); result.Error != nil {
		return 0, fmt.Errorf("database error: %w", result.Error)
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

	result := storage.db.
		Where("expires_at IS NOT NULL AND expires_at < ?", now).
		Delete(&models.KeyValueStore{})

	if result.Error != nil {
		slog.Error("error cleaning up expired entries from key_value_store", slog.Any("error", result.Error))
	}
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
