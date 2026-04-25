package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/plugins/api-key/repositories"
	"github.com/Authula/authula/plugins/api-key/types"
	rootservices "github.com/Authula/authula/services"
)

type apiKeyService struct {
	config               apiKeyServiceConfig
	userService          rootservices.UserService
	tokenService         rootservices.TokenService
	accessControlService rootservices.AccessControlService
	rateLimiterService   rootservices.RateLimiterService
	organizationService  rootservices.OrganizationService
	apiKeyRepo           repositories.ApiKeyRepository
}

type apiKeyServiceConfig struct {
	allowOrgKeys  bool
	defaultPrefix string
}

func NewApiKeyService(
	pluginConfig types.ApiKeyPluginConfig,
	userService rootservices.UserService,
	tokenService rootservices.TokenService,
	accessControlService rootservices.AccessControlService,
	rateLimiterService rootservices.RateLimiterService,
	organizationService rootservices.OrganizationService,
	apiKeyRepo repositories.ApiKeyRepository,
) ApiKeyService {
	return &apiKeyService{
		config: apiKeyServiceConfig{
			allowOrgKeys:  pluginConfig.AllowOrgKeys,
			defaultPrefix: pluginConfig.DefaultPrefix,
		},
		userService:          userService,
		tokenService:         tokenService,
		accessControlService: accessControlService,
		rateLimiterService:   rateLimiterService,
		organizationService:  organizationService,
		apiKeyRepo:           apiKeyRepo,
	}
}

func (s *apiKeyService) Create(ctx context.Context, req types.CreateApiKeyRequest) (*types.CreateApiKeyResponse, error) {
	if req.OwnerType != types.OwnerTypeUser && req.OwnerType != types.OwnerTypeOrganization {
		return nil, fmt.Errorf("%w: owner_type must be 'user' or 'organization'", internalerrors.ErrBadRequest)
	}

	if err := s.validatePermissions(ctx, req.Permissions); err != nil {
		return nil, err
	}

	switch req.OwnerType {
	case types.OwnerTypeUser:
		user, err := s.userService.GetByID(ctx, req.OwnerID)
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
		exists, err := s.organizationService.ExistsByID(ctx, req.OwnerID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("%w: organization not found", internalerrors.ErrNotFound)
		}
	}

	prefix := s.config.defaultPrefix
	if req.Prefix != nil {
		prefix = *req.Prefix
	}

	rawKey, err := s.tokenService.Generate()
	if err != nil {
		return nil, err
	}

	apiKeyValue := prefix + rawKey
	keyHash := s.tokenService.Hash(apiKeyValue)

	start := ""
	if len(rawKey) > 4 {
		start = rawKey[:4]
	}

	last := ""
	if len(rawKey) > 4 {
		last = rawKey[len(rawKey)-4:]
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rateLimitEnabled := false
	if req.RateLimitEnabled != nil {
		rateLimitEnabled = *req.RateLimitEnabled
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		expiresAt = req.ExpiresAt
	}

	permissions := req.Permissions

	metadata := make(map[string]any)
	if len(req.Metadata) > 0 {
		metadata = req.Metadata
	}

	apiKey := &types.ApiKey{
		ID:               util.GenerateUUID(),
		KeyHash:          keyHash,
		Name:             req.Name,
		OwnerType:        req.OwnerType,
		OwnerID:          req.OwnerID,
		Prefix:           &prefix,
		Start:            start,
		Last:             last,
		Enabled:          enabled,
		RateLimitEnabled: rateLimitEnabled,
		ExpiresAt:        expiresAt,
		Permissions:      permissions,
		Metadata:         metadata,
	}

	created, err := s.apiKeyRepo.Create(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	return &types.CreateApiKeyResponse{
		RawApiKey: apiKeyValue,
		ApiKey:    created,
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

	page := req.Page
	if page <= 0 {
		page = 1
	}

	items, total, err := s.apiKeyRepo.GetAll(ctx, req.OwnerType, req.OwnerID, page, limit)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*types.ApiKey{}
	}

	return &types.GetAllApiKeysResponse{
		Items: items,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *apiKeyService) Update(ctx context.Context, id string, req types.UpdateApiKeyData) (*types.ApiKey, error) {
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
	if req.RateLimitEnabled != nil {
		apiKey.RateLimitEnabled = *req.RateLimitEnabled
	}
	if req.LastRequestedAt != nil {
		apiKey.LastRequestedAt = req.LastRequestedAt
	}
	if req.ExpiresAt != nil {
		apiKey.ExpiresAt = req.ExpiresAt
	}
	if err := s.validatePermissions(ctx, req.Permissions); err != nil {
		return nil, err
	}
	if len(req.Permissions) > 0 {
		apiKey.Permissions = req.Permissions
	}
	if len(req.Metadata) > 0 {
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
	if s.rateLimiterService != nil && apiKey.RateLimitEnabled {
		if err := s.rateLimiterService.DeleteRule(ctx, apiKey.KeyHash); err != nil {
			return err
		}
	}
	return s.apiKeyRepo.Delete(ctx, id)
}

func (s *apiKeyService) DeleteExpired(ctx context.Context) error {
	return s.apiKeyRepo.DeleteExpired(ctx)
}

func (s *apiKeyService) DeleteAllByOwner(ctx context.Context, ownerType string, ownerID string) error {
	return s.apiKeyRepo.DeleteAllByOwner(ctx, ownerType, ownerID)
}

func (s *apiKeyService) Verify(ctx context.Context, req types.VerifyApiKeyRequest) (*types.VerifyApiKeyResult, error) {
	keyHash := s.tokenService.Hash(req.Key)
	apiKey, err := s.apiKeyRepo.GetByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	if apiKey == nil || !apiKey.Enabled || apiKey.ExpiresAt != nil && time.Now().UTC().After(*apiKey.ExpiresAt) {
		return &types.VerifyApiKeyResult{Valid: false}, nil
	}

	return &types.VerifyApiKeyResult{Valid: true, ApiKey: apiKey}, nil
}

func (s *apiKeyService) ValidatePermissionKeys(ctx context.Context, permissionKeys []string) error {
	return s.validatePermissions(ctx, permissionKeys)
}

func (s *apiKeyService) validatePermissions(ctx context.Context, permissionKeys []string) error {
	if len(permissionKeys) == 0 {
		return nil
	}

	if err := s.accessControlService.ValidatePermissionKeys(ctx, permissionKeys); err != nil {
		if errors.Is(err, internalerrors.ErrNotFound) {
			return fmt.Errorf("%w: one or more permissions were not found", internalerrors.ErrNotFound)
		}
		return err
	}

	return nil
}
