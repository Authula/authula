package types

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestJWTPluginConfig_ApplyDefaults(t *testing.T) {
	t.Parallel()

	defaults := JWTPluginConfig{
		KeyRotationInterval:    30 * 24 * time.Hour,
		KeyRotationGracePeriod: time.Hour,
		ExpiresIn:              15 * time.Minute,
		RefreshExpiresIn:       7 * 24 * time.Hour,
		JWKSCacheTTL:           24 * time.Hour,
		RefreshGracePeriod:     10 * time.Second,
		AutoCleanup:            new(true),
		CleanupInterval:        time.Hour,
	}

	tests := []struct {
		name   string
		config JWTPluginConfig
		want   JWTPluginConfig
	}{
		{
			name:   "zero_config_gets_every_default",
			config: JWTPluginConfig{},
			want:   defaults,
		},
		{
			name: "custom_values_are_preserved",
			config: JWTPluginConfig{
				Enabled:                true,
				KeyRotationInterval:    48 * time.Hour,
				KeyRotationGracePeriod: 5 * time.Minute,
				ExpiresIn:              30 * time.Minute,
				RefreshExpiresIn:       14 * 24 * time.Hour,
				JWKSCacheTTL:           2 * time.Hour,
				RefreshGracePeriod:     time.Minute,
				AutoCleanup:            new(false),
				CleanupInterval:        30 * time.Minute,
				Issuer:                 "https://auth.example.com",
				Audience:               "my-service",
			},
			want: JWTPluginConfig{
				Enabled:                true,
				KeyRotationInterval:    48 * time.Hour,
				KeyRotationGracePeriod: 5 * time.Minute,
				ExpiresIn:              30 * time.Minute,
				RefreshExpiresIn:       14 * 24 * time.Hour,
				JWKSCacheTTL:           2 * time.Hour,
				RefreshGracePeriod:     time.Minute,
				AutoCleanup:            new(false),
				CleanupInterval:        30 * time.Minute,
				Issuer:                 "https://auth.example.com",
				Audience:               "my-service",
			},
		},
		{
			name: "partial_config_fills_only_missing_fields",
			config: JWTPluginConfig{
				ExpiresIn:    5 * time.Minute,
				JWKSCacheTTL: time.Hour,
			},
			want: JWTPluginConfig{
				KeyRotationInterval:    defaults.KeyRotationInterval,
				KeyRotationGracePeriod: defaults.KeyRotationGracePeriod,
				ExpiresIn:              5 * time.Minute,
				RefreshExpiresIn:       defaults.RefreshExpiresIn,
				JWKSCacheTTL:           time.Hour,
				RefreshGracePeriod:     defaults.RefreshGracePeriod,
				AutoCleanup:            defaults.AutoCleanup,
				CleanupInterval:        defaults.CleanupInterval,
			},
		},
		{
			name: "negative_durations_are_treated_as_configured",
			config: JWTPluginConfig{
				ExpiresIn:          -time.Minute,
				RefreshGracePeriod: -time.Second,
			},
			want: JWTPluginConfig{
				KeyRotationInterval:    defaults.KeyRotationInterval,
				KeyRotationGracePeriod: defaults.KeyRotationGracePeriod,
				ExpiresIn:              -time.Minute,
				RefreshExpiresIn:       defaults.RefreshExpiresIn,
				JWKSCacheTTL:           defaults.JWKSCacheTTL,
				RefreshGracePeriod:     -time.Second,
				AutoCleanup:            defaults.AutoCleanup,
				CleanupInterval:        defaults.CleanupInterval,
			},
		},
		{
			name: "non_positive_cleanup_interval_gets_default",
			config: JWTPluginConfig{
				CleanupInterval: -time.Minute,
			},
			want: JWTPluginConfig{
				KeyRotationInterval:    defaults.KeyRotationInterval,
				KeyRotationGracePeriod: defaults.KeyRotationGracePeriod,
				ExpiresIn:              defaults.ExpiresIn,
				RefreshExpiresIn:       defaults.RefreshExpiresIn,
				JWKSCacheTTL:           defaults.JWKSCacheTTL,
				RefreshGracePeriod:     defaults.RefreshGracePeriod,
				AutoCleanup:            defaults.AutoCleanup,
				CleanupInterval:        defaults.CleanupInterval,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := tt.config
			config.ApplyDefaults()

			require.Equal(t, tt.want, config)
		})
	}
}

func TestJWTPluginConfig_ApplyDefaultsFromGlobal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       JWTPluginConfig
		baseURL      string
		basePath     string
		wantIssuer   string
		wantAudience string
	}{
		{
			name:         "derives_defaults_from_global_config",
			baseURL:      "http://localhost:8080",
			basePath:     "/api/auth",
			wantIssuer:   "http://localhost:8080/api/auth",
			wantAudience: DefaultAudience,
		},
		{
			name:         "trims_separators_between_url_and_path",
			baseURL:      "https://auth.example.com/",
			basePath:     "/api/auth/",
			wantIssuer:   "https://auth.example.com/api/auth",
			wantAudience: DefaultAudience,
		},
		{
			name:         "handles_empty_base_path",
			baseURL:      "https://auth.example.com",
			basePath:     "",
			wantIssuer:   "https://auth.example.com",
			wantAudience: DefaultAudience,
		},
		{
			name: "preserves_explicit_values",
			config: JWTPluginConfig{
				Issuer:   "https://issuer.example.com",
				Audience: "my-service",
			},
			baseURL:      "http://localhost:8080",
			basePath:     "/api/auth",
			wantIssuer:   "https://issuer.example.com",
			wantAudience: "my-service",
		},
		{
			name:         "leaves_issuer_empty_without_base_url",
			baseURL:      "",
			basePath:     "",
			wantIssuer:   "",
			wantAudience: DefaultAudience,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := tt.config
			config.ApplyDefaultsFromGlobal(tt.baseURL, tt.basePath)

			if config.Issuer != tt.wantIssuer {
				t.Errorf("Issuer = %q, want %q", config.Issuer, tt.wantIssuer)
			}
			if config.Audience != tt.wantAudience {
				t.Errorf("Audience = %q, want %q", config.Audience, tt.wantAudience)
			}
		})
	}
}

func TestJWTPluginConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  JWTPluginConfig
		wantErr string
	}{
		{
			name: "valid",
			config: JWTPluginConfig{
				Issuer:   "https://auth.example.com",
				Audience: DefaultAudience,
			},
		},
		{
			name: "missing_issuer",
			config: JWTPluginConfig{
				Audience: DefaultAudience,
			},
			wantErr: "issuer must not be empty",
		},
		{
			name: "blank_issuer",
			config: JWTPluginConfig{
				Issuer:   "   ",
				Audience: DefaultAudience,
			},
			wantErr: "issuer must not be empty",
		},
		{
			name: "missing_audience",
			config: JWTPluginConfig{
				Issuer: "https://auth.example.com",
			},
			wantErr: "audience must not be empty",
		},
		{
			name: "blank_audience",
			config: JWTPluginConfig{
				Issuer:   "https://auth.example.com",
				Audience: " ",
			},
			wantErr: "audience must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("Validate() error = nil, want %q", tt.wantErr)
			}
			if got := err.Error(); !strings.Contains(got, tt.wantErr) {
				t.Fatalf("Validate() error = %q, want it to contain %q", got, tt.wantErr)
			}
		})
	}
}
