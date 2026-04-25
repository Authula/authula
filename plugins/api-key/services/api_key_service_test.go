package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalerrors "github.com/Authula/authula/internal/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	roottests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	apiKeyTests "github.com/Authula/authula/plugins/api-key/tests"
	"github.com/Authula/authula/plugins/api-key/types"
)

type apiKeyServiceFixture struct {
	mockUserService          *roottests.MockUserService
	mockTokenService         *roottests.MockTokenService
	mockOrgService           *internaltests.MockOrganizationService
	mockAccessControlService *apiKeyTests.MockAccessControlService
	mockApiKeyService        *apiKeyService
	mockApiKeyRepo           *apiKeyTests.MockApiKeyRepository
}

func newApiKeyServiceFixture(pluginConfig types.ApiKeyPluginConfig) *apiKeyServiceFixture {
	mockOrgService := &internaltests.MockOrganizationService{}
	mockUserService := &roottests.MockUserService{}
	mockTokenService := &roottests.MockTokenService{}
	mockRateLimiterService := &roottests.MockRateLimitService{}
	mockAccessControlService := &apiKeyTests.MockAccessControlService{}

	mockApiKeyRepo := &apiKeyTests.MockApiKeyRepository{}
	service := NewApiKeyService(pluginConfig, mockUserService, mockTokenService, mockAccessControlService, mockRateLimiterService, mockOrgService, mockApiKeyRepo).(*apiKeyService)
	return &apiKeyServiceFixture{mockApiKeyService: service, mockApiKeyRepo: mockApiKeyRepo, mockUserService: mockUserService, mockTokenService: mockTokenService, mockOrgService: mockOrgService, mockAccessControlService: mockAccessControlService}
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
		req            types.CreateApiKeyRequest
		setup          func(*apiKeyServiceFixture)
		wantErr        error
		assertResponse func(*testing.T, *types.CreateApiKeyResponse)
	}{
		{name: "invalid_owner_type", req: types.CreateApiKeyRequest{Name: "Key", OwnerType: "invalid", OwnerID: userID}, wantErr: internalerrors.ErrBadRequest},
		{name: "user_not_found", req: types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeUser, OwnerID: userID}, setup: func(f *apiKeyServiceFixture) {
			f.mockUserService.On("GetByID", mock.Anything, userID).Return((*models.User)(nil), nil).Once()
		}, wantErr: internalerrors.ErrNotFound},
		{name: "user_success", config: types.ApiKeyPluginConfig{DefaultPrefix: "pref-"}, req: types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeUser, OwnerID: userID, Prefix: &prefix, Enabled: &enabled, RateLimitEnabled: &rateLimitEnabled, RateLimitTimeWindow: &rateLimitWindow, RateLimitMaxRequests: &rateLimitMaxRequests, ExpiresAt: &expiresAt, Metadata: metadata}, setup: func(f *apiKeyServiceFixture) {
			f.mockUserService.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID}, nil).Once()
			f.mockTokenService.On("Generate").Return(rawApiKey, nil).Once()
			f.mockTokenService.On("Hash", prefixedApiKey).Return("hashed-key").Once()
			f.mockApiKeyRepo.On("Create", mock.Anything, mock.MatchedBy(func(apiKey *types.ApiKey) bool {
				return apiKey != nil && apiKey.ID != "" && apiKey.KeyHash == "hashed-key" && apiKey.Name == "Key" && apiKey.OwnerType == types.OwnerTypeUser && apiKey.OwnerID == userID && apiKey.Start == expectedStart && apiKey.Last == expectedLast && apiKey.Prefix != nil && *apiKey.Prefix == "pref-" && !apiKey.Enabled && apiKey.RateLimitEnabled && apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Equal(expiresAt) && apiKey.Metadata != nil && len(apiKey.Metadata) == len(metadata)
			})).Return(&types.ApiKey{ID: "api-key-1"}, nil).Once()
		}, assertResponse: func(t *testing.T, resp *types.CreateApiKeyResponse) {
			require.NotNil(t, resp)
			assert.Equal(t, prefixedApiKey, resp.RawApiKey)
			require.NotNil(t, resp.ApiKey)
			assert.Equal(t, "api-key-1", resp.ApiKey.ID)
		}},
		{name: "organization_disabled", req: types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeOrganization, OwnerID: orgID}, wantErr: internalerrors.ErrForbidden},
		{name: "organization_service_nil", config: types.ApiKeyPluginConfig{AllowOrgKeys: true}, req: types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeOrganization, OwnerID: orgID}, setup: nil, wantErr: internalerrors.ErrUnprocessableEntity},
		{name: "organization_not_found", config: types.ApiKeyPluginConfig{AllowOrgKeys: true}, req: types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeOrganization, OwnerID: orgID}, setup: func(f *apiKeyServiceFixture) {
			f.mockOrgService.On("ExistsByID", mock.Anything, orgID).Return(false, nil).Once()
		}, wantErr: internalerrors.ErrNotFound},
		{name: "generate_error", req: types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeUser, OwnerID: userID}, setup: func(f *apiKeyServiceFixture) {
			f.mockUserService.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID}, nil).Once()
			f.mockTokenService.On("Generate").Return("", errors.New("boom")).Once()
		}, wantErr: errors.New("boom")},
		{name: "unknown_permission_rejected", req: types.CreateApiKeyRequest{Name: "Key", OwnerType: types.OwnerTypeUser, OwnerID: userID, Permissions: []string{"missing.permission"}}, setup: func(f *apiKeyServiceFixture) {
			f.mockAccessControlService.On("ValidatePermissionKeys", mock.Anything, []string{"missing.permission"}).Return(internalerrors.ErrNotFound).Once()
		}, wantErr: internalerrors.ErrNotFound},
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

			resp, err := fixture.mockApiKeyService.Create(context.Background(), tc.req)
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
			fixture.mockApiKeyRepo.AssertExpectations(t)
		})
	}
}

