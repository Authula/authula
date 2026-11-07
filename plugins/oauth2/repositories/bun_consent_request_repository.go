package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/oauth2/types"
)

type BunConsentRequestRepository struct {
	db bun.IDB
}

func NewBunConsentRequestRepository(db bun.IDB) ConsentRequestRepository {
	return &BunConsentRequestRepository{db: db}
}

func (r *BunConsentRequestRepository) Create(ctx context.Context, request *types.OAuthConsentRequest) (*types.OAuthConsentRequest, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(request).Exec(ctx); err != nil {
			return err
		}
		return tx.NewSelect().Model(request).WherePK().Scan(ctx)
	})
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (r *BunConsentRequestRepository) GetByRequestHash(ctx context.Context, requestHash string) (*types.OAuthConsentRequest, error) {
	request := new(types.OAuthConsentRequest)
	if err := r.db.NewSelect().Model(request).Where("request_hash = ?", requestHash).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return request, nil
}

func (r *BunConsentRequestRepository) Consume(ctx context.Context, requestHash string, consumedAt time.Time) (*types.OAuthConsentRequest, error) {
	var request *types.OAuthConsentRequest
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		result, err := tx.NewUpdate().
			Model((*types.OAuthConsentRequest)(nil)).
			Set("consumed_at = ?", consumedAt.UTC()).
			Where("request_hash = ? AND consumed_at IS NULL AND expires_at > ?", requestHash, consumedAt.UTC()).
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

		request = new(types.OAuthConsentRequest)
		return tx.NewSelect().Model(request).Where("request_hash = ?", requestHash).Scan(ctx)
	})
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (r *BunConsentRequestRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*types.OAuthConsentRequest)(nil)).
		Where("expires_at < ?", before.UTC()).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (r *BunConsentRequestRepository) WithTx(tx bun.IDB) ConsentRequestRepository {
	return &BunConsentRequestRepository{db: tx}
}
