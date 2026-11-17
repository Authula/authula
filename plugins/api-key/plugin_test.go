package apikey

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	apiconstants "github.com/Authula/authula/plugins/api-key/constants"
	apirepositories "github.com/Authula/authula/plugins/api-key/repositories"
	apiKeyTests "github.com/Authula/authula/plugins/api-key/tests"
	"github.com/Authula/authula/plugins/api-key/types"
	rootservices "github.com/Authula/authula/services"
)

func TestApiKeyPlugin_NewDefaultsAndMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		config           types.ApiKeyPluginConfig
		wantHeader       string
		wantInterval     time.Duration
		wantAllowOrgKeys bool
	}{
		{
			name:         "applies_defaults",
			config:       types.ApiKeyPluginConfig{},
			wantHeader:   "X-AUTHULA-API-KEY",
			wantInterval: time.Minute,
		},
		{
			name:             "preserves_custom_values",
			config:           types.ApiKeyPluginConfig{Header: "X-Custom", CleanupInterval: 5 * time.Second, AllowOrgKeys: true},
			wantHeader:       "X-Custom",
			wantInterval:     5 * time.Second,
			wantAllowOrgKeys: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(tc.config)

			require.NotNil(t, plugin)
			assert.Equal(t, tc.wantHeader, plugin.config.Header)
			assert.Equal(t, tc.wantInterval, plugin.config.CleanupInterval)
			assert.Equal(t, tc.wantAllowOrgKeys, plugin.config.AllowOrgKeys)

			metadata := plugin.Metadata()
			assert.Equal(t, models.PluginApiKey.String(), metadata.ID)
			assert.Equal(t, "1.0.0", metadata.Version)
			assert.NotEmpty(t, metadata.Description)

			assert.Equal(t, plugin.config, plugin.Config())
			assert.Equal(t, []string{models.PluginAccessControl.String()}, plugin.DependsOn())
		})
	}
}

func TestApiKeyPlugin_Routes(t *testing.T) {
	t.Parallel()

	plugin := New(types.ApiKeyPluginConfig{})

	routes := plugin.Routes()

	expected := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api-keys"},
		{http.MethodGet, "/api-keys"},
		{http.MethodGet, "/api-keys/{id}"},
		{http.MethodPatch, "/api-keys/{id}"},
		{http.MethodDelete, "/api-keys/{id}"},
		{http.MethodPost, "/api-keys/verify"},
	}

	require.Len(t, routes, len(expected))
	for i, want := range expected {
		assert.Equal(t, want.method, routes[i].Method)
		assert.Equal(t, want.path, routes[i].Path)
		assert.Len(t, routes[i].Middleware, 1)
		assert.NotNil(t, routes[i].Handler)
	}
}

func TestApiKeyPlugin_HooksAndMigrations(t *testing.T) {
	t.Parallel()

	plugin := New(types.ApiKeyPluginConfig{Header: "X-Test-API-Key"})

	hooks := plugin.Hooks()
	require.Len(t, hooks, 1)
	assert.Equal(t, models.HookBefore, hooks[0].Stage)
	assert.Equal(t, 7, hooks[0].Order)
	assert.NotNil(t, hooks[0].Matcher)
	assert.NotNil(t, hooks[0].Handler)

	assert.NotEmpty(t, plugin.Migrations("sqlite"))
	assert.Empty(t, plugin.Migrations("unknown-provider"))
}

