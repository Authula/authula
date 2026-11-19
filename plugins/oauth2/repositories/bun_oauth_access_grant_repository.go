package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/oauth2/types"
)

type BunAccessGrantRepository struct {
	db bun.IDB
}

func NewBunAccessGrantRepository(db bun.IDB) AccessGrantRepository {
	return &BunAccessGrantRepository{db: db}
}

func (r *BunAccessGrantRepository) Create(ctx context.Context, grant *types.OAuthAccessGrant) (*types.OAuthAccessGrant, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(grant).Exec(ctx); err != nil {
			return err
		}
		return tx.NewSelect().Model(grant).WherePK().Scan(ctx)
	})
	if err != nil {
		return nil, err
	}
	return grant, nil
}

func (r *BunAccessGrantRepository) GetByID(ctx context.Context, id string) (*types.OAuthAccessGrant, error) {
	return r.getBy(ctx, "id", id)
}

func (r *BunAccessGrantRepository) GetByJTI(ctx context.Context, jti string) (*types.OAuthAccessGrant, error) {
	return r.getBy(ctx, "jti", jti)
}

func (r *BunAccessGrantRepository) getBy(ctx context.Context, column string, value string) (*types.OAuthAccessGrant, error) {
	grant := new(types.OAuthAccessGrant)
	if err := r.db.NewSelect().Model(grant).Where("? = ?", bun.Ident(column), value).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return grant, nil
}

func (r *BunAccessGrantRepository) Revoke(ctx context.Context, jti string, revokedAt time.Time) (bool, error) {
	result, err := r.db.NewUpdate().
		Model((*types.OAuthAccessGrant)(nil)).
		Set("revoked_at = ?", revokedAt.UTC()).
		Where("jti = ? AND revoked_at IS NULL", jti).
		Exec(ctx)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *BunAccessGrantRepository) RevokeFamily(ctx context.Context, familyID string, revokedAt time.Time) (int64, error) {
	result, err := r.db.NewUpdate().
		Model((*types.OAuthAccessGrant)(nil)).
		Set("revoked_at = ?", revokedAt.UTC()).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *BunAccessGrantRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*types.OAuthAccessGrant)(nil)).
		Where("expires_at < ?", before.UTC()).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *BunAccessGrantRepository) WithTx(tx bun.IDB) AccessGrantRepository {
	return &BunAccessGrantRepository{db: tx}
}
