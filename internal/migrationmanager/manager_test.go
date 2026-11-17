package migrationmanager_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/internal/migrationmanager"
	internaltests "github.com/Authula/authula/internal/tests"
	migrationsmodule "github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
)

// fakePlugin is the smallest valid models.Plugin. It intentionally provides no
// migrations so tests can assert that non-migratable plugins are skipped.
type fakePlugin struct {
	metadata models.PluginMetadata
}

func newFakePlugin(id string) *fakePlugin {
	return &fakePlugin{metadata: models.PluginMetadata{ID: id, Version: "1.0.0"}}
}

func (p *fakePlugin) Metadata() models.PluginMetadata    { return p.metadata }
func (p *fakePlugin) Config() any                        { return nil }
func (p *fakePlugin) Init(_ *models.PluginContext) error { return nil }
func (p *fakePlugin) Close() error                       { return nil }

// fakeMigratablePlugin adds the PluginWithMigrations capability.
type fakeMigratablePlugin struct {
	*fakePlugin
	providerMigrations map[string][]migrationsmodule.Migration
	dependsOn          []string
}

func newFakeMigratablePlugin(id string, dependsOn []string, providerMigrations map[string][]migrationsmodule.Migration) *fakeMigratablePlugin {
	return &fakeMigratablePlugin{
		fakePlugin:         newFakePlugin(id),
		providerMigrations: providerMigrations,
		dependsOn:          dependsOn,
	}
}

func (p *fakeMigratablePlugin) Migrations(provider string) []migrationsmodule.Migration {
	return p.providerMigrations[provider]
}

func (p *fakeMigratablePlugin) DependsOn() []string { return p.dependsOn }

func pluginWithID(id string) models.Plugin {
	return newFakePlugin(id)
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

func TestManager_Migrator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		withMigrator bool
		nilReceiver  bool
		wantNil      bool
	}{
		{
			name:         "returns the migrator the manager was built with",
			withMigrator: true,
		},
		{
			name:    "returns nil when built without a migrator",
			wantNil: true,
		},
		{
			name:        "returns nil for a nil manager",
			nilReceiver: true,
			wantNil:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var migrator *migrationsmodule.Migrator
			if tt.withMigrator {
				db := internaltests.NewSQLiteIntegrationDB(t)
				var err error
				migrator, err = migrationsmodule.NewMigrator(db, &internaltests.MockLogger{})
				require.NoError(t, err)
			}

			manager := migrationmanager.NewManager(migrator)
			if tt.nilReceiver {
				manager = nil
			}

			got := manager.Migrator()

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			assert.Same(t, migrator, got)
		})
	}
}

