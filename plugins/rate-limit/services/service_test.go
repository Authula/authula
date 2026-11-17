package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	plugintests "github.com/Authula/authula/plugins/rate-limit/tests"
	"github.com/Authula/authula/plugins/rate-limit/types"
)

func TestRateLimiterService_GetValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(*plugintests.FakeRateLimitProvider)
		wantNil bool
		wantErr error
	}{
		{
			name: "existing_value",
			setup: func(p *plugintests.FakeRateLimitProvider) {
				p.Store["key-1"] = types.RateLimitRuleRecord{Key: "key-1", WindowSeconds: 60, MaxRequests: 5}
			},
		},
		{
			name:    "missing_value_returns_nil",
			setup:   func(*plugintests.FakeRateLimitProvider) {},
			wantNil: true,
		},
		{
			name: "provider_error",
			setup: func(p *plugintests.FakeRateLimitProvider) {
				p.GetValueErr = errors.New("boom")
			},
			wantErr: errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			provider := plugintests.NewFakeRateLimitProvider()
			tc.setup(provider)
			service := NewRateLimiterService(provider)

			value, err := service.GetValue(context.Background(), "key-1")

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
				assert.Nil(t, value)
			} else {
				require.NoError(t, err)
			}
			if tc.wantNil {
				assert.Nil(t, value)
			} else if tc.wantErr == nil {
				require.NotNil(t, value)
			}
		})
	}
}

func TestRateLimiterService_CheckAndIncrement(t *testing.T) {
	t.Parallel()

	resetAt := time.Unix(4000, 0)
	tests := []struct {
		name      string
		provider  *plugintests.FakeRateLimitProvider
		wantAllow bool
		wantCount int
		wantErr   error
	}{
		{
			name:      "allowed",
			provider:  plugintests.NewFakeRateLimitProvider().WithCheckResult(true, 2, resetAt, nil),
			wantAllow: true,
			wantCount: 2,
		},
		{
			name:      "blocked",
			provider:  plugintests.NewFakeRateLimitProvider().WithCheckResult(false, 11, resetAt, nil),
			wantAllow: false,
			wantCount: 11,
		},
		{
			name:     "provider_error",
			provider: plugintests.NewFakeRateLimitProvider().WithCheckError(errors.New("boom")),
			wantErr:  errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewRateLimiterService(tc.provider)

			allowed, count, gotReset, err := service.CheckAndIncrement(context.Background(), "key-1", time.Minute, 10)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantAllow, allowed)
			assert.Equal(t, tc.wantCount, count)
			assert.Equal(t, resetAt, gotReset)
			assert.Equal(t, "key-1", tc.provider.LastCheckKey)
			assert.Equal(t, time.Minute, tc.provider.LastCheckWindow)
			assert.Equal(t, 10, tc.provider.LastCheckMax)
		})
	}
}

func TestRateLimiterService_SetRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider *plugintests.FakeRateLimitProvider
		wantErr  error
	}{
		{
			name:     "success",
			provider: plugintests.NewFakeRateLimitProvider(),
		},
		{
			name:     "provider_error",
			provider: plugintests.NewFakeRateLimitProvider().WithSetError(errors.New("boom")),
			wantErr:  errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewRateLimiterService(tc.provider)

			err := service.SetRule(context.Background(), "key-1", 2*time.Minute, 7)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "key-1", tc.provider.LastSetKey)
			assert.Equal(t, 2*time.Minute, tc.provider.LastSetWindow)
			assert.Equal(t, 7, tc.provider.LastSetMax)
		})
	}
}

func TestRateLimiterService_GetRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		provider   *plugintests.FakeRateLimitProvider
		wantNil    bool
		wantWindow time.Duration
		wantMax    int
		wantErr    error
	}{
		{
			name:       "found",
			provider:   plugintests.NewFakeRateLimitProvider().WithExistingRule("key-1", 90*time.Second, 13),
			wantWindow: 90 * time.Second,
			wantMax:    13,
		},
		{
			name:     "not_found_returns_nil",
			provider: plugintests.NewFakeRateLimitProvider(),
			wantNil:  true,
		},
		{
			name:     "provider_error",
			provider: plugintests.NewFakeRateLimitProvider().WithGetError(errors.New("boom")),
			wantErr:  errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewRateLimiterService(tc.provider)

			rule, err := service.GetRule(context.Background(), "key-1")

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
				assert.Nil(t, rule)
				return
			}
			require.NoError(t, err)
			if tc.wantNil {
				assert.Nil(t, rule)
				return
			}
			require.NotNil(t, rule)
			assert.Equal(t, tc.wantWindow, rule.Window)
			assert.Equal(t, tc.wantMax, rule.MaxRequests)
		})
	}
}

func TestRateLimiterService_DeleteRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider *plugintests.FakeRateLimitProvider
		wantErr  error
	}{
		{
			name:     "success",
			provider: plugintests.NewFakeRateLimitProvider().WithExistingRule("key-1", time.Minute, 5),
		},
		{
			name:     "provider_error",
			provider: &plugintests.FakeRateLimitProvider{Store: map[string]types.RateLimitRuleRecord{}, DeleteRuleErr: errors.New("boom")},
			wantErr:  errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewRateLimiterService(tc.provider)

			err := service.DeleteRule(context.Background(), "key-1")

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
				return
			}
			require.NoError(t, err)
			_, ok := tc.provider.Store["key-1"]
			assert.False(t, ok)
		})
	}
}
