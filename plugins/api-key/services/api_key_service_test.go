package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	apiKeyTests "github.com/Authula/authula/plugins/api-key/tests"
	"github.com/Authula/authula/plugins/api-key/types"
)

type apiKeyServiceFixture struct {
	mockUserService          *internaltests.MockUserService
	mockTokenService         *internaltests.MockTokenService
	mockOrgService           *internaltests.MockOrganizationService
	mockAccessControlService *apiKeyTests.MockAccessControlService
	mockRateLimiterService   *internaltests.MockRateLimitService
	mockApiKeyService        *apiKeyService
	mockApiKeyRepo           *apiKeyTests.MockApiKeyRepository
}

func newApiKeyServiceFixture(pluginConfig types.ApiKeyPluginConfig) *apiKeyServiceFixture {
	mockOrgService := &internaltests.MockOrganizationService{}
	mockUserService := &internaltests.MockUserService{}
	mockTokenService := &internaltests.MockTokenService{}
	mockRateLimiterService := &internaltests.MockRateLimitService{}
	mockAccessControlService := &apiKeyTests.MockAccessControlService{}

	mockApiKeyRepo := &apiKeyTests.MockApiKeyRepository{}
	service := NewApiKeyService(pluginConfig, mockUserService, mockTokenService, mockAccessControlService, mockRateLimiterService, mockOrgService, mockApiKeyRepo).(*apiKeyService)
	return &apiKeyServiceFixture{mockApiKeyService: service, mockApiKeyRepo: mockApiKeyRepo, mockUserService: mockUserService, mockTokenService: mockTokenService, mockOrgService: mockOrgService, mockAccessControlService: mockAccessControlService, mockRateLimiterService: mockRateLimiterService}
}

func userActor(id string) *models.Actor {
	return &models.Actor{ID: id, Type: models.ActorUser, Scopes: []string{"read", "write"}}
}