func TestManager_PlanPlugins(t *testing.T) {
	t.Parallel()

	widgetMigrations := []migrationsmodule.Migration{{Version: "20260101000000_widgets"}}
	auditMigrations := []migrationsmodule.Migration{
		{Version: "20260102000000_audit"},
		{Version: "20260103000000_audit_more"},
	}
	reportMigrations := []migrationsmodule.Migration{{Version: "20260104000000_reports"}}

	widgets := newFakeMigratablePlugin("widgets", []string{"core"}, map[string][]migrationsmodule.Migration{
		"sqlite": widgetMigrations,
	})
	audit := newFakeMigratablePlugin("audit", nil, map[string][]migrationsmodule.Migration{
		"sqlite": auditMigrations,
	})
	postgresOnly := newFakeMigratablePlugin("postgres_only", nil, map[string][]migrationsmodule.Migration{
		"postgres": reportMigrations,
	})
	reports := newFakeMigratablePlugin("reports", nil, map[string][]migrationsmodule.Migration{
		"sqlite": reportMigrations,
	})

	manager := migrationmanager.NewManager(nil)

	tests := []struct {
		name     string
		plugins  []models.Plugin
		selector migrationmanager.PluginSelector
		want     []migrationsmodule.MigrationSet
	}{
		{
			name: "empty plugin list returns nil",
		},
		{
			name:    "plugins without migrations are skipped",
			plugins: []models.Plugin{pluginWithID("plain")},
		},
		{
			name:    "plugin without migrations for the provider is skipped",
			plugins: []models.Plugin{postgresOnly},
		},
		{
			name:    "migratable plugin is planned with metadata and dependencies",
			plugins: []models.Plugin{widgets},
			want: []migrationsmodule.MigrationSet{
				{
					PluginID:   "widgets",
					DependsOn:  []string{"core"},
					Migrations: widgetMigrations,
				},
			},
		},
		{
			name:    "nil selector includes every migratable plugin in order",
			plugins: []models.Plugin{widgets, pluginWithID("plain"), audit},
			want: []migrationsmodule.MigrationSet{
				{PluginID: "widgets", DependsOn: []string{"core"}, Migrations: widgetMigrations},
				{PluginID: "audit", Migrations: auditMigrations},
			},
		},
		{
			name:     "selector filters plugins",
			plugins:  []models.Plugin{widgets, audit, reports},
			selector: migrationmanager.OnlyPluginIDs("audit"),
			want: []migrationsmodule.MigrationSet{
				{PluginID: "audit", Migrations: auditMigrations},
			},
		},
		{
			name:     "selector excluding everything yields nil",
			plugins:  []models.Plugin{widgets, audit},
			selector: migrationmanager.ExceptPluginIDs("widgets", "audit"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := manager.PlanPlugins("sqlite", tt.plugins, tt.selector)

			if len(tt.want) == 0 {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOnlyPluginIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ids       []string
		pluginIDs []string
		wantNil   bool
		want      []bool
	}{
		{
			name:    "no ids returns nil selector",
			wantNil: true,
		},
		{
			name:    "blank ids are ignored and return nil selector",
			ids:     []string{" ", "", "\t"},
			wantNil: true,
		},
		{
			name:      "matches trimmed ids",
			ids:       []string{" widgets ", "audit"},
			pluginIDs: []string{"widgets", "audit", "other"},
			want:      []bool{true, true, false},
		},
		{
			name:      "unknown id matches nothing",
			ids:       []string{"missing"},
			pluginIDs: []string{"widgets"},
			want:      []bool{false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			selector := migrationmanager.OnlyPluginIDs(tt.ids...)

			if tt.wantNil {
				assert.Nil(t, selector)
				return
			}

			require.NotNil(t, selector)
			for i, pluginID := range tt.pluginIDs {
				assert.Equal(t, tt.want[i], selector(pluginWithID(pluginID)), "plugin %q", pluginID)
			}
		})
	}
}

func TestExceptPluginIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ids       []string
		pluginIDs []string
		wantNil   bool
		want      []bool
	}{
		{
			name:    "no ids returns nil selector",
			wantNil: true,
		},
		{
			name:    "blank ids are ignored and return nil selector",
			ids:     []string{"", "  "},
			wantNil: true,
		},
		{
			name:      "excludes trimmed ids",
			ids:       []string{" widgets "},
			pluginIDs: []string{"widgets", "audit"},
			want:      []bool{false, true},
		},
		{
			name:      "unknown id excludes nothing",
			ids:       []string{"missing"},
			pluginIDs: []string{"widgets", "audit"},
			want:      []bool{true, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			selector := migrationmanager.ExceptPluginIDs(tt.ids...)

			if tt.wantNil {
				assert.Nil(t, selector)
				return
			}

			require.NotNil(t, selector)
			for i, pluginID := range tt.pluginIDs {
				assert.Equal(t, tt.want[i], selector(pluginWithID(pluginID)), "plugin %q", pluginID)
			}
		})
	}
}

func TestComposeSelectors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		selectors []migrationmanager.PluginSelector
		pluginIDs []string
		wantNil   bool
		want      []bool
	}{
		{
			name:    "no selectors returns nil",
			wantNil: true,
		},
		{
			name:      "all nil selectors return nil",
			selectors: []migrationmanager.PluginSelector{nil, nil},
			wantNil:   true,
		},
		{
			name: "nil selectors are skipped",
			selectors: []migrationmanager.PluginSelector{
				nil,
				migrationmanager.OnlyPluginIDs("widgets"),
			},
			pluginIDs: []string{"widgets", "audit"},
			want:      []bool{true, false},
		},
		{
			name: "only and except selectors compose",
			selectors: []migrationmanager.PluginSelector{
				migrationmanager.OnlyPluginIDs("widgets", "audit"),
				migrationmanager.ExceptPluginIDs("audit"),
			},
			pluginIDs: []string{"widgets", "audit", "reports"},
			want:      []bool{true, false, false},
		},
		{
			name: "short-circuits when a selector rejects",
			selectors: []migrationmanager.PluginSelector{
				func(models.Plugin) bool { return false },
				func(models.Plugin) bool { panic("selector called after a rejection") },
			},
			pluginIDs: []string{"widgets"},
			want:      []bool{false},
		},
		{
			name: "all selectors must match",
			selectors: []migrationmanager.PluginSelector{
				func(models.Plugin) bool { return true },
				func(models.Plugin) bool { return true },
			},
			pluginIDs: []string{"widgets"},
			want:      []bool{true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			selector := migrationmanager.ComposeSelectors(tt.selectors...)

			if tt.wantNil {
				assert.Nil(t, selector)
				return
			}

			require.NotNil(t, selector)
			for i, pluginID := range tt.pluginIDs {
				assert.Equal(t, tt.want[i], selector(pluginWithID(pluginID)), "plugin %q", pluginID)
			}
		})
	}
}

