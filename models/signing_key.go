package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Signing key purposes. Each purpose has its own independently rotated key so
// that plugin tokens and OIDC tokens never share a signing key.
const (
	SigningKeyPurposeJWTPlugin       = "jwt_plugin"
	SigningKeyPurposeOIDCIDToken     = "oidc_id_token"
	SigningKeyPurposeOIDCAccessToken = "oidc_access_token"
)

// Supported JWT signing algorithms. RS256 is the default because it is the
// OIDC baseline. EdDSA is deliberately not supported.
const (
	SigningKeyAlgorithmRS256 = "RS256"
	SigningKeyAlgorithmES256 = "ES256"
	SigningKeyAlgorithmPS256 = "PS256"
)

const (
	// DefaultSigningKeyAlgorithm is used whenever a purpose has no explicit
	// algorithm.
	DefaultSigningKeyAlgorithm = SigningKeyAlgorithmRS256
	// DefaultRSAKeySize is the minimum RSA modulus size accepted by the service.
	DefaultRSAKeySize = 2048
)

var (
	// ErrSigningKeyNotFound is returned when no usable key matches the lookup.
	ErrSigningKeyNotFound = errors.New("jwt signing key not found")
	// ErrUnsupportedSigningAlgorithm is returned for algorithms outside the
	// supported set.
	ErrUnsupportedSigningAlgorithm = errors.New("unsupported jwt signing algorithm")
	// ErrInvalidSigningKeyPurpose is returned for unknown key purposes.
	ErrInvalidSigningKeyPurpose = errors.New("invalid jwt signing key purpose")
)

// SigningKey is a persisted JWT signing key. Private keys are stored encrypted;
// public keys are stored as PEM-encoded PKIX structures.
type SigningKey struct {
	bun.BaseModel `bun:"table:jwks"`

	ID         string     `json:"id" bun:"column:id,pk"`
	KeyID      string     `json:"key_id" bun:"column:key_id,unique"`
	Algorithm  string     `json:"algorithm" bun:"column:algorithm"`
	KeyPurpose string     `json:"key_purpose" bun:"column:key_purpose"`
	PublicKey  string     `json:"public_key" bun:"column:public_key"`
	PrivateKey string     `json:"-" bun:"column:private_key"`
	IsActive   bool       `json:"is_active" bun:"column:is_active"`
	CreatedAt  time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty" bun:"column:expires_at"`
}

// ValidateSigningKeyAlgorithm reports whether algorithm is part of the
// supported set.
func ValidateSigningKeyAlgorithm(algorithm string) error {
	switch algorithm {
	case SigningKeyAlgorithmRS256, SigningKeyAlgorithmES256, SigningKeyAlgorithmPS256:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedSigningAlgorithm, algorithm)
	}
}

// ValidateSigningKeyPurpose reports whether purpose is a known key purpose.
func ValidateSigningKeyPurpose(purpose string) error {
	switch purpose {
	case SigningKeyPurposeJWTPlugin, SigningKeyPurposeOIDCIDToken, SigningKeyPurposeOIDCAccessToken:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidSigningKeyPurpose, purpose)
	}
}
