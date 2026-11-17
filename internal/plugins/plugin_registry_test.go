package plugins_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/internal/migrationmanager"
	"github.com/Authula/authula/internal/plugins"
	internaltests "github.com/Authula/authula/internal/tests"
	migrationsmodule "github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
)

type fakePlugin struct {
	metadata models.PluginMetadata

	mu         sync.Mutex
	initCalls  int
	closeCalls int
	initCtx    *models.PluginContext
	initErr    error
	closeErr   error

	providerMigrations map[string][]migrationsmodule.Migration
	dependsOn          []string
}

func newFakePlugin(id string) *fakePlugin {
	return &fakePlugin{metadata: models.PluginMetadata{ID: id, Version: "1.0.0"}}
}

func newMigratableFakePlugin(id string, migrations []migrationsmodule.Migration) *fakePlugin {
	plugin := newFakePlugin(id)
	plugin.providerMigrations = map[string][]migrationsmodule.Migration{"sqlite": migrations}
	return plugin
}

func (p *fakePlugin) Metadata() models.PluginMetadata { return p.metadata }
func (p *fakePlugin) Config() any                     { return nil }

func (p *fakePlugin) Init(ctx *models.PluginContext) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.initCalls++
	p.initCtx = ctx
	return p.initErr
}

func (p *fakePlugin) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.closeCalls++
	return p.closeErr
}

func (p *fakePlugin) Migrations(provider string) []migrationsmodule.Migration {
	return p.providerMigrations[provider]
}

func (p *fakePlugin) DependsOn() []string { return p.dependsOn }

func (p *fakePlugin) stats() (initCalls, closeCalls int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.initCalls, p.closeCalls
}

func (p *fakePlugin) context() *models.PluginContext {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.initCtx
}

type enabledPluginConfig struct {
	Enabled bool `json:"enabled"`
}

func configWithPlugins(provider string, pluginsConfig models.PluginsConfig) *models.Config {
	return &models.Config{
		Database: models.DatabaseConfig{Provider: provider},
		Plugins:  pluginsConfig,
	}
}

func newSQLiteRegistry(t *testing.T) (*plugins.PluginRegistry, *bun.DB, *migrationmanager.Manager) {
	t.Helper()

	db := internaltests.NewSQLiteIntegrationDB(t)
	migrator, err := migrationsmodule.NewMigrator(db, &internaltests.MockLogger{})
	require.NoError(t, err)

	manager := migrationmanager.NewManager(migrator)
	registry := plugins.NewPluginRegistry(
		configWithPlugins("sqlite", models.PluginsConfig{}),
		&internaltests.MockLogger{},
		db,
		manager,
		plugins.NewServiceRegistry(),
		&internaltests.MockEventBus{},
	)

	return registry, db, manager
}

func sqliteTableMigrations(version, table string) []migrationsmodule.Migration {
	return []migrationsmodule.Migration{
		{
			Version: version,
			Up: func(ctx context.Context, tx bun.Tx) error {
				return migrationsmodule.ExecStatements(ctx, tx, `CREATE TABLE `+table+` (id TEXT PRIMARY KEY);`)
			},
			Down: func(ctx context.Context, tx bun.Tx) error {
				return migrationsmodule.ExecStatements(ctx, tx, `DROP TABLE IF EXISTS `+table+`;`)
			},
		},
	}
}

func assertTableExists(t *testing.T, ctx context.Context, db *bun.DB, table string) {
	t.Helper()

	var count int
	err := db.NewSelect().TableExpr(table).ColumnExpr("COUNT(*)").Scan(ctx, &count)
	require.NoError(t, err)
}

func assertTableMissing(t *testing.T, ctx context.Context, db *bun.DB, table string) {
	t.Helper()

	var count int
	err := db.NewSelect().TableExpr(table).ColumnExpr("COUNT(*)").Scan(ctx, &count)
	require.Error(t, err)
}

