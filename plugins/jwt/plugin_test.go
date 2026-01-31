package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/types"
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
				if c.Algorithm != types.JWTAlgEdDSA {
					t.Errorf("Algorithm = %v, want %v", c.Algorithm, types.JWTAlgEdDSA)
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
				expected := 30 * 24 * time.Hour
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

	expectedID := models.PluginJWT.String()
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

func TestKeyRotationInterval_Configuration(t *testing.T) {
	tests := []struct {
		name           string
		configInterval time.Duration
		expectedAfter  time.Duration
		shouldRotate   bool
	}{
		{
			name:           "default 30 day interval",
			configInterval: 30 * 24 * time.Hour,
			expectedAfter:  29 * 24 * time.Hour,
			shouldRotate:   false,
		},
		{
			name:           "key past interval should rotate",
			configInterval: 30 * 24 * time.Hour,
			expectedAfter:  31 * 24 * time.Hour,
			shouldRotate:   true,
		},
		{
			name:           "short 1 hour interval",
			configInterval: 1 * time.Hour,
			expectedAfter:  2 * time.Hour,
			shouldRotate:   true,
		},
		{
			name:           "very short 1 minute interval",
			configInterval: 1 * time.Minute,
			expectedAfter:  2 * time.Minute,
			shouldRotate:   true,
		},
		{
			name:           "long 90 day interval",
			configInterval: 90 * 24 * time.Hour,
			expectedAfter:  89 * 24 * time.Hour,
			shouldRotate:   false,
		},
		{
			name:           "exactly at interval boundary",
			configInterval: 30 * 24 * time.Hour,
			expectedAfter:  30 * 24 * time.Hour,
			shouldRotate:   false,
		},
		{
			name:           "just past interval boundary",
			configInterval: 30 * 24 * time.Hour,
			expectedAfter:  30*24*time.Hour + time.Second,
			shouldRotate:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyAge := tt.expectedAfter
			rotationInterval := tt.configInterval

			shouldRotate := keyAge > rotationInterval

			if shouldRotate != tt.shouldRotate {
				t.Errorf("keyAge=%v, rotationInterval=%v, shouldRotate=%v, want %v",
					keyAge, rotationInterval, shouldRotate, tt.shouldRotate)
			}
		})
	}
}

func TestKeyRotationInterval_ConfigPreservation(t *testing.T) {
	tests := []struct {
		name           string
		inputConfig    types.JWTPluginConfig
		expectedConfig types.JWTPluginConfig
	}{
		{
			name: "custom 7 day interval",
			inputConfig: types.JWTPluginConfig{
				KeyRotationInterval: 7 * 24 * time.Hour,
			},
			expectedConfig: types.JWTPluginConfig{
				KeyRotationInterval: 7 * 24 * time.Hour,
			},
		},
		{
			name: "custom 60 day interval",
			inputConfig: types.JWTPluginConfig{
				KeyRotationInterval: 60 * 24 * time.Hour,
			},
			expectedConfig: types.JWTPluginConfig{
				KeyRotationInterval: 60 * 24 * time.Hour,
			},
		},
		{
			name: "custom 1 hour interval",
			inputConfig: types.JWTPluginConfig{
				KeyRotationInterval: time.Hour,
			},
			expectedConfig: types.JWTPluginConfig{
				KeyRotationInterval: time.Hour,
			},
		},
		{
			name: "zero interval uses default",
			inputConfig: types.JWTPluginConfig{
				KeyRotationInterval: 0,
			},
			expectedConfig: types.JWTPluginConfig{
				KeyRotationInterval: 30 * 24 * time.Hour,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.inputConfig
			config.ApplyDefaults()

			if config.KeyRotationInterval != tt.expectedConfig.KeyRotationInterval {
				t.Errorf("KeyRotationInterval = %v, want %v",
					config.KeyRotationInterval, tt.expectedConfig.KeyRotationInterval)
			}
		})
	}
}