func TestApiKeyServiceCreate(t *testing.T) {
	t.Parallel()

	userID := "user-1"
	orgID := "org-1"
	rawApiKey := "a1b2c3d4e5f6g7h8i9j0"
	prefix := "pref-"
	prefixedApiKey := prefix + rawApiKey
	expectedStart := rawApiKey[:4]
	expectedLast := rawApiKey[len(rawApiKey)-4:]
	enabled := false
	rateLimitEnabled := true
	rateLimitWindow := 60
	rateLimitMaxRequests := 100
	expiresAt := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	metadata := map[string]any{"source": "test"}

	tests := []struct {
		name           string
		config         types.ApiKeyPluginConfig
		actor          *models.Actor
		req            types.CreateApiKeyRequest
		setup          func(*apiKeyServiceFixture)
		wantErr        error
		assertResponse func(*testing.T, *types.CreateApiKeyResponse)
	}{
		{
			name:    "invalid_owner_type",
			actor:   userActor(userID),
			req:     types.CreateApiKeyRequest{Name: "Key", OwnerType: "invalid", OwnerID: userID},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "different_user_id_rejected",
			actor:   userActor(userID),
			req:     types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeUser, OwnerID: "other-user"},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:  "user_not_found",
			actor: userActor(userID),
			req:   types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeUser, OwnerID: userID},
			setup: func(f *apiKeyServiceFixture) {
				f.mockUserService.On("GetByID", mock.Anything, userID).Return((*models.User)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:   "user_success",
			actor:  userActor(userID),
			config: types.ApiKeyPluginConfig{DefaultPrefix: "pref-"},
			req:    types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeUser, OwnerID: userID, Prefix: &prefix, Enabled: &enabled, RateLimitEnabled: &rateLimitEnabled, RateLimitTimeWindow: &rateLimitWindow, RateLimitMaxRequests: &rateLimitMaxRequests, ExpiresAt: &expiresAt, Metadata: metadata},
			setup: func(f *apiKeyServiceFixture) {
				f.mockUserService.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID}, nil).Once()
				f.mockTokenService.On("Generate").Return(rawApiKey, nil).Once()
				f.mockTokenService.On("Hash", prefixedApiKey).Return("hashed-key").Once()
				f.mockApiKeyRepo.On("Create", mock.Anything, mock.MatchedBy(func(apiKey *types.ApiKey) bool {
					return apiKey != nil && apiKey.ID != "" && apiKey.KeyHash == "hashed-key" && apiKey.Name == "Key" && apiKey.OwnerType == types.OwnerTypeUser && apiKey.OwnerID == userID && apiKey.Start == expectedStart && apiKey.Last == expectedLast && apiKey.Prefix != nil && *apiKey.Prefix == "pref-" && !apiKey.Enabled && apiKey.RateLimitEnabled && apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Equal(expiresAt) && apiKey.Metadata != nil && len(apiKey.Metadata) == len(metadata)
				})).Return(&types.ApiKey{ID: "api-key-1", KeyHash: "hashed-key"}, nil).Once()
				f.mockRateLimiterService.On("SetRule", mock.Anything, "hashed-key", time.Duration(rateLimitWindow)*time.Second, rateLimitMaxRequests).Return(nil).Once()
			},
			assertResponse: func(t *testing.T, resp *types.CreateApiKeyResponse) {
				require.NotNil(t, resp)
				assert.Equal(t, prefixedApiKey, resp.RawApiKey)
				require.NotNil(t, resp.ApiKey)
				assert.Equal(t, "api-key-1", resp.ApiKey.ID)
			},
		},
		{
			name:   "org_create_privilege_escalation_rejected",
			actor:  &models.Actor{ID: userID, Type: models.ActorUser, Scopes: []string{"org:api-key:create"}},
			config: types.ApiKeyPluginConfig{AllowOrgKeys: true},
			req:    types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeOrganization, OwnerID: orgID, Permissions: []string{"admin"}},
			setup: func(f *apiKeyServiceFixture) {
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, orgID).Return([]string{"org:api-key:create"}, nil).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:   "org_create_member_required",
			actor:  &models.Actor{ID: userID, Type: models.ActorUser, Scopes: []string{"org:api-key:create"}},
			config: types.ApiKeyPluginConfig{AllowOrgKeys: true},
			req:    types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeOrganization, OwnerID: orgID, Permissions: []string{"read"}},
			setup: func(f *apiKeyServiceFixture) {
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, orgID).Return(([]string)(nil), coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:   "org_create_without_create_permission_rejected",
			actor:  &models.Actor{ID: userID, Type: models.ActorUser, Scopes: []string{"organizations:members:read"}},
			config: types.ApiKeyPluginConfig{AllowOrgKeys: true},
			req:    types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeOrganization, OwnerID: orgID, Permissions: []string{"organizations:members:read"}},
			setup: func(f *apiKeyServiceFixture) {
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, orgID).Return([]string{"organizations:members:read"}, nil).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:   "org_create_success",
			actor:  &models.Actor{ID: userID, Type: models.ActorUser, Scopes: []string{"org:api-key:create", "read"}},
			config: types.ApiKeyPluginConfig{AllowOrgKeys: true},
			req:    types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeOrganization, OwnerID: orgID, Permissions: []string{"read"}},
			setup: func(f *apiKeyServiceFixture) {
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, orgID).Return([]string{"org:api-key:create", "read"}, nil).Once()
				f.mockAccessControlService.On("ValidatePermissionKeys", mock.Anything, []string{"read"}).Return(nil).Once()
				f.mockOrgService.On("ExistsByID", mock.Anything, orgID).Return(true, nil).Once()
				f.mockTokenService.On("Generate").Return(rawApiKey, nil).Once()
				f.mockTokenService.On("Hash", rawApiKey).Return("hashed-key").Once()
				f.mockApiKeyRepo.On("Create", mock.Anything, mock.MatchedBy(func(apiKey *types.ApiKey) bool {
					return apiKey != nil && apiKey.OwnerType == types.OwnerTypeOrganization && apiKey.OwnerID == orgID && len(apiKey.Permissions) == 1 && apiKey.Permissions[0] == "read"
				})).Return(&types.ApiKey{ID: "api-key-1", KeyHash: "hashed-key"}, nil).Once()
			},
			assertResponse: func(t *testing.T, resp *types.CreateApiKeyResponse) {
				require.NotNil(t, resp)
				assert.Equal(t, rawApiKey, resp.RawApiKey)
			},
		},
		{
			name:    "organization_disabled",
			actor:   userActor(userID),
			req:     types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeOrganization, OwnerID: orgID},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:    "organization_service_nil",
			actor:   userActor(userID),
			config:  types.ApiKeyPluginConfig{AllowOrgKeys: true},
			req:     types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeOrganization, OwnerID: orgID},
			setup:   nil,
			wantErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name:   "organization_not_found",
			actor:  &models.Actor{ID: userID, Type: models.ActorUser, Scopes: []string{"org:api-key:create"}},
			config: types.ApiKeyPluginConfig{AllowOrgKeys: true},
			req:    types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeOrganization, OwnerID: orgID},
			setup: func(f *apiKeyServiceFixture) {
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, orgID).Return([]string{"org:api-key:create"}, nil).Once()
				f.mockOrgService.On("ExistsByID", mock.Anything, orgID).Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:  "generate_error",
			actor: userActor(userID),
			req:   types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeUser, OwnerID: userID},
			setup: func(f *apiKeyServiceFixture) {
				f.mockUserService.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID}, nil).Once()
				f.mockTokenService.On("Generate").Return("", errors.New("some error")).Once()
			},
			wantErr: errors.New("some error"),
		},
		{
			name:  "unknown_permission_rejected",
			actor: &models.Actor{ID: userID, Type: models.ActorUser, Scopes: []string{"missing.permission"}},
			req:   types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeUser, OwnerID: userID, Permissions: []string{"missing.permission"}},
			setup: func(f *apiKeyServiceFixture) {
				f.mockAccessControlService.On("ValidatePermissionKeys", mock.Anything, []string{"missing.permission"}).Return(coreerrors.ErrNotFound).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newApiKeyServiceFixture(tc.config)
			if tc.name == "organization_service_nil" {
				fixture.mockApiKeyService.organizationService = nil
			}
			if tc.setup != nil {
				tc.setup(fixture)
			}

			resp, err := fixture.mockApiKeyService.Create(context.Background(), tc.actor, tc.req)
			if tc.wantErr == nil {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tc.assertResponse != nil {
					tc.assertResponse(t, resp)
				}
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
				assert.Nil(t, resp)
			}

			fixture.mockUserService.AssertExpectations(t)
			fixture.mockTokenService.AssertExpectations(t)
			fixture.mockOrgService.AssertExpectations(t)
			fixture.mockAccessControlService.AssertExpectations(t)
			fixture.mockRateLimiterService.AssertExpectations(t)
			fixture.mockApiKeyRepo.AssertExpectations(t)
		})
	}
}

