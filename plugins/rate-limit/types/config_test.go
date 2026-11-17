package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/env"
)

func TestRateLimitProviderType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider RateLimitProviderType
		want     string
	}{
		{name: "memory", provider: RateLimitProviderInMemory, want: "memory"},
		{name: "redis", provider: RateLimitProviderRedis, want: "redis"},
		{name: "database", provider: RateLimitProviderDatabase, want: "database"},
		{name: "custom", provider: RateLimitProviderType("custom"), want: "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.provider.String())
		})
	}
}

func TestRateLimitPluginConfig_ApplyDefaultsOutsideProduction(t *testing.T) {
	t.Setenv(env.EnvGoEnvironment, "development")

	tests := []struct {
		name               string
		config             RateLimitPluginConfig
		wantEnabled        bool
		wantWindow         time.Duration
		wantMax            int
		wantPrefix         string
		wantCustomRuleKeys []string
	}{
		{
			name:               "zero_config_receives_defaults",
			wantWindow:         time.Minute,
			wantMax:            100,
			wantPrefix:         "ratelimit:",
			wantCustomRuleKeys: []string{},
		},
		{
			name: "explicit_values_are_preserved",
			config: RateLimitPluginConfig{
				Enabled:      true,
				Window:       30 * time.Second,
				Max:          5,
				Prefix:       "custom:",
				HashClientIP: true,
				CustomRules: map[string]RateLimitRule{
					"/admin": {Window: time.Minute, Max: 10},
				},
				Provider: RateLimitProviderRedis,
			},
			wantEnabled:        true,
			wantWindow:         30 * time.Second,
			wantMax:            5,
			wantPrefix:         "custom:",
			wantCustomRuleKeys: []string{"/admin"},
		},
		{
			name: "negative_values_are_preserved",
			config: RateLimitPluginConfig{
				Window: -time.Minute,
				Max:    -1,
			},
			wantWindow:         -time.Minute,
			wantMax:            -1,
			wantPrefix:         "ratelimit:",
			wantCustomRuleKeys: []string{},
		},
		{
			name:               "explicit_empty_custom_rules_are_preserved",
			config:             RateLimitPluginConfig{CustomRules: map[string]RateLimitRule{}},
			wantWindow:         time.Minute,
			wantMax:            100,
			wantPrefix:         "ratelimit:",
			wantCustomRuleKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.config
			config.ApplyDefaults()

			require.Equal(t, tt.wantEnabled, config.Enabled)
			require.Equal(t, tt.wantWindow, config.Window)
			require.Equal(t, tt.wantMax, config.Max)
			require.Equal(t, tt.wantPrefix, config.Prefix)
			require.NotNil(t, config.CustomRules)
			require.ElementsMatch(t, tt.wantCustomRuleKeys, mapKeys(config.CustomRules))
		})
	}
}

func TestRateLimitPluginConfig_ApplyDefaultsInProduction(t *testing.T) {
	t.Setenv(env.EnvGoEnvironment, "production")

	tests := []struct {
		name        string
		config      RateLimitPluginConfig
		wantEnabled bool
	}{
		{
			name:        "enables_rate_limiting_by_default",
			wantEnabled: true,
		},
		{
			name:        "keeps_rate_limiting_enabled",
			config:      RateLimitPluginConfig{Enabled: true},
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.config
			config.ApplyDefaults()

			require.Equal(t, tt.wantEnabled, config.Enabled)
			require.Equal(t, time.Minute, config.Window)
			require.Equal(t, 100, config.Max)
			require.Equal(t, "ratelimit:", config.Prefix)
			require.NotNil(t, config.CustomRules)
		})
	}
}

func TestRateLimitPluginConfig_JSONOmitsEmptyOptionals(t *testing.T) {
	t.Parallel()

	config := RateLimitPluginConfig{
		Enabled:  true,
		Window:   time.Minute,
		Max:      100,
		Provider: RateLimitProviderInMemory,
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &fields))
	require.Contains(t, fields, "enabled")
	require.Contains(t, fields, "window")
	require.Contains(t, fields, "max")
	require.Contains(t, fields, "custom_rules")
	require.Contains(t, fields, "provider")
	require.NotContains(t, fields, "prefix")
	require.NotContains(t, fields, "hash_client_ip")
	require.NotContains(t, fields, "memory")
	require.NotContains(t, fields, "database")
}

func mapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
