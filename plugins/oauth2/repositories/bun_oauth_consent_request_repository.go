package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
			return fmt.Errorf("insert oauth consent request: %w", err)
		}
		if err := tx.NewSelect().Model(request).WherePK().Scan(ctx); err != nil {
			return fmt.Errorf("reload created oauth consent request: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return request, nil
}

func (r *BunConsentRequestRepository) GetByID(ctx context.Context, id string) (*types.OAuthConsentRequest, error) {
	return r.getBy(ctx, "id", id)
}

func (r *BunConsentRequestRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*types.OAuthConsentRequest, error) {
	return r.getBy(ctx, "token_hash", tokenHash)
}

func (r *BunConsentRequestRepository) GetByCompletionCodeHash(ctx context.Context, completionCodeHash string) (*types.OAuthConsentRequest, error) {
	return r.getBy(ctx, "completion_code_hash", completionCodeHash)
}

func (r *BunConsentRequestRepository) getBy(ctx context.Context, column string, value string) (*types.OAuthConsentRequest, error) {
	request := new(types.OAuthConsentRequest)
	if err := r.db.NewSelect().Model(request).Where("? = ?", bun.Ident(column), value).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s=%s", ErrConsentRequestNotFound, column, value)
		}
		return nil, fmt.Errorf("get oauth consent request by %s: %w", column, err)
	}
	return request, nil
}

func (r *BunConsentRequestRepository) Consume(ctx context.Context, tokenHash string, consumedAt time.Time) (*types.OAuthConsentRequest, error) {
	var request *types.OAuthConsentRequest
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		result, err := tx.NewUpdate().
			Model((*types.OAuthConsentRequest)(nil)).
			Set("consumed_at = ?", consumedAt.UTC()).
			Where("token_hash = ? AND consumed_at IS NULL AND expires_at > ?", tokenHash, consumedAt.UTC()).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("consume oauth consent request: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read affected rows for oauth consent request consumption: %w", err)
		}
		if affected == 0 {
			return nil
		}

		request = new(types.OAuthConsentRequest)
		if err := tx.NewSelect().Model(request).Where("token_hash = ?", tokenHash).Scan(ctx); err != nil {
			return fmt.Errorf("reload consumed oauth consent request: %w", err)
		}
		return nil
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
		return 0, fmt.Errorf("delete expired oauth consent requests: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read affected rows for expired oauth consent request delete: %w", err)
	}
	return affected, nil
}

func (r *BunConsentRequestRepository) WithTx(tx bun.IDB) ConsentRequestRepository {
	return &BunConsentRequestRepository{db: tx}
}
