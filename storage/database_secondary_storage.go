package storage

import (
	"context"
	"fmt"
	"strconv"
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
}

func NewDatabaseSecondaryStorage(db *gorm.DB, config *models.SecondaryStorageDatabaseOptions) *DatabaseSecondaryStorage {
	cleanupInterval := 1 * time.Minute
	if config != nil {
		if config.CleanupInterval != 0 {
			cleanupInterval = config.CleanupInterval
		}
	}

	storage := &DatabaseSecondaryStorage{
		db:              db,
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
		done:            make(chan struct{}),
	}

	go storage.cleanupExpiredEntries()

	return storage
}

// Get retrieves a value from the database by key.
// Returns an error if the key does not exist or has expired.
func (storage *DatabaseSecondaryStorage) Get(ctx context.Context, key string) (any, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	var entry models.KeyValueStore
	result := storage.db.WithContext(ctx).Where("key = ?", key).First(&entry)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("database error: %w", result.Error)
	}

	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		return nil, fmt.Errorf("key expired: %s", key)
	}

	return entry.Value, nil
}

// Set stores a value in the database with an optional TTL.
// The value must be a string. If ttl is nil, the entry will not expire.
func (storage *DatabaseSecondaryStorage) Set(ctx context.Context, key string, value any, ttl *time.Duration) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
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
// Returns an error if the key does not exist.
func (storage *DatabaseSecondaryStorage) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	result := storage.db.WithContext(ctx).Where("key = ?", key).Delete(&models.KeyValueStore{})

	if result.Error != nil {
		return fmt.Errorf("database error: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("key not found: %s", key)
	}

	return nil
}

// Incr increments the integer value stored at key by 1.
// If the key does not exist, it is initialized to 0 and then incremented to 1.
// If ttl is provided, it will be set or updated on the key.
func (storage *DatabaseSecondaryStorage) Incr(ctx context.Context, key string, ttl *time.Duration) (int, error) {
	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	var count int

	var entry models.KeyValueStore
	result := storage.db.WithContext(ctx).Where("key = ?", key).First(&entry)

	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return 0, fmt.Errorf("database error: %w", result.Error)
	}

	if result.Error == nil {
		if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
			count = 0
		} else {
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
		// Log error but don't fail - cleanup is best-effort
		// In production, this should be logged properly
		fmt.Printf("error cleaning up expired entries: %v\n", result.Error)
	}
}

// Close gracefully shuts down the storage by stopping the cleanup goroutine.
// This should be called when the application is shutting down.
func (storage *DatabaseSecondaryStorage) Close() error {
	close(storage.stopCleanup)
	<-storage.done
	return nil
}