func TestApiKeyServiceGetAll(t *testing.T) {
	t.Parallel()

	ownerType := types.OwnerTypeUser
	ownerID := "user-1"
	orgOwnerType := types.OwnerTypeOrganization
	orgID := "org-1"

	tests := []struct {
		name    string
		actor   *models.Actor
		req     types.GetApiKeysRequest
		setup   func(*apiKeyServiceFixture)
		wantErr error
	}{
		{
			name:  "normalize_defaults",
			actor: userActor(ownerID),
			req:   types.GetApiKeysRequest{},
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetAll", mock.Anything, &ownerType, &ownerID, 1, 10).Return([]*types.ApiKey{{ID: "api-key-1"}}, 1, nil).Once()
			},
		},
		{
			name:  "normalize_nil_items_to_empty_slice",
			actor: userActor(ownerID),
			req:   types.GetApiKeysRequest{},
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetAll", mock.Anything, &ownerType, &ownerID, 1, 10).Return(([]*types.ApiKey)(nil), 0, nil).Once()
			},
		},
		{
			name:  "clamp_limit_and_page",
			actor: userActor(ownerID),
			req:   types.GetApiKeysRequest{Page: -2, Limit: 500, OwnerType: &ownerType, OwnerID: &ownerID},
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetAll", mock.Anything, &ownerType, &ownerID, 1, 100).Return([]*types.ApiKey{{ID: "api-key-1"}}, 1, nil).Once()
			},
		},
		{
			name:  "repo_error",
			actor: userActor(ownerID),
			req:   types.GetApiKeysRequest{Page: 1, Limit: 25},
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetAll", mock.Anything, &ownerType, &ownerID, 1, 25).Return(nil, 0, errors.New("some error")).Once()
			},
			wantErr: errors.New("some error"),
		},
		{
			name:    "org_list_requires_owner_id",
			actor:   userActor(ownerID),
			req:     types.GetApiKeysRequest{OwnerType: &orgOwnerType},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:  "org_list_non_member_rejected",
			actor: userActor(ownerID),
			req:   types.GetApiKeysRequest{OwnerType: &orgOwnerType, OwnerID: &orgID},
			setup: func(f *apiKeyServiceFixture) {
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, ownerID, orgID).Return(([]string)(nil), coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:  "org_list_success",
			actor: userActor(ownerID),
			req:   types.GetApiKeysRequest{OwnerType: &orgOwnerType, OwnerID: &orgID},
			setup: func(f *apiKeyServiceFixture) {
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, ownerID, orgID).Return([]string{"org:api-key:list"}, nil).Once()
				f.mockApiKeyRepo.On("GetAll", mock.Anything, &orgOwnerType, &orgID, 1, 10).Return([]*types.ApiKey{{ID: "api-key-1"}}, 1, nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			if tc.setup != nil {
				tc.setup(fixture)
			}

			resp, err := fixture.mockApiKeyService.GetAll(context.Background(), tc.actor, tc.req)
			if tc.wantErr == nil {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tc.name == "normalize_defaults" {
					assert.Equal(t, 10, resp.Limit)
					assert.Equal(t, 1, resp.Page)
				}
				if tc.name == "normalize_nil_items_to_empty_slice" {
					assert.NotNil(t, resp.Items)
					assert.Empty(t, resp.Items)
				}
				if tc.name == "clamp_limit_and_page" {
					assert.Equal(t, 100, resp.Limit)
					assert.Equal(t, 1, resp.Page)
				}
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
				assert.Nil(t, resp)
			}
			fixture.mockApiKeyRepo.AssertExpectations(t)
		})
	}
}

func TestApiKeyServiceGetByID(t *testing.T) {
	t.Parallel()

	userID := "user-1"

	tests := []struct {
		name    string
		actor   *models.Actor
		setup   func(*apiKeyServiceFixture)
		wantErr error
	}{
		{
			name:  "not_found",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:  "repo_error",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), errors.New("some error")).Once()
			},
			wantErr: errors.New("some error"),
		},
		{
			name:  "other_users_key_rejected",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: "other-user"}, nil).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:  "success",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID}, nil).Once()
			},
		},
		{
			name:  "org_key_non_member_rejected",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeOrganization, OwnerID: "org-1"}, nil).Once()
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, "org-1").Return(([]string)(nil), coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:  "org_key_without_read_permission_rejected",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeOrganization, OwnerID: "org-1"}, nil).Once()
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, "org-1").Return([]string{"org:api-key:create"}, nil).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:  "org_key_success",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeOrganization, OwnerID: "org-1"}, nil).Once()
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, "org-1").Return([]string{"org:api-key:read"}, nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			tc.setup(fixture)

			resp, err := fixture.mockApiKeyService.GetByID(context.Background(), tc.actor, "api-key-1")
			if tc.wantErr == nil {
				require.NoError(t, err)
				require.NotNil(t, resp)
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			}
			fixture.mockApiKeyRepo.AssertExpectations(t)
		})
	}
}

