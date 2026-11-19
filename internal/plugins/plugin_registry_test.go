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
	initOrder          *[]string
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
	if p.initOrder != nil {
		*p.initOrder = append(*p.initOrder, p.metadata.ID)
	}
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

// fakePluginWithoutDependencies deliberately omits PluginWithDependencies so
// tests can exercise plugins that are independent by definition.
type fakePluginWithoutDependencies struct {
	metadata  models.PluginMetadata
	initCalls int
}

func newFakePluginWithoutDependencies(id string) *fakePluginWithoutDependencies {
	return &fakePluginWithoutDependencies{metadata: models.PluginMetadata{ID: id, Version: "1.0.0"}}
}

func (p *fakePluginWithoutDependencies) Metadata() models.PluginMetadata { return p.metadata }
func (p *fakePluginWithoutDependencies) Config() any                     { return nil }
func (p *fakePluginWithoutDependencies) Close() error                    { return nil }

func (p *fakePluginWithoutDependencies) Init(*models.PluginContext) error {
	p.initCalls++
	return nil
}

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

func pluginIDs(registered []models.Plugin) []string {
	ids := make([]string, 0, len(registered))
	for _, plugin := range registered {
		ids = append(ids, plugin.Metadata().ID)
	}
	return ids
}

func TestPluginRegistry_Register(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pluginIDs []string
		wantIDs   []string
		wantErr   string
	}{
		{
			name:      "preserves registration order",
			pluginIDs: []string{"alpha", "beta"},
			wantIDs:   []string{"alpha", "beta"},
		},
		{
			name:      "rejects duplicate plugin IDs",
			pluginIDs: []string{"alpha", "alpha"},
			wantIDs:   []string{"alpha"},
			wantErr:   `plugin with ID "alpha" is already registered`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := plugins.NewPluginRegistry(nil, &internaltests.MockLogger{}, nil, nil, nil, nil)

			var err error
			for _, pluginID := range tt.pluginIDs {
				if err = registry.Register(newFakePlugin(pluginID)); err != nil {
					break
				}
			}

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantIDs, pluginIDs(registry.Plugins()))
		})
	}
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