func TestPluginRegistry_Register(t *testing.T) {
	t.Parallel()

	t.Run("preserves registration order", func(t *testing.T) {
		t.Parallel()

		registry := plugins.NewPluginRegistry(nil, &internaltests.MockLogger{}, nil, nil, nil, nil)
		alpha := newFakePlugin("alpha")
		beta := newFakePlugin("beta")

		require.NoError(t, registry.Register(alpha))
		require.NoError(t, registry.Register(beta))

		assert.Equal(t, []models.Plugin{alpha, beta}, registry.Plugins())
	})

	t.Run("rejects duplicate plugin IDs", func(t *testing.T) {
		t.Parallel()

		registry := plugins.NewPluginRegistry(nil, &internaltests.MockLogger{}, nil, nil, nil, nil)

		require.NoError(t, registry.Register(newFakePlugin("alpha")))
		err := registry.Register(newFakePlugin("alpha"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), `plugin with ID "alpha" is already registered`)
		assert.Len(t, registry.Plugins(), 1)
	})
}

func TestPluginRegistry_InitAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    *models.Config
		pluginIDs []string
		configure func(plugin *fakePlugin)
		wantInits map[string]int
		wantErr   error
	}{
		{
			name: "initializes enabled plugins only",
			config: configWithPlugins("sqlite", models.PluginsConfig{
				"alpha": map[string]any{"enabled": true},
				"beta":  map[string]any{"enabled": false},
				"gamma": map[string]any{},
			}),
			pluginIDs: []string{"alpha", "beta", "gamma"},
			wantInits: map[string]int{"alpha": 1, "beta": 0, "gamma": 1},
		},
		{
			name: "honours pre-parsed typed plugin configs",
			config: &models.Config{
				PreParsedConfigs: map[string]any{
					"alpha": enabledPluginConfig{Enabled: true},
					"beta":  enabledPluginConfig{Enabled: false},
				},
			},
			pluginIDs: []string{"alpha", "beta"},
			wantInits: map[string]int{"alpha": 1, "beta": 0},
		},
		{
			name:      "nil config disables every plugin",
			config:    nil,
			pluginIDs: []string{"alpha"},
			wantInits: map[string]int{"alpha": 0},
		},
		{
			name: "returns the initialization error and stops",
			config: configWithPlugins("sqlite", models.PluginsConfig{
				"alpha": map[string]any{"enabled": true},
				"beta":  map[string]any{"enabled": true},
			}),
			pluginIDs: []string{"alpha", "beta"},
			configure: func(plugin *fakePlugin) {
				if plugin.Metadata().ID == "alpha" {
					plugin.initErr = errInitFailed
				}
			},
			wantInits: map[string]int{"alpha": 1, "beta": 0},
			wantErr:   errInitFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := plugins.NewPluginRegistry(tt.config, &internaltests.MockLogger{}, nil, nil, nil, nil)
			registered := make(map[string]*fakePlugin, len(tt.pluginIDs))
			for _, pluginID := range tt.pluginIDs {
				plugin := newFakePlugin(pluginID)
				if tt.configure != nil {
					tt.configure(plugin)
				}
				require.NoError(t, registry.Register(plugin))
				registered[pluginID] = plugin
			}

			err := registry.InitAll()

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			for pluginID, wantInits := range tt.wantInits {
				initCalls, _ := registered[pluginID].stats()
				assert.Equal(t, wantInits, initCalls, "plugin %q init calls", pluginID)
			}
		})
	}
}

var errInitFailed = errors.New("init failed")

func TestPluginRegistry_InitAllWiresPluginContext(t *testing.T) {
	t.Parallel()

	db := internaltests.NewSQLiteIntegrationDB(t)
	logger := &internaltests.MockLogger{}
	serviceRegistry := plugins.NewServiceRegistry()
	eventBus := &internaltests.MockEventBus{}
	cfg := configWithPlugins("sqlite", models.PluginsConfig{"alpha": map[string]any{"enabled": true}})
	registry := plugins.NewPluginRegistry(cfg, logger, db, nil, serviceRegistry, eventBus)
	plugin := newFakePlugin("alpha")
	require.NoError(t, registry.Register(plugin))

	require.NoError(t, registry.InitAll())

	initCtx := plugin.context()
	require.NotNil(t, initCtx)
	assert.Same(t, db, initCtx.DB)
	assert.Same(t, logger, initCtx.Logger)
	assert.Same(t, serviceRegistry, initCtx.ServiceRegistry)
	assert.Same(t, eventBus, initCtx.EventBus)
	assert.Same(t, registry, initCtx.PluginRegistry)
	require.NotNil(t, initCtx.GetConfig)
	assert.Same(t, cfg, initCtx.GetConfig())
}

