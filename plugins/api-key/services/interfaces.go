package services

import (
	"context"

	"github.com/Authula/authula/plugins/api-key/types"
)

type ApiKeyService interface {
	Create(ctx context.Context, req types.CreateApiKeyRequest) (*types.CreateApiKeyResponse, error)
	GetByID(ctx context.Context, id string) (*types.ApiKey, error)
	GetAll(ctx context.Context, req types.GetApiKeysRequest) (*types.GetAllApiKeysResponse, error)
	Update(ctx context.Context, id string, req types.UpdateApiKeyRequest) (*types.ApiKey, error)
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) error
	DeleteAllByOwner(ctx context.Context, ownerType string, referenceID string) error
	Verify(ctx context.Context, req types.VerifyApiKeyRequest) (*types.VerifyApiKeyResult, error)
}