func TestPluginRegistry_InitAllDependencyValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		config     *models.Config
		dependsOn  map[string][]string
		registered []string
		wantErr    string
	}{
		{
			name: "enabled plugin with disabled dependency",
			config: configWithPlugins("sqlite", models.PluginsConfig{
				"alpha": map[string]any{"enabled": true},
				"beta":  map[string]any{"enabled": false},
			}),
			dependsOn:  map[string][]string{"alpha": {"beta"}},
			registered: []string{"alpha", "beta"},
			wantErr:    `plugin "alpha" requires plugin "beta" to be enabled`,
		},
		{
			name: "enabled plugin with unregistered dependency",
			config: configWithPlugins("sqlite", models.PluginsConfig{
				"alpha": map[string]any{"enabled": true},
			}),
			dependsOn:  map[string][]string{"alpha": {"missing"}},
			registered: []string{"alpha"},
			wantErr:    `plugin "alpha" requires plugin "missing" to be enabled`,
		},
		{
			name: "core dependency is ignored",
			config: configWithPlugins("sqlite", models.PluginsConfig{
				"alpha": map[string]any{"enabled": true},
			}),
			dependsOn:  map[string][]string{"alpha": {migrationsmodule.CorePluginID}},
			registered: []string{"alpha"},
		},
		{
			name: "dependency cycle is detected",
			config: configWithPlugins("sqlite", models.PluginsConfig{
				"alpha": map[string]any{"enabled": true},
				"beta":  map[string]any{"enabled": true},
			}),
			dependsOn:  map[string][]string{"alpha": {"beta"}, "beta": {"alpha"}},
			registered: []string{"alpha", "beta"},
			wantErr:    "plugin dependency cycle detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := plugins.NewPluginRegistry(tt.config, &internaltests.MockLogger{}, nil, nil, nil, nil)
			registered := make(map[string]*fakePlugin, len(tt.registered))
			for _, pluginID := range tt.registered {
				plugin := newFakePlugin(pluginID)
				plugin.dependsOn = tt.dependsOn[pluginID]
				require.NoError(t, registry.Register(plugin))
				registered[pluginID] = plugin
			}

			err := registry.InitAll()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				for pluginID, plugin := range registered {
					initCalls, _ := plugin.stats()
					assert.Zero(t, initCalls, "plugin %q must not be initialized when validation fails", pluginID)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestPluginRegistry_InitAllInitializesDependenciesFirst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		registered []string
		dependsOn  map[string][]string
		wantOrder  []string
	}{
		{
			name:       "reverse registration order still initializes dependencies first",
			registered: []string{"oauth2", "jwt"},
			dependsOn:  map[string][]string{"oauth2": {"jwt"}},
			wantOrder:  []string{"jwt", "oauth2"},
		},
		{
			name:       "resolves a dependency chain",
			registered: []string{"alpha", "beta", "gamma"},
			dependsOn:  map[string][]string{"alpha": {"beta"}, "beta": {"gamma"}},
			wantOrder:  []string{"gamma", "beta", "alpha"},
		},
		{
			name:       "registration order breaks ties",
			registered: []string{"beta", "alpha"},
			dependsOn:  map[string][]string{},
			wantOrder:  []string{"beta", "alpha"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pluginConfig := make(models.PluginsConfig, len(tt.registered))
			for _, pluginID := range tt.registered {
				pluginConfig[pluginID] = map[string]any{"enabled": true}
			}
			registry := plugins.NewPluginRegistry(configWithPlugins("sqlite", pluginConfig), &internaltests.MockLogger{}, nil, nil, nil, nil)

			var initOrder []string
			for _, pluginID := range tt.registered {
				plugin := newFakePlugin(pluginID)
				plugin.dependsOn = tt.dependsOn[pluginID]
				plugin.initOrder = &initOrder
				require.NoError(t, registry.Register(plugin))
			}

			require.NoError(t, registry.InitAll())
			assert.Equal(t, tt.wantOrder, initOrder)
		})
	}
}

func TestPluginRegistry_InitAllIgnoresInvalidDependencyEntries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		registered []string
		dependsOn  map[string][]string
		wantOrder  []string
	}{
		{
			name:       "duplicate dependency entries are ignored",
			registered: []string{"alpha", "beta"},
			dependsOn:  map[string][]string{"alpha": {"beta", "beta", "beta"}},
			wantOrder:  []string{"beta", "alpha"},
		},
		{
			name:       "empty and whitespace-only entries are ignored",
			registered: []string{"alpha", "beta"},
			dependsOn:  map[string][]string{"alpha": {"", "   ", "\t\n", migrationsmodule.CorePluginID}},
			wantOrder:  []string{"alpha", "beta"},
		},
		{
			name:       "dependency entries are trimmed before resolution",
			registered: []string{"alpha", "beta"},
			dependsOn:  map[string][]string{"alpha": {" beta "}},
			wantOrder:  []string{"beta", "alpha"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pluginConfig := make(models.PluginsConfig, len(tt.registered))
			for _, pluginID := range tt.registered {
				pluginConfig[pluginID] = map[string]any{"enabled": true}
			}
			registry := plugins.NewPluginRegistry(configWithPlugins("sqlite", pluginConfig), &internaltests.MockLogger{}, nil, nil, nil, nil)

			var initOrder []string
			for _, pluginID := range tt.registered {
				plugin := newFakePlugin(pluginID)
				plugin.dependsOn = tt.dependsOn[pluginID]
				plugin.initOrder = &initOrder
				require.NoError(t, registry.Register(plugin))
			}

			require.NoError(t, registry.InitAll())
			assert.Equal(t, tt.wantOrder, initOrder)
		})
	}
}

func TestPluginRegistry_InitAllDisabledPluginDependenciesDoNotBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		registered []string
		dependsOn  map[string][]string
		enabled    map[string]bool
		wantOrder  []string
	}{
		{
			name:       "disabled plugin depending on an unregistered plugin",
			registered: []string{"alpha", "omega"},
			dependsOn:  map[string][]string{"omega": {"missing"}},
			enabled:    map[string]bool{"alpha": true, "omega": false},
			wantOrder:  []string{"alpha"},
		},
		{
			name:       "disabled plugin depending on a disabled plugin",
			registered: []string{"alpha", "omega", "sigma"},
			dependsOn:  map[string][]string{"omega": {"sigma"}},
			enabled:    map[string]bool{"alpha": true, "omega": false, "sigma": false},
			wantOrder:  []string{"alpha"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pluginConfig := make(models.PluginsConfig, len(tt.registered))
			for _, pluginID := range tt.registered {
				pluginConfig[pluginID] = map[string]any{"enabled": tt.enabled[pluginID]}
			}
			registry := plugins.NewPluginRegistry(configWithPlugins("sqlite", pluginConfig), &internaltests.MockLogger{}, nil, nil, nil, nil)

			var initOrder []string
			for _, pluginID := range tt.registered {
				plugin := newFakePlugin(pluginID)
				plugin.dependsOn = tt.dependsOn[pluginID]
				plugin.initOrder = &initOrder
				require.NoError(t, registry.Register(plugin))
			}

			require.NoError(t, registry.InitAll())
			assert.Equal(t, tt.wantOrder, initOrder)
		})
	}
}