func TestApiKeyServiceUpdate(t *testing.T) {
	t.Parallel()

	userID := "user-1"
	orgID := "org-1"
	name := "updated"
	permissions := []string{"read", "write"}
	metadata := map[string]any{"scope": "all"}

	tests := []struct {
		name    string
		actor   *models.Actor
		setup   func(*apiKeyServiceFixture)
		wantErr error
	}{
		{
			name:  "not_found",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:  "repo_error_on_lookup",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), errors.New("some error")).Once()
			},
			wantErr: errors.New("some error"),
		},
		{
			name:  "other_users_key_rejected",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: "other-user"}, nil).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:  "success_selective_update",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockAccessControlService.On("ValidatePermissionKeys", mock.Anything, permissions).Return(nil).Once()
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", Name: "old", Enabled: true, OwnerType: types.OwnerTypeUser, OwnerID: userID}, nil).Once()
				f.mockApiKeyRepo.On("Update", mock.Anything, mock.MatchedBy(func(apiKey *types.ApiKey) bool {
					return apiKey != nil && apiKey.ID == "api-key-1" && apiKey.Name == name && !apiKey.Enabled && apiKey.RateLimitEnabled && len(apiKey.Permissions) == len(permissions) && apiKey.Metadata != nil && len(apiKey.Metadata) == len(metadata)
				})).Return(&types.ApiKey{ID: "api-key-1", Name: name}, nil).Once()
			},
		},
		{
			name:  "permission_validation_failure",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockAccessControlService.On("ValidatePermissionKeys", mock.Anything, permissions).Return(coreerrors.ErrNotFound).Once()
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", Name: "old", Enabled: true, OwnerType: types.OwnerTypeUser, OwnerID: userID}, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:  "org_update_allowed",
			actor: &models.Actor{ID: userID, Type: models.ActorUser, Scopes: []string{"org:api-key:update", "read", "write"}},
			setup: func(f *apiKeyServiceFixture) {
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, orgID).Return([]string{"org:api-key:update", "read", "write"}, nil).Once()
				f.mockAccessControlService.On("ValidatePermissionKeys", mock.Anything, permissions).Return(nil).Once()
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", Name: "old", Enabled: true, OwnerType: types.OwnerTypeOrganization, OwnerID: orgID}, nil).Once()
				f.mockApiKeyRepo.On("Update", mock.Anything, mock.MatchedBy(func(apiKey *types.ApiKey) bool {
					return apiKey != nil && apiKey.ID == "api-key-1" && apiKey.Name == name && !apiKey.Enabled && apiKey.RateLimitEnabled && len(apiKey.Permissions) == len(permissions) && apiKey.Metadata != nil && len(apiKey.Metadata) == len(metadata)
				})).Return(&types.ApiKey{ID: "api-key-1", Name: name}, nil).Once()
			},
		},
		{
			name:  "org_update_non_member_rejected",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", Name: "old", Enabled: true, OwnerType: types.OwnerTypeOrganization, OwnerID: orgID}, nil).Once()
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, orgID).Return(([]string)(nil), coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:  "org_update_without_update_permission_rejected",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", Name: "old", Enabled: true, OwnerType: types.OwnerTypeOrganization, OwnerID: orgID}, nil).Once()
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, orgID).Return([]string{"org:api-key:read"}, nil).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
	}

	req := types.UpdateApiKeyData{
		Name:             &name,
		Enabled:          new(true),
		RateLimitEnabled: new(true),
		LastRequestedAt:  new(time.Now().UTC()),
		ExpiresAt:        nil,
		Permissions:      permissions,
		Metadata:         metadata,
	}
	*req.Enabled = false
	*req.RateLimitEnabled = true

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			tc.setup(fixture)

			resp, err := fixture.mockApiKeyService.Update(context.Background(), tc.actor, "api-key-1", req)
			if tc.wantErr == nil {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, "api-key-1", resp.ID)
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
				assert.Nil(t, resp)
			}
			fixture.mockApiKeyRepo.AssertExpectations(t)
		})
	}
}

func TestApiKeyServiceDelete(t *testing.T) {
	t.Parallel()

	userID := "user-1"

	tests := []struct {
		name    string
		actor   *models.Actor
		setup   func(*apiKeyServiceFixture)
		wantErr error
	}{
		{
			name:  "not_found",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:  "other_users_key_rejected",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: "other-user"}, nil).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:  "success",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID}, nil).Once()
				f.mockApiKeyRepo.On("Delete", mock.Anything, "api-key-1").Return(nil).Once()
			},
		},
		{
			name:  "org_key_non_member_rejected",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeOrganization, OwnerID: "org-1"}, nil).Once()
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, "org-1").Return(([]string)(nil), coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:  "org_key_delete_success",
			actor: userActor(userID),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeOrganization, OwnerID: "org-1"}, nil).Once()
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, "org-1").Return([]string{"org:api-key:delete"}, nil).Once()
				f.mockApiKeyRepo.On("Delete", mock.Anything, "api-key-1").Return(nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			tc.setup(fixture)

			err := fixture.mockApiKeyService.Delete(context.Background(), tc.actor, "api-key-1")
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			}
			fixture.mockApiKeyRepo.AssertExpectations(t)
		})
	}
}

