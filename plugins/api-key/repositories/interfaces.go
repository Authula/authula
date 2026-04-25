package repositories

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/api-key/types"
)

type ApiKeyRepository interface {
	Create(ctx context.Context, apiKey *types.ApiKey) (*types.ApiKey, error)
	GetAll(ctx context.Context, ownerType *string, referenceID *string, page int, limit int) ([]*types.ApiKey, int, error)
	GetByID(ctx context.Context, id string) (*types.ApiKey, error)
	GetByKeyHash(ctx context.Context, keyHash string) (*types.ApiKey, error)
	Update(ctx context.Context, apiKey *types.ApiKey) (*types.ApiKey, error)
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) error
	DeleteAllByOwner(ctx context.Context, ownerType string, referenceID string) error
	WithTx(tx bun.IDB) ApiKeyRepository
}
