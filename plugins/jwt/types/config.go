package types

import (
	"time"
)

type JWTPluginConfig struct {
	Enabled                bool          `json:"enabled" toml:"enabled"`
	KeyRotationInterval    time.Duration `json:"key_rotation_interval" toml:"key_rotation_interval"`         // Default: 30 days
	KeyRotationGracePeriod time.Duration `json:"key_rotation_grace_period" toml:"key_rotation_grace_period"` // Grace period for old key validity after rotation, default: 1 hour
	ExpiresIn              time.Duration `json:"expires_in" toml:"expires_in"`                               // Access token TTL
	RefreshExpiresIn       time.Duration `json:"refresh_expires_in" toml:"refresh_expires_in"`               // Refresh token TTL
	JWKSCacheTTL           time.Duration `json:"jwks_cache_ttl" toml:"jwks_cache_ttl"`                       // Cache TTL for JWKS, default 24 hours
	RefreshGracePeriod     time.Duration `json:"refresh_grace_period" toml:"refresh_grace_period"`           // Grace period for refresh token reuse, default 10s
}

func (c *JWTPluginConfig) ApplyDefaults() {
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
