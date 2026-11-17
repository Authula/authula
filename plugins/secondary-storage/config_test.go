package secondarystorage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSecondaryStorageProviderString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider SecondaryStorageProvider
		want     string
	}{
		{name: "memory", provider: SecondaryStorageProviderMemory, want: "memory"},
		{name: "database", provider: SecondaryStorageProviderDatabase, want: "database"},
		{name: "redis", provider: SecondaryStorageProviderRedis, want: "redis"},
		{name: "custom value is preserved", provider: SecondaryStorageProvider("custom"), want: "custom"},
		{name: "empty value stays empty", provider: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.provider.String())
		})
	}
}

func TestSecondaryStoragePluginConfigApplyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		config            SecondaryStoragePluginConfig
		wantProvider      SecondaryStorageProvider
		wantEnabled       bool
		wantMemoryCleanup time.Duration
	}{
		{
			name:         "empty provider defaults to memory",
			config:       SecondaryStoragePluginConfig{},
			wantProvider: SecondaryStorageProviderMemory,
		},
		{
			name:         "explicit memory is preserved",
			config:       SecondaryStoragePluginConfig{Provider: SecondaryStorageProviderMemory, Enabled: true},
			wantProvider: SecondaryStorageProviderMemory,
			wantEnabled:  true,
		},
		{
			name:         "explicit database is preserved",
			config:       SecondaryStoragePluginConfig{Provider: SecondaryStorageProviderDatabase},
			wantProvider: SecondaryStorageProviderDatabase,
		},
		{
			name:         "explicit redis is preserved",
			config:       SecondaryStoragePluginConfig{Provider: SecondaryStorageProviderRedis},
			wantProvider: SecondaryStorageProviderRedis,
		},
		{
			name: "provider-specific settings are untouched",
			config: SecondaryStoragePluginConfig{
				Memory: &MemoryStorageConfig{CleanupInterval: 5 * time.Second},
			},
			wantProvider:      SecondaryStorageProviderMemory,
			wantMemoryCleanup: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := tt.config
			config.ApplyDefaults()

			assert.Equal(t, tt.wantProvider, config.Provider)
			assert.Equal(t, tt.wantEnabled, config.Enabled)
			if tt.config.Memory != nil {
				assert.Equal(t, tt.wantMemoryCleanup, config.Memory.CleanupInterval)
			}
			if tt.config.Database != nil {
				assert.Same(t, tt.config.Database, config.Database)
			}
			if tt.config.Redis != nil {
				assert.Same(t, tt.config.Redis, config.Redis)
			}
		})
	}
}

func TestSecondaryStoragePluginConfigApplyDefaultsIsIdempotent(t *testing.T) {
	t.Parallel()

	config := SecondaryStoragePluginConfig{}

	config.ApplyDefaults()
	config.ApplyDefaults()

	assert.Equal(t, SecondaryStorageProviderMemory, config.Provider)
}
