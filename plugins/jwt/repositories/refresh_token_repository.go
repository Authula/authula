package repositories

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/types"
)

// RefreshTokenRepository provides data access for refresh token records
type RefreshTokenRepository interface {
	// StoreRefreshToken saves a refresh token record
	StoreRefreshToken(ctx context.Context, record *types.RefreshToken) error

	// GetRefreshToken retrieves a refresh token by hash
	GetRefreshToken(ctx context.Context, tokenHash string) (*types.RefreshToken, error)

	// RevokeRefreshToken marks a token as revoked
	RevokeRefreshToken(ctx context.Context, tokenHash string) error

	// RevokeAllSessionTokens revokes all refresh tokens for a session
	RevokeAllSessionTokens(ctx context.Context, sessionID string) error

	// SetLastReuseAttempt updates the last reuse attempt timestamp for a token
	SetLastReuseAttempt(ctx context.Context, tokenHash string) error

	// CleanupExpiredTokens removes expired refresh token records
	CleanupExpiredTokens(ctx context.Context) error
}
