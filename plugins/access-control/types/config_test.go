package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccessControlPluginConfig_ApplyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      AccessControlPluginConfig
		wantEnabled bool
	}{
		{
			name:        "zero_config_stays_disabled",
			wantEnabled: false,
		},
		{
			name:        "enabled_config_is_preserved",
			config:      AccessControlPluginConfig{Enabled: true},
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := tt.config
			config.ApplyDefaults()

			require.Equal(t, tt.wantEnabled, config.Enabled)
		})
	}
}
