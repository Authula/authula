package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTOTPPluginConfig_ApplyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                      string
		config                    TOTPPluginConfig
		wantBackupCodeCount       int
		wantTrustedDeviceDuration time.Duration
		wantCleanupInterval       time.Duration
		wantPendingTokenExpiry    time.Duration
		wantSameSite              string
	}{
		{
			name:                      "zero_config_receives_defaults",
			wantBackupCodeCount:       10,
			wantTrustedDeviceDuration: 30 * 24 * time.Hour,
			wantCleanupInterval:       0,
			wantPendingTokenExpiry:    5 * time.Minute,
			wantSameSite:              "lax",
		},
		{
			name:                      "auto_cleanup_enables_default_interval",
			config:                    TOTPPluginConfig{TrustedDevicesAutoCleanup: true},
			wantBackupCodeCount:       10,
			wantTrustedDeviceDuration: 30 * 24 * time.Hour,
			wantCleanupInterval:       time.Hour,
			wantPendingTokenExpiry:    5 * time.Minute,
			wantSameSite:              "lax",
		},
		{
			name: "explicit_cleanup_interval_is_preserved_with_auto_cleanup",
			config: TOTPPluginConfig{
				TrustedDevicesAutoCleanup:     true,
				TrustedDevicesCleanupInterval: 2 * time.Hour,
			},
			wantBackupCodeCount:       10,
			wantTrustedDeviceDuration: 30 * 24 * time.Hour,
			wantCleanupInterval:       2 * time.Hour,
			wantPendingTokenExpiry:    5 * time.Minute,
			wantSameSite:              "lax",
		},
		{
			name: "explicit_cleanup_interval_is_preserved_without_auto_cleanup",
			config: TOTPPluginConfig{
				TrustedDevicesCleanupInterval: 2 * time.Hour,
			},
			wantBackupCodeCount:       10,
			wantTrustedDeviceDuration: 30 * 24 * time.Hour,
			wantCleanupInterval:       2 * time.Hour,
			wantPendingTokenExpiry:    5 * time.Minute,
			wantSameSite:              "lax",
		},
		{
			name: "explicit_values_are_preserved",
			config: TOTPPluginConfig{
				BackupCodeCount:       5,
				TrustedDeviceDuration: 7 * 24 * time.Hour,
				PendingTokenExpiry:    10 * time.Minute,
				SameSite:              "strict",
			},
			wantBackupCodeCount:       5,
			wantTrustedDeviceDuration: 7 * 24 * time.Hour,
			wantCleanupInterval:       0,
			wantPendingTokenExpiry:    10 * time.Minute,
			wantSameSite:              "strict",
		},
		{
			name: "negative_values_are_preserved",
			config: TOTPPluginConfig{
				BackupCodeCount:       -1,
				TrustedDeviceDuration: -time.Minute,
				PendingTokenExpiry:    -time.Minute,
			},
			wantBackupCodeCount:       -1,
			wantTrustedDeviceDuration: -time.Minute,
			wantCleanupInterval:       0,
			wantPendingTokenExpiry:    -time.Minute,
			wantSameSite:              "lax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := tt.config
			config.ApplyDefaults()

			require.Equal(t, tt.wantBackupCodeCount, config.BackupCodeCount)
			require.Equal(t, tt.wantTrustedDeviceDuration, config.TrustedDeviceDuration)
			require.Equal(t, tt.wantCleanupInterval, config.TrustedDevicesCleanupInterval)
			require.Equal(t, tt.wantPendingTokenExpiry, config.PendingTokenExpiry)
			require.Equal(t, tt.wantSameSite, config.SameSite)
		})
	}
}
