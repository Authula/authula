package apikey

import (
	"context"

	apiservices "github.com/Authula/authula/plugins/api-key/services"
	"github.com/Authula/authula/plugins/api-key/types"
)

type API struct {
	service apiservices.ApiKeyService
}

func NewAPI(service apiservices.ApiKeyService) *API {
	return &API{service: service}
}

func (a *API) Create(ctx context.Context, req types.CreateApiKeyRequest) (*types.CreateApiKeyResponse, error) {
	return a.service.Create(ctx, req)
}

func (a *API) Verify(ctx context.Context, req types.VerifyApiKeyRequest) (*types.VerifyApiKeyResult, error) {
	return a.service.Verify(ctx, req)
}

func (a *API) GetByID(ctx context.Context, id string) (*types.ApiKey, error) {
	return a.service.GetByID(ctx, id)
}

func (a *API) GetAll(ctx context.Context, req types.GetApiKeysRequest) (*types.GetAllApiKeysResponse, error) {
	return a.service.GetAll(ctx, req)
}

func (a *API) Update(ctx context.Context, id string, req types.UpdateApiKeyRequest) (*types.ApiKey, error) {
	return a.service.Update(ctx, id, req)
}

func (a *API) Delete(ctx context.Context, id string) error {
	return a.service.Delete(ctx, id)
}

func (a *API) DeleteExpired(ctx context.Context) error {
	return a.service.DeleteExpired(ctx)
}

func (a *API) DeleteAllByOwner(ctx context.Context, ownerType string, referenceID string) error {
	return a.service.DeleteAllByOwner(ctx, ownerType, referenceID)
}