func TestApiKeyServiceDeleteAllByOwner(t *testing.T) {
	t.Parallel()

	userID := "user-1"
	orgID := "org-1"

	tests := []struct {
		name      string
		actor     *models.Actor
		ownerType string
		ownerID   string
		setup     func(*apiKeyServiceFixture)
		wantErr   error
	}{
		{
			name:      "user_success",
			actor:     userActor(userID),
			ownerType: types.OwnerTypeUser,
			ownerID:   userID,
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("DeleteAllByOwner", mock.Anything, types.OwnerTypeUser, userID).Return(nil).Once()
			},
		},
		{
			name:      "other_users_keys_rejected",
			actor:     userActor(userID),
			ownerType: types.OwnerTypeUser,
			ownerID:   "other-user",
			wantErr:   coreerrors.ErrForbidden,
		},
		{
			name:      "org_non_member_rejected",
			actor:     userActor(userID),
			ownerType: types.OwnerTypeOrganization,
			ownerID:   orgID,
			setup: func(f *apiKeyServiceFixture) {
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, orgID).Return(([]string)(nil), coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name:      "org_success",
			actor:     userActor(userID),
			ownerType: types.OwnerTypeOrganization,
			ownerID:   orgID,
			setup: func(f *apiKeyServiceFixture) {
				f.mockOrgService.On("GetUserPermissionsInOrganization", mock.Anything, userID, orgID).Return([]string{"org:api-key:delete"}, nil).Once()
				f.mockApiKeyRepo.On("DeleteAllByOwner", mock.Anything, types.OwnerTypeOrganization, orgID).Return(nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			if tc.setup != nil {
				tc.setup(fixture)
			}

			err := fixture.mockApiKeyService.DeleteAllByOwner(context.Background(), tc.actor, tc.ownerType, tc.ownerID)
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			}
			fixture.mockApiKeyRepo.AssertExpectations(t)
		})
	}
}

func TestApiKeyServiceVerify(t *testing.T) {
	t.Parallel()

	expiresAt := time.Now().UTC().Add(time.Hour)
	expiredAt := time.Now().UTC().Add(-time.Hour)

	tests := []struct {
		name    string
		setup   func(*apiKeyServiceFixture)
		wantErr error
		wantVal bool
	}{
		{name: "repo_error", setup: func(f *apiKeyServiceFixture) {
			f.mockTokenService.On("Hash", "raw-key").Return("hashed-key").Once()
			f.mockApiKeyRepo.On("GetByKeyHash", mock.Anything, "hashed-key").Return((*types.ApiKey)(nil), errors.New("some error")).Once()
		}, wantErr: errors.New("some error")},
		{name: "invalid_key_missing", setup: func(f *apiKeyServiceFixture) {
			f.mockTokenService.On("Hash", "raw-key").Return("hashed-key").Once()
			f.mockApiKeyRepo.On("GetByKeyHash", mock.Anything, "hashed-key").Return((*types.ApiKey)(nil), nil).Once()
		}, wantVal: false},
		{name: "invalid_key_disabled", setup: func(f *apiKeyServiceFixture) {
			f.mockTokenService.On("Hash", "raw-key").Return("hashed-key").Once()
			f.mockApiKeyRepo.On("GetByKeyHash", mock.Anything, "hashed-key").Return(&types.ApiKey{ID: "api-key-1", Enabled: false}, nil).Once()
		}, wantVal: false},
		{name: "invalid_key_expired", setup: func(f *apiKeyServiceFixture) {
			f.mockTokenService.On("Hash", "raw-key").Return("hashed-key").Once()
			f.mockApiKeyRepo.On("GetByKeyHash", mock.Anything, "hashed-key").Return(&types.ApiKey{ID: "api-key-1", Enabled: true, ExpiresAt: &expiredAt}, nil).Once()
		}, wantVal: false},
		{name: "invalid_key_exhausted", setup: func(f *apiKeyServiceFixture) {
			f.mockTokenService.On("Hash", "raw-key").Return("hashed-key").Once()
			f.mockApiKeyRepo.On("GetByKeyHash", mock.Anything, "hashed-key").Return(&types.ApiKey{ID: "api-key-1", Enabled: true}, nil).Once()
		}, wantVal: true},
		{name: "success", setup: func(f *apiKeyServiceFixture) {
			f.mockTokenService.On("Hash", "raw-key").Return("hashed-key").Once()
			f.mockApiKeyRepo.On("GetByKeyHash", mock.Anything, "hashed-key").Return(&types.ApiKey{ID: "api-key-1", Enabled: true, ExpiresAt: &expiresAt}, nil).Once()
		}, wantVal: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			tc.setup(fixture)

			resp, err := fixture.mockApiKeyService.Verify(context.Background(), types.VerifyApiKeyRequest{Key: "raw-key"})
			if tc.wantErr == nil {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tc.wantVal, resp.Valid)
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			}
			fixture.mockTokenService.AssertExpectations(t)
			fixture.mockApiKeyRepo.AssertExpectations(t)
		})
	}
}

func TestApiKeyServiceDeleteExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(*apiKeyServiceFixture)
		wantErr error
	}{
		{
			name: "success",
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("DeleteExpired", mock.Anything).Return(nil).Once()
			},
		},
		{
			name: "repository_error",
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("DeleteExpired", mock.Anything).Return(errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			tc.setup(fixture)

			err := fixture.mockApiKeyService.DeleteExpired(context.Background())

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
			fixture.mockApiKeyRepo.AssertExpectations(t)
		})
	}
}

