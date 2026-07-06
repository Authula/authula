package apikey

import (
	"context"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/api-key/types"
	"github.com/Authula/authula/plugins/api-key/usecases"
)

type API struct {
	useCases *usecases.UseCases
}

func NewAPI(useCases *usecases.UseCases) *API {
	return &API{useCases: useCases}
}

func (a *API) Create(ctx context.Context, actor *models.Actor, req types.CreateApiKeyRequest) (*types.CreateApiKeyResponse, error) {
	return a.useCases.Create(ctx, actor, req)
}

func (a *API) GetByID(ctx context.Context, actor *models.Actor, id string) (*types.ApiKey, error) {
	return a.useCases.GetByID(ctx, actor, id)
}

func (a *API) GetAll(ctx context.Context, actor *models.Actor, req types.GetApiKeysRequest) (*types.GetAllApiKeysResponse, error) {
	return a.useCases.GetAll(ctx, actor, req)
}

func (a *API) Update(ctx context.Context, actor *models.Actor, id string, req types.UpdateApiKeyData) (*types.ApiKey, error) {
	return a.useCases.Update(ctx, actor, id, req)
}

func (a *API) Delete(ctx context.Context, actor *models.Actor, id string) error {
	return a.useCases.Delete(ctx, actor, id)
}

func (a *API) DeleteExpired(ctx context.Context) error {
	return a.useCases.DeleteExpired(ctx)
}

func (a *API) DeleteAllByOwner(ctx context.Context, actor *models.Actor, ownerType string, ownerID string) error {
	return a.useCases.DeleteAllByOwner(ctx, actor, ownerType, ownerID)
}

func (a *API) RecordLastRequest(ctx context.Context, id string, timestamp time.Time) (*types.ApiKey, error) {
	return a.useCases.RecordLastRequest(ctx, id, timestamp)
}

func (a *API) Verify(ctx context.Context, req types.VerifyApiKeyRequest) (*types.VerifyApiKeyResult, error) {
	return a.useCases.Verify(ctx, req)
}
