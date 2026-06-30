package apikey

import (
	"context"

	"github.com/Authula/authula/models"
	apiservices "github.com/Authula/authula/plugins/api-key/services"
	"github.com/Authula/authula/plugins/api-key/types"
)

type API struct {
	service apiservices.ApiKeyService
}

func NewAPI(service apiservices.ApiKeyService) *API {
	return &API{service: service}
}

func (a *API) Create(ctx context.Context, actor *models.Actor, req types.CreateApiKeyRequest) (*types.CreateApiKeyResponse, error) {
	return a.service.Create(ctx, actor, req)
}

func (a *API) GetByID(ctx context.Context, actor *models.Actor, id string) (*types.ApiKey, error) {
	return a.service.GetByID(ctx, actor, id)
}

func (a *API) GetAll(ctx context.Context, actor *models.Actor, req types.GetApiKeysRequest) (*types.GetAllApiKeysResponse, error) {
	return a.service.GetAll(ctx, actor, req)
}

func (a *API) Update(ctx context.Context, actor *models.Actor, id string, req types.UpdateApiKeyData) (*types.ApiKey, error) {
	return a.service.Update(ctx, actor, id, req)
}

func (a *API) Delete(ctx context.Context, actor *models.Actor, id string) error {
	return a.service.Delete(ctx, actor, id)
}

func (a *API) DeleteExpired(ctx context.Context) error {
	return a.service.DeleteExpired(ctx)
}

func (a *API) DeleteAllByOwner(ctx context.Context, actor *models.Actor, ownerType string, ownerID string) error {
	return a.service.DeleteAllByOwner(ctx, actor, ownerType, ownerID)
}

func (a *API) Verify(ctx context.Context, req types.VerifyApiKeyRequest) (*types.VerifyApiKeyResult, error) {
	return a.service.Verify(ctx, req)
}