func TestApiKeyPlugin_InitErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    types.ApiKeyPluginConfig
		registry  func(*apiKeyTests.MockAccessControlService) models.ServiceRegistry
		configFn  func() *models.Config
		wantError string
		wantPerms bool
	}{
		{
			name:   "missing_user_service",
			config: types.ApiKeyPluginConfig{},
			registry: func(*apiKeyTests.MockAccessControlService) models.ServiceRegistry {
				return &internaltests.TestServiceRegistry{Services: map[string]any{}}
			},
			wantError: "required service " + models.ServiceUser.String(),
		},
		{
			name:   "missing_token_service",
			config: types.ApiKeyPluginConfig{},
			registry: func(*apiKeyTests.MockAccessControlService) models.ServiceRegistry {
				return &internaltests.TestServiceRegistry{Services: map[string]any{
					models.ServiceUser.String(): &internaltests.MockUserService{},
				}}
			},
			wantError: "required service " + models.ServiceToken.String(),
		},
		{
			name:   "missing_access_control_service",
			config: types.ApiKeyPluginConfig{},
			registry: func(*apiKeyTests.MockAccessControlService) models.ServiceRegistry {
				return &internaltests.TestServiceRegistry{Services: map[string]any{
					models.ServiceUser.String():  &internaltests.MockUserService{},
					models.ServiceToken.String(): &internaltests.MockTokenService{},
				}}
			},
			wantError: "required service " + models.ServiceAccessControl.String(),
		},
		{
			name:   "ensure_permissions_failure",
			config: types.ApiKeyPluginConfig{},
			registry: func(ac *apiKeyTests.MockAccessControlService) models.ServiceRegistry {
				ac.On("EnsurePermissions", mock.Anything, mock.Anything).Return(errors.New("boom")).Once()
				return &internaltests.TestServiceRegistry{Services: map[string]any{
					models.ServiceUser.String():          &internaltests.MockUserService{},
					models.ServiceToken.String():         &internaltests.MockTokenService{},
					models.ServiceAccessControl.String(): ac,
				}}
			},
			wantError: "failed to ensure api key permissions",
			wantPerms: true,
		},
		{
			name:   "org_keys_enabled_without_org_service",
			config: types.ApiKeyPluginConfig{AllowOrgKeys: true},
			registry: func(ac *apiKeyTests.MockAccessControlService) models.ServiceRegistry {
				ac.On("EnsurePermissions", mock.Anything, mock.Anything).Return(nil).Once()
				return &internaltests.TestServiceRegistry{Services: map[string]any{
					models.ServiceUser.String():          &internaltests.MockUserService{},
					models.ServiceToken.String():         &internaltests.MockTokenService{},
					models.ServiceAccessControl.String(): ac,
				}}
			},
			wantError: "allow_org_keys is enabled",
			wantPerms: true,
		},
		{
			name:   "invalid_plugin_config",
			config: types.ApiKeyPluginConfig{},
			configFn: func() *models.Config {
				return &models.Config{Plugins: map[string]any{models.PluginApiKey.String(): "not-a-config"}}
			},
			registry: func(*apiKeyTests.MockAccessControlService) models.ServiceRegistry {
				return &internaltests.TestServiceRegistry{Services: map[string]any{}}
			},
			wantError: "expected a map",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			accessControl := &apiKeyTests.MockAccessControlService{}
			configFn := tc.configFn
			if configFn == nil {
				configFn = func() *models.Config { return &models.Config{} }
			}

			plugin := New(tc.config)
			err := plugin.Init(&models.PluginContext{
				Logger:          &internaltests.MockLogger{},
				ServiceRegistry: tc.registry(accessControl),
				GetConfig:       configFn,
			})

			require.Error(t, err)
			assert.ErrorContains(t, err, tc.wantError)
			if tc.wantPerms {
				accessControl.AssertExpectations(t)
			}
		})
	}
}