func TestApiKeyServiceRecordLastRequest(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		setup       func(*apiKeyServiceFixture)
		wantErr     error
		wantNil     bool
		wantUpdated bool
	}{
		{
			name: "success",
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1"}, nil).Once()
				f.mockApiKeyRepo.On("Update", mock.Anything, mock.MatchedBy(func(apiKey *types.ApiKey) bool {
					return apiKey != nil && apiKey.LastRequestedAt != nil && apiKey.LastRequestedAt.Equal(timestamp)
				})).Return(&types.ApiKey{ID: "api-key-1", LastRequestedAt: &timestamp}, nil).Once()
			},
			wantUpdated: true,
		},
		{
			name: "not_found",
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
			wantNil: true,
		},
		{
			name: "lookup_error",
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
			wantNil: true,
		},
		{
			name: "update_error",
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1"}, nil).Once()
				f.mockApiKeyRepo.On("Update", mock.Anything, mock.Anything).Return(nil, errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			tc.setup(fixture)

			apiKey, err := fixture.mockApiKeyService.RecordLastRequest(context.Background(), "api-key-1", timestamp)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
			if tc.wantNil {
				assert.Nil(t, apiKey)
			} else {
				require.NotNil(t, apiKey)
			}
			if tc.wantUpdated {
				require.NotNil(t, apiKey.LastRequestedAt)
				assert.Equal(t, timestamp, *apiKey.LastRequestedAt)
			}
			fixture.mockApiKeyRepo.AssertExpectations(t)
		})
	}
}

func TestApiKeyServiceValidatePermissionKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		permissions []string
		setup       func(*apiKeyServiceFixture)
		wantErr     error
	}{
		{
			name:        "empty_permissions_skip_validation",
			permissions: nil,
		},
		{
			name:        "valid_permissions",
			permissions: []string{"read"},
			setup: func(f *apiKeyServiceFixture) {
				f.mockAccessControlService.On("ValidatePermissionKeys", mock.Anything, []string{"read"}).Return(nil).Once()
			},
		},
		{
			name:        "not_found_is_mapped",
			permissions: []string{"missing"},
			setup: func(f *apiKeyServiceFixture) {
				f.mockAccessControlService.On("ValidatePermissionKeys", mock.Anything, []string{"missing"}).Return(coreerrors.ErrNotFound).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:        "other_error_propagated",
			permissions: []string{"read"},
			setup: func(f *apiKeyServiceFixture) {
				f.mockAccessControlService.On("ValidatePermissionKeys", mock.Anything, []string{"read"}).Return(errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			if tc.setup != nil {
				tc.setup(fixture)
			}

			err := fixture.mockApiKeyService.ValidatePermissionKeys(context.Background(), tc.permissions)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
			fixture.mockAccessControlService.AssertExpectations(t)
		})
	}
}

func TestApiKeyServiceDelete_RateLimitAndRepositoryErrors(t *testing.T) {
	t.Parallel()

	userID := "user-1"

	tests := []struct {
		name            string
		rateLimitEnable bool
		removeLimiter   bool
		setup           func(*apiKeyServiceFixture)
		wantErr         error
	}{
		{
			name:            "rate_limit_delete_rule_error_aborts",
			rateLimitEnable: true,
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID, KeyHash: "hash", RateLimitEnabled: true}, nil).Once()
				f.mockRateLimiterService.On("DeleteRule", mock.Anything, "hash").Return(errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
		{
			name:            "rate_limit_delete_rule_success",
			rateLimitEnable: true,
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID, KeyHash: "hash", RateLimitEnabled: true}, nil).Once()
				f.mockRateLimiterService.On("DeleteRule", mock.Anything, "hash").Return(nil).Once()
				f.mockApiKeyRepo.On("Delete", mock.Anything, "api-key-1").Return(nil).Once()
			},
		},
		{
			name:            "nil_rate_limiter_skips_rule_cleanup",
			rateLimitEnable: true,
			removeLimiter:   true,
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID, KeyHash: "hash", RateLimitEnabled: true}, nil).Once()
				f.mockApiKeyRepo.On("Delete", mock.Anything, "api-key-1").Return(nil).Once()
			},
		},
		{
			name: "lookup_repository_error",
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "repository_delete_error",
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID}, nil).Once()
				f.mockApiKeyRepo.On("Delete", mock.Anything, "api-key-1").Return(errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			if tc.removeLimiter {
				fixture.mockApiKeyService.rateLimiterService = nil
			}
			tc.setup(fixture)

			err := fixture.mockApiKeyService.Delete(context.Background(), userActor(userID), "api-key-1")

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
			fixture.mockApiKeyRepo.AssertExpectations(t)
			fixture.mockRateLimiterService.AssertExpectations(t)
		})
	}
}

