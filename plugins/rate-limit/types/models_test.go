package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimit_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	expiresAt := time.Date(2026, time.November, 17, 10, 0, 0, 0, time.UTC)
	want := RateLimit{
		Key:       "ratelimit:user-1",
		Count:     3,
		ExpiresAt: expiresAt,
	}

	data, err := json.Marshal(want)
	require.NoError(t, err)
	require.JSONEq(t, `{"key":"ratelimit:user-1","count":3,"expires_at":"2026-11-17T10:00:00Z"}`, string(data))

	var got RateLimit
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}

func TestRateLimitRuleRecord_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	want := RateLimitRuleRecord{
		Key:           "ratelimit:/admin",
		WindowSeconds: 60,
		MaxRequests:   10,
	}

	data, err := json.Marshal(want)
	require.NoError(t, err)
	require.JSONEq(t, `{"key":"ratelimit:/admin","window_seconds":60,"max_requests":10}`, string(data))

	var got RateLimitRuleRecord
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}
