package services

import (
	"context"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/Authula/authula/plugins/jwt/types"
)

type TokenService interface {
	GenerateUserToken(ctx context.Context, userID string, sessionID string, extraClaims map[string]any) (*types.TokenPair, error)
	GenerateMachineToken(ctx context.Context, clientID string, organizationID string, scopes []string) (*types.TokenPair, error)
	ExtractClaims(ctx context.Context, token string) (map[string]any, error)
}

type KeyService interface {
	GenerateKeysIfMissing(ctx context.Context) error
	GetActiveKey(ctx context.Context) (*types.JWKS, error)
	IsKeyRotationDue(ctx context.Context, rotationInterval time.Duration) bool
	// RotateKeysIfNeeded rotates keys if they're past the rotation interval
	// gracePeriod specifies how long old keys remain valid after rotation
	// Returns true if rotation occurred, false otherwise
	RotateKeysIfNeeded(ctx context.Context, rotationInterval time.Duration, gracePeriod time.Duration, invalidateCacheFunc func(context.Context) error) (bool, error)
}

type RefreshTokenRepository interface {
	StoreRefreshToken(ctx context.Context, record *types.RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*types.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllSessionTokens(ctx context.Context, sessionID string) error
	SetLastReuseAttempt(ctx context.Context, tokenHash string) error
	CleanupExpiredTokens(ctx context.Context) error
}

type RefreshTokenService interface {
	RefreshTokens(ctx context.Context, refreshToken string) (*types.RefreshTokenResponse, error)
	StoreInitialRefreshToken(ctx context.Context, refreshToken string, sessionID string, expiresAt time.Time) error
}

type BlacklistService interface {
	BlacklistToken(ctx context.Context, jti string, expiresAt time.Time) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
	BlacklistAllSessionTokens(ctx context.Context, sessionID string, expiresAt time.Time) error
	CleanupExpired(ctx context.Context) error
}

type CacheService interface {
	GetCachedJWKS(ctx context.Context) (jwk.Set, error)
	FetchJWKSFromDatabase(ctx context.Context) (jwk.Set, error)
	CacheJWKS(ctx context.Context, set jwk.Set) error
	InvalidateCache(ctx context.Context) error
	GetJWKSWithFallback(ctx context.Context) (jwk.Set, error)
}
