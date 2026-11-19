package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/oauth2/types"
)

type BunRefreshTokenRepository struct {
	db bun.IDB
}

func NewBunRefreshTokenRepository(db bun.IDB) RefreshTokenRepository {
	return &BunRefreshTokenRepository{db: db}
}

func (r *BunRefreshTokenRepository) Create(ctx context.Context, token *types.OAuthRefreshToken) (*types.OAuthRefreshToken, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(token).Exec(ctx); err != nil {
			return err
		}
		return tx.NewSelect().Model(token).WherePK().Scan(ctx)
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (r *BunRefreshTokenRepository) GetByID(ctx context.Context, id string) (*types.OAuthRefreshToken, error) {
	return r.getBy(ctx, "id", id)
}

func (r *BunRefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*types.OAuthRefreshToken, error) {
	return r.getBy(ctx, "token_hash", tokenHash)
}

func (r *BunRefreshTokenRepository) getBy(ctx context.Context, column string, value string) (*types.OAuthRefreshToken, error) {
	token := new(types.OAuthRefreshToken)
	if err := r.db.NewSelect().Model(token).Where("? = ?", bun.Ident(column), value).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return token, nil
}

// Rotate consumes the old refresh token and stores its replacement in one
// transaction. The conditional update guarantees that only one caller can
// consume a given token, even under concurrent requests.
func (r *BunRefreshTokenRepository) Rotate(ctx context.Context, oldTokenHash string, replacement *types.OAuthRefreshToken, consumedAt time.Time) (*types.OAuthRefreshToken, error) {
	if replacement == nil {
		return nil, errors.New("replacement refresh token is required")
	}

	var consumed *types.OAuthRefreshToken
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		result, err := tx.NewUpdate().
			Model((*types.OAuthRefreshToken)(nil)).
			Set("consumed_at = ?", consumedAt.UTC()).
			Where("token_hash = ? AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at > ?", oldTokenHash, consumedAt.UTC()).
			Exec(ctx)
		if err != nil {
			return err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return nil
		}

		if _, err := tx.NewInsert().Model(replacement).Exec(ctx); err != nil {
			return err
		}

		consumed = new(types.OAuthRefreshToken)
		return tx.NewSelect().Model(consumed).Where("token_hash = ?", oldTokenHash).Scan(ctx)
	})
	if err != nil {
		return nil, err
	}
	return consumed, nil
}

func (r *BunRefreshTokenRepository) RevokeFamily(ctx context.Context, familyID string, revokedAt time.Time) (int64, error) {
	result, err := r.db.NewUpdate().
		Model((*types.OAuthRefreshToken)(nil)).
		Set("revoked_at = ?", revokedAt.UTC()).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *BunRefreshTokenRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*types.OAuthRefreshToken)(nil)).
		Where("expires_at < ?", before.UTC()).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *BunRefreshTokenRepository) WithTx(tx bun.IDB) RefreshTokenRepository {
	return &BunRefreshTokenRepository{db: tx}
}
