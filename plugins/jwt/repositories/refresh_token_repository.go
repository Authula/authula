package repositories

import (
	"context"

	"github.com/Authula/authula/plugins/jwt/types"
)

type RefreshTokenRepository interface {
	StoreRefreshToken(ctx context.Context, record *types.RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*types.RefreshToken, error)
	RotateRefreshToken(ctx context.Context, oldTokenHash string, newRecord *types.RefreshToken) (bool, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllSessionTokens(ctx context.Context, sessionID string) error
	SetLastReuseAttempt(ctx context.Context, tokenHash string) error
	CleanupExpiredTokens(ctx context.Context) error
}
