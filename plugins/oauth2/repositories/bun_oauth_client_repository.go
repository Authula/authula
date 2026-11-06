package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/oauth2/types"
)

const oauthClientListMaxLimit = 100

type BunOAuthClientRepository struct {
	db bun.IDB
}

func NewBunOAuthClientRepository(db bun.IDB) OAuthClientRepository {
	return &BunOAuthClientRepository{db: db}
}

func (r *BunOAuthClientRepository) Create(ctx context.Context, client *types.OAuthClient) (*types.OAuthClient, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(client).Exec(ctx); err != nil {
			return err
		}
		return tx.NewSelect().Model(client).WherePK().Scan(ctx)
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (r *BunOAuthClientRepository) GetByID(ctx context.Context, id string) (*types.OAuthClient, error) {
	return r.getBy(ctx, "id", id, false)
}

func (r *BunOAuthClientRepository) GetByIDIncludingDeleted(ctx context.Context, id string) (*types.OAuthClient, error) {
	return r.getBy(ctx, "id", id, true)
}

func (r *BunOAuthClientRepository) GetByClientID(ctx context.Context, clientID string) (*types.OAuthClient, error) {
	return r.getBy(ctx, "client_id", clientID, false)
}

func (r *BunOAuthClientRepository) GetByClientIDIncludingDeleted(ctx context.Context, clientID string) (*types.OAuthClient, error) {
	return r.getBy(ctx, "client_id", clientID, true)
}

func (r *BunOAuthClientRepository) getBy(ctx context.Context, column string, value string, includeDeleted bool) (*types.OAuthClient, error) {
	client := new(types.OAuthClient)
	query := r.db.NewSelect().Model(client).Where("? = ?", bun.Ident(column), value)
	if !includeDeleted {
		query = query.Where("deleted_at IS NULL")
	}
	if err := query.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return client, nil
}

func (r *BunOAuthClientRepository) List(ctx context.Context, filter OAuthClientListFilter) ([]*types.OAuthClient, error) {
	clients := make([]*types.OAuthClient, 0)
	query := r.db.NewSelect().Model(&clients)

	if filter.OwnerType != "" {
		query = query.Where("owner_type = ?", filter.OwnerType)
	}
	if filter.OwnerID != "" {
		query = query.Where("owner_id = ?", filter.OwnerID)
	}
	if filter.OrganizationID != nil {
		query = query.Where("organization_id = ?", *filter.OrganizationID)
	}
	if !filter.IncludeDeleted {
		query = query.Where("deleted_at IS NULL")
	}
	if filter.Cursor != nil {
		query = query.Where(
			"(created_at < ? OR (created_at = ? AND id < ?))",
			filter.Cursor.CreatedAt,
			filter.Cursor.CreatedAt,
			filter.Cursor.ID,
		)
	}

	limit := filter.Limit
	if limit <= 0 || limit > oauthClientListMaxLimit {
		limit = oauthClientListMaxLimit
	}

	if err := query.OrderExpr("created_at DESC, id DESC").Limit(limit).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*types.OAuthClient{}, nil
		}
		return nil, err
	}

	return clients, nil
}

func (r *BunOAuthClientRepository) Update(ctx context.Context, client *types.OAuthClient, expectedUpdatedAt *time.Time) (*types.OAuthClient, error) {
	updated := false
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		query := tx.NewUpdate().Model(client).WherePK()
		if expectedUpdatedAt != nil {
			query = query.Where("updated_at = ?", expectedUpdatedAt.UTC())
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

		updated = true
		return tx.NewSelect().Model(client).WherePK().Scan(ctx)
	})
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, ErrOAuthClientConflict
	}

	return client, nil
}

func (r *BunOAuthClientRepository) SoftDelete(ctx context.Context, id string, deletedAt time.Time) (bool, error) {
	result, err := r.db.NewUpdate().
		Model((*types.OAuthClient)(nil)).
		Set("deleted_at = ?", deletedAt.UTC()).
		Set("is_active = ?", false).
		Set("updated_at = ?", deletedAt.UTC()).
		Where("id = ? AND deleted_at IS NULL", id).
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

func (r *BunOAuthClientRepository) WithTx(tx bun.IDB) OAuthClientRepository {
	return &BunOAuthClientRepository{db: tx}
}
