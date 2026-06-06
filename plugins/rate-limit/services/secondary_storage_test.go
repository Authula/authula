package services

import (
	"context"
	"strings"
	"testing"
	"time"
)

type dummySecondaryStorage struct {
	values map[string]any
	ttls   map[string]time.Duration
}

func newDummySecondaryStorage() *dummySecondaryStorage {
	return &dummySecondaryStorage{values: map[string]any{}, ttls: map[string]time.Duration{}}
}

func (s *dummySecondaryStorage) Get(_ context.Context, key string) (any, error) {
	return s.values[key], nil
}
func (s *dummySecondaryStorage) Set(_ context.Context, key string, value any, ttl *time.Duration) error {
	s.values[key] = value
	if ttl != nil {
		s.ttls[key] = *ttl
	} else {
		delete(s.ttls, key)
	}
	return nil
}
func (s *dummySecondaryStorage) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	delete(s.ttls, key)
	return nil
}
func (s *dummySecondaryStorage) Incr(_ context.Context, key string, ttl *time.Duration) (int, error) {
	if s.values[key] == nil {
		s.values[key] = 1
		if ttl != nil {
			s.ttls[key] = *ttl
		}
		return 1, nil
	}
	count := s.values[key].(int) + 1
	s.values[key] = count
	return count, nil
}
func (s *dummySecondaryStorage) TTL(_ context.Context, key string) (*time.Duration, error) {
	if ttl, ok := s.ttls[key]; ok {
		return &ttl, nil
	}
	return nil, nil
}
func (s *dummySecondaryStorage) Scan(_ context.Context, prefix string) ([]string, error) {
	var keys []string
	for key := range s.values {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (s *dummySecondaryStorage) Close() error { return nil }

func TestSecondaryStorageProviderRuleLifecycle(t *testing.T) {
	t.Parallel()

	storage := newDummySecondaryStorage()
	provider := NewSecondaryStorageProvider("custom", storage)
	ctx := context.Background()
	window := 4 * time.Minute
	maxRequests := 21

	if err := provider.SetRule(ctx, "api-key-1", window, maxRequests); err != nil {
		t.Fatalf("SetRule returned error: %v", err)
	}

	gotWindow, gotMaxRequests, found, err := provider.GetRule(ctx, "api-key-1")
	if err != nil {
		t.Fatalf("GetRule returned error: %v", err)
	}
	if !found || gotWindow != window || gotMaxRequests != maxRequests {
		t.Fatalf("unexpected rule: found=%v window=%v max=%d", found, gotWindow, gotMaxRequests)
	}

	if err := provider.DeleteRule(ctx, "api-key-1"); err != nil {
		t.Fatalf("DeleteRule returned error: %v", err)
	}

	_, _, found, err = provider.GetRule(ctx, "api-key-1")
	if err != nil {
		t.Fatalf("GetRule after delete returned error: %v", err)
	}
	if found {
		t.Fatal("expected deleted rule to be absent")
	}
}
