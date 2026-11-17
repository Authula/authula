package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/rate-limit/services"
	plugintests "github.com/Authula/authula/plugins/rate-limit/tests"
	"github.com/Authula/authula/plugins/rate-limit/types"
	rootservices "github.com/Authula/authula/services"
)

func TestInMemoryProvider(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		key         string
		window      time.Duration
		maxRequests int
		calls       []struct {
			allowed bool
			count   int
		}
	}{
		{
			name:        "request sequence with overflow",
			key:         "test:key",
			window:      1 * time.Minute,
			maxRequests: 5,
			calls: []struct {
				allowed bool
				count   int
			}{
				{allowed: true, count: 1},
				{allowed: true, count: 2},
				{allowed: true, count: 3},
				{allowed: true, count: 4},
				{allowed: true, count: 5},
				{allowed: false, count: 6},
			},
		},
		{
			name:        "different key starts at one",
			key:         "different:key",
			window:      1 * time.Minute,
			maxRequests: 5,
			calls: []struct {
				allowed bool
				count   int
			}{
				{allowed: true, count: 1},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			provider := services.NewInMemoryProvider()
			t.Cleanup(func() {
				if err := provider.Close(); err != nil {
					t.Fatalf("failed to close provider: %v", err)
				}
			})

			ctx := context.Background()
			for _, call := range tc.calls {
				allowed, count, _, err := provider.CheckAndIncrement(ctx, tc.key, tc.window, tc.maxRequests)
				if err != nil {
					t.Fatalf("unexpected error for key %q: %v", tc.key, err)
				}
				if allowed != call.allowed || count != call.count {
					t.Fatalf("expected allowed=%v count=%d, got allowed=%v count=%d", call.allowed, call.count, allowed, count)
				}
			}
		})
	}
}

func TestRateLimitPluginConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		config     types.RateLimitPluginConfig
		expectedID string
	}{
		{
			name: "default in-memory config",
			config: types.RateLimitPluginConfig{
				Enabled:  true,
				Window:   1 * time.Minute,
				Max:      100,
				Prefix:   "ratelimit:",
				Provider: types.RateLimitProviderInMemory,
			},
			expectedID: models.PluginRateLimit.String(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(tc.config)
			metadata := plugin.Metadata()

			if metadata.ID != tc.expectedID {
				t.Fatalf("expected plugin ID %q, got %q", tc.expectedID, metadata.ID)
			}
			if plugin.Config() == nil {
				t.Fatal("plugin config should not be nil")
			}
		})
	}
}

func newTestRateLimitPlugin(t *testing.T) (*RateLimitPlugin, types.RateLimitProvider) {
	t.Helper()

	provider := services.NewInMemoryProvider()
	t.Cleanup(func() {
		if err := provider.Close(); err != nil {
			t.Fatalf("failed to close provider: %v", err)
		}
	})

	return &RateLimitPlugin{provider: provider, logger: &internaltests.MockLogger{}}, provider
}

func TestRateLimitPlugin_MetadataAndHooks(t *testing.T) {
	t.Parallel()

	plugin := New(types.RateLimitPluginConfig{})

	metadata := plugin.Metadata()
	require.Equal(t, models.PluginRateLimit.String(), metadata.ID)
	require.NotEmpty(t, metadata.Version)
	require.NotEmpty(t, metadata.Description)
	require.NotNil(t, plugin.Config())
	require.Nil(t, plugin.DependsOn())

	hooks := plugin.Hooks()
	require.Len(t, hooks, 3)
	assert.Equal(t, models.HookOnRequest, hooks[0].Stage)
	assert.Equal(t, 0, hooks[0].Order)
	assert.Equal(t, models.HookBefore, hooks[1].Stage)
	assert.Equal(t, 15, hooks[1].Order)
	assert.Equal(t, models.HookAfter, hooks[2].Stage)
	assert.Equal(t, 0, hooks[2].Order)
	for _, hook := range hooks {
		assert.NotNil(t, hook.Handler)
	}
}

func TestRateLimitPlugin_Migrations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		provider  types.RateLimitProviderType
		wantEmpty bool
	}{
		{name: "database_provider_has_migrations", provider: types.RateLimitProviderDatabase},
		{name: "memory_provider_has_no_migrations", provider: types.RateLimitProviderInMemory, wantEmpty: true},
		{name: "redis_provider_has_no_migrations", provider: types.RateLimitProviderRedis, wantEmpty: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(types.RateLimitPluginConfig{Provider: tc.provider})

			migrations := plugin.Migrations("sqlite")

			if tc.wantEmpty {
				assert.Empty(t, migrations)
			} else {
				assert.NotEmpty(t, migrations)
			}
		})
	}
}