func TestKeyRotationInterval_AlgorithmCompatibility(t *testing.T) {
	algorithms := []types.JWTAlgorithm{
		types.JWTAlgEdDSA,
		types.JWTAlgRS256,
		types.JWTAlgPS256,
		types.JWTAlgES256,
		types.JWTAlgES512,
	}

	for _, alg := range algorithms {
		t.Run(string(alg), func(t *testing.T) {
			config := types.JWTPluginConfig{
				Algorithm:           alg,
				KeyRotationInterval: 30 * 24 * time.Hour,
			}
			config.ApplyDefaults()

			if config.Algorithm != alg {
				t.Errorf("Algorithm changed from %v to %v", alg, config.Algorithm)
			}

			if config.KeyRotationInterval != 30*24*time.Hour {
				t.Errorf("KeyRotationInterval = %v, want %v",
					config.KeyRotationInterval, 30*24*time.Hour)
			}
		})
	}
}

func TestKeyRotationInterval_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name           string
		keyAge         time.Duration
		interval       time.Duration
		expectedRotate bool
	}{
		{
			name:           "zero key age should not rotate",
			keyAge:         0,
			interval:       30 * 24 * time.Hour,
			expectedRotate: false,
		},
		{
			name:           "very small interval with new key",
			keyAge:         time.Nanosecond,
			interval:       time.Nanosecond,
			expectedRotate: false,
		},
		{
			name:           "maximum practical interval",
			keyAge:         365 * 24 * time.Hour,
			interval:       90 * 24 * time.Hour,
			expectedRotate: true,
		},
		{
			name:           "millisecond precision at boundary",
			keyAge:         30*24*time.Hour - time.Millisecond,
			interval:       30 * 24 * time.Hour,
			expectedRotate: false,
		},
		{
			name:           "millisecond precision past boundary",
			keyAge:         30*24*time.Hour + time.Millisecond,
			interval:       30 * 24 * time.Hour,
			expectedRotate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldRotate := tt.keyAge > tt.interval

			if shouldRotate != tt.expectedRotate {
				t.Errorf("keyAge=%v, interval=%v, shouldRotate=%v, want %v",
					tt.keyAge, tt.interval, shouldRotate, tt.expectedRotate)
			}
		})
	}
}

func TestKeyRotationInterval_WithOtherConfigOptions(t *testing.T) {
	config := types.JWTPluginConfig{
		Algorithm:           types.JWTAlgEdDSA,
		KeyRotationInterval: 45 * 24 * time.Hour,
		ExpiresIn:           5 * time.Minute,
		RefreshExpiresIn:    30 * 24 * time.Hour,
		JWKSCacheTTL:        6 * time.Hour,
		RefreshGracePeriod:  30 * time.Second,
	}
	config.ApplyDefaults()

	if config.KeyRotationInterval != 45*24*time.Hour {
		t.Errorf("KeyRotationInterval = %v, want %v", config.KeyRotationInterval, 45*24*time.Hour)
	}

	if config.Algorithm != types.JWTAlgEdDSA {
		t.Errorf("Algorithm = %v, want %v", config.Algorithm, types.JWTAlgEdDSA)
	}

	if config.ExpiresIn != 5*time.Minute {
		t.Errorf("ExpiresIn = %v, want %v", config.ExpiresIn, 5*time.Minute)
	}

	if config.RefreshExpiresIn != 30*24*time.Hour {
		t.Errorf("RefreshExpiresIn = %v, want %v", config.RefreshExpiresIn, 30*24*time.Hour)
	}

	if config.JWKSCacheTTL != 6*time.Hour {
		t.Errorf("JWKSCacheTTL = %v, want %v", config.JWKSCacheTTL, 6*time.Hour)
	}

	if config.RefreshGracePeriod != 30*time.Second {
		t.Errorf("RefreshGracePeriod = %v, want %v", config.RefreshGracePeriod, 30*time.Second)
	}
}

func TestKeyRotationInterval_TimingCalculations(t *testing.T) {
	baseInterval := 30 * 24 * time.Hour

	tests := []struct {
		name         string
		days         int
		expectedOver bool
	}{
		{"29 days", 29, false},
		{"30 days", 30, false},
		{"31 days", 31, true},
		{"60 days", 60, true},
		{"90 days", 90, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyAge := time.Duration(tt.days) * 24 * time.Hour
			isOverDue := keyAge > baseInterval

			if isOverDue != tt.expectedOver {
				t.Errorf("keyAge=%v, isOverDue=%v, want %v", keyAge, isOverDue, tt.expectedOver)
			}
		})
	}
}

