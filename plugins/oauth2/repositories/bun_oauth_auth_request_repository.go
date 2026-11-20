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

type BunAuthRequestRepository struct {
	db bun.IDB
}

func NewBunAuthRequestRepository(db bun.IDB) AuthRequestRepository {
	return &BunAuthRequestRepository{db: db}
}

func (r *BunAuthRequestRepository) Create(ctx context.Context, request *types.OAuthAuthRequest) (*types.OAuthAuthRequest, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(request).Exec(ctx); err != nil {
			return fmt.Errorf("insert oauth auth request: %w", err)
		}
		if err := tx.NewSelect().Model(request).WherePK().Scan(ctx); err != nil {
			return fmt.Errorf("reload created oauth auth request: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return request, nil
}

func (r *BunAuthRequestRepository) GetByID(ctx context.Context, id string) (*types.OAuthAuthRequest, error) {
	return r.getBy(ctx, "id", id)
}

func (r *BunAuthRequestRepository) GetByBindingHash(ctx context.Context, bindingHash string) (*types.OAuthAuthRequest, error) {
	return r.getBy(ctx, "binding_hash", bindingHash)
}

func (r *BunAuthRequestRepository) GetByCodeHash(ctx context.Context, codeHash string) (*types.OAuthAuthRequest, error) {
	return r.getBy(ctx, "code_hash", codeHash)
}

func (r *BunAuthRequestRepository) getBy(ctx context.Context, column string, value string) (*types.OAuthAuthRequest, error) {
	request := new(types.OAuthAuthRequest)
	if err := r.db.NewSelect().Model(request).Where("? = ?", bun.Ident(column), value).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s=%s", ErrAuthRequestNotFound, column, value)
		}
		return nil, fmt.Errorf("get oauth auth request by %s: %w", column, err)
	}
	return request, nil
}

func (r *BunAuthRequestRepository) ConsumeCode(ctx context.Context, codeHash string, clientID string, consumedAt time.Time) (*types.OAuthAuthRequest, error) {
	var request *types.OAuthAuthRequest
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		query := tx.NewUpdate().
			Model((*types.OAuthAuthRequest)(nil)).
			Set("code_consumed_at = ?", consumedAt.UTC()).
			Where("code_hash = ? AND code_consumed_at IS NULL AND code_issued_at IS NOT NULL AND expires_at > ?", codeHash, consumedAt.UTC())
		if clientID != "" {
			query = query.Where("client_id = ?", clientID)
		}

		result, err := query.Exec(ctx)
		if err != nil {
			return fmt.Errorf("consume oauth authorization code: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read affected rows for oauth authorization code consumption: %w", err)
		}
		if affected == 0 {
			return nil
		}

		request = new(types.OAuthAuthRequest)
		if err := tx.NewSelect().Model(request).Where("code_hash = ?", codeHash).Scan(ctx); err != nil {
			return fmt.Errorf("reload consumed oauth auth request: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return request, nil
}

func (r *BunAuthRequestRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*types.OAuthAuthRequest)(nil)).
		Where("expires_at < ?", before.UTC()).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete expired oauth auth requests: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read affected rows for expired oauth auth request delete: %w", err)
	}
	return affected, nil
}

func (r *BunAuthRequestRepository) WithTx(tx bun.IDB) AuthRequestRepository {
	return &BunAuthRequestRepository{db: tx}
}
