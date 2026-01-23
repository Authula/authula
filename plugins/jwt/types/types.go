package types

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

// Algorithm represents supported JWT/JWKS algorithms
type Algorithm string

const (
	AlgEdDSA  Algorithm = "EdDSA"
	AlgRS256  Algorithm = "RS256"
	AlgPS256  Algorithm = "PS256"
	AlgES256  Algorithm = "ES256"
	AlgES512  Algorithm = "ES512"
	AlgECDHES Algorithm = "ECDH-ES"
)

func (a Algorithm) String() string {
	return string(a)
}

type TokenType string

const (
	TokenTypeAccess  TokenType = "access_token"
	TokenTypeRefresh TokenType = "refresh_token"
)

func (t TokenType) String() string {
	return string(t)
}

// ParseAlgorithm parses a string into an Algorithm, accepting only canonical names (case-insensitive input)
func ParseAlgorithm(s string) (Algorithm, error) {
	switch s {
	case "eddsa":
		return AlgEdDSA, nil
	case "rs256":
		return AlgRS256, nil
	case "ps256":
		return AlgPS256, nil
	case "es256":
		return AlgES256, nil
	case "es512":
		return AlgES512, nil
	case "ecdh-es":
		return AlgECDHES, nil
	default:
		return "", errors.New("unsupported jwt algorithm")
	}
}

// ValidateAlgorithm enforces that the algorithm can be used for JWT signing
func ValidateAlgorithm(alg Algorithm) error {
	switch alg {
	case AlgEdDSA, AlgRS256, AlgPS256, AlgES256, AlgES512:
		return nil
	case AlgECDHES:
		return errors.New("ECDH-ES cannot be used for JWT signing")
	default:
		return errors.New("unsupported JWT algorithm")
	}
}

// JWTPluginConfig configures the JWKS-based JWT plugin
type JWTPluginConfig struct {
	Enabled                bool          `json:"enabled" toml:"enabled"`
	Algorithm              Algorithm     `json:"algorithm" toml:"algorithm"`                                 // EdDSA (default), RS256, PS256, ES256, ES512
	KeyRotationInterval    time.Duration `json:"key_rotation_interval" toml:"key_rotation_interval"`         // Default: 30 days
	KeyRotationGracePeriod time.Duration `json:"key_rotation_grace_period" toml:"key_rotation_grace_period"` // Grace period for old key validity after rotation, default: 1 hour
	ExpiresIn              time.Duration `json:"expires_in" toml:"expires_in"`                               // Access token TTL
	RefreshExpiresIn       time.Duration `json:"refresh_expires_in" toml:"refresh_expires_in"`               // Refresh token TTL
	JWKSCacheTTL           time.Duration `json:"jwks_cache_ttl" toml:"jwks_cache_ttl"`                       // Cache TTL for JWKS, default 24 hours
	RefreshGracePeriod     time.Duration `json:"refresh_grace_period" toml:"refresh_grace_period"`           // Grace period for refresh token reuse, default 10s
	DisableIPLogging       bool          `json:"disable_ip_logging" toml:"disable_ip_logging"`               // Disable IP address logging for GDPR compliance
}

// ApplyDefaults returns sensible defaults for the JWT plugin
func (c *JWTPluginConfig) ApplyDefaults() {
	if c.Algorithm == "" {
		c.Algorithm = AlgEdDSA
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
		c.Algorithm = AlgEdDSA
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

// JWKS represents a cryptographic key pair for signing and verification
type JWKS struct {
	bun.BaseModel `bun:"table:jwks"`

	ID         string     `json:"id" bun:"column:id,pk,notnull"`
	PublicKey  string     `json:"public_key" bun:"column:public_key,notnull"`
	PrivateKey string     `json:"private_key" bun:"column:private_key,notnull"`
	CreatedAt  time.Time  `json:"created_at" bun:"column:created_at,nullzero,notnull,default:current_timestamp"`
	ExpiresAt  *time.Time `json:"expires_at" bun:"column:expires_at,nullzero"`
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
	Type      string `json:"type"` // "access_token" or "refresh_token"
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

	ID               string     `json:"id" bun:"column:id,pk,notnull"`
	SessionID        string     `json:"session_id" bun:"column:session_id,notnull"`
	TokenHash        string     `json:"token_hash" bun:"column:token_hash,notnull,unique"`
	ExpiresAt        time.Time  `json:"expires_at" bun:"column:expires_at,notnull"`
	IsRevoked        bool       `json:"is_revoked" bun:"column:is_revoked,notnull,default:false"`
	RevokedAt        *time.Time `json:"revoked_at" bun:"column:revoked_at,nullzero"`
	LastReuseAttempt *time.Time `json:"last_reuse_attempt" bun:"column:last_reuse_attempt,nullzero"`
	CreatedAt        time.Time  `json:"created_at" bun:"column:created_at,nullzero,notnull,default:current_timestamp"`
}

var _ bun.BeforeAppendModelHook = (*RefreshTokenRecord)(nil)

func (s *RefreshTokenRecord) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		s.CreatedAt = time.Now()
	}
	return nil
}