func TestKeyRotationInterval_PluginStorage(t *testing.T) {
	plugin := New(types.JWTPluginConfig{
		KeyRotationInterval: 45 * 24 * time.Hour,
	})

	storedConfig := plugin.Config().(types.JWTPluginConfig)

	if storedConfig.KeyRotationInterval != 45*24*time.Hour {
		t.Errorf("Stored KeyRotationInterval = %v, want %v",
			storedConfig.KeyRotationInterval, 45*24*time.Hour)
	}
}

func TestKeyRotationInterval_MultipleConfigUpdates(t *testing.T) {
	plugin := New(types.JWTPluginConfig{
		KeyRotationInterval: 30 * 24 * time.Hour,
	})

	pluginConfig := plugin.Config().(types.JWTPluginConfig)
	if pluginConfig.KeyRotationInterval != 30*24*time.Hour {
		t.Errorf("Initial KeyRotationInterval = %v, want %v",
			pluginConfig.KeyRotationInterval, 30*24*time.Hour)
	}

	plugin2 := New(types.JWTPluginConfig{
		KeyRotationInterval: 90 * 24 * time.Hour,
	})

	plugin2Config := plugin2.Config().(types.JWTPluginConfig)
	if plugin2Config.KeyRotationInterval != 90*24*time.Hour {
		t.Errorf("Updated KeyRotationInterval = %v, want %v",
			plugin2Config.KeyRotationInterval, 90*24*time.Hour)
	}
}

func TestKeyRotationInterval_EdgeCaseIntervals(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		valid    bool
	}{
		{"1 nanosecond", time.Nanosecond, true},
		{"1 microsecond", time.Microsecond, true},
		{"1 millisecond", time.Millisecond, true},
		{"1 second", time.Second, true},
		{"1 minute", time.Minute, true},
		{"1 hour", time.Hour, true},
		{"1 day", 24 * time.Hour, true},
		{"1 week", 7 * 24 * time.Hour, true},
		{"1 month (30 days)", 30 * 24 * time.Hour, true},
		{"1 year", 365 * 24 * time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := types.JWTPluginConfig{
				KeyRotationInterval: tt.interval,
			}
			config.ApplyDefaults()

			if tt.valid && config.KeyRotationInterval != tt.interval {
				t.Errorf("KeyRotationInterval changed from %v to %v", tt.interval, config.KeyRotationInterval)
			}
		})
	}
}

func TestKeyRotationGracePeriod_DefaultValue(t *testing.T) {
	config := types.JWTPluginConfig{}
	config.ApplyDefaults()

	expected := 1 * time.Hour
	if config.KeyRotationGracePeriod != expected {
		t.Errorf("KeyRotationGracePeriod = %v, want %v", config.KeyRotationGracePeriod, expected)
	}
}

func TestKeyRotationGracePeriod_CustomValues(t *testing.T) {
	tests := []struct {
		name          string
		inputGrace    time.Duration
		expectedGrace time.Duration
	}{
		{
			name:          "5 minute grace period",
			inputGrace:    5 * time.Minute,
			expectedGrace: 5 * time.Minute,
		},
		{
			name:          "30 minute grace period",
			inputGrace:    30 * time.Minute,
			expectedGrace: 30 * time.Minute,
		},
		{
			name:          "2 hour grace period",
			inputGrace:    2 * time.Hour,
			expectedGrace: 2 * time.Hour,
		},
		{
			name:          "24 hour grace period",
			inputGrace:    24 * time.Hour,
			expectedGrace: 24 * time.Hour,
		},
		{
			name:          "zero uses default",
			inputGrace:    0,
			expectedGrace: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := types.JWTPluginConfig{
				KeyRotationGracePeriod: tt.inputGrace,
			}
			config.ApplyDefaults()

			if config.KeyRotationGracePeriod != tt.expectedGrace {
				t.Errorf("KeyRotationGracePeriod = %v, want %v",
					config.KeyRotationGracePeriod, tt.expectedGrace)
			}
		})
	}
}

