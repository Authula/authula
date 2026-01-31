package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/types"
)

type refreshTokenRepositoryImpl struct {
	db bun.IDB
}

func NewRefreshTokenRepository(db bun.IDB) RefreshTokenRepository {
	return &refreshTokenRepositoryImpl{db: db}
}

func (r *refreshTokenRepositoryImpl) StoreRefreshToken(ctx context.Context, record *types.RefreshToken) error {
	if _, err := r.db.NewInsert().Model(record).Exec(ctx); err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepositoryImpl) GetRefreshToken(ctx context.Context, tokenHash string) (*types.RefreshToken, error) {
	var record types.RefreshToken
	err := r.db.NewSelect().Model(&record).Where("token_hash = ?", tokenHash).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	return &record, nil
}

func (r *refreshTokenRepositoryImpl) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model(&types.RefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Set("is_revoked = ?, revoked_at = ?", true, now).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepositoryImpl) RevokeAllSessionTokens(ctx context.Context, sessionID string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model(&types.RefreshToken{}).
		Where("session_id = ?", sessionID).
		Set("is_revoked = ?, revoked_at = ?", true, now).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to revoke session tokens: %w", err)
	}
	return nil
}

func (r *refreshTokenRepositoryImpl) SetLastReuseAttempt(ctx context.Context, tokenHash string) error {
	_, err := r.db.NewUpdate().
		Model(&types.RefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Where("last_reuse_attempt IS NULL").
		Set("last_reuse_attempt = ?", time.Now()).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update last reuse attempt: %w", err)
	}
	return nil
}

func (r *refreshTokenRepositoryImpl) CleanupExpiredTokens(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model(&types.RefreshToken{}).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}
	return nil
}