func TestApiKeyServiceGetAll(t *testing.T) {
	t.Parallel()

	ownerType := types.OwnerTypeUser
	ownerID := "user-1"

	tests := []struct {
		name    string
		req     types.GetApiKeysRequest
		setup   func(*apiKeyServiceFixture)
		wantErr error
	}{
		{name: "normalize_defaults", req: types.GetApiKeysRequest{}, setup: func(f *apiKeyServiceFixture) {
			f.mockApiKeyRepo.On("GetAll", mock.Anything, (*string)(nil), (*string)(nil), 1, 10).Return([]*types.ApiKey{{ID: "api-key-1"}}, 1, nil).Once()
		}},
		{name: "normalize_nil_items_to_empty_slice", req: types.GetApiKeysRequest{}, setup: func(f *apiKeyServiceFixture) {
			f.mockApiKeyRepo.On("GetAll", mock.Anything, (*string)(nil), (*string)(nil), 1, 10).Return(([]*types.ApiKey)(nil), 0, nil).Once()
		}},
		{name: "clamp_limit_and_page", req: types.GetApiKeysRequest{Page: -2, Limit: 500, OwnerType: &ownerType, OwnerID: &ownerID}, setup: func(f *apiKeyServiceFixture) {
			f.mockApiKeyRepo.On("GetAll", mock.Anything, &ownerType, &ownerID, 1, 100).Return([]*types.ApiKey{{ID: "api-key-1"}}, 1, nil).Once()
		}},
		{name: "repo_error", req: types.GetApiKeysRequest{Page: 1, Limit: 25}, setup: func(f *apiKeyServiceFixture) {
			f.mockApiKeyRepo.On("GetAll", mock.Anything, (*string)(nil), (*string)(nil), 1, 25).Return(nil, 0, errors.New("boom")).Once()
		}, wantErr: errors.New("boom")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			tc.setup(fixture)

			resp, err := fixture.mockApiKeyService.GetAll(context.Background(), tc.req)
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

	tests := []struct {
		name    string
		setup   func(*apiKeyServiceFixture)
		wantErr error
	}{
		{name: "not_found", setup: func(f *apiKeyServiceFixture) {
			f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), nil).Once()
		}, wantErr: internalerrors.ErrNotFound},
		{name: "repo_error", setup: func(f *apiKeyServiceFixture) {
			f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), errors.New("boom")).Once()
		}, wantErr: errors.New("boom")},
		{name: "success", setup: func(f *apiKeyServiceFixture) {
			f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1"}, nil).Once()
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			tc.setup(fixture)

			resp, err := fixture.mockApiKeyService.GetByID(context.Background(), "api-key-1")
			if tc.wantErr == nil {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, "api-key-1", resp.ID)
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

	name := "updated"
	permissions := []string{"read", "write"}
	metadata := map[string]any{"scope": "all"}

	tests := []struct {
		name    string
		setup   func(*apiKeyServiceFixture)
		wantErr error
	}{
		{name: "not_found", setup: func(f *apiKeyServiceFixture) {
			f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), nil).Once()
		}, wantErr: internalerrors.ErrNotFound},
		{name: "repo_error_on_lookup", setup: func(f *apiKeyServiceFixture) {
			f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), errors.New("boom")).Once()
		}, wantErr: errors.New("boom")},
		{name: "success_selective_update", setup: func(f *apiKeyServiceFixture) {
			f.mockAccessControlService.On("ValidatePermissionKeys", mock.Anything, permissions).Return(nil).Once()
			f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", Name: "old", Enabled: true}, nil).Once()
			f.mockApiKeyRepo.On("Update", mock.Anything, mock.MatchedBy(func(apiKey *types.ApiKey) bool {
				return apiKey != nil && apiKey.ID == "api-key-1" && apiKey.Name == name && !apiKey.Enabled && apiKey.RateLimitEnabled && len(apiKey.Permissions) == len(permissions) && apiKey.Metadata != nil && len(apiKey.Metadata) == len(metadata)
			})).Return(&types.ApiKey{ID: "api-key-1", Name: name}, nil).Once()
		}},
		{name: "permission_validation_failure", setup: func(f *apiKeyServiceFixture) {
			f.mockAccessControlService.On("ValidatePermissionKeys", mock.Anything, permissions).Return(internalerrors.ErrNotFound).Once()
			f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", Name: "old", Enabled: true}, nil).Once()
		}, wantErr: internalerrors.ErrNotFound},
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

			resp, err := fixture.mockApiKeyService.Update(context.Background(), "api-key-1", req)
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

	tests := []struct {
		name    string
		setup   func(*apiKeyServiceFixture)
		wantErr error
	}{
		{name: "not_found", setup: func(f *apiKeyServiceFixture) {
			f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), nil).Once()
		}, wantErr: internalerrors.ErrNotFound},
		{name: "success", setup: func(f *apiKeyServiceFixture) {
			f.mockApiKeyRepo.On("GetByID", mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1"}, nil).Once()
			f.mockApiKeyRepo.On("Delete", mock.Anything, "api-key-1").Return(nil).Once()
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := newApiKeyServiceFixture(types.ApiKeyPluginConfig{})
			tc.setup(fixture)

			err := fixture.mockApiKeyService.Delete(context.Background(), "api-key-1")
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
			f.mockApiKeyRepo.On("GetByKeyHash", mock.Anything, "hashed-key").Return((*types.ApiKey)(nil), errors.New("boom")).Once()
		}, wantErr: errors.New("boom")},
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