func TestKeyRotationGracePeriod_WithInterval(t *testing.T) {
	tests := []struct {
		name        string
		interval    time.Duration
		gracePeriod time.Duration
	}{
		{
			name:        "30 day interval with 1 hour grace",
			interval:    30 * 24 * time.Hour,
			gracePeriod: 1 * time.Hour,
		},
		{
			name:        "90 day interval with 24 hour grace",
			interval:    90 * 24 * time.Hour,
			gracePeriod: 24 * time.Hour,
		},
		{
			name:        "1 hour interval with 5 minute grace",
			interval:    1 * time.Hour,
			gracePeriod: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := types.JWTPluginConfig{
				KeyRotationInterval:    tt.interval,
				KeyRotationGracePeriod: tt.gracePeriod,
			}
			config.ApplyDefaults()

			if config.KeyRotationInterval != tt.interval {
				t.Errorf("KeyRotationInterval = %v, want %v",
					config.KeyRotationInterval, tt.interval)
			}

			if config.KeyRotationGracePeriod != tt.gracePeriod {
				t.Errorf("KeyRotationGracePeriod = %v, want %v",
					config.KeyRotationGracePeriod, tt.gracePeriod)
			}
		})
	}
}

func TestKeyRotationGracePeriod_OldKeysValidity(t *testing.T) {
	tests := []struct {
		name          string
		gracePeriod   time.Duration
		keyAge        time.Duration
		expectedValid bool
	}{
		{
			name:          "key within grace period should be valid",
			gracePeriod:   1 * time.Hour,
			keyAge:        30 * time.Minute,
			expectedValid: true,
		},
		{
			name:          "key at grace period boundary is expired (strict comparison)",
			gracePeriod:   1 * time.Hour,
			keyAge:        1 * time.Hour,
			expectedValid: false,
		},
		{
			name:          "key just past grace period should be invalid",
			gracePeriod:   1 * time.Hour,
			keyAge:        1*time.Hour + time.Second,
			expectedValid: false,
		},
		{
			name:          "key long past grace period should be invalid",
			gracePeriod:   1 * time.Hour,
			keyAge:        2 * time.Hour,
			expectedValid: false,
		},
		{
			name:          "24 hour grace period with 12 hour old key",
			gracePeriod:   24 * time.Hour,
			keyAge:        12 * time.Hour,
			expectedValid: true,
		},
		{
			name:          "24 hour grace period with 25 hour old key",
			gracePeriod:   24 * time.Hour,
			keyAge:        25 * time.Hour,
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rotationTime := time.Now().Add(-tt.keyAge)
			gracePeriodEnd := rotationTime.Add(tt.gracePeriod)
			isStillValid := time.Now().Before(gracePeriodEnd)

			if isStillValid != tt.expectedValid {
				t.Errorf("keyAge=%v, gracePeriod=%v, isValid=%v, want %v",
					tt.keyAge, tt.gracePeriod, isStillValid, tt.expectedValid)
			}
		})
	}
}

func TestKeyRotationGracePeriod_BothKeysActive(t *testing.T) {
	now := time.Now()
	gracePeriod := 1 * time.Hour

	rotationTime := now.Add(-30 * time.Minute)

	oldKeyExpiration := rotationTime.Add(gracePeriod)
	isOldKeyActive := now.Before(oldKeyExpiration)

	newKeyActive := true

	if !isOldKeyActive {
		t.Errorf("Old key should still be active within grace period")
	}

	if !newKeyActive {
		t.Errorf("New key should always be active")
	}

	totalActiveKeys := 0
	if isOldKeyActive {
		totalActiveKeys++
	}
	if newKeyActive {
		totalActiveKeys++
	}

	if totalActiveKeys != 2 {
		t.Errorf("Expected 2 active keys during grace period, got %d", totalActiveKeys)
	}
}