func TestManager_RequiresMigrator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	manager := migrationmanager.NewManager(nil)
	var nilManager *migrationmanager.Manager

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "RunCore",
			call: func() error { return manager.RunCore(ctx, "sqlite") },
		},
		{
			name: "DropCore",
			call: func() error { return manager.DropCore(ctx, "sqlite") },
		},
		{
			name: "RunPlugins",
			call: func() error { return manager.RunPlugins(ctx, "sqlite", []models.Plugin{pluginWithID("widgets")}, nil) },
		},
		{
			name: "DropPlugins",
			call: func() error { return manager.DropPlugins(ctx, "sqlite", []models.Plugin{pluginWithID("widgets")}, nil) },
		},
		{
			name: "nil receiver RunCore",
			call: func() error { return nilManager.RunCore(ctx, "sqlite") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.call()

			require.Error(t, err)
			assert.Contains(t, err.Error(), "migrator not initialized")
		})
	}
}

func TestManager_RunCoreAndDropCoreSQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := internaltests.NewSQLiteIntegrationDB(t)
	migrator, err := migrationsmodule.NewMigrator(db, &internaltests.MockLogger{})
	require.NoError(t, err)
	manager := migrationmanager.NewManager(migrator)

	err = manager.RunCore(ctx, "sqlite")
	require.NoError(t, err)
	assertTableExists(t, ctx, db, "users")

	err = manager.DropCore(ctx, "sqlite")
	require.NoError(t, err)
	assertTableMissing(t, ctx, db, "users")
}

func TestManager_CoreMigrationsRejectUnsupportedProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(ctx context.Context, manager *migrationmanager.Manager) error
	}{
		{
			name: "RunCore",
			call: func(ctx context.Context, manager *migrationmanager.Manager) error {
				return manager.RunCore(ctx, "oracle")
			},
		},
		{
			name: "DropCore",
			call: func(ctx context.Context, manager *migrationmanager.Manager) error {
				return manager.DropCore(ctx, "oracle")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := internaltests.NewSQLiteIntegrationDB(t)
			migrator, err := migrationsmodule.NewMigrator(db, &internaltests.MockLogger{})
			require.NoError(t, err)
			manager := migrationmanager.NewManager(migrator)

			err = tt.call(context.Background(), manager)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported database provider")
		})
	}
}

func TestManager_RunPluginsSQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := internaltests.NewSQLiteIntegrationDB(t)
	migrator, err := migrationsmodule.NewMigrator(db, &internaltests.MockLogger{})
	require.NoError(t, err)
	manager := migrationmanager.NewManager(migrator)

	widgets := newFakeMigratablePlugin("widgets", nil, map[string][]migrationsmodule.Migration{
		"sqlite": sqliteTableMigrations("20260101000000_widgets", "widget_items"),
	})
	plain := pluginWithID("plain")
	unselected := newFakeMigratablePlugin("reports", nil, map[string][]migrationsmodule.Migration{
		"sqlite": sqliteTableMigrations("20260102000000_reports", "report_items"),
	})
	plugins := []models.Plugin{plain, widgets, unselected}

	t.Run("plans nothing for plugins without migrations", func(t *testing.T) {
		require.NoError(t, manager.RunPlugins(ctx, "sqlite", []models.Plugin{plain}, nil))
	})

	t.Run("plans nothing when the selector rejects every plugin", func(t *testing.T) {
		require.NoError(t, manager.RunPlugins(ctx, "sqlite", plugins, migrationmanager.ExceptPluginIDs("widgets", "reports")))
		assertTableMissing(t, ctx, db, "widget_items")
		assertTableMissing(t, ctx, db, "report_items")
	})

	t.Run("applies migrations for selected plugins", func(t *testing.T) {
		require.NoError(t, manager.RunPlugins(ctx, "sqlite", plugins, migrationmanager.OnlyPluginIDs("widgets")))
		assertTableExists(t, ctx, db, "widget_items")
		assertTableMissing(t, ctx, db, "report_items")
	})

	t.Run("is idempotent for already applied migrations", func(t *testing.T) {
		require.NoError(t, manager.RunPlugins(ctx, "sqlite", plugins, migrationmanager.OnlyPluginIDs("widgets")))
		assertTableExists(t, ctx, db, "widget_items")
	})

	t.Run("rolls back selected plugin migrations", func(t *testing.T) {
		require.NoError(t, manager.DropPlugins(ctx, "sqlite", plugins, migrationmanager.OnlyPluginIDs("widgets")))
		assertTableMissing(t, ctx, db, "widget_items")
	})

	t.Run("rolls back nothing when no plugin matches", func(t *testing.T) {
		require.NoError(t, manager.DropPlugins(ctx, "sqlite", plugins, migrationmanager.OnlyPluginIDs("missing")))
	})

	t.Run("rolls back nothing for plugins without migrations", func(t *testing.T) {
		require.NoError(t, manager.DropPlugins(ctx, "sqlite", []models.Plugin{plain}, nil))
	})
}
