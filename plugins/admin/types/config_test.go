package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdminPluginConfig_ApplyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config AdminPluginConfig
		want   time.Duration
	}{
		{
			name: "zero_expiry_receives_default",
			want: 15 * time.Minute,
		},
		{
			name:   "explicit_expiry_is_preserved",
			config: AdminPluginConfig{ImpersonationMaxExpiresIn: 2 * time.Hour},
			want:   2 * time.Hour,
		},
		{
			name:   "negative_expiry_is_preserved",
			config: AdminPluginConfig{ImpersonationMaxExpiresIn: -time.Minute},
			want:   -time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := tt.config
			config.ApplyDefaults()

			require.Equal(t, tt.want, config.ImpersonationMaxExpiresIn)
		})
	}
}
