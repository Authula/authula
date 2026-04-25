package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/api-key/types"
)

type BunApiKeyRepository struct {
	db bun.IDB
}

func NewBunApiKeyRepository(db bun.IDB) ApiKeyRepository {
	return &BunApiKeyRepository{db: db}
}

func (r *BunApiKeyRepository) Create(ctx context.Context, apiKey *types.ApiKey) (*types.ApiKey, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(apiKey).Exec(ctx)
		if err != nil {
			return err
		}
		return tx.NewSelect().Model(apiKey).WherePK().Scan(ctx)
	})
	return apiKey, err
}

func (r *BunApiKeyRepository) GetAll(ctx context.Context, ownerType *string, referenceID *string, page int, limit int) ([]*types.ApiKey, int, error) {
	var apiKeys []*types.ApiKey
	q := r.db.NewSelect().Model(&apiKeys)

	if ownerType != nil {
		q = q.Where("owner_type = ?", *ownerType)
	}
	if referenceID != nil {
		q = q.Where("reference_id = ?", *referenceID)
	}

	total, err := q.
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return apiKeys, total, nil
}

func (r *BunApiKeyRepository) GetByID(ctx context.Context, id string) (*types.ApiKey, error) {
	apiKey := new(types.ApiKey)
	err := r.db.NewSelect().Model(apiKey).Where("id = ?", id).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return apiKey, err
}

func (r *BunApiKeyRepository) GetByKeyHash(ctx context.Context, keyHash string) (*types.ApiKey, error) {
	apiKey := new(types.ApiKey)
	err := r.db.NewSelect().Model(apiKey).Where("key_hash = ?", keyHash).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return apiKey, err
}

func (r *BunApiKeyRepository) Update(ctx context.Context, apiKey *types.ApiKey) (*types.ApiKey, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model(apiKey).WherePK().Exec(ctx)
		if err != nil {
			return err
		}
		return tx.NewSelect().Model(apiKey).WherePK().Scan(ctx)
	})

	return apiKey, err
}

func (r *BunApiKeyRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model(&types.ApiKey{}).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *BunApiKeyRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.NewDelete().Model(&types.ApiKey{}).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now().UTC()).
		Exec(ctx)

	return err
}

func (r *BunApiKeyRepository) DeleteAllByOwner(ctx context.Context, ownerType string, referenceID string) error {
	_, err := r.db.NewDelete().Model(&types.ApiKey{}).
		Where("owner_type = ? AND reference_id = ?", ownerType, referenceID).
		Exec(ctx)

	return err
}

func (r *BunApiKeyRepository) WithTx(tx bun.IDB) ApiKeyRepository {
	return &BunApiKeyRepository{db: tx}
}
