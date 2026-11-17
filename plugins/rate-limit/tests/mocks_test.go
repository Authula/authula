package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/rate-limit/types"
)

type storageStub struct {
	models.SecondaryStorage
}

type registration struct {
	name    string
	service any
}

func storeKeys(p *FakeRateLimitProvider) []string {
	keys := make([]string, 0, len(p.Store))
	for key := range p.Store {
		keys = append(keys, key)
	}
	return keys
}

func TestNewFakeRateLimitProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		check func(t *testing.T, p *FakeRateLimitProvider)
	}{
		{
			name: "initializes an empty store",
			check: func(t *testing.T, p *FakeRateLimitProvider) {
				assert.NotNil(t, p.Store)
				assert.Empty(t, p.Store)
			},
		},
		{
			name: "reports fake as the provider name",
			check: func(t *testing.T, p *FakeRateLimitProvider) {
				assert.Equal(t, "fake", p.GetName())
			},
		},
		{
			name: "closes without error",
			check: func(t *testing.T, p *FakeRateLimitProvider) {
				assert.NoError(t, p.Close())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.check(t, NewFakeRateLimitProvider())
		})
	}
}

func TestFakeRateLimitProvider_Builders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		build  func(p *FakeRateLimitProvider) *FakeRateLimitProvider
		verify func(t *testing.T, p *FakeRateLimitProvider)
	}{
		{
			name: "WithCheckResult sets allowed, count, reset and error",
			build: func(p *FakeRateLimitProvider) *FakeRateLimitProvider {
				return p.WithCheckResult(true, 3, time.Unix(2000, 0), errors.New("boom"))
			},
			verify: func(t *testing.T, p *FakeRateLimitProvider) {
				assert.True(t, p.CheckAllowed)
				assert.Equal(t, 3, p.CheckCount)
				assert.Equal(t, time.Unix(2000, 0), p.CheckReset)
				assert.EqualError(t, p.CheckErr, "boom")
			},
		},
		{
			name: "WithCheckError sets only the check error",
			build: func(p *FakeRateLimitProvider) *FakeRateLimitProvider {
				return p.WithCheckError(errors.New("boom"))
			},
			verify: func(t *testing.T, p *FakeRateLimitProvider) {
				assert.False(t, p.CheckAllowed)
				assert.Equal(t, 0, p.CheckCount)
				assert.True(t, p.CheckReset.IsZero())
				assert.EqualError(t, p.CheckErr, "boom")
			},
		},
		{
			name: "WithExistingRule stores the window in seconds",
			build: func(p *FakeRateLimitProvider) *FakeRateLimitProvider {
				return p.WithExistingRule("key", 90*time.Second, 7)
			},
			verify: func(t *testing.T, p *FakeRateLimitProvider) {
				assert.Equal(t, types.RateLimitRuleRecord{Key: "key", WindowSeconds: 90, MaxRequests: 7}, p.Store["key"])
			},
		},
		{
			name: "WithGetError sets the get error",
			build: func(p *FakeRateLimitProvider) *FakeRateLimitProvider {
				return p.WithGetError(errors.New("boom"))
			},
			verify: func(t *testing.T, p *FakeRateLimitProvider) {
				assert.EqualError(t, p.GetErr, "boom")
				assert.NoError(t, p.SetErr)
			},
		},
		{
			name: "WithSetError sets the set error",
			build: func(p *FakeRateLimitProvider) *FakeRateLimitProvider {
				return p.WithSetError(errors.New("boom"))
			},
			verify: func(t *testing.T, p *FakeRateLimitProvider) {
				assert.EqualError(t, p.SetErr, "boom")
				assert.NoError(t, p.GetErr)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := NewFakeRateLimitProvider()

			got := tc.build(p)

			assert.Same(t, p, got)
			tc.verify(t, p)
		})
	}
}