func TestApiKeyServiceUpdate_RateLimitAndRepositoryErrors(t *testing.T) {
	t.Parallel()

	userID := "user-1"
	window := 60
	maxRequests := 30
	lastRequestedAt := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		req           types.UpdateApiKeyData
		removeLimiter bool
		setup         func(*apiKeyServiceFixture)
		wantErr       error
	}{
		{
			name: "enable_rate_limit_stores_rule",
			req:  types.UpdateApiKeyData{RateLimitEnabled: new(true), RateLimitTimeWindow: &window, RateLimitMaxRequests: &maxRequests},
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID, KeyHash: "hash"}, nil).Once()
				f.mockRateLimiterService.On("SetRule", mock.Anything, "hash", time.Minute, maxRequests).Return(nil).Once()
				f.mockApiKeyRepo.On("Update", mock.Anything, mock.MatchedBy(func(apiKey *types.ApiKey) bool {
					return apiKey != nil && apiKey.RateLimitEnabled
				})).Return(&types.ApiKey{ID: "api-key-1"}, nil).Once()
			},
		},
		{
			name: "enabled_without_window_skips_rule",
			req:  types.UpdateApiKeyData{RateLimitEnabled: new(true)},
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID, KeyHash: "hash"}, nil).Once()
				f.mockApiKeyRepo.On("Update", mock.Anything, mock.Anything).Return(&types.ApiKey{ID: "api-key-1"}, nil).Once()
			},
		},
		{
			name: "disable_rate_limit_deletes_rule",
			req:  types.UpdateApiKeyData{RateLimitEnabled: new(false)},
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID, KeyHash: "hash", RateLimitEnabled: true}, nil).Once()
				f.mockRateLimiterService.On("DeleteRule", mock.Anything, "hash").Return(nil).Once()
				f.mockApiKeyRepo.On("Update", mock.Anything, mock.Anything).Return(&types.ApiKey{ID: "api-key-1"}, nil).Once()
			},
		},
		{
			name: "set_rule_error_aborts",
			req:  types.UpdateApiKeyData{RateLimitEnabled: new(true), RateLimitTimeWindow: &window, RateLimitMaxRequests: &maxRequests},
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID, KeyHash: "hash"}, nil).Once()
				f.mockRateLimiterService.On("SetRule", mock.Anything, "hash", time.Minute, maxRequests).Return(errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
		{
			name: "delete_rule_error_aborts",
			req:  types.UpdateApiKeyData{RateLimitEnabled: new(false)},
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID, KeyHash: "hash", RateLimitEnabled: true}, nil).Once()
				f.mockRateLimiterService.On("DeleteRule", mock.Anything, "hash").Return(errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
		{
			name:          "nil_rate_limiter_skips_rule_management",
			req:           types.UpdateApiKeyData{RateLimitEnabled: new(true), RateLimitTimeWindow: &window, RateLimitMaxRequests: &maxRequests},
			removeLimiter: true,
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID, KeyHash: "hash"}, nil).Once()
				f.mockApiKeyRepo.On("Update", mock.Anything, mock.Anything).Return(&types.ApiKey{ID: "api-key-1"}, nil).Once()
			},
		},
		{
			name: "timestamps_are_applied",
			req:  types.UpdateApiKeyData{LastRequestedAt: &lastRequestedAt, ExpiresAt: &expiresAt},
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID}, nil).Once()
				f.mockApiKeyRepo.On("Update", mock.Anything, mock.MatchedBy(func(apiKey *types.ApiKey) bool {
					return apiKey != nil && apiKey.LastRequestedAt != nil && apiKey.LastRequestedAt.Equal(lastRequestedAt) && apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Equal(expiresAt)
				})).Return(&types.ApiKey{ID: "api-key-1"}, nil).Once()
			},
		},
		{
			name: "repository_update_error",
			req:  types.UpdateApiKeyData{},
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: userID}, nil).Once()
				f.mockApiKeyRepo.On("Update", mock.Anything, mock.Anything).Return(nil, errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			if tc.removeLimiter {
				fixture.mockApiKeyService.rateLimiterService = nil
			}
			tc.setup(fixture)

			apiKey, err := fixture.mockApiKeyService.Update(context.Background(), userActor(userID), "api-key-1", tc.req)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
				assert.Nil(t, apiKey)
			} else {
				require.NoError(t, err)
				require.NotNil(t, apiKey)
			}
			fixture.mockApiKeyRepo.AssertExpectations(t)
			fixture.mockRateLimiterService.AssertExpectations(t)
		})
	}
}

