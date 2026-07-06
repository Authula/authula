package usecases

import (
	"context"
	"time"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
	apiconstants "github.com/Authula/authula/plugins/api-key/constants"
	apiservices "github.com/Authula/authula/plugins/api-key/services"
	"github.com/Authula/authula/plugins/api-key/types"
	rootservices "github.com/Authula/authula/services"
)

type UseCases struct {
	service    apiservices.ApiKeyService
	authorizer rootservices.Authorizer
}

func NewUseCases(service apiservices.ApiKeyService, authorizer rootservices.Authorizer) *UseCases {
	return &UseCases{service: service, authorizer: authorizer}
}

func (u *UseCases) Create(ctx context.Context, actor *models.Actor, req types.CreateApiKeyRequest) (*types.CreateApiKeyResponse, error) {
	if req.OwnerType == types.OwnerTypeOrganization {
		if err := u.authorizer.AuthorizeScope(ctx, actor, apiconstants.OrgApiKeyCreate); err != nil {
			return nil, err
		}
	}
	return u.service.Create(ctx, actor, req)
}

func (u *UseCases) GetByID(ctx context.Context, actor *models.Actor, id string) (*types.ApiKey, error) {
	apiKey, err := u.service.GetByID(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return apiKey, nil
	}
	if apiKey.OwnerType == types.OwnerTypeOrganization {
		if err := u.authorizer.AuthorizeScope(ctx, actor, apiconstants.OrgApiKeyRead); err != nil {
			return nil, err
		}
	}
	return apiKey, nil
}

func (u *UseCases) GetAll(ctx context.Context, actor *models.Actor, req types.GetApiKeysRequest) (*types.GetAllApiKeysResponse, error) {
	if req.OwnerType != nil && *req.OwnerType == types.OwnerTypeOrganization {
		if err := u.authorizer.AuthorizeScope(ctx, actor, apiconstants.OrgApiKeyList); err != nil {
			return nil, err
		}
	}
	return u.service.GetAll(ctx, actor, req)
}

func (u *UseCases) Update(ctx context.Context, actor *models.Actor, id string, req types.UpdateApiKeyData) (*types.ApiKey, error) {
	apiKey, err := u.service.GetByID(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, internalerrors.ErrNotFound
	}
	if apiKey.OwnerType == types.OwnerTypeOrganization {
		if err := u.authorizer.AuthorizeScope(ctx, actor, apiconstants.OrgApiKeyUpdate); err != nil {
			return nil, err
		}
	}
	return u.service.Update(ctx, actor, id, req)
}

func (u *UseCases) Delete(ctx context.Context, actor *models.Actor, id string) error {
	apiKey, err := u.service.GetByID(ctx, actor, id)
	if err != nil {
		return err
	}
	if apiKey == nil {
		return nil
	}
	if apiKey.OwnerType == types.OwnerTypeOrganization {
		if err := u.authorizer.AuthorizeScope(ctx, actor, apiconstants.OrgApiKeyDelete); err != nil {
			return err
		}
	}
	return u.service.Delete(ctx, actor, id)
}

func (u *UseCases) DeleteExpired(ctx context.Context) error {
	return u.service.DeleteExpired(ctx)
}

func (u *UseCases) DeleteAllByOwner(ctx context.Context, actor *models.Actor, ownerType string, ownerID string) error {
	if ownerType == types.OwnerTypeOrganization {
		if err := u.authorizer.AuthorizeScope(ctx, actor, apiconstants.OrgApiKeyDelete); err != nil {
			return err
		}
	}
	return u.service.DeleteAllByOwner(ctx, actor, ownerType, ownerID)
}

func (u *UseCases) RecordLastRequest(ctx context.Context, id string, timestamp time.Time) (*types.ApiKey, error) {
	return u.service.RecordLastRequest(ctx, id, timestamp)
}

func (u *UseCases) Verify(ctx context.Context, req types.VerifyApiKeyRequest) (*types.VerifyApiKeyResult, error) {
	return u.service.Verify(ctx, req)
}