func TestFakeRateLimitProvider_GetValue(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")

	tests := []struct {
		name        string
		arrange     func(p *FakeRateLimitProvider)
		key         string
		expected    any
		expectedErr error
	}{
		{
			name: "returns the get value error",
			arrange: func(p *FakeRateLimitProvider) {
				p.GetValueErr = errBoom
			},
			key:         "key",
			expectedErr: errBoom,
		},
		{
			name:     "returns nil for a missing key",
			key:      "missing",
			expected: nil,
		},
		{
			name: "returns count and expiry for a stored rule",
			arrange: func(p *FakeRateLimitProvider) {
				p.WithExistingRule("key", time.Minute, 5)
			},
			key: "key",
			expected: types.RateLimitValue{
				Count:     5,
				ExpiresAt: time.Unix(60, 0),
			},
		},
		{
			name: "get value error takes precedence over a stored rule",
			arrange: func(p *FakeRateLimitProvider) {
				p.WithExistingRule("key", time.Minute, 5)
				p.GetValueErr = errBoom
			},
			key:         "key",
			expectedErr: errBoom,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := NewFakeRateLimitProvider()
			if tc.arrange != nil {
				tc.arrange(p)
			}

			got, err := p.GetValue(context.Background(), tc.key)

			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestFakeRateLimitProvider_CheckAndIncrement(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")

	tests := []struct {
		name            string
		arrange         func(p *FakeRateLimitProvider)
		key             string
		window          time.Duration
		maxRequests     int
		expectedAllowed bool
		expectedCount   int
		expectedReset   time.Time
		expectedErr     error
	}{
		{
			name: "returns the configured result",
			arrange: func(p *FakeRateLimitProvider) {
				p.WithCheckResult(true, 3, time.Unix(2000, 0), nil)
			},
			key:             "key",
			window:          time.Minute,
			maxRequests:     5,
			expectedAllowed: true,
			expectedCount:   3,
			expectedReset:   time.Unix(2000, 0),
		},
		{
			name: "returns the configured error",
			arrange: func(p *FakeRateLimitProvider) {
				p.WithCheckError(errBoom)
			},
			key:           "key",
			window:        time.Minute,
			maxRequests:   5,
			expectedReset: time.Unix(1000, 0),
			expectedErr:   errBoom,
		},
		{
			name:          "defaults the reset time when none is configured",
			key:           "key",
			window:        time.Minute,
			maxRequests:   5,
			expectedReset: time.Unix(1000, 0),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := NewFakeRateLimitProvider()
			if tc.arrange != nil {
				tc.arrange(p)
			}

			allowed, count, reset, err := p.CheckAndIncrement(context.Background(), tc.key, tc.window, tc.maxRequests)

			assert.Equal(t, tc.key, p.LastCheckKey)
			assert.Equal(t, tc.window, p.LastCheckWindow)
			assert.Equal(t, tc.maxRequests, p.LastCheckMax)
			assert.Equal(t, tc.expectedAllowed, allowed)
			assert.Equal(t, tc.expectedCount, count)
			assert.Equal(t, tc.expectedReset, reset)
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func TestFakeRateLimitProvider_SetRule(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")

	tests := []struct {
		name           string
		arrange        func(p *FakeRateLimitProvider)
		key            string
		window         time.Duration
		maxRequests    int
		expectedErr    error
		expectedStored bool
	}{
		{
			name:           "stores the rule",
			key:            "key",
			window:         time.Minute,
			maxRequests:    5,
			expectedStored: true,
		},
		{
			name: "returns the configured error without storing the rule",
			arrange: func(p *FakeRateLimitProvider) {
				p.WithSetError(errBoom)
			},
			key:         "key",
			window:      time.Minute,
			maxRequests: 5,
			expectedErr: errBoom,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := NewFakeRateLimitProvider()
			if tc.arrange != nil {
				tc.arrange(p)
			}

			err := p.SetRule(context.Background(), tc.key, tc.window, tc.maxRequests)

			assert.Equal(t, 1, p.SetCalls)
			assert.Equal(t, tc.key, p.LastSetKey)
			assert.Equal(t, tc.window, p.LastSetWindow)
			assert.Equal(t, tc.maxRequests, p.LastSetMax)
			assert.ErrorIs(t, err, tc.expectedErr)

			if tc.expectedStored {
				assert.Equal(t, types.RateLimitRuleRecord{Key: tc.key, WindowSeconds: 60, MaxRequests: tc.maxRequests}, p.Store[tc.key])
				return
			}
			assert.NotContains(t, p.Store, tc.key)
		})
	}
}

func TestFakeRateLimitProvider_GetRule(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")

	tests := []struct {
		name           string
		arrange        func(p *FakeRateLimitProvider)
		key            string
		expectedWindow time.Duration
		expectedMax    int
		expectedFound  bool
		expectedErr    error
	}{
		{
			name: "returns a stored rule",
			arrange: func(p *FakeRateLimitProvider) {
				p.WithExistingRule("key", time.Minute, 5)
			},
			key:            "key",
			expectedWindow: time.Minute,
			expectedMax:    5,
			expectedFound:  true,
		},
		{
			name:          "returns not found for a missing key",
			key:           "missing",
			expectedFound: false,
		},
		{
			name: "returns the configured error",
			arrange: func(p *FakeRateLimitProvider) {
				p.WithGetError(errBoom)
			},
			key:         "key",
			expectedErr: errBoom,
		},
		{
			name: "get error takes precedence over a stored rule",
			arrange: func(p *FakeRateLimitProvider) {
				p.WithExistingRule("key", time.Minute, 5)
				p.WithGetError(errBoom)
			},
			key:         "key",
			expectedErr: errBoom,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := NewFakeRateLimitProvider()
			if tc.arrange != nil {
				tc.arrange(p)
			}

			window, maxRequests, found, err := p.GetRule(context.Background(), tc.key)

			assert.Equal(t, 1, p.GetCalls)
			assert.Equal(t, tc.expectedWindow, window)
			assert.Equal(t, tc.expectedMax, maxRequests)
			assert.Equal(t, tc.expectedFound, found)
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func TestFakeRateLimitProvider_DeleteRule(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")

	tests := []struct {
		name         string
		arrange      func(p *FakeRateLimitProvider)
		key          string
		expectedErr  error
		expectedKeys []string
	}{
		{
			name: "deletes an existing rule",
			arrange: func(p *FakeRateLimitProvider) {
				p.WithExistingRule("key", time.Minute, 5)
			},
			key:          "key",
			expectedKeys: []string{},
		},
		{
			name: "leaves other rules untouched when the key is missing",
			arrange: func(p *FakeRateLimitProvider) {
				p.WithExistingRule("other", time.Minute, 5)
			},
			key:          "missing",
			expectedKeys: []string{"other"},
		},
		{
			name: "returns the configured error and keeps the rule",
			arrange: func(p *FakeRateLimitProvider) {
				p.WithExistingRule("key", time.Minute, 5)
				p.DeleteRuleErr = errBoom
			},
			key:          "key",
			expectedErr:  errBoom,
			expectedKeys: []string{"key"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := NewFakeRateLimitProvider()
			if tc.arrange != nil {
				tc.arrange(p)
			}

			err := p.DeleteRule(context.Background(), tc.key)

			assert.ErrorIs(t, err, tc.expectedErr)
			assert.ElementsMatch(t, tc.expectedKeys, storeKeys(p))
		})
	}
}

func TestStubSecondaryStorageService(t *testing.T) {
	t.Parallel()

	storage := &storageStub{}

	tests := []struct {
		name            string
		stub            *StubSecondaryStorageService
		expectedStorage models.SecondaryStorage
		expectedName    string
	}{
		{
			name:            "returns the configured storage and name",
			stub:            &StubSecondaryStorageService{Storage: storage, Name: "memory"},
			expectedStorage: storage,
			expectedName:    "memory",
		},
		{
			name:         "returns nil storage when unset",
			stub:         &StubSecondaryStorageService{Name: "custom"},
			expectedName: "custom",
		},
		{
			name:            "returns an empty name when unset",
			stub:            &StubSecondaryStorageService{Storage: storage},
			expectedStorage: storage,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.stub.GetStorage()

			if tc.expectedStorage == nil {
				assert.Nil(t, got)
			} else {
				assert.Same(t, tc.expectedStorage, got)
			}
			assert.Equal(t, tc.expectedName, tc.stub.GetProviderName())
		})
	}
}

func TestRecordingServiceRegistry_Register(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		registrations      []registration
		expectedRegistered []string
		expectedServices   map[string]any
	}{
		{
			name:               "leaves the registry empty when nothing is registered",
			expectedRegistered: nil,
			expectedServices:   nil,
		},
		{
			name: "initializes the services map on first registration",
			registrations: []registration{
				{name: "cache", service: "svc-a"},
			},
			expectedRegistered: []string{"cache"},
			expectedServices:   map[string]any{"cache": "svc-a"},
		},
		{
			name: "records every registration in order",
			registrations: []registration{
				{name: "a", service: 1},
				{name: "b", service: 2},
			},
			expectedRegistered: []string{"a", "b"},
			expectedServices:   map[string]any{"a": 1, "b": 2},
		},
		{
			name: "overwrites the service but records both registrations",
			registrations: []registration{
				{name: "a", service: 1},
				{name: "a", service: 2},
			},
			expectedRegistered: []string{"a", "a"},
			expectedServices:   map[string]any{"a": 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &RecordingServiceRegistry{}
			for _, reg := range tc.registrations {
				r.Register(reg.name, reg.service)
			}

			assert.Equal(t, tc.expectedRegistered, r.Registered)
			assert.Equal(t, tc.expectedServices, r.Services)
		})
	}
}

func TestRecordingServiceRegistry_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		registry *RecordingServiceRegistry
		lookup   string
		expected any
	}{
		{
			name:     "returns nil for a nil registry",
			registry: nil,
			lookup:   "cache",
			expected: nil,
		},
		{
			name:     "returns nil when no services are registered",
			registry: &RecordingServiceRegistry{},
			lookup:   "cache",
			expected: nil,
		},
		{
			name: "returns the registered service",
			registry: &RecordingServiceRegistry{
				Services: map[string]any{"cache": "svc-a"},
			},
			lookup:   "cache",
			expected: "svc-a",
		},
		{
			name: "returns nil for an unknown name",
			registry: &RecordingServiceRegistry{
				Services: map[string]any{"cache": "svc-a"},
			},
			lookup:   "missing",
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, tc.registry.Get(tc.lookup))
		})
	}
}
