package accesscontrol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/internal/plugins"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func newTestPluginContext(config *models.Config) *models.PluginContext {
	return &models.PluginContext{
		Logger:          &internaltests.MockLogger{},
		ServiceRegistry: plugins.NewServiceRegistry(),
		GetConfig: func() *models.Config {
			if config != nil {
				return config
			}
			return &models.Config{}
		},
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	plugin := New(types.AccessControlPluginConfig{Enabled: true})

	require.NotNil(t, plugin)
	require.True(t, plugin.Config().(types.AccessControlPluginConfig).Enabled)
}

func TestAccessControlPluginMetadata(t *testing.T) {
	t.Parallel()

	plugin := New(types.AccessControlPluginConfig{})

	metadata := plugin.Metadata()
	require.Equal(t, models.PluginAccessControl.String(), metadata.ID)
	require.Equal(t, "1.0.0", metadata.Version)
	require.NotEmpty(t, metadata.Description)
}

func TestAccessControlPluginInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		pluginConfig    types.AccessControlPluginConfig
		config          *models.Config
		closeDB         bool
		initCalls       int
		wantErrContains string
		verify          func(t *testing.T, plugin *AccessControlPlugin, ctx *models.PluginContext)
	}{
		{
			name:         "creates services, permissions and api",
			pluginConfig: types.AccessControlPluginConfig{Enabled: true},
			initCalls:    1,
			verify: func(t *testing.T, plugin *AccessControlPlugin, ctx *models.PluginContext) {
				require.NotNil(t, plugin.useCases)
				require.NotNil(t, plugin.accessControlService)
				require.NotNil(t, ctx.ServiceRegistry.Get(models.ServiceAccessControl.String()))
			},
		},
		{
			name:         "is idempotent when permissions already exist",
			pluginConfig: types.AccessControlPluginConfig{},
			initCalls:    2,
		},
		{
			name:         "loads pre-parsed config",
			pluginConfig: types.AccessControlPluginConfig{},
			config: &models.Config{
				Plugins: map[string]any{},
				PreParsedConfigs: map[string]any{
					models.PluginAccessControl.String(): types.AccessControlPluginConfig{Enabled: true},
				},
			},
			initCalls: 1,
			verify: func(t *testing.T, plugin *AccessControlPlugin, ctx *models.PluginContext) {
				require.True(t, plugin.Config().(types.AccessControlPluginConfig).Enabled)
			},
		},
		{
			name:            "fails when permissions cannot be ensured",
			pluginConfig:    types.AccessControlPluginConfig{},
			closeDB:         true,
			initCalls:       1,
			wantErrContains: "failed to ensure access control permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := accesscontroltests.SetupRepoDB(t)
			if tt.closeDB {
				require.NoError(t, db.Close())
			}

			ctx := newTestPluginContext(tt.config)
			ctx.DB = db

			plugin := New(tt.pluginConfig)

			var err error
			for range tt.initCalls {
				if err = plugin.Init(ctx); err != nil {
					break
				}
			}

			if tt.wantErrContains != "" {
				require.ErrorContains(t, err, tt.wantErrContains)
				require.Nil(t, plugin.Api)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, plugin.Api)

			permissions, err := plugin.Api.GetAllPermissions(context.Background())
			require.NoError(t, err)
			require.Len(t, permissions, 19)

			if tt.verify != nil {
				tt.verify(t, plugin, ctx)
			}
		})
	}
}

func TestAccessControlPluginMigrations(t *testing.T) {
	t.Parallel()

	plugin := New(types.AccessControlPluginConfig{})

	require.NotEmpty(t, plugin.Migrations("sqlite"))
	require.Empty(t, plugin.DependsOn())
	require.NoError(t, plugin.Close())
}
