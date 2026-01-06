package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/types"
)

func TestJWTPluginConfig_DefaultConfig(t *testing.T) {
	tests := []struct {
		name   string
		config types.JWTPluginConfig
		check  func(*testing.T, types.JWTPluginConfig)
	}{
		{
			name:   "sets default algorithm",
			config: types.JWTPluginConfig{},
			check: func(t *testing.T, c types.JWTPluginConfig) {
				if c.Algorithm != "ed25519" {
					t.Errorf("Algorithm = %v, want ed25519", c.Algorithm)
				}
			},
		},
		{
			name:   "preserves custom algorithm",
			config: types.JWTPluginConfig{Algorithm: "rs256"},
			check: func(t *testing.T, c types.JWTPluginConfig) {
				if c.Algorithm != "rs256" {
					t.Errorf("Algorithm = %v, want rs256", c.Algorithm)
				}
			},
		},
		{
			name:   "sets default key rotation interval",
			config: types.JWTPluginConfig{},
			check: func(t *testing.T, c types.JWTPluginConfig) {
				expected := 90 * 24 * time.Hour
				if c.KeyRotationInterval != expected {
					t.Errorf("KeyRotationInterval = %v, want %v", c.KeyRotationInterval, expected)
				}
			},
		},
		{
			name:   "sets default access token expiry",
			config: types.JWTPluginConfig{},
			check: func(t *testing.T, c types.JWTPluginConfig) {
				expected := 15 * time.Minute
				if c.ExpiresIn != expected {
					t.Errorf("ExpiresIn = %v, want %v", c.ExpiresIn, expected)
				}
			},
		},
		{
			name:   "sets default refresh token expiry",
			config: types.JWTPluginConfig{},
			check: func(t *testing.T, c types.JWTPluginConfig) {
				expected := 7 * 24 * time.Hour
				if c.RefreshExpiresIn != expected {
					t.Errorf("RefreshExpiresIn = %v, want %v", c.RefreshExpiresIn, expected)
				}
			},
		},
		{
			name:   "sets default JWKS cache TTL",
			config: types.JWTPluginConfig{},
			check: func(t *testing.T, c types.JWTPluginConfig) {
				expected := 24 * time.Hour
				if c.JWKSCacheTTL != expected {
					t.Errorf("JWKSCacheTTL = %v, want %v", c.JWKSCacheTTL, expected)
				}
			},
		},
		{
			name: "preserves custom values",
			config: types.JWTPluginConfig{
				Algorithm:           "es256",
				KeyRotationInterval: 30 * 24 * time.Hour,
				ExpiresIn:           30 * time.Minute,
				RefreshExpiresIn:    14 * 24 * time.Hour,
				JWKSCacheTTL:        12 * time.Hour,
			},
			check: func(t *testing.T, c types.JWTPluginConfig) {
				if c.Algorithm != "es256" {
					t.Errorf("Algorithm = %v, want es256", c.Algorithm)
				}
				if c.KeyRotationInterval != 30*24*time.Hour {
					t.Errorf("KeyRotationInterval not preserved")
				}
				if c.ExpiresIn != 30*time.Minute {
					t.Errorf("ExpiresIn not preserved")
				}
				if c.RefreshExpiresIn != 14*24*time.Hour {
					t.Errorf("RefreshExpiresIn not preserved")
				}
				if c.JWKSCacheTTL != 12*time.Hour {
					t.Errorf("JWKSCacheTTL not preserved")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.config
			config.ApplyDefaults()
			tt.check(t, config)
		})
	}
}

func TestJWTPlugin_Metadata(t *testing.T) {
	plugin := New(types.JWTPluginConfig{})
	metadata := plugin.Metadata()

	if metadata.ID == "" {
		t.Error("Plugin ID is empty")
	}

	if metadata.Version == "" {
		t.Error("Plugin version is empty")
	}

	if metadata.Description == "" {
		t.Error("Plugin description is empty")
	}

	expectedID := "jwt"
	if metadata.ID != expectedID {
		t.Errorf("Plugin ID = %v, want %v", metadata.ID, expectedID)
	}
}

func TestJWTPlugin_Migrations(t *testing.T) {
	plugin := New(types.JWTPluginConfig{})
	ctx := context.Background()

	// Test that migrations returns a non-nil embed.FS for postgres
	migrations, err := plugin.Migrations(ctx, "postgres")
	if err != nil {
		t.Errorf("Migrations() error = %v, want nil", err)
	}

	if migrations == nil {
		t.Errorf("Migrations() returned nil, want non-nil embed.FS")
	}

	// Test that migrations returns a non-nil embed.FS for mysql
	migrations, err = plugin.Migrations(ctx, "mysql")
	if err != nil {
		t.Errorf("Migrations() error = %v, want nil", err)
	}

	if migrations == nil {
		t.Errorf("Migrations() returned nil, want non-nil embed.FS for mysql")
	}
}

func TestJWTPlugin_Config(t *testing.T) {
	config := types.JWTPluginConfig{
		Algorithm: "es256",
		ExpiresIn: 30 * time.Minute,
	}

	plugin := New(config)
	returnedConfig := plugin.Config()

	if returnedConfig == nil {
		t.Fatal("Config() returned nil")
	}

	cfg, ok := returnedConfig.(types.JWTPluginConfig)
	if !ok {
		t.Fatal("Config() did not return types.JWTPluginConfig type")
	}

	if cfg.Algorithm != config.Algorithm {
		t.Errorf("Config Algorithm = %v, want %v", cfg.Algorithm, config.Algorithm)
	}
}