func TestApiKeyPlugin_InitSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		config          types.ApiKeyPluginConfig
		withRateLimiter bool
		withOrgService  bool
	}{
		{
			name:   "minimal_services",
			config: types.ApiKeyPluginConfig{},
		},
		{
			name:            "all_optional_services",
			config:          types.ApiKeyPluginConfig{AllowOrgKeys: true, DefaultPrefix: "ak_"},
			withRateLimiter: true,
			withOrgService:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userService := &internaltests.MockUserService{}
			tokenService := &internaltests.MockTokenService{}
			accessControl := &apiKeyTests.MockAccessControlService{}
			accessControl.On("EnsurePermissions", mock.Anything, mock.MatchedBy(func(defs []rootservices.PermissionDefinition) bool {
				return len(defs) == 6 && defs[0].Key == apiconstants.All
			})).Return(nil).Once()

			services := map[string]any{
				models.ServiceUser.String():          userService,
				models.ServiceToken.String():         tokenService,
				models.ServiceAccessControl.String(): accessControl,
			}
			if tc.withRateLimiter {
				services[models.ServiceRateLimit.String()] = &internaltests.MockRateLimitService{}
			}
			if tc.withOrgService {
				services[models.ServiceOrganization.String()] = &internaltests.MockOrganizationService{}
			}

			plugin := New(tc.config)
			err := plugin.Init(&models.PluginContext{
				Logger:          &internaltests.MockLogger{},
				ServiceRegistry: &internaltests.TestServiceRegistry{Services: services},
				GetConfig:       func() *models.Config { return &models.Config{} },
			})

			require.NoError(t, err)
			require.NotNil(t, plugin.useCases)
			require.NotNil(t, plugin.Api)
			assert.Same(t, userService, plugin.userService)
			assert.Same(t, accessControl, plugin.accessControlService)
			if tc.withRateLimiter {
				assert.NotNil(t, plugin.rateLimiterService)
			} else {
				assert.Nil(t, plugin.rateLimiterService)
			}
			if tc.withOrgService {
				assert.NotNil(t, plugin.organizationService)
			} else {
				assert.Nil(t, plugin.organizationService)
			}
			assert.Len(t, plugin.Routes(), 6)
			assert.Len(t, plugin.Hooks(), 1)
			accessControl.AssertExpectations(t)
		})
	}
}

func TestApiKeyPlugin_InitWithAutoCleanup(t *testing.T) {
	t.Parallel()

	db := internaltests.NewSQLiteIntegrationDB(t)
	ctx := context.Background()
	_, err := db.NewCreateTable().Model((*types.ApiKey)(nil)).IfNotExists().Exec(ctx)
	require.NoError(t, err)

	accessControl := &apiKeyTests.MockAccessControlService{}
	accessControl.On("EnsurePermissions", mock.Anything, mock.Anything).Return(nil).Once()

	plugin := New(types.ApiKeyPluginConfig{AutoCleanup: true, CleanupInterval: 10 * time.Millisecond})
	repo := apirepositories.NewBunApiKeyRepository(db)
	now := time.Now().UTC()
	expiredKey := &types.ApiKey{
		ID:        "expired-key",
		KeyHash:   "hash",
		Name:      "expired",
		OwnerType: types.OwnerTypeUser,
		OwnerID:   "user-1",
		Start:     "abcd",
		Last:      "efgh",
		Enabled:   true,
		ExpiresAt: new(now.Add(-time.Hour)),
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err = repo.Create(ctx, expiredKey)
	require.NoError(t, err)

	err = plugin.Init(&models.PluginContext{
		DB:     db,
		Logger: &internaltests.MockLogger{},
		ServiceRegistry: &internaltests.TestServiceRegistry{Services: map[string]any{
			models.ServiceUser.String():          &internaltests.MockUserService{},
			models.ServiceToken.String():         &internaltests.MockTokenService{},
			models.ServiceAccessControl.String(): accessControl,
		}},
		GetConfig: func() *models.Config { return &models.Config{} },
	})
	require.NoError(t, err)
	require.NotNil(t, plugin.stopCleanup)
	require.NotNil(t, plugin.done)

	require.Eventually(t, func() bool {
		stored, err := repo.GetByID(ctx, "expired-key")
		return err == nil && stored == nil
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, plugin.Close())
	assert.True(t, plugin.config.AutoCleanup)
	accessControl.AssertExpectations(t)
}

func TestApiKeyPlugin_CloseWithoutAutoCleanup(t *testing.T) {
	t.Parallel()

	plugin := New(types.ApiKeyPluginConfig{})

	require.NoError(t, plugin.Close())
}