func TestPluginRegistry_SetConfigProvider(t *testing.T) {
	t.Parallel()

	first := configWithPlugins("sqlite", models.PluginsConfig{"alpha": map[string]any{"enabled": true}})
	second := configWithPlugins("sqlite", models.PluginsConfig{"alpha": map[string]any{"enabled": false}})
	registry := plugins.NewPluginRegistry(first, &internaltests.MockLogger{}, nil, nil, nil, nil)
	plugin := newFakePlugin("alpha")
	require.NoError(t, registry.Register(plugin))

	assert.Same(t, first, registry.GetConfig())

	registry.SetConfigProvider(func() *models.Config { return second })
	assert.Same(t, second, registry.GetConfig())

	require.NoError(t, registry.InitAll())
	initCalls, _ := plugin.stats()
	assert.Zero(t, initCalls)
}

func TestPluginRegistry_SetConfigProviderIgnoresNil(t *testing.T) {
	t.Parallel()

	cfg := configWithPlugins("sqlite", nil)
	registry := plugins.NewPluginRegistry(cfg, &internaltests.MockLogger{}, nil, nil, nil, nil)

	registry.SetConfigProvider(nil)

	assert.Same(t, cfg, registry.GetConfig())
}

func TestPluginRegistry_RunMigrationsWithoutManager(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("nil migration manager is a no-op", func(t *testing.T) {
		t.Parallel()

		registry := plugins.NewPluginRegistry(configWithPlugins("sqlite", nil), &internaltests.MockLogger{}, nil, nil, nil, nil)
		require.NoError(t, registry.RunMigrations(ctx))
	})

	t.Run("manager without migrator is a no-op", func(t *testing.T) {
		t.Parallel()

		registry := plugins.NewPluginRegistry(
			configWithPlugins("sqlite", nil),
			&internaltests.MockLogger{},
			nil,
			migrationmanager.NewManager(nil),
			nil,
			nil,
		)
		require.NoError(t, registry.RunMigrations(ctx))
	})

	t.Run("nil config returns an error", func(t *testing.T) {
		t.Parallel()

		registry, _, _ := newSQLiteRegistry(t)
		registry.SetConfigProvider(func() *models.Config { return nil })

		err := registry.RunMigrations(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "config provider returned nil config")
	})
}

func TestPluginRegistry_RunMigrationsAppliesEnabledPlugins(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry, db, _ := newSQLiteRegistry(t)
	registry.SetConfigProvider(func() *models.Config {
		return configWithPlugins("sqlite", models.PluginsConfig{
			"widgets":  map[string]any{"enabled": true},
			"disabled": map[string]any{"enabled": false},
		})
	})
	require.NoError(t, registry.Register(newMigratableFakePlugin("widgets", sqliteTableMigrations("20260101000000_widgets", "plugin_items"))))
	require.NoError(t, registry.Register(newMigratableFakePlugin("disabled", sqliteTableMigrations("20260102000000_disabled", "disabled_items"))))

	require.NoError(t, registry.RunMigrations(ctx))

	assertTableExists(t, ctx, db, "plugin_items")
	assertTableMissing(t, ctx, db, "disabled_items")
}