func TestApiKeyServiceCreate_RateLimitAndRepositoryPaths(t *testing.T) {
	t.Parallel()

	userID := "user-1"
	rawApiKey := "abcdefghij"
	window := 60
	maxRequests := 10
	rateLimitEnabled := true

	tests := []struct {
		name          string
		req           types.CreateApiKeyRequest
		config        types.ApiKeyPluginConfig
		removeLimiter bool
		setup         func(*apiKeyServiceFixture)
		wantErr       error
		wantRawKey    string
	}{
		{
			name:   "set_rule_error_aborts",
			config: types.ApiKeyPluginConfig{DefaultPrefix: "ak_"},
			req:    types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeUser, OwnerID: userID, RateLimitEnabled: &rateLimitEnabled, RateLimitTimeWindow: &window, RateLimitMaxRequests: &maxRequests},
			setup: func(f *apiKeyServiceFixture) {
				f.mockUserService.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID}, nil).Once()
				f.mockTokenService.On("Generate").Return(rawApiKey, nil).Once()
				f.mockTokenService.On("Hash", "ak_"+rawApiKey).Return("hashed-key").Once()
				f.mockApiKeyRepo.On("Create", mock.Anything, mock.Anything).Return(&types.ApiKey{ID: "api-key-1", KeyHash: "hashed-key"}, nil).Once()
				f.mockRateLimiterService.On("SetRule", mock.Anything, "hashed-key", time.Minute, maxRequests).Return(errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
		{
			name:          "nil_rate_limiter_skips_set_rule",
			config:        types.ApiKeyPluginConfig{DefaultPrefix: "ak_"},
			req:           types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeUser, OwnerID: userID, RateLimitEnabled: &rateLimitEnabled, RateLimitTimeWindow: &window, RateLimitMaxRequests: &maxRequests},
			removeLimiter: true,
			setup: func(f *apiKeyServiceFixture) {
				f.mockUserService.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID}, nil).Once()
				f.mockTokenService.On("Generate").Return(rawApiKey, nil).Once()
				f.mockTokenService.On("Hash", "ak_"+rawApiKey).Return("hashed-key").Once()
				f.mockApiKeyRepo.On("Create", mock.Anything, mock.Anything).Return(&types.ApiKey{ID: "api-key-1", KeyHash: "hashed-key"}, nil).Once()
			},
			wantRawKey: "ak_" + rawApiKey,
		},
		{
			name:   "repository_create_error",
			config: types.ApiKeyPluginConfig{DefaultPrefix: "ak_"},
			req:    types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeUser, OwnerID: userID},
			setup: func(f *apiKeyServiceFixture) {
				f.mockUserService.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID}, nil).Once()
				f.mockTokenService.On("Generate").Return(rawApiKey, nil).Once()
				f.mockTokenService.On("Hash", "ak_"+rawApiKey).Return("hashed-key").Once()
				f.mockApiKeyRepo.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
		{
			name:   "short_raw_key_has_empty_start_and_last",
			config: types.ApiKeyPluginConfig{},
			req:    types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeUser, OwnerID: userID},
			setup: func(f *apiKeyServiceFixture) {
				f.mockUserService.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID}, nil).Once()
				f.mockTokenService.On("Generate").Return("abc", nil).Once()
				f.mockTokenService.On("Hash", "abc").Return("hashed-key").Once()
				f.mockApiKeyRepo.On("Create", mock.Anything, mock.MatchedBy(func(apiKey *types.ApiKey) bool {
					return apiKey != nil && apiKey.Start == "" && apiKey.Last == "" && apiKey.Prefix != nil && *apiKey.Prefix == ""
				})).Return(&types.ApiKey{ID: "api-key-1", KeyHash: "hashed-key"}, nil).Once()
			},
			wantRawKey: "abc",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(tc.config)
			if tc.removeLimiter {
				fixture.mockApiKeyService.rateLimiterService = nil
			}
			tc.setup(fixture)

			resp, err := fixture.mockApiKeyService.Create(context.Background(), userActor(userID), tc.req)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tc.wantRawKey, resp.RawApiKey)
			}
			fixture.mockUserService.AssertExpectations(t)
			fixture.mockTokenService.AssertExpectations(t)
			fixture.mockRateLimiterService.AssertExpectations(t)
			fixture.mockApiKeyRepo.AssertExpectations(t)
		})
	}
}

func TestApiKeyService_AuthorizationEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		actor   *models.Actor
		setup   func(*apiKeyServiceFixture)
		call    func(*apiKeyService, *models.Actor) error
		wantErr error
	}{
		{
			name:  "org_key_with_nil_actor_is_unauthorized",
			actor: nil,
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeOrganization, OwnerID: "org-1"}, nil).Once()
			},
			call: func(s *apiKeyService, actor *models.Actor) error {
				_, err := s.GetByID(context.Background(), actor, "api-key-1")
				return err
			},
			wantErr: coreerrors.ErrUnauthorized,
		},
		{
			name:  "org_key_without_organization_service_is_unprocessable",
			actor: userActor("user-1"),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyService.organizationService = nil
				f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeOrganization, OwnerID: "org-1"}, nil).Once()
			},
			call: func(s *apiKeyService, actor *models.Actor) error {
				_, err := s.GetByID(context.Background(), actor, "api-key-1")
				return err
			},
			wantErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name:  "delete_all_by_owner_unknown_type_still_delegates",
			actor: userActor("user-1"),
			setup: func(f *apiKeyServiceFixture) {
				f.mockApiKeyRepo.On("DeleteAllByOwner", mock.Anything, "service", "svc-1").Return(nil).Once()
			},
			call: func(s *apiKeyService, actor *models.Actor) error {
				return s.DeleteAllByOwner(context.Background(), actor, "service", "svc-1")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			tc.setup(fixture)

			err := tc.call(fixture.mockApiKeyService, tc.actor)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
			fixture.mockApiKeyRepo.AssertExpectations(t)
		})
	}
}

func TestHasPermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		scopes   []string
		required string
		want     bool
	}{
		{name: "exact_match", scopes: []string{"read"}, required: "read", want: true},
		{name: "wildcard_all", scopes: []string{"*"}, required: "anything", want: true},
		{name: "prefix_wildcard_match", scopes: []string{"org:api-key:*"}, required: "org:api-key:read", want: true},
		{name: "prefix_wildcard_without_prefix_match", scopes: []string{"users:*"}, required: "org:api-key:read", want: false},
		{name: "no_match", scopes: []string{"read"}, required: "write", want: false},
		{name: "empty_scopes", scopes: nil, required: "read", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, hasPermission(tc.scopes, tc.required))
		})
	}
}