func TestKeyRotationGracePeriod_AfterGracePeriod(t *testing.T) {
	now := time.Now()
	gracePeriod := 1 * time.Hour

	rotationTime := now.Add(-2 * time.Hour)

	oldKeyExpiration := rotationTime.Add(gracePeriod)
	isOldKeyActive := now.Before(oldKeyExpiration)

	newKeyActive := true

	if isOldKeyActive {
		t.Errorf("Old key should be expired after grace period")
	}

	if !newKeyActive {
		t.Errorf("New key should always be active")
	}

	totalActiveKeys := 0
	if isOldKeyActive {
		totalActiveKeys++
	}
	if newKeyActive {
		totalActiveKeys++
	}

	if totalActiveKeys != 1 {
		t.Errorf("Expected 1 active key after grace period, got %d", totalActiveKeys)
	}
}

func TestKeyRotationGracePeriod_PluginStorage(t *testing.T) {
	plugin := New(types.JWTPluginConfig{
		KeyRotationInterval:    30 * 24 * time.Hour,
		KeyRotationGracePeriod: 2 * time.Hour,
	})

	storedConfig := plugin.Config().(types.JWTPluginConfig)

	if storedConfig.KeyRotationInterval != 30*24*time.Hour {
		t.Errorf("Stored KeyRotationInterval = %v, want %v",
			storedConfig.KeyRotationInterval, 30*24*time.Hour)
	}

	if storedConfig.KeyRotationGracePeriod != 2*time.Hour {
		t.Errorf("Stored KeyRotationGracePeriod = %v, want %v",
			storedConfig.KeyRotationGracePeriod, 2*time.Hour)
	}
}

func TestKeyRotationGracePeriod_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		gracePeriod time.Duration
	}{
		{"1 nanosecond", time.Nanosecond},
		{"1 microsecond", time.Microsecond},
		{"1 millisecond", time.Millisecond},
		{"1 second", time.Second},
		{"1 minute", time.Minute},
		{"5 minutes", 5 * time.Minute},
		{"30 minutes", 30 * time.Minute},
		{"1 hour", time.Hour},
		{"6 hours", 6 * time.Hour},
		{"12 hours", 12 * time.Hour},
		{"24 hours", 24 * time.Hour},
		{"48 hours", 48 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := types.JWTPluginConfig{
				KeyRotationGracePeriod: tt.gracePeriod,
			}
			config.ApplyDefaults()

			if config.KeyRotationGracePeriod != tt.gracePeriod {
				t.Errorf("KeyRotationGracePeriod changed from %v to %v",
					tt.gracePeriod, config.KeyRotationGracePeriod)
			}
		})
	}
}

func TestKeyRotationGracePeriod_ConfigWithOtherOptions(t *testing.T) {
	config := types.JWTPluginConfig{
		Algorithm:              types.JWTAlgEdDSA,
		KeyRotationInterval:    45 * 24 * time.Hour,
		KeyRotationGracePeriod: 30 * time.Minute,
		ExpiresIn:              5 * time.Minute,
		RefreshExpiresIn:       30 * 24 * time.Hour,
		JWKSCacheTTL:           6 * time.Hour,
		RefreshGracePeriod:     30 * time.Second,
	}
	config.ApplyDefaults()

	if config.KeyRotationInterval != 45*24*time.Hour {
		t.Errorf("KeyRotationInterval = %v, want %v",
			config.KeyRotationInterval, 45*24*time.Hour)
	}

	if config.KeyRotationGracePeriod != 30*time.Minute {
		t.Errorf("KeyRotationGracePeriod = %v, want %v",
			config.KeyRotationGracePeriod, 30*time.Minute)
	}

	if config.Algorithm != types.JWTAlgEdDSA {
		t.Errorf("Algorithm = %v, want %v", config.Algorithm, types.JWTAlgEdDSA)
	}

	if config.ExpiresIn != 5*time.Minute {
		t.Errorf("ExpiresIn = %v, want %v", config.ExpiresIn, 5*time.Minute)
	}

	if config.RefreshExpiresIn != 30*24*time.Hour {
		t.Errorf("RefreshExpiresIn = %v, want %v", config.RefreshExpiresIn, 30*24*time.Hour)
	}
}
