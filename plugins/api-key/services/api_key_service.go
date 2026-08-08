package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	apikeyconstants "github.com/Authula/authula/plugins/api-key/constants"
	"github.com/Authula/authula/plugins/api-key/repositories"
	"github.com/Authula/authula/plugins/api-key/types"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type apiKeyService struct {
	config               apiKeyServiceConfig
	userService          rootservices.UserService
	tokenService         rootservices.TokenService
	accessControlService rootservices.AccessControlService
	rateLimiterService   rootservices.RateLimiterService
	organizationService  OrganizationLookupService
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
	organizationService OrganizationLookupService,
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

func (s *apiKeyService) Create(ctx context.Context, actor *models.Actor, req types.CreateApiKeyRequest) (*types.CreateApiKeyResponse, error) {
	if err := s.authorizeCreate(ctx, actor, req); err != nil {
		return nil, err
	}

	if err := s.validatePermissions(ctx, req.Permissions); err != nil {
		return nil, err
	}

	if req.OwnerType != types.OwnerTypeUser && req.OwnerType != types.OwnerTypeOrganization {
		return nil, fmt.Errorf("%w: owner_type must be 'user' or 'organization'", coreerrors.ErrBadRequest)
	}

	switch req.OwnerType {
	case types.OwnerTypeUser:
		user, err := s.userService.GetByID(ctx, req.OwnerID)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, fmt.Errorf("%w: user not found", coreerrors.ErrNotFound)
		}
	case types.OwnerTypeOrganization:
		if !s.config.allowOrgKeys {
			return nil, fmt.Errorf("%w: organization-owned keys are not enabled", coreerrors.ErrForbidden)
		}
		if s.organizationService == nil {
			return nil, fmt.Errorf("%w: organization service is not available", coreerrors.ErrUnprocessableEntity)
		}
		exists, err := s.organizationService.ExistsByID(ctx, req.OwnerID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("%w: organization not found", coreerrors.ErrNotFound)
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

	if rateLimitEnabled && req.RateLimitTimeWindow != nil && req.RateLimitMaxRequests != nil && s.rateLimiterService != nil {
		if err := s.rateLimiterService.SetRule(ctx, created.KeyHash, time.Duration(*req.RateLimitTimeWindow)*time.Second, *req.RateLimitMaxRequests); err != nil {
			return nil, err
		}
	}

	return &types.CreateApiKeyResponse{
		RawApiKey: apiKeyValue,
		ApiKey:    created,
	}, nil
}

func (s *apiKeyService) authorizeCreate(ctx context.Context, actor *models.Actor, req types.CreateApiKeyRequest) error {
	switch req.OwnerType {
	case types.OwnerTypeUser:
		if req.OwnerID != "" && req.OwnerID != actor.ID {
			return fmt.Errorf("%w: cannot create API key for another user", coreerrors.ErrForbidden)
		}
		if err := s.validatePermissionsSubset(actor.Scopes, req.Permissions); err != nil {
			return err
		}
	case types.OwnerTypeOrganization:
		if !s.config.allowOrgKeys {
			return fmt.Errorf("%w: organization-owned keys are not enabled", coreerrors.ErrForbidden)
		}
		perms, err := s.authorizeOrgKeyAccess(ctx, actor, req.OwnerID, apikeyconstants.OrgApiKeyCreate)
		if err != nil {
			return err
		}
		if err := s.validatePermissionsSubset(perms, req.Permissions); err != nil {
			return err
		}
	}
	return nil
}

// authorizeOrgKeyAccess verifies that the actor is a member of the owning organization
// and holds the required organization api-key permission via their membership role.
// It returns the caller's permissions within the organization.
func (s *apiKeyService) authorizeOrgKeyAccess(ctx context.Context, actor *models.Actor, orgID string, requiredPermission string) ([]string, error) {
	if actor == nil || actor.ID == "" {
		return nil, coreerrors.ErrUnauthorized
	}
	if s.organizationService == nil {
		return nil, fmt.Errorf("%w: organization service is not available", coreerrors.ErrUnprocessableEntity)
	}
	perms, err := s.organizationService.GetUserPermissionsInOrganization(ctx, actor.ID, orgID)
	if err != nil {
		return nil, err
	}
	if !hasPermission(perms, requiredPermission) {
		return nil, fmt.Errorf("%w: permission %q is not within your organization permissions", coreerrors.ErrForbidden, requiredPermission)
	}
	return perms, nil
}

func (s *apiKeyService) GetByID(ctx context.Context, actor *models.Actor, id string) (*types.ApiKey, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, coreerrors.ErrNotFound
	}

	switch apiKey.OwnerType {
	case types.OwnerTypeUser:
		if apiKey.OwnerID != actor.ID {
			return nil, fmt.Errorf("%w: you do not have access to this API key", coreerrors.ErrForbidden)
		}
	case types.OwnerTypeOrganization:
		if _, err := s.authorizeOrgKeyAccess(ctx, actor, apiKey.OwnerID, apikeyconstants.OrgApiKeyRead); err != nil {
			return nil, err
		}
	}

	return apiKey, nil
}

func (s *apiKeyService) GetAll(ctx context.Context, actor *models.Actor, req types.GetApiKeysRequest) (*types.GetAllApiKeysResponse, error) {
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

	switch {
	case req.OwnerType == nil || *req.OwnerType == types.OwnerTypeUser:
		ownerID := actor.ID
		req.OwnerID = &ownerID
		ownerType := types.OwnerTypeUser
		req.OwnerType = &ownerType
	case *req.OwnerType == types.OwnerTypeOrganization:
		if req.OwnerID == nil || *req.OwnerID == "" {
			return nil, fmt.Errorf("%w: owner_id is required for organization keys", coreerrors.ErrBadRequest)
		}
		if _, err := s.authorizeOrgKeyAccess(ctx, actor, *req.OwnerID, apikeyconstants.OrgApiKeyList); err != nil {
			return nil, err
		}
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

func (s *apiKeyService) Update(ctx context.Context, actor *models.Actor, id string, req types.UpdateApiKeyData) (*types.ApiKey, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, coreerrors.ErrNotFound
	}

	switch apiKey.OwnerType {
	case types.OwnerTypeUser:
		if apiKey.OwnerID != actor.ID {
			return nil, fmt.Errorf("%w: you do not have access to this API key", coreerrors.ErrForbidden)
		}
		if len(req.Permissions) > 0 {
			if err := s.validatePermissionsSubset(actor.Scopes, req.Permissions); err != nil {
				return nil, err
			}
		}
	case types.OwnerTypeOrganization:
		perms, err := s.authorizeOrgKeyAccess(ctx, actor, apiKey.OwnerID, apikeyconstants.OrgApiKeyUpdate)
		if err != nil {
			return nil, err
		}
		if len(req.Permissions) > 0 {
			if err := s.validatePermissionsSubset(perms, req.Permissions); err != nil {
				return nil, err
			}
		}
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

	if req.RateLimitEnabled != nil && s.rateLimiterService != nil {
		if *req.RateLimitEnabled && req.RateLimitTimeWindow != nil && req.RateLimitMaxRequests != nil {
			if err := s.rateLimiterService.SetRule(ctx, apiKey.KeyHash, time.Duration(*req.RateLimitTimeWindow)*time.Second, *req.RateLimitMaxRequests); err != nil {
				return nil, err
			}
		} else if !*req.RateLimitEnabled {
			if err := s.rateLimiterService.DeleteRule(ctx, apiKey.KeyHash); err != nil {
				return nil, err
			}
		}
	}

	return s.apiKeyRepo.Update(ctx, apiKey)
}

func (s *apiKeyService) Delete(ctx context.Context, actor *models.Actor, id string) error {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if apiKey == nil {
		return coreerrors.ErrNotFound
	}

	switch apiKey.OwnerType {
	case types.OwnerTypeUser:
		if apiKey.OwnerID != actor.ID {
			return fmt.Errorf("%w: you do not have access to this API key", coreerrors.ErrForbidden)
		}
	case types.OwnerTypeOrganization:
		if _, err := s.authorizeOrgKeyAccess(ctx, actor, apiKey.OwnerID, apikeyconstants.OrgApiKeyDelete); err != nil {
			return err
		}
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

func (s *apiKeyService) DeleteAllByOwner(ctx context.Context, actor *models.Actor, ownerType string, ownerID string) error {
	switch ownerType {
	case types.OwnerTypeUser:
		if ownerID != actor.ID {
			return fmt.Errorf("%w: you cannot delete API keys for another user", coreerrors.ErrForbidden)
		}
	case types.OwnerTypeOrganization:
		if _, err := s.authorizeOrgKeyAccess(ctx, actor, ownerID, apikeyconstants.OrgApiKeyDelete); err != nil {
			return err
		}
	}
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

func (s *apiKeyService) RecordLastRequest(ctx context.Context, id string, timestamp time.Time) (*types.ApiKey, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, coreerrors.ErrNotFound
	}
	apiKey.LastRequestedAt = &timestamp
	return s.apiKeyRepo.Update(ctx, apiKey)
}

func (s *apiKeyService) ValidatePermissionKeys(ctx context.Context, permissionKeys []string) error {
	return s.validatePermissions(ctx, permissionKeys)
}

func (s *apiKeyService) validatePermissionsSubset(actorScopes []string, keyPermissions []string) error {
	if len(keyPermissions) == 0 {
		return nil
	}
	for _, perm := range keyPermissions {
		if !hasPermission(actorScopes, perm) {
			return fmt.Errorf("%w: permission %q is not within your scopes", coreerrors.ErrForbidden, perm)
		}
	}
	return nil
}

func hasPermission(scopes []string, required string) bool {
	for _, scope := range scopes {
		if scope == required || scope == "*" {
			return true
		}
		if strings.HasSuffix(scope, "*") && strings.HasPrefix(required, strings.TrimSuffix(scope, "*")) {
			return true
		}
	}
	return false
}

func (s *apiKeyService) validatePermissions(ctx context.Context, permissionKeys []string) error {
	if len(permissionKeys) == 0 {
		return nil
	}

	if err := s.accessControlService.ValidatePermissionKeys(ctx, permissionKeys); err != nil {
		if errors.Is(err, coreerrors.ErrNotFound) {
			return fmt.Errorf("%w: one or more permissions were not found", coreerrors.ErrNotFound)
		}
		return err
	}

	return nil
}
