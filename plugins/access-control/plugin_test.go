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

	t.Run("creates services, permissions and api", func(t *testing.T) {
		t.Parallel()

		db := accesscontroltests.SetupRepoDB(t)
		ctx := newTestPluginContext(nil)
		ctx.DB = db

		plugin := New(types.AccessControlPluginConfig{Enabled: true})
		require.NoError(t, plugin.Init(ctx))

		require.NotNil(t, plugin.Api)
		require.NotNil(t, plugin.useCases)
		require.NotNil(t, plugin.accessControlService)
		require.NotNil(t, ctx.ServiceRegistry.Get(models.ServiceAccessControl.String()))

		permissions, err := plugin.Api.GetAllPermissions(context.Background())
		require.NoError(t, err)
		require.Len(t, permissions, 19)
	})

	t.Run("is idempotent when permissions already exist", func(t *testing.T) {
		t.Parallel()

		db := accesscontroltests.SetupRepoDB(t)
		ctx := newTestPluginContext(nil)
		ctx.DB = db

		plugin := New(types.AccessControlPluginConfig{})
		require.NoError(t, plugin.Init(ctx))
		require.NoError(t, plugin.Init(ctx))

		permissions, err := plugin.Api.GetAllPermissions(context.Background())
		require.NoError(t, err)
		require.Len(t, permissions, 19)
	})

	t.Run("loads pre-parsed config", func(t *testing.T) {
		t.Parallel()

		db := accesscontroltests.SetupRepoDB(t)
		config := &models.Config{
			Plugins: map[string]any{},
			PreParsedConfigs: map[string]any{
				models.PluginAccessControl.String(): types.AccessControlPluginConfig{Enabled: true},
			},
		}
		ctx := newTestPluginContext(config)
		ctx.DB = db

		plugin := New(types.AccessControlPluginConfig{})
		require.NoError(t, plugin.Init(ctx))
		require.True(t, plugin.Config().(types.AccessControlPluginConfig).Enabled)
	})

	t.Run("fails when permissions cannot be ensured", func(t *testing.T) {
		t.Parallel()

		db := accesscontroltests.SetupRepoDB(t)
		require.NoError(t, db.Close())

		ctx := newTestPluginContext(nil)
		ctx.DB = db

		plugin := New(types.AccessControlPluginConfig{})
		err := plugin.Init(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to ensure access control permissions")
		require.Nil(t, plugin.Api)
	})
}

func TestAccessControlPluginMigrations(t *testing.T) {
	t.Parallel()

	plugin := New(types.AccessControlPluginConfig{})

	require.NotEmpty(t, plugin.Migrations("sqlite"))
	require.Empty(t, plugin.DependsOn())
	require.NoError(t, plugin.Close())
}
