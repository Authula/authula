package services

import (
	"context"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/types"
)

type ApiKeyService interface {
	Create(ctx context.Context, actor *models.Actor, req types.CreateApiKeyRequest) (*types.CreateApiKeyResponse, error)
	GetByID(ctx context.Context, actor *models.Actor, id string) (*types.ApiKey, error)
	GetAll(ctx context.Context, actor *models.Actor, req types.GetApiKeysRequest) (*types.GetAllApiKeysResponse, error)
	Update(ctx context.Context, actor *models.Actor, id string, req types.UpdateApiKeyData) (*types.ApiKey, error)
	Delete(ctx context.Context, actor *models.Actor, id string) error
	RecordLastRequest(ctx context.Context, id string, timestamp time.Time) (*types.ApiKey, error)
	DeleteExpired(ctx context.Context) error
	DeleteAllByOwner(ctx context.Context, actor *models.Actor, ownerType string, ownerID string) error
	Verify(ctx context.Context, req types.VerifyApiKeyRequest) (*types.VerifyApiKeyResult, error)
	ValidatePermissionKeys(ctx context.Context, permissionKeys []string) error
}
