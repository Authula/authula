package types

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

// JWTPluginConfig configures the JWKS-based JWT plugin
type JWTPluginConfig struct {
	Enabled             bool          `json:"enabled" toml:"enabled"`
	Algorithm           string        `json:"algorithm" toml:"algorithm"`                         // ed25519 (default), rs256, rs2048, rs4096, es256, es384, es512
	KeyRotationInterval time.Duration `json:"key_rotation_interval" toml:"key_rotation_interval"` // Default: 90 days
	ExpiresIn           time.Duration `json:"expires_in" toml:"expires_in"`                       // Access token TTL
	RefreshExpiresIn    time.Duration `json:"refresh_expires_in" toml:"refresh_expires_in"`       // Refresh token TTL
	JWKSCacheTTL        time.Duration `json:"jwks_cache_ttl" toml:"jwks_cache_ttl"`               // Cache TTL for JWKS, default 24 hours
	RefreshGracePeriod  time.Duration `json:"refresh_grace_period" toml:"refresh_grace_period"`   // Grace period for refresh token reuse, default 10s
	DisableIPLogging    bool          `json:"disable_ip_logging" toml:"disable_ip_logging"`       // Disable IP address logging for GDPR compliance
}

// ApplyDefaults returns sensible defaults for the JWT plugin
func (c *JWTPluginConfig) ApplyDefaults() {
	if c.Algorithm == "" {
		c.Algorithm = "ed25519"
	}
	if c.KeyRotationInterval == 0 {
		c.KeyRotationInterval = 90 * 24 * time.Hour
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

// JWKS represents a cryptographic key pair for signing and verification
type JWKS struct {
	bun.BaseModel `bun:"table:jwks"`

	ID         string     `json:"id" bun:",pk,notnull"`
	PublicKey  string     `json:"public_key" bun:",notnull"`
	PrivateKey string     `json:"private_key" bun:",notnull"`
	CreatedAt  time.Time  `json:"created_at" bun:",nullzero,notnull,default:current_timestamp"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

var _ bun.BeforeAppendModelHook = (*JWKS)(nil)

func (s *JWKS) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		s.CreatedAt = time.Now()
	}
	return nil
}

// Claims represents standard JWT claims
type Claims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"sid"`
	Type      string `json:"type"` // "access" or "refresh"
	Sub       string `json:"sub"`
	Iss       string `json:"iss"`
	Aud       string `json:"aud"`
	Exp       int64  `json:"exp"`
	Iat       int64  `json:"iat"`
	Nbf       int64  `json:"nbf,omitempty"`
	Jti       string `json:"jti"`
}

// TokenPair holds both access and refresh tokens
type TokenPair struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    time.Duration `json:"expires_in"`
	TokenType    string        `json:"token_type"`
}

// RefreshTokenRecord stores refresh token metadata for rotation tracking
type RefreshTokenRecord struct {
	bun.BaseModel `bun:"table:refresh_tokens"`

	ID               string     `json:"id" bun:"pk,notnull"`
	SessionID        string     `json:"session_id" bun:"notnull"`
	TokenHash        string     `json:"token_hash" bun:"notnull,unique"` // SHA256 hash of refresh token
	ExpiresAt        time.Time  `json:"expires_at" bun:",notnull"`
	IsRevoked        bool       `json:"is_revoked" bun:",notnull,default:false"`
	RevokedAt        *time.Time `json:"revoked_at"`
	LastReuseAttempt *time.Time `json:"last_reuse_attempt" bun:""` // Tracks first reuse attempt within grace period
	CreatedAt        time.Time  `json:"created_at" bun:",nullzero,notnull,default:current_timestamp"`
}

var _ bun.BeforeAppendModelHook = (*RefreshTokenRecord)(nil)

func (s *RefreshTokenRecord) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		s.CreatedAt = time.Now()
	}
	return nil
}
