package storage

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// storageEntry represents a single entry in the memory storage with expiration support.
type storageEntry struct {
	value     string
	expiresAt *time.Time
}

// MemorySecondaryStorage is an in-memory implementation of SecondaryStorage.
type MemorySecondaryStorage struct {
	mu    sync.RWMutex
	store map[string]*storageEntry
	// cleanupInterval controls how often expired entries are cleaned up.
	cleanupInterval time.Duration
	// stopCleanup is used to signal the cleanup goroutine to stop.
	stopCleanup chan struct{}
	// done signals that the cleanup goroutine has stopped.
	done chan struct{}
}

func NewMemorySecondaryStorage(config models.SecondaryStorageMemoryOptions) *MemorySecondaryStorage {
	cleanupInterval := 1 * time.Minute
	if config.CleanupInterval != 0 {
		cleanupInterval = config.CleanupInterval
	}

	storage := &MemorySecondaryStorage{
		store:           make(map[string]*storageEntry),
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
		done:            make(chan struct{}),
	}

	go storage.cleanupExpiredEntries()

	return storage
}

// Get retrieves a value from memory by key.
// Returns nil if the key does not exist or has expired.
func (storage *MemorySecondaryStorage) Get(ctx context.Context, key string) (any, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	storage.mu.RLock()
	defer storage.mu.RUnlock()

	entry, exists := storage.store[key]
	if !exists {
		return nil, nil
	}

	if entry.expiresAt != nil && time.Now().After(*entry.expiresAt) {
		return nil, nil
	}

	return entry.value, nil
}

// Set stores a value in memory with an optional TTL.
// The value must be a string. If ttl is nil, the entry will not expire.
func (storage *MemorySecondaryStorage) Set(ctx context.Context, key string, value any, ttl *time.Duration) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	valueStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be of type string, got %T", value)
	}

	storage.mu.Lock()
	defer storage.mu.Unlock()

	entry := &storageEntry{
		value: valueStr,
	}

	if ttl != nil {
		expiresAt := time.Now().Add(*ttl)
		entry.expiresAt = &expiresAt
	}

	storage.store[key] = entry

	return nil
}

// Delete removes a key from storage.
// This operation is idempotent: no error is returned if the key does not exist.
func (storage *MemorySecondaryStorage) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	storage.mu.Lock()
	defer storage.mu.Unlock()

	delete(storage.store, key)

	return nil
}

// Incr increments the integer value stored at key by 1.
// If the key does not exist, it is initialized to 0 and then incremented to 1.
// If ttl is provided, it will be set or updated on the key.
func (storage *MemorySecondaryStorage) Incr(ctx context.Context, key string, ttl *time.Duration) (int, error) {
	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	storage.mu.Lock()
	defer storage.mu.Unlock()

	var count int

	if entry, exists := storage.store[key]; exists {
		if entry.expiresAt != nil && time.Now().After(*entry.expiresAt) {
			count = 0
		} else {
			if num, err := strconv.Atoi(entry.value); err == nil {
				count = num
			} else {
				return 0, fmt.Errorf("value at key %s is not a valid integer: %w", key, err)
			}
		}
	}

	count++

	entry := &storageEntry{
		value: strconv.Itoa(count),
	}

	if ttl != nil {
		expiresAt := time.Now().Add(*ttl)
		entry.expiresAt = &expiresAt
	}

	storage.store[key] = entry

	return count, nil
}

// cleanupExpiredEntries runs periodically to remove expired entries from storage.
// This prevents memory leaks from entries with TTL that are never accessed.
func (storage *MemorySecondaryStorage) cleanupExpiredEntries() {
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

// removeExpiredEntries removes all expired entries from storage.
func (storage *MemorySecondaryStorage) removeExpiredEntries() {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	now := time.Now()
	for key, entry := range storage.store {
		if entry.expiresAt != nil && now.After(*entry.expiresAt) {
			delete(storage.store, key)
		}
	}
}

// Close gracefully shuts down the storage by stopping the cleanup goroutine.
func (storage *MemorySecondaryStorage) Close() error {
	close(storage.stopCleanup)
	<-storage.done
	return nil
}