func TestRateLimitPlugin_Init(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         types.RateLimitPluginConfig
		withDB         bool
		services       map[string]any
		configFn       func() *models.Config
		wantError      string
		wantProvider   string
		wantRegistered bool
	}{
		{
			name:   "memory_provider_defaults",
			config: types.RateLimitPluginConfig{},
			services: map[string]any{
				models.ServiceSecondaryStorage.String(): &plugintests.StubSecondaryStorageService{Storage: nil, Name: "none"},
			},
			wantProvider:   "memory",
			wantRegistered: true,
		},
		{
			name:           "memory_provider_with_config",
			config:         types.RateLimitPluginConfig{Provider: types.RateLimitProviderInMemory, Memory: &types.MemoryStorageConfig{CleanupInterval: time.Hour}},
			services:       map[string]any{},
			wantProvider:   "memory",
			wantRegistered: true,
		},
		{
			name:           "redis_without_secondary_storage_falls_back_to_memory",
			config:         types.RateLimitPluginConfig{Provider: types.RateLimitProviderRedis},
			services:       map[string]any{},
			wantProvider:   "memory",
			wantRegistered: true,
		},
		{
			name:           "redis_with_secondary_storage_uses_it",
			config:         types.RateLimitPluginConfig{Provider: types.RateLimitProviderRedis},
			services:       map[string]any{models.ServiceSecondaryStorage.String(): &plugintests.StubSecondaryStorageService{Storage: &internaltests.MockSecondaryStorage{}, Name: "redis"}},
			wantProvider:   "redis",
			wantRegistered: true,
		},
		{
			name:           "database_without_db_falls_back_to_memory",
			config:         types.RateLimitPluginConfig{Provider: types.RateLimitProviderDatabase},
			services:       map[string]any{},
			wantProvider:   "memory",
			wantRegistered: true,
		},
		{
			name:           "database_with_db_uses_database_provider",
			config:         types.RateLimitPluginConfig{Provider: types.RateLimitProviderDatabase, Database: &types.DatabaseStorageConfig{CleanupInterval: time.Hour}},
			withDB:         true,
			services:       map[string]any{},
			wantProvider:   "database",
			wantRegistered: true,
		},
		{
			name:      "invalid_plugin_config_errors",
			config:    types.RateLimitPluginConfig{},
			services:  map[string]any{},
			wantError: "expected a map",
			configFn: func() *models.Config {
				return &models.Config{Plugins: map[string]any{models.PluginRateLimit.String(): "invalid"}}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(tc.config)
			registry := &plugintests.RecordingServiceRegistry{Services: tc.services}
			configFn := tc.configFn
			if configFn == nil {
				configFn = func() *models.Config { return &models.Config{} }
			}
			pluginCtx := &models.PluginContext{
				Logger:          &internaltests.MockLogger{},
				ServiceRegistry: registry,
				GetConfig:       configFn,
			}
			if tc.withDB {
				pluginCtx.DB = internaltests.NewSQLiteIntegrationDB(t)
			}

			err := plugin.Init(pluginCtx)

			if tc.wantError != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, plugin.provider)
			assert.Equal(t, tc.wantProvider, plugin.provider.GetName())

			if tc.wantRegistered {
				assert.Contains(t, registry.Registered, models.ServiceRateLimit.String())
				service, ok := registry.Services[models.ServiceRateLimit.String()].(rootservices.RateLimiterService)
				require.True(t, ok)
				require.NotNil(t, service)
			}
			require.NoError(t, plugin.Close())
		})
	}
}

func TestRateLimitPlugin_TrySecondaryStorage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		registry models.ServiceRegistry
		wantNil  bool
		wantName string
	}{
		{
			name:     "missing_service",
			registry: &internaltests.TestServiceRegistry{Services: map[string]any{}},
			wantNil:  true,
		},
		{
			name: "nil_storage",
			registry: &internaltests.TestServiceRegistry{Services: map[string]any{
				models.ServiceSecondaryStorage.String(): &plugintests.StubSecondaryStorageService{Storage: nil, Name: "redis"},
			}},
			wantNil: true,
		},
		{
			name: "configured_storage",
			registry: &internaltests.TestServiceRegistry{Services: map[string]any{
				models.ServiceSecondaryStorage.String(): &plugintests.StubSecondaryStorageService{Storage: &internaltests.MockSecondaryStorage{}, Name: "custom"},
			}},
			wantName: "custom",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plugin := &RateLimitPlugin{pluginCtx: &models.PluginContext{ServiceRegistry: tc.registry}}

			provider := plugin.trySecondaryStorage()

			if tc.wantNil {
				assert.Nil(t, provider)
				return
			}
			require.NotNil(t, provider)
			assert.Equal(t, tc.wantName, provider.GetName())
			require.NoError(t, provider.Close())
		})
	}
}
