package types

import (
	"time"
)

// JWTPluginConfig configures the JWKS-based JWT plugin
type JWTPluginConfig struct {
	Enabled                bool          `json:"enabled" toml:"enabled"`
	Algorithm              JWTAlgorithm  `json:"algorithm" toml:"algorithm"`                                 // EdDSA (default), RS256, PS256, ES256, ES512
	KeyRotationInterval    time.Duration `json:"key_rotation_interval" toml:"key_rotation_interval"`         // Default: 30 days
	KeyRotationGracePeriod time.Duration `json:"key_rotation_grace_period" toml:"key_rotation_grace_period"` // Grace period for old key validity after rotation, default: 1 hour
	ExpiresIn              time.Duration `json:"expires_in" toml:"expires_in"`                               // Access token TTL
	RefreshExpiresIn       time.Duration `json:"refresh_expires_in" toml:"refresh_expires_in"`               // Refresh token TTL
	JWKSCacheTTL           time.Duration `json:"jwks_cache_ttl" toml:"jwks_cache_ttl"`                       // Cache TTL for JWKS, default 24 hours
	RefreshGracePeriod     time.Duration `json:"refresh_grace_period" toml:"refresh_grace_period"`           // Grace period for refresh token reuse, default 10s
}

// ApplyDefaults returns sensible defaults for the JWT plugin
func (c *JWTPluginConfig) ApplyDefaults() {
	if c.Algorithm == "" {
		c.Algorithm = JWTAlgEdDSA
	}
	if c.KeyRotationInterval == 0 {
		c.KeyRotationInterval = 30 * 24 * time.Hour
	}
	if c.KeyRotationGracePeriod == 0 {
		c.KeyRotationGracePeriod = 1 * time.Hour
	}
	if c.ExpiresIn == 0 {
		c.ExpiresIn = 15 * time.Minute
	}
	if c.RefreshExpiresIn == 0 {
		c.RefreshExpiresIn = 7 * 24 * time.Hour
	}
	if c.JWKSCacheTTL == 0 {
		c.JWKSCacheTTL = 24 * time.Hour
	}
	if c.RefreshGracePeriod == 0 {
		c.RefreshGracePeriod = 10 * time.Second
	}
}

// NormalizeAlgorithm normalizes and validates the algorithm string. Use when
// parsing config or on update to catch legacy or unsupported values.
func (c *JWTPluginConfig) NormalizeAlgorithm() error {
	if c.Algorithm == "" {
		c.Algorithm = JWTAlgEdDSA
		return nil
	}
	parsed, err := ParseAlgorithm(string(c.Algorithm))
	if err != nil {
		return err
	}
	if err := ValidateAlgorithm(parsed); err != nil {
		return err
	}
	c.Algorithm = parsed
	return nil
}
