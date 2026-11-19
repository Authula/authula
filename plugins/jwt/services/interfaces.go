package services

import (
	"context"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/types"
)

type TokenServiceConfig struct {
	ExpiresIn        time.Duration
	RefreshExpiresIn time.Duration
	Issuer           string
	Audience         string
}

type TokenService interface {
	GenerateUserToken(ctx context.Context, userID string, sessionID string, extraClaims map[string]any) (*types.TokenPair, error)
	GenerateMachineToken(ctx context.Context, clientID string, organizationID string, scopes []string) (*types.TokenPair, error)
	ValidateRefreshToken(ctx context.Context, token string) (map[string]any, error)
}

// SigningKeyService is the JWT plugin's internal contract for purpose-scoped,
// algorithm-aware signing keys. The plugin exposes the same operations publicly
// through services.JWTService.
type SigningKeyService interface {
	GetActiveKey(ctx context.Context, purpose, algorithm string) (*models.SigningKey, error)
	GetKeyByID(ctx context.Context, kid string) (*models.SigningKey, error)
	GetPublicKeys(ctx context.Context, purpose string) ([]models.SigningKey, error)
	GenerateKey(ctx context.Context, purpose, algorithm string) error
	RotateKey(ctx context.Context, purpose, algorithm string) error
	RotateKeyWithGrace(ctx context.Context, purpose, algorithm string, gracePeriod time.Duration) error
}

type KeyService interface {
	GenerateKeysIfMissing(ctx context.Context) error
	GetActiveKey(ctx context.Context) (*models.SigningKey, error)
	GetPublicKeys(ctx context.Context) ([]models.SigningKey, error)
	IsKeyRotationDue(ctx context.Context, rotationInterval time.Duration) bool
	// RotateKeysIfNeeded rotates keys if they're past the rotation interval
	// gracePeriod specifies how long old keys remain valid after rotation
	// Returns true if rotation occurred, false otherwise
	RotateKeysIfNeeded(ctx context.Context, rotationInterval time.Duration, gracePeriod time.Duration, invalidateCacheFunc func(context.Context) error) (bool, error)
}

type RefreshTokenRepository interface {
	StoreRefreshToken(ctx context.Context, record *types.RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*types.RefreshToken, error)
	RotateRefreshToken(ctx context.Context, oldTokenHash string, newRecord *types.RefreshToken) (bool, error)
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
