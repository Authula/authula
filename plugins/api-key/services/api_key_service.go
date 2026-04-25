package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/plugins/api-key/repositories"
	"github.com/Authula/authula/plugins/api-key/types"
	rootservices "github.com/Authula/authula/services"
)

type apiKeyService struct {
	config              apiKeyServiceConfig
	userService         rootservices.UserService
	tokenService        rootservices.TokenService
	organizationService rootservices.OrganizationService
	apiKeyRepo          repositories.ApiKeyRepository
}

type apiKeyServiceConfig struct {
	allowOrgKeys  bool
	defaultPrefix string
}

func NewApiKeyService(
	pluginConfig types.ApiKeyPluginConfig,
	userService rootservices.UserService,
	tokenService rootservices.TokenService,
	organizationService rootservices.OrganizationService,
	apiKeyRepo repositories.ApiKeyRepository,
) ApiKeyService {
	return &apiKeyService{
		config: apiKeyServiceConfig{
			allowOrgKeys:  pluginConfig.AllowOrgKeys,
			defaultPrefix: pluginConfig.DefaultPrefix,
		},
		userService:         userService,
		tokenService:        tokenService,
		organizationService: organizationService,
		apiKeyRepo:          apiKeyRepo,
	}
}

func (s *apiKeyService) Create(ctx context.Context, req types.CreateApiKeyRequest) (*types.CreateApiKeyResponse, error) {
	if req.OwnerType != types.OwnerTypeUser && req.OwnerType != types.OwnerTypeOrganization {
		return nil, fmt.Errorf("%w: owner_type must be 'user' or 'organization'", internalerrors.ErrBadRequest)
	}

	switch req.OwnerType {
	case types.OwnerTypeUser:
		user, err := s.userService.GetByID(ctx, req.ReferenceID)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, fmt.Errorf("%w: user not found", internalerrors.ErrNotFound)
		}
	case types.OwnerTypeOrganization:
		if !s.config.allowOrgKeys {
			return nil, fmt.Errorf("%w: organization-owned keys are not enabled", internalerrors.ErrForbidden)
		}
		if s.organizationService == nil {
			return nil, fmt.Errorf("%w: organization service is not available", internalerrors.ErrUnprocessableEntity)
		}
		exists, err := s.organizationService.ExistsByID(ctx, req.ReferenceID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("%w: organization not found", internalerrors.ErrNotFound)
		}
	}

	rawKey, err := s.tokenService.Generate()
	if err != nil {
		return nil, err
	}

	keyHash := s.tokenService.Hash(rawKey)

	prefix := s.config.defaultPrefix
	if req.Prefix != nil {
		prefix = *req.Prefix
	}

	start := rawKey
	if len(start) > 8 {
		start = start[:8]
	}
	if prefix != "" {
		start = prefix + start
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rateLimitEnabled := false
	if req.RateLimitEnabled != nil {
		rateLimitEnabled = *req.RateLimitEnabled
	}

	rateLimitTimeWindow := 0
	if req.RateLimitTimeWindow != nil {
		rateLimitTimeWindow = *req.RateLimitTimeWindow
	}

	rateLimitMaxRequests := 0
	if req.RateLimitMaxRequests != nil {
		rateLimitMaxRequests = *req.RateLimitMaxRequests
	}

	requestsRemaining := 0
	if req.RequestsRemaining != nil {
		requestsRemaining = *req.RequestsRemaining
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		expiresAt = req.ExpiresAt
	}

	permissions := req.Permissions

	metadata := json.RawMessage("{}")
	if len(req.Metadata) > 0 && string(req.Metadata) != "null" {
		metadata = req.Metadata
	}

	apiKey := &types.ApiKey{
		ID:                   util.GenerateUUID(),
		KeyHash:              keyHash,
		Name:                 req.Name,
		OwnerType:            req.OwnerType,
		ReferenceID:          req.ReferenceID,
		Start:                start,
		Prefix:               &prefix,
		Enabled:              enabled,
		RateLimitEnabled:     rateLimitEnabled,
		RateLimitTimeWindow:  &rateLimitTimeWindow,
		RateLimitMaxRequests: &rateLimitMaxRequests,
		RequestsRemaining:    &requestsRemaining,
		ExpiresAt:            expiresAt,
		Permissions:          permissions,
		Metadata:             metadata,
	}

	created, err := s.apiKeyRepo.Create(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	return &types.CreateApiKeyResponse{
		ApiKey:    created,
		RawApiKey: rawKey,
	}, nil
}

func (s *apiKeyService) GetByID(ctx context.Context, id string) (*types.ApiKey, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, internalerrors.ErrNotFound
	}
	return apiKey, nil
}

func (s *apiKeyService) GetAll(ctx context.Context, req types.GetApiKeysRequest) (*types.GetAllApiKeysResponse, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	page := max(req.Page, 0)

	items, total, err := s.apiKeyRepo.GetAll(ctx, req.OwnerType, req.ReferenceID, page, limit)
	if err != nil {
		return nil, err
	}

	return &types.GetAllApiKeysResponse{
		Items: items,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *apiKeyService) Update(ctx context.Context, id string, req types.UpdateApiKeyRequest) (*types.ApiKey, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, internalerrors.ErrNotFound
	}

	if req.Name != nil {
		apiKey.Name = *req.Name
	}
	if req.Enabled != nil {
		apiKey.Enabled = *req.Enabled
	}
	if req.ExpiresAt != nil {
		apiKey.ExpiresAt = req.ExpiresAt
	}
	if req.RateLimitEnabled != nil {
		apiKey.RateLimitEnabled = *req.RateLimitEnabled
	}
	if req.RateLimitTimeWindow != nil {
		apiKey.RateLimitTimeWindow = req.RateLimitTimeWindow
	}
	if req.RateLimitMaxRequests != nil {
		apiKey.RateLimitMaxRequests = req.RateLimitMaxRequests
	}
	if req.RequestsRemaining != nil {
		apiKey.RequestsRemaining = req.RequestsRemaining
	}
	if len(req.Permissions) > 0 {
		apiKey.Permissions = req.Permissions
	}
	if len(req.Metadata) > 0 && string(req.Metadata) != "null" {
		apiKey.Metadata = req.Metadata
	}

	return s.apiKeyRepo.Update(ctx, apiKey)
}

func (s *apiKeyService) Delete(ctx context.Context, id string) error {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if apiKey == nil {
		return internalerrors.ErrNotFound
	}
	return s.apiKeyRepo.Delete(ctx, id)
}

func (s *apiKeyService) DeleteExpired(ctx context.Context) error {
	return s.apiKeyRepo.DeleteExpired(ctx)
}

func (s *apiKeyService) DeleteAllByOwner(ctx context.Context, ownerType string, referenceID string) error {
	return s.apiKeyRepo.DeleteAllByOwner(ctx, ownerType, referenceID)
}

func (s *apiKeyService) Verify(ctx context.Context, req types.VerifyApiKeyRequest) (*types.VerifyApiKeyResult, error) {
	keyHash := s.tokenService.Hash(req.Key)
	apiKey, err := s.apiKeyRepo.GetByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	if apiKey == nil ||
		!apiKey.Enabled ||
		apiKey.ExpiresAt != nil && time.Now().UTC().After(*apiKey.ExpiresAt) ||
		apiKey.RequestsRemaining != nil && *apiKey.RequestsRemaining <= 0 {
		return &types.VerifyApiKeyResult{Valid: false}, nil
	}

	return &types.VerifyApiKeyResult{Valid: true, ApiKey: apiKey}, nil
}