func TestPluginRegistry_RunMigrationsSkipsDisabledPlugins(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry, db, _ := newSQLiteRegistry(t)
	registry.SetConfigProvider(func() *models.Config {
		return configWithPlugins("sqlite", models.PluginsConfig{
			"widgets": map[string]any{"enabled": false},
		})
	})
	require.NoError(t, registry.Register(newMigratableFakePlugin("widgets", sqliteTableMigrations("20260101000000_widgets", "plugin_items"))))

	require.NoError(t, registry.RunMigrations(ctx))

	assertTableMissing(t, ctx, db, "plugin_items")
}

func TestPluginRegistry_DropMigrationsWithoutManager(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("nil migration manager is a no-op", func(t *testing.T) {
		t.Parallel()

		registry := plugins.NewPluginRegistry(configWithPlugins("sqlite", nil), &internaltests.MockLogger{}, nil, nil, nil, nil)
		require.NoError(t, registry.DropMigrations(ctx))
	})

	t.Run("manager without migrator is a no-op", func(t *testing.T) {
		t.Parallel()

		registry := plugins.NewPluginRegistry(
			configWithPlugins("sqlite", nil),
			&internaltests.MockLogger{},
			nil,
			migrationmanager.NewManager(nil),
			nil,
			nil,
		)
		require.NoError(t, registry.DropMigrations(ctx))
	})

	t.Run("nil config returns an error", func(t *testing.T) {
		t.Parallel()

		registry, _, _ := newSQLiteRegistry(t)
		registry.SetConfigProvider(func() *models.Config { return nil })

		err := registry.DropMigrations(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "config provider returned nil config")
	})
}

func TestPluginRegistry_DropMigrationsIgnoresEnablement(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry, db, _ := newSQLiteRegistry(t)
	registry.SetConfigProvider(func() *models.Config {
		return configWithPlugins("sqlite", models.PluginsConfig{
			"widgets": map[string]any{"enabled": true},
		})
	})
	require.NoError(t, registry.Register(newMigratableFakePlugin("widgets", sqliteTableMigrations("20260101000000_widgets", "plugin_items"))))
	require.NoError(t, registry.RunMigrations(ctx))
	assertTableExists(t, ctx, db, "plugin_items")

	registry.SetConfigProvider(func() *models.Config {
		return configWithPlugins("sqlite", models.PluginsConfig{
			"widgets": map[string]any{"enabled": false},
		})
	})
	require.NoError(t, registry.DropMigrations(ctx))

	assertTableMissing(t, ctx, db, "plugin_items")
}

func TestPluginRegistry_CloseAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		closeErr   bool
		wantErrors int
	}{
		{name: "closes every plugin"},
		{name: "continues after a close error", closeErr: true, wantErrors: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := &internaltests.RecordingLogger{}
			registry := plugins.NewPluginRegistry(nil, logger, nil, nil, nil, nil)
			alpha := newFakePlugin("alpha")
			beta := newFakePlugin("beta")
			if tt.closeErr {
				alpha.closeErr = errors.New("close failed")
			}
			require.NoError(t, registry.Register(alpha))
			require.NoError(t, registry.Register(beta))

			registry.CloseAll()

			_, alphaCloses := alpha.stats()
			_, betaCloses := beta.stats()
			assert.Equal(t, 1, alphaCloses)
			assert.Equal(t, 1, betaCloses)
			assert.Len(t, logger.ErrorCalls, tt.wantErrors)
		})
	}
}

func TestPluginRegistry_GetPlugin(t *testing.T) {
	t.Parallel()

	registry := plugins.NewPluginRegistry(nil, &internaltests.MockLogger{}, nil, nil, nil, nil)
	alpha := newFakePlugin("alpha")
	require.NoError(t, registry.Register(alpha))

	assert.Same(t, alpha, registry.GetPlugin("alpha"))
	assert.Nil(t, registry.GetPlugin("missing"))
}

func TestPluginRegistry_GetConfigReflectsRegisteredConfig(t *testing.T) {
	t.Parallel()

	cfg := configWithPlugins("postgres", nil)
	registry := plugins.NewPluginRegistry(cfg, &internaltests.MockLogger{}, nil, nil, nil, nil)

	assert.Same(t, cfg, registry.GetConfig())
}
