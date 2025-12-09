package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// storageEntry represents a single entry in the memory storage with expiration support.
type storageEntry struct {
	value     []byte
	expiresAt *time.Time
}

// MemoryStorage is an in-memory implementation of SecondaryStorage.
type MemoryStorage struct {
	mu    sync.RWMutex
	store map[string]*storageEntry
	// cleanupTickDuration controls how often expired entries are cleaned up.
	cleanupTickDuration time.Duration
	// stopCleanup is used to signal the cleanup goroutine to stop.
	stopCleanup chan struct{}
	// done signals that the cleanup goroutine has stopped.
	done chan struct{}
}

func NewMemoryStorage() *MemoryStorage {
	storage := &MemoryStorage{
		store:               make(map[string]*storageEntry),
		cleanupTickDuration: 1 * time.Minute,
		stopCleanup:         make(chan struct{}),
		done:                make(chan struct{}),
	}

	go storage.cleanupExpiredEntries()

	return storage
}

// Get retrieves a value from memory by key.
// Returns an error if the key does not exist or has expired.
func (storage *MemoryStorage) Get(ctx context.Context, key string) ([]byte, error) {
	// Check context cancellation early.
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	storage.mu.RLock()
	defer storage.mu.RUnlock()

	entry, exists := storage.store[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	if entry.expiresAt != nil && time.Now().After(*entry.expiresAt) {
		return nil, fmt.Errorf("key expired: %s", key)
	}

	// Return a copy of the value to prevent external mutations.
	value := make([]byte, len(entry.value))
	copy(value, entry.value)

	return value, nil
}

// Set stores a value in memory with an optional TTL.
// If ttl is nil, the entry will not expire.
func (storage *MemoryStorage) Set(ctx context.Context, key string, value any, ttl *time.Duration) error {
	// Check context cancellation early.
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	// Type assert to []byte.
	valueBytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("value must be of type []byte, got %T", value)
	}

	storage.mu.Lock()
	defer storage.mu.Unlock()

	// Create a copy of the value to prevent external mutations.
	valueCopy := make([]byte, len(valueBytes))
	copy(valueCopy, valueBytes)

	entry := &storageEntry{
		value: valueCopy,
	}

	// Set expiration time if TTL is provided.
	if ttl != nil {
		expiresAt := time.Now().Add(*ttl)
		entry.expiresAt = &expiresAt
	}

	storage.store[key] = entry

	return nil
}

// Delete removes a key from storage.
// Returns an error if the key does not exist.
func (storage *MemoryStorage) Delete(ctx context.Context, key string) error {
	// Check context cancellation early.
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	storage.mu.Lock()
	defer storage.mu.Unlock()

	if _, exists := storage.store[key]; !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	delete(storage.store, key)

	return nil
}

// cleanupExpiredEntries runs periodically to remove expired entries from storage.
// This prevents memory leaks from entries with TTL that are never accessed.
func (storage *MemoryStorage) cleanupExpiredEntries() {
	ticker := time.NewTicker(storage.cleanupTickDuration)
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
func (storage *MemoryStorage) removeExpiredEntries() {
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
func (storage *MemoryStorage) Close() error {
	close(storage.stopCleanup)
	<-storage.done
	return nil
}
