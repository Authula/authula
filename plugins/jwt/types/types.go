package types

import (
	"errors"
	"time"

	"github.com/uptrace/bun"
)

type JWTAlgorithm string

const (
	JWTAlgEdDSA  JWTAlgorithm = "eddsa"
	JWTAlgRS256  JWTAlgorithm = "rs256"
	JWTAlgPS256  JWTAlgorithm = "ps256"
	JWTAlgES256  JWTAlgorithm = "es256"
	JWTAlgES512  JWTAlgorithm = "es512"
	JWTAlgECDHES JWTAlgorithm = "ecdh-es"
)

func (a JWTAlgorithm) String() string {
	return string(a)
}

type JWTTokenType string

const (
	JWTTokenTypeAccess  JWTTokenType = "access_token"
	JWTTokenTypeRefresh JWTTokenType = "refresh_token"
)

func (t JWTTokenType) String() string {
	return string(t)
}

// ParseAlgorithm parses a string into an Algorithm, accepting only canonical names (case-insensitive input)
func ParseAlgorithm(s string) (JWTAlgorithm, error) {
	switch s {
	case "eddsa":
		return JWTAlgEdDSA, nil
	case "rs256":
		return JWTAlgRS256, nil
	case "ps256":
		return JWTAlgPS256, nil
	case "es256":
		return JWTAlgES256, nil
	case "es512":
		return JWTAlgES512, nil
	case "ecdh-es":
		return JWTAlgECDHES, nil
	default:
		return "", errors.New("unsupported jwt algorithm")
	}
}

// ValidateAlgorithm enforces that the algorithm can be used for JWT signing
func ValidateAlgorithm(alg JWTAlgorithm) error {
	switch alg {
	case JWTAlgEdDSA, JWTAlgRS256, JWTAlgPS256, JWTAlgES256, JWTAlgES512:
		return nil
	case JWTAlgECDHES:
		return errors.New("ECDH-ES cannot be used for JWT signing")
	default:
		return errors.New("unsupported JWT algorithm")
	}
}

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

// JWKS represents a cryptographic key pair for signing and verification
type JWKS struct {
	bun.BaseModel `bun:"table:jwks"`

	ID         string     `json:"id" bun:"column:id,pk"`
	PublicKey  string     `json:"public_key" bun:"column:public_key"`
	PrivateKey string     `json:"private_key" bun:"column:private_key"`
	ExpiresAt  *time.Time `json:"expires_at" bun:"column:expires_at"`
	CreatedAt  time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
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

// RefreshToken represents a stored refresh token in the database
type RefreshToken struct {
	bun.BaseModel `bun:"table:refresh_tokens"`

	ID               string     `json:"id" bun:"column:id,pk"`
	SessionID        string     `json:"session_id" bun:"column:session_id"`
	TokenHash        string     `json:"token_hash" bun:"column:token_hash"`
	ExpiresAt        time.Time  `json:"expires_at" bun:"column:expires_at"`
	IsRevoked        bool       `json:"is_revoked" bun:"column:is_revoked"`
	RevokedAt        *time.Time `json:"revoked_at" bun:"column:revoked_at"`
	LastReuseAttempt *time.Time `json:"last_reuse_attempt" bun:"column:last_reuse_attempt"`
	CreatedAt        time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
}
