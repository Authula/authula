package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/oauth2/types"
)

type BunAuthorizationCodeRepository struct {
	db bun.IDB
}

func NewBunAuthorizationCodeRepository(db bun.IDB) AuthorizationCodeRepository {
	return &BunAuthorizationCodeRepository{db: db}
}

func (r *BunAuthorizationCodeRepository) Create(ctx context.Context, code *types.OAuthAuthorizationCode) (*types.OAuthAuthorizationCode, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(code).Exec(ctx); err != nil {
			return err
		}
		return tx.NewSelect().Model(code).WherePK().Scan(ctx)
	})
	if err != nil {
		return nil, err
	}

	return code, nil
}

func (r *BunAuthorizationCodeRepository) GetByCodeHash(ctx context.Context, codeHash string) (*types.OAuthAuthorizationCode, error) {
	code := new(types.OAuthAuthorizationCode)
	if err := r.db.NewSelect().Model(code).Where("code_hash = ?", codeHash).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return code, nil
}

func (r *BunAuthorizationCodeRepository) Consume(ctx context.Context, codeHash string, clientID string, consumedAt time.Time) (*types.OAuthAuthorizationCode, error) {
	var code *types.OAuthAuthorizationCode
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		query := tx.NewUpdate().
			Model((*types.OAuthAuthorizationCode)(nil)).
			Set("consumed_at = ?", consumedAt.UTC()).
			Where("code_hash = ? AND consumed_at IS NULL AND expires_at > ?", codeHash, consumedAt.UTC())
		if clientID != "" {
			query = query.Where("client_id = ?", clientID)
		}

		result, err := query.Exec(ctx)
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

		code = new(types.OAuthAuthorizationCode)
		return tx.NewSelect().Model(code).Where("code_hash = ?", codeHash).Scan(ctx)
	})
	if err != nil {
		return nil, err
	}

	return code, nil
}

func (r *BunAuthorizationCodeRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*types.OAuthAuthorizationCode)(nil)).
		Where("expires_at < ?", before.UTC()).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (r *BunAuthorizationCodeRepository) WithTx(tx bun.IDB) AuthorizationCodeRepository {
	return &BunAuthorizationCodeRepository{db: tx}
}
