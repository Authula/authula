package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Authula/authula/plugins/rate-limit/types"
)

type dummySecondaryStorage struct {
	values  map[string]any
	ttls    map[string]time.Duration
	getErr  error
	setErr  error
	incrErr error
	ttlErr  error
}

func newDummySecondaryStorage() *dummySecondaryStorage {
	return &dummySecondaryStorage{values: map[string]any{}, ttls: map[string]time.Duration{}}
}

func (s *dummySecondaryStorage) Get(_ context.Context, key string) (any, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.values[key], nil
}
func (s *dummySecondaryStorage) Set(_ context.Context, key string, value any, ttl *time.Duration) error {
	if s.setErr != nil {
		return s.setErr
	}
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
	if s.incrErr != nil {
		return 0, s.incrErr
	}
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
	if s.ttlErr != nil {
		return nil, s.ttlErr
	}
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

func TestSecondaryStorageProvider_GetNameAndClose(t *testing.T) {
	t.Parallel()

	provider := NewSecondaryStorageProvider("redis", newDummySecondaryStorage())

	if provider.GetName() != "redis" {
		t.Fatalf("expected provider name redis, got %q", provider.GetName())
	}
	if err := provider.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestSecondaryStorageProvider_GetValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seed    map[string]any
		getErr  error
		wantErr bool
	}{
		{
			name: "returns_rate_limit_value",
			seed: map[string]any{"key-1": types.RateLimitValue{Count: 3, ExpiresAt: time.Unix(1000, 0)}},
		},
		{
			name:    "missing_value_reports_unexpected_type",
			wantErr: true,
		},
		{
			name:    "storage_error",
			getErr:  errors.New("boom"),
			wantErr: true,
		},
		{
			name:    "unexpected_type",
			seed:    map[string]any{"key-1": "wrong"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			storage := newDummySecondaryStorage()
			storage.getErr = tc.getErr
			storage.values = tc.seed
			if storage.values == nil {
				storage.values = map[string]any{}
			}
			provider := NewSecondaryStorageProvider("custom", storage)

			value, err := provider.GetValue(context.Background(), "key-1")

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			rateLimitValue, ok := value.(types.RateLimitValue)
			if !ok {
				t.Fatalf("expected RateLimitValue, got %T", value)
			}
			if rateLimitValue.Count != 3 {
				t.Fatalf("expected count 3, got %d", rateLimitValue.Count)
			}
		})
	}
}

func TestSecondaryStorageProvider_CheckAndIncrement(t *testing.T) {
	t.Parallel()

	window := time.Minute

	tests := []struct {
		name        string
		seed        map[string]any
		ttls        map[string]time.Duration
		getErr      error
		incrErr     error
		ttlErr      error
		cancel      bool
		wantErr     bool
		wantAllowed bool
		wantCount   int
		wantReset   bool
	}{
		{
			name:        "new_entry_uses_window",
			wantAllowed: true,
			wantCount:   1,
			wantReset:   true,
		},
		{
			name:        "existing_entry_uses_ttl",
			seed:        map[string]any{"key-1": 4},
			ttls:        map[string]time.Duration{"key-1": 30 * time.Second},
			wantAllowed: true,
			wantCount:   5,
			wantReset:   true,
		},
		{
			name:        "existing_entry_without_ttl_falls_back_to_window",
			seed:        map[string]any{"key-1": 9},
			wantAllowed: false,
			wantCount:   10,
			wantReset:   true,
		},
		{
			name:        "ttl_error_falls_back_to_window",
			seed:        map[string]any{"key-1": 1},
			ttlErr:      errors.New("ttl boom"),
			wantAllowed: true,
			wantCount:   2,
			wantReset:   true,
		},
		{
			name:    "storage_get_error",
			getErr:  errors.New("boom"),
			wantErr: true,
		},
		{
			name:    "increment_error",
			incrErr: errors.New("boom"),
			wantErr: true,
		},
		{
			name:    "canceled_context",
			cancel:  true,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			storage := newDummySecondaryStorage()
			if tc.seed != nil {
				storage.values = tc.seed
			}
			if tc.ttls != nil {
				storage.ttls = tc.ttls
			}
			storage.getErr = tc.getErr
			storage.incrErr = tc.incrErr
			storage.ttlErr = tc.ttlErr
			provider := NewSecondaryStorageProvider("custom", storage)

			ctx := context.Background()
			if tc.cancel {
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
			}

			allowed, count, resetAt, err := provider.CheckAndIncrement(ctx, "key-1", window, 5)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tc.wantAllowed || count != tc.wantCount {
				t.Fatalf("expected allowed=%v count=%d, got allowed=%v count=%d", tc.wantAllowed, tc.wantCount, allowed, count)
			}
			if tc.wantReset && resetAt.Before(time.Now()) {
				t.Fatalf("expected reset time in the future, got %v", resetAt)
			}
		})
	}
}

func TestSecondaryStorageProvider_SetRuleErrors(t *testing.T) {
	t.Parallel()

	storage := newDummySecondaryStorage()
	storage.setErr = errors.New("boom")
	provider := NewSecondaryStorageProvider("custom", storage)

	if err := provider.SetRule(context.Background(), "key-1", time.Minute, 5); err == nil {
		t.Fatal("expected set error, got nil")
	}
}

func TestSecondaryStorageProvider_GetRuleVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		raw        any
		getErr     error
		wantFound  bool
		wantWindow time.Duration
		wantMax    int
		wantErr    bool
	}{
		{
			name:       "string_payload",
			raw:        `{"WindowSeconds":90,"MaxRequests":13}`,
			wantFound:  true,
			wantWindow: 90 * time.Second,
			wantMax:    13,
		},
		{
			name:       "bytes_payload",
			raw:        []byte(`{"WindowSeconds":30,"MaxRequests":7}`),
			wantFound:  true,
			wantWindow: 30 * time.Second,
			wantMax:    7,
		},
		{
			name:    "missing_rule",
			wantErr: false,
		},
		{
			name:    "storage_error",
			getErr:  errors.New("boom"),
			wantErr: true,
		},
		{
			name:    "unexpected_type",
			raw:     42,
			wantErr: true,
		},
		{
			name:    "invalid_json",
			raw:     "{not-json",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			storage := newDummySecondaryStorage()
			storage.getErr = tc.getErr
			if tc.raw != nil {
				storage.values[ruleKeyPrefix+"key-1"] = tc.raw
			}
			provider := NewSecondaryStorageProvider("custom", storage)

			window, maxRequests, found, err := provider.GetRule(context.Background(), "key-1")

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if found != tc.wantFound {
				t.Fatalf("expected found=%v, got %v", tc.wantFound, found)
			}
			if !tc.wantFound {
				return
			}
			if window != tc.wantWindow || maxRequests != tc.wantMax {
				t.Fatalf("unexpected rule: window=%v max=%d", window, maxRequests)
			}
		})
	}
}