func TestPluginRegistry_InitAllDependencyCycleError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		registered []string
		dependsOn  map[string][]string
		wantErr    string
	}{
		{
			name:       "lists every plugin stuck in the cycle alphabetically",
			registered: []string{"delta", "charlie", "bravo"},
			dependsOn: map[string][]string{
				"bravo":   {"delta"},
				"charlie": {"bravo"},
				"delta":   {"charlie"},
			},
			wantErr: `plugin dependency cycle detected among "bravo", "charlie", "delta"`,
		},
		{
			name:       "includes plugins that depend on the cycle",
			registered: []string{"zulu", "alpha", "charlie", "delta"},
			dependsOn: map[string][]string{
				"alpha":   {"delta"},
				"charlie": {"delta"},
				"delta":   {"charlie"},
			},
			wantErr: `plugin dependency cycle detected among "alpha", "charlie", "delta"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pluginConfig := make(models.PluginsConfig, len(tt.registered))
			for _, pluginID := range tt.registered {
				pluginConfig[pluginID] = map[string]any{"enabled": true}
			}
			registry := plugins.NewPluginRegistry(configWithPlugins("sqlite", pluginConfig), &internaltests.MockLogger{}, nil, nil, nil, nil)
			registered := make([]*fakePlugin, 0, len(tt.registered))

			for _, pluginID := range tt.registered {
				plugin := newFakePlugin(pluginID)
				plugin.dependsOn = tt.dependsOn[pluginID]
				require.NoError(t, registry.Register(plugin))
				registered = append(registered, plugin)
			}

			err := registry.InitAll()

			assert.EqualError(t, err, tt.wantErr)
			for _, plugin := range registered {
				initCalls, _ := plugin.stats()
				assert.Zero(t, initCalls, "plugin %q must not be initialized when ordering fails", plugin.Metadata().ID)
			}
		})
	}
}

func TestPluginRegistry_InitAllSupportsPluginsWithoutDependencies(t *testing.T) {
	t.Parallel()

	pluginConfig := models.PluginsConfig{
		"planner": map[string]any{"enabled": true},
		"oauth2":  map[string]any{"enabled": true},
		"jwt":     map[string]any{"enabled": true},
	}
	registry := plugins.NewPluginRegistry(configWithPlugins("sqlite", pluginConfig), &internaltests.MockLogger{}, nil, nil, nil, nil)

	planner := newFakePluginWithoutDependencies("planner")
	oauth2 := newFakePlugin("oauth2")
	oauth2.dependsOn = []string{"jwt"}
	jwtPlugin := newFakePlugin("jwt")

	require.NoError(t, registry.Register(planner))
	require.NoError(t, registry.Register(oauth2))
	require.NoError(t, registry.Register(jwtPlugin))

	require.NoError(t, registry.InitAll())

	assert.Equal(t, 1, planner.initCalls)
	oauth2Inits, _ := oauth2.stats()
	jwtInits, _ := jwtPlugin.stats()
	assert.Equal(t, 1, oauth2Inits)
	assert.Equal(t, 1, jwtInits)
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

	tests := []struct {
		name          string
		provider      func() *models.Config
		wantConfig    *models.Config
		wantInitCalls int
	}{
		{
			name:          "replaces the config provider",
			provider:      func() *models.Config { return second },
			wantConfig:    second,
			wantInitCalls: 0,
		},
		{
			name:          "ignores a nil provider",
			provider:      nil,
			wantConfig:    first,
			wantInitCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := plugins.NewPluginRegistry(first, &internaltests.MockLogger{}, nil, nil, nil, nil)
			plugin := newFakePlugin("alpha")
			require.NoError(t, registry.Register(plugin))

			registry.SetConfigProvider(tt.provider)

			assert.Same(t, tt.wantConfig, registry.GetConfig())
			require.NoError(t, registry.InitAll())
			initCalls, _ := plugin.stats()
			assert.Equal(t, tt.wantInitCalls, initCalls)
		})
	}
}

func TestPluginRegistry_RunMigrationsWithoutManager(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		newRegistry func(t *testing.T) *plugins.PluginRegistry
		wantErr     string
	}{
		{
			name: "nil migration manager is a no-op",
			newRegistry: func(t *testing.T) *plugins.PluginRegistry {
				return plugins.NewPluginRegistry(configWithPlugins("sqlite", nil), &internaltests.MockLogger{}, nil, nil, nil, nil)
			},
		},
		{
			name: "manager without migrator is a no-op",
			newRegistry: func(t *testing.T) *plugins.PluginRegistry {
				return plugins.NewPluginRegistry(
					configWithPlugins("sqlite", nil),
					&internaltests.MockLogger{},
					nil,
					migrationmanager.NewManager(nil),
					nil,
					nil,
				)
			},
		},
		{
			name: "nil config returns an error",
			newRegistry: func(t *testing.T) *plugins.PluginRegistry {
				registry, _, _ := newSQLiteRegistry(t)
				registry.SetConfigProvider(func() *models.Config { return nil })
				return registry
			},
			wantErr: "config provider returned nil config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := tt.newRegistry(t)

			err := registry.RunMigrations(context.Background())

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPluginRegistry_RunMigrations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		pluginsConfig     models.PluginsConfig
		plugins           []*fakePlugin
		wantTables        []string
		wantMissingTables []string
	}{
		{
			name: "applies migrations for enabled plugins only",
			pluginsConfig: models.PluginsConfig{
				"widgets":  map[string]any{"enabled": true},
				"disabled": map[string]any{"enabled": false},
			},
			plugins: []*fakePlugin{
				newMigratableFakePlugin("widgets", sqliteTableMigrations("20260101000000_widgets", "plugin_items")),
				newMigratableFakePlugin("disabled", sqliteTableMigrations("20260102000000_disabled", "disabled_items")),
			},
			wantTables:        []string{"plugin_items"},
			wantMissingTables: []string{"disabled_items"},
		},
		{
			name: "skips migrations for disabled plugins",
			pluginsConfig: models.PluginsConfig{
				"widgets": map[string]any{"enabled": false},
			},
			plugins: []*fakePlugin{
				newMigratableFakePlugin("widgets", sqliteTableMigrations("20260101000000_widgets", "plugin_items")),
			},
			wantMissingTables: []string{"plugin_items"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			registry, db, _ := newSQLiteRegistry(t)
			registry.SetConfigProvider(func() *models.Config {
				return configWithPlugins("sqlite", tt.pluginsConfig)
			})
			for _, plugin := range tt.plugins {
				require.NoError(t, registry.Register(plugin))
			}

			require.NoError(t, registry.RunMigrations(ctx))

			for _, table := range tt.wantTables {
				assertTableExists(t, ctx, db, table)
			}
			for _, table := range tt.wantMissingTables {
				assertTableMissing(t, ctx, db, table)
			}
		})
	}
}

func TestPluginRegistry_DropMigrationsWithoutManager(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		newRegistry func(t *testing.T) *plugins.PluginRegistry
		wantErr     string
	}{
		{
			name: "nil migration manager is a no-op",
			newRegistry: func(t *testing.T) *plugins.PluginRegistry {
				return plugins.NewPluginRegistry(configWithPlugins("sqlite", nil), &internaltests.MockLogger{}, nil, nil, nil, nil)
			},
		},
		{
			name: "manager without migrator is a no-op",
			newRegistry: func(t *testing.T) *plugins.PluginRegistry {
				return plugins.NewPluginRegistry(
					configWithPlugins("sqlite", nil),
					&internaltests.MockLogger{},
					nil,
					migrationmanager.NewManager(nil),
					nil,
					nil,
				)
			},
		},
		{
			name: "nil config returns an error",
			newRegistry: func(t *testing.T) *plugins.PluginRegistry {
				registry, _, _ := newSQLiteRegistry(t)
				registry.SetConfigProvider(func() *models.Config { return nil })
				return registry
			},
			wantErr: "config provider returned nil config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := tt.newRegistry(t)

			err := registry.DropMigrations(context.Background())

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
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

	tests := []struct {
		name      string
		pluginID  string
		wantFound bool
	}{
		{name: "returns the registered plugin", pluginID: "alpha", wantFound: true},
		{name: "returns nil for an unknown plugin", pluginID: "missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := plugins.NewPluginRegistry(nil, &internaltests.MockLogger{}, nil, nil, nil, nil)
			alpha := newFakePlugin("alpha")
			require.NoError(t, registry.Register(alpha))

			got := registry.GetPlugin(tt.pluginID)

			if tt.wantFound {
				assert.Same(t, alpha, got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestPluginRegistry_GetConfigReflectsRegisteredConfig(t *testing.T) {
	t.Parallel()

	cfg := configWithPlugins("postgres", nil)
	registry := plugins.NewPluginRegistry(cfg, &internaltests.MockLogger{}, nil, nil, nil, nil)

	assert.Same(t, cfg, registry.GetConfig())
}
