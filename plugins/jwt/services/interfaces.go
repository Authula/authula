package services

import (
	"context"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/types"
)

// RefreshTokenResponse contains the result of a token refresh operation
type RefreshTokenResponse struct {
	AccessToken  string
	RefreshToken string
}

// JwtService defines the JWT operations
type JwtService interface {
	GenerateTokens(ctx context.Context, userID string, sessionID string) (*types.TokenPair, error)
}

// KeyService manages cryptographic key generation, rotation, and retrieval
type KeyService interface {
	// GenerateKeysIfMissing generates the initial key pair if none exist in the database
	GenerateKeysIfMissing(ctx context.Context) error

	// GetActiveKey retrieves the currently active (non-expired) key
	GetActiveKey(ctx context.Context) (*types.JWKS, error)

	// IsKeyRotationDue returns true if the active key's age exceeds the rotation interval
	IsKeyRotationDue(ctx context.Context, rotationInterval time.Duration) bool

	// RotateKeysIfNeeded rotates keys if they're past the rotation interval
	// gracePeriod specifies how long old keys remain valid after rotation
	// Returns true if rotation occurred, false otherwise
	RotateKeysIfNeeded(ctx context.Context, rotationInterval time.Duration, gracePeriod time.Duration, invalidateCacheFunc func(context.Context) error) (bool, error)
}

// RefreshTokenStorage defines storage operations for refresh tokens
type RefreshTokenStorage interface {
	StoreRefreshToken(ctx context.Context, record *types.RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*types.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	SetLastReuseAttempt(ctx context.Context, tokenHash string) error
	RevokeAllSessionTokens(ctx context.Context, sessionID string) error
}

// RefreshTokenRepository defines data access operations for refresh tokens
type RefreshTokenRepository interface {
	StoreRefreshToken(ctx context.Context, record *types.RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*types.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllSessionTokens(ctx context.Context, sessionID string) error
	SetLastReuseAttempt(ctx context.Context, tokenHash string) error
	CleanupExpiredTokens(ctx context.Context) error
}

// RefreshTokenService handles refresh token operations
type RefreshTokenService interface {
	// RefreshTokens refreshes the access and refresh tokens using the provided refresh token
	RefreshTokens(ctx context.Context, refreshToken string) (*RefreshTokenResponse, error)

	// StoreInitialRefreshToken stores the initial refresh token along with its session ID and expiration time
	StoreInitialRefreshToken(ctx context.Context, refreshToken string, sessionID string, expiresAt time.Time) error
}

// BlacklistService handles token blacklisting/revocation
type BlacklistService interface {
	// BlacklistToken adds a token JTI to the blacklist with TTL
	BlacklistToken(ctx context.Context, jti string, expiresAt time.Time) error

	// IsBlacklisted checks if a token JTI is blacklisted
	IsBlacklisted(ctx context.Context, jti string) (bool, error)

	// BlacklistAllSessionTokens blacklists all tokens for a session
	BlacklistAllSessionTokens(ctx context.Context, sessionID string, expiresAt time.Time) error

	// CleanupExpired removes expired blacklist entries (for non-TTL stores)
	CleanupExpired(ctx context.Context) error
}

// CacheService manages JWKS caching with database fallback
type CacheService interface {
	// GetCachedJWKS retrieves JWKS from cache if available and not expired
	GetCachedJWKS(ctx context.Context) (jwk.Set, error)

	// FetchJWKSFromDatabase loads all non-expired public keys from the database
	FetchJWKSFromDatabase(ctx context.Context) (jwk.Set, error)

	// CacheJWKS stores the JWKS in the cache with the configured TTL
	CacheJWKS(ctx context.Context, set jwk.Set) error

	// InvalidateCache removes the cached JWKS immediately and fetches fresh from DB
	InvalidateCache(ctx context.Context) error

	// GetJWKSWithFallback retrieves JWKS from cache with database fallback
	GetJWKSWithFallback(ctx context.Context) (jwk.Set, error)
}
