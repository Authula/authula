package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestApiKeyPluginConfig_ApplyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       ApiKeyPluginConfig
		wantHeader   string
		wantInterval time.Duration
	}{
		{
			name:         "zero_config_receives_defaults",
			wantHeader:   "X-AUTHULA-API-KEY",
			wantInterval: time.Minute,
		},
		{
			name:         "explicit_values_are_preserved",
			config:       ApiKeyPluginConfig{Header: "X-Custom-Key", CleanupInterval: 5 * time.Minute},
			wantHeader:   "X-Custom-Key",
			wantInterval: 5 * time.Minute,
		},
		{
			name:         "negative_cleanup_interval_receives_default",
			config:       ApiKeyPluginConfig{CleanupInterval: -time.Minute},
			wantHeader:   "X-AUTHULA-API-KEY",
			wantInterval: time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := tt.config
			config.ApplyDefaults()

			require.Equal(t, tt.wantHeader, config.Header)
			require.Equal(t, tt.wantInterval, config.CleanupInterval)
		})
	}
}

func TestApiKeyPluginConfig_ApplyDefaultsLeavesOtherFieldsIntact(t *testing.T) {
	t.Parallel()

	config := ApiKeyPluginConfig{
		AllowOrgKeys:  true,
		DefaultPrefix: "myapp",
		AutoCleanup:   true,
	}
	config.ApplyDefaults()

	require.True(t, config.AllowOrgKeys)
	require.Equal(t, "myapp", config.DefaultPrefix)
	require.True(t, config.AutoCleanup)
}
