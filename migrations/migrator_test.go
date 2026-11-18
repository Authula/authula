package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"

	"github.com/Authula/authula/internal/sqlitedsn"
)

type testLogger struct {
	testing.TB
}

func (l testLogger) Debug(msg string, args ...any) {}
func (l testLogger) Info(msg string, args ...any)  {}
func (l testLogger) Warn(msg string, args ...any)  {}
func (l testLogger) Error(msg string, args ...any) {}

func newTestDB(t *testing.T) *bun.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	sqlDB, err := sql.Open("sqlite", sqlitedsn.Build(dsn))
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func TestMigrator_MigrateAndRollback(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	ctx := context.Background()

	migrator, err := NewMigrator(db, testLogger{t})
	assert.NoError(t, err)

	set := MigrationSet{
		PluginID: "test_plugin",
		Migrations: []Migration{
			{
				Version: "001_create_table",
				Up: func(ctx context.Context, tx bun.Tx) error {
					_, err := tx.ExecContext(ctx, `CREATE TABLE test_entities (id TEXT PRIMARY KEY);`)
					return err
				},
				Down: func(ctx context.Context, tx bun.Tx) error {
					_, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS test_entities;`)
					return err
				},
			},
		},
	}

	err = migrator.Migrate(ctx, []MigrationSet{set})
	assert.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO test_entities (id) VALUES (?)`, "abc")
	assert.NoError(t, err)

	err = migrator.RollbackAll(ctx, []MigrationSet{set})
	assert.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO test_entities (id) VALUES (?)`, "def")
	assert.Error(t, err)
}

func TestMigrator_DependencyOrdering(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	ctx := context.Background()

	migrator, err := NewMigrator(db, testLogger{t})
	assert.NoError(t, err)

	parent := MigrationSet{
		PluginID:  "parent",
		DependsOn: nil,
		Migrations: []Migration{
			{
				Version: "001_parent",
				Up: func(ctx context.Context, tx bun.Tx) error {
					_, err := tx.ExecContext(ctx, `CREATE TABLE parent_entities (id TEXT PRIMARY KEY);`)
					return err
				},
				Down: func(ctx context.Context, tx bun.Tx) error {
					_, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS parent_entities;`)
					return err
				},
			},
		},
	}

	child := MigrationSet{
		PluginID:  "child",
		DependsOn: []string{"parent"},
		Migrations: []Migration{
			{
				Version: "001_child",
				Up: func(ctx context.Context, tx bun.Tx) error {
					_, err := tx.ExecContext(ctx, `CREATE TABLE child_entities (id TEXT PRIMARY KEY, parent_id TEXT NOT NULL, FOREIGN KEY(parent_id) REFERENCES parent_entities(id));`)
					return err
				},
				Down: func(ctx context.Context, tx bun.Tx) error {
					_, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS child_entities;`)
					return err
				},
			},
		},
	}

	err = migrator.Migrate(ctx, []MigrationSet{child, parent})
	assert.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO parent_entities (id) VALUES (?)`, "parent-1")
	assert.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO child_entities (id, parent_id) VALUES (?, ?)`, "child-1", "parent-1")
	assert.NoError(t, err)

	err = migrator.RollbackAll(ctx, []MigrationSet{parent, child})
	assert.NoError(t, err)
}

func TestMigrator_DetectsCycles(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	ctx := context.Background()

	migrator, err := NewMigrator(db, testLogger{t})
	assert.NoError(t, err)

	setA := MigrationSet{PluginID: "a", DependsOn: []string{"b"}}
	setB := MigrationSet{PluginID: "b", DependsOn: []string{"a"}}

	err = migrator.Migrate(ctx, []MigrationSet{setA, setB})
	assert.Error(t, err)
}

func noopMigration(context.Context, bun.Tx) error {
	return nil
}

func withSchemaTable(table string) MigratorOption {
	return func(m *Migrator) {
		m.table = table
	}
}

func newTableMigration(version, table string) Migration {
	return Migration{
		Version: version,
		Up: func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s (id TEXT PRIMARY KEY);", table))
			return err
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s;", table))
			return err
		},
	}
}

func mustNewMigrator(t *testing.T, db *bun.DB, opts ...MigratorOption) *Migrator {
	t.Helper()

	migrator, err := NewMigrator(db, testLogger{t}, opts...)
	require.NoError(t, err)
	return migrator
}

func tableExistsInSchema(t *testing.T, db *bun.DB, table string) bool {
	t.Helper()

	var count int
	err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table,
	).Scan(&count)
	require.NoError(t, err)
	return count > 0
}

func TestNewMigrator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		nilDB     bool
		opts      []MigratorOption
		wantTable string
		wantErr   string
	}{
		{name: "applies defaults", wantTable: DefaultSchemaTable},
		{name: "applies custom table option", opts: []MigratorOption{withSchemaTable("custom_migrations")}, wantTable: "custom_migrations"},
		{name: "rejects a non bun DB", nilDB: true, wantErr: "migrator requires *bun.DB instance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var db bun.IDB
			if !tt.nilDB {
				db = newTestDB(t)
			}
			migrator, err := NewMigrator(db, testLogger{t}, tt.opts...)
			if tt.wantErr != "" {
				assert.Nil(t, migrator)
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, migrator)
			assert.Equal(t, tt.wantTable, migrator.table)
			assert.Equal(t, "sqlite", migrator.dialect)
		})
	}
}

func TestMigrator_createSchemaTable(t *testing.T) {
	t.Parallel()

	t.Run("is idempotent", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db)

		require.NoError(t, migrator.createSchemaTable(ctx))
		require.NoError(t, migrator.createSchemaTable(ctx))
		assert.True(t, tableExistsInSchema(t, db, DefaultSchemaTable))
	})

	t.Run("uses the configured table", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db, withSchemaTable("custom_migrations"))

		require.NoError(t, migrator.createSchemaTable(ctx))
		assert.True(t, tableExistsInSchema(t, db, "custom_migrations"))
		assert.False(t, tableExistsInSchema(t, db, DefaultSchemaTable))
	})

	t.Run("wraps creation errors", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		require.NoError(t, db.Close())
		migrator := mustNewMigrator(t, db)

		err := migrator.createSchemaTable(context.Background())
		require.Error(t, err)
		assert.ErrorContains(t, err, "create auth_schema_migrations table")
	})
}

func TestMigrator_tableExpr(t *testing.T) {
	t.Parallel()

	migrator := &Migrator{table: DefaultSchemaTable}

	tests := []struct {
		name      string
		withAlias bool
		wantExpr  string
	}{
		{name: "with alias", withAlias: true, wantExpr: "? AS schema_migration"},
		{name: "without alias", withAlias: false, wantExpr: "?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expr, args := migrator.tableExpr(tt.withAlias)
			assert.Equal(t, tt.wantExpr, expr)
			require.Len(t, args, 1)
			assert.Equal(t, bun.Ident(DefaultSchemaTable), args[0])
		})
	}
}

func TestExecStatements(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		statements        []string
		wantErr           string
		wantTables        []string
		wantMissingTables []string
	}{
		{
			name:       "skips blank statements",
			statements: []string{"", "   ", "\n\t", "  \n "},
		},
		{
			name: "executes multiple statements",
			statements: []string{
				"CREATE TABLE exec_statements_one (id TEXT);",
				"CREATE TABLE exec_statements_two (id TEXT);",
			},
			wantTables: []string{"exec_statements_one", "exec_statements_two"},
		},
		{
			name: "wraps errors and rolls back",
			statements: []string{
				"CREATE TABLE exec_statements_rolled_back (id TEXT);",
				"THIS IS NOT SQL",
			},
			wantErr:           "exec statement failed",
			wantMissingTables: []string{"exec_statements_rolled_back"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDB(t)
			err := db.RunInTx(context.Background(), nil, func(ctx context.Context, tx bun.Tx) error {
				return ExecStatements(ctx, tx, tt.statements...)
			})
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			for _, table := range tt.wantTables {
				assert.True(t, tableExistsInSchema(t, db, table))
			}
			for _, table := range tt.wantMissingTables {
				assert.False(t, tableExistsInSchema(t, db, table))
			}
		})
	}
}

func TestValidateSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		plugin  string
		wantErr string
	}{
		{name: "valid plugin ID", plugin: "plugin_a"},
		{name: "empty plugin ID", plugin: "", wantErr: "plugin ID cannot be empty"},
		{name: "whitespace plugin ID", plugin: " \t\n ", wantErr: "plugin ID cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateSet(MigrationSet{PluginID: tt.plugin})
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestValidateMigration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		migration Migration
		wantErr   string
	}{
		{
			name:      "valid migration",
			migration: Migration{Version: "001", Up: noopMigration},
		},
		{
			name:      "empty version",
			migration: Migration{Version: "", Up: noopMigration},
			wantErr:   "migration version cannot be empty for plugin plugin_a",
		},
		{
			name:      "whitespace version",
			migration: Migration{Version: " \t ", Up: noopMigration},
			wantErr:   "migration version cannot be empty for plugin plugin_a",
		},
		{
			name:      "missing Up function",
			migration: Migration{Version: "001"},
			wantErr:   "migration plugin_a:001 missing Up function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateMigration("plugin_a", tt.migration)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestFindMigration(t *testing.T) {
	t.Parallel()

	set := MigrationSet{
		PluginID: "plugin_a",
		Migrations: []Migration{
			newTableMigration("001", "find_migration_first"),
			newTableMigration("002", "find_migration_second"),
		},
	}

	tests := []struct {
		name    string
		set     MigrationSet
		version string
		wantOK  bool
	}{
		{name: "finds a known version", set: set, version: "002", wantOK: true},
		{name: "reports an unknown version", set: set, version: "999"},
		{name: "reports an empty set", set: MigrationSet{PluginID: "empty"}, version: "001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			migration, ok := findMigration(tt.set, tt.version)
			if !tt.wantOK {
				assert.False(t, ok)
				assert.Equal(t, Migration{}, migration)
				return
			}
			require.True(t, ok)
			assert.Equal(t, tt.version, migration.Version)
			assert.NotNil(t, migration.Up)
			assert.NotNil(t, migration.Down)
		})
	}
}

func TestMigrator_MigrateValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		sets    []MigrationSet
		wantErr string
	}{
		{
			name: "no sets is a no-op",
			sets: nil,
		},
		{
			name:    "empty plugin ID",
			sets:    []MigrationSet{{PluginID: " ", Migrations: []Migration{newTableMigration("001", "validate_plugin")}}},
			wantErr: "plugin ID cannot be empty",
		},
		{
			name:    "empty version",
			sets:    []MigrationSet{{PluginID: "plugin_a", Migrations: []Migration{{Up: noopMigration}}}},
			wantErr: "migration version cannot be empty for plugin plugin_a",
		},
		{
			name:    "missing Up function",
			sets:    []MigrationSet{{PluginID: "plugin_a", Migrations: []Migration{{Version: "001"}}}},
			wantErr: "migration plugin_a:001 missing Up function",
		},
		{
			name: "duplicate version",
			sets: []MigrationSet{{PluginID: "plugin_a", Migrations: []Migration{
				{Version: "001", Up: noopMigration},
				{Version: "001", Up: noopMigration},
			}}},
			wantErr: "duplicate migration version 001 for plugin plugin_a",
		},
		{
			name:    "missing dependency",
			sets:    []MigrationSet{{PluginID: "plugin_a", DependsOn: []string{"missing"}}},
			wantErr: "plugin plugin_a depends on missing which has no registered migrations",
		},
		{
			name: "duplicate plugin set",
			sets: []MigrationSet{
				{PluginID: "plugin_a"},
				{PluginID: "plugin_a"},
			},
			wantErr: "duplicate migration set for plugin plugin_a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDB(t)
			migrator := mustNewMigrator(t, db)

			err := migrator.Migrate(context.Background(), tt.sets)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestMigrator_MigrateIsNoOpWhenUpToDate(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	ctx := context.Background()
	migrator := mustNewMigrator(t, db)

	upCalls := 0
	set := MigrationSet{
		PluginID: "up_to_date_plugin",
		Migrations: []Migration{{
			Version: "001",
			Up: func(context.Context, bun.Tx) error {
				upCalls++
				return nil
			},
			Down: noopMigration,
		}},
	}

	require.NoError(t, migrator.Migrate(ctx, []MigrationSet{set}))
	require.NoError(t, migrator.Migrate(ctx, []MigrationSet{set}))
	assert.Equal(t, 1, upCalls)

	applied, err := migrator.ListApplied(ctx, "up_to_date_plugin")
	require.NoError(t, err)
	assert.Len(t, applied, 1)
}

func TestMigrator_MigrateFailureRollsBack(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	ctx := context.Background()
	migrator := mustNewMigrator(t, db)

	set := MigrationSet{
		PluginID: "rollback_failure_plugin",
		Migrations: []Migration{{
			Version: "001",
			Up: func(ctx context.Context, tx bun.Tx) error {
				if _, err := tx.ExecContext(ctx, "CREATE TABLE migrate_rollback (id TEXT);"); err != nil {
					return err
				}
				return errors.New("boom")
			},
			Down: noopMigration,
		}},
	}

	err := migrator.Migrate(ctx, []MigrationSet{set})
	require.Error(t, err)
	assert.ErrorContains(t, err, "up failed")

	assert.False(t, tableExistsInSchema(t, db, "migrate_rollback"))
	applied, err := migrator.ListApplied(ctx, "rollback_failure_plugin")
	require.NoError(t, err)
	assert.Empty(t, applied)
}

func TestMigrator_RollbackLast(t *testing.T) {
	t.Parallel()

	t.Run("no applied migrations", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		migrator := mustNewMigrator(t, db)
		set := MigrationSet{
			PluginID:   "empty_plugin",
			Migrations: []Migration{newTableMigration("001", "rollback_last_empty")},
		}

		require.NoError(t, migrator.RollbackLast(context.Background(), set))
		assert.False(t, tableExistsInSchema(t, db, "rollback_last_empty"))
	})

	t.Run("rolls back the only applied migration", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db)
		set := MigrationSet{
			PluginID:   "single_plugin",
			Migrations: []Migration{newTableMigration("001", "rollback_last_single")},
		}

		require.NoError(t, migrator.Migrate(ctx, []MigrationSet{set}))
		require.NoError(t, migrator.RollbackLast(ctx, set))

		assert.False(t, tableExistsInSchema(t, db, "rollback_last_single"))
		applied, err := migrator.ListApplied(ctx, "single_plugin")
		require.NoError(t, err)
		assert.Empty(t, applied)
	})

	t.Run("rolls back the latest applied migration", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db)
		set := MigrationSet{
			PluginID: "stack_plugin",
			Migrations: []Migration{
				newTableMigration("001", "rollback_last_first"),
				newTableMigration("002", "rollback_last_second"),
			},
		}

		require.NoError(t, migrator.Migrate(ctx, []MigrationSet{set}))
		require.NoError(t, migrator.RollbackLast(ctx, set))

		assert.True(t, tableExistsInSchema(t, db, "rollback_last_first"))
		assert.False(t, tableExistsInSchema(t, db, "rollback_last_second"))

		applied, err := migrator.ListApplied(ctx, "stack_plugin")
		require.NoError(t, err)
		require.Len(t, applied, 1)
		assert.Equal(t, "001", applied[0].Version)
	})

	t.Run("requires a migration definition", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db)
		set := MigrationSet{
			PluginID:   "unknown_definition_plugin",
			Migrations: []Migration{newTableMigration("001", "unknown_definition_table")},
		}

		require.NoError(t, migrator.Migrate(ctx, []MigrationSet{set}))

		otherSet := MigrationSet{
			PluginID:   "unknown_definition_plugin",
			Migrations: []Migration{newTableMigration("002", "unknown_definition_other")},
		}
		err := migrator.RollbackLast(ctx, otherSet)
		require.Error(t, err)
		assert.ErrorContains(t, err, "migration definition for unknown_definition_plugin:001 not found")
		assert.True(t, tableExistsInSchema(t, db, "unknown_definition_table"))
	})

	t.Run("rejects an empty plugin ID", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		migrator := mustNewMigrator(t, db)

		err := migrator.RollbackLast(context.Background(), MigrationSet{PluginID: ""})
		require.Error(t, err)
		assert.ErrorContains(t, err, "plugin ID cannot be empty")
	})

	t.Run("returns Down errors", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db)
		set := MigrationSet{
			PluginID: "down_error_plugin",
			Migrations: []Migration{{
				Version: "001",
				Up: func(ctx context.Context, tx bun.Tx) error {
					_, err := tx.ExecContext(ctx, "CREATE TABLE rollback_last_down_error (id TEXT);")
					return err
				},
				Down: func(context.Context, bun.Tx) error {
					return errors.New("boom")
				},
			}},
		}

		require.NoError(t, migrator.Migrate(ctx, []MigrationSet{set}))
		err := migrator.RollbackLast(ctx, set)
		require.Error(t, err)
		assert.ErrorContains(t, err, "down failed")

		assert.True(t, tableExistsInSchema(t, db, "rollback_last_down_error"))
		applied, err := migrator.ListApplied(ctx, "down_error_plugin")
		require.NoError(t, err)
		assert.Len(t, applied, 1)
	})

	t.Run("requires a Down function", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db)
		set := MigrationSet{
			PluginID: "missing_down_plugin",
			Migrations: []Migration{{
				Version: "001",
				Up: func(ctx context.Context, tx bun.Tx) error {
					_, err := tx.ExecContext(ctx, "CREATE TABLE rollback_last_missing_down (id TEXT);")
					return err
				},
			}},
		}

		require.NoError(t, migrator.Migrate(ctx, []MigrationSet{set}))
		err := migrator.RollbackLast(ctx, set)
		require.Error(t, err)
		assert.ErrorContains(t, err, "missing Down function")
	})

	t.Run("returns schema table errors", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		require.NoError(t, db.Close())
		migrator := mustNewMigrator(t, db)

		err := migrator.RollbackLast(context.Background(), MigrationSet{PluginID: "closed_plugin"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "create auth_schema_migrations table")
	})
}

func TestMigrator_RollbackAll(t *testing.T) {
	t.Parallel()

	t.Run("no sets is a no-op", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		migrator := mustNewMigrator(t, db)
		assert.NoError(t, migrator.RollbackAll(context.Background(), nil))
	})

	t.Run("no applied migrations", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db)
		set := MigrationSet{
			PluginID:   "never_applied_plugin",
			Migrations: []Migration{newTableMigration("001", "never_applied_table")},
		}

		require.NoError(t, migrator.RollbackAll(ctx, []MigrationSet{set}))
		assert.False(t, tableExistsInSchema(t, db, "never_applied_table"))
	})

	t.Run("skips migrations that were never applied", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db)

		appliedSet := MigrationSet{
			PluginID: "partial_plugin",
			Migrations: []Migration{
				newTableMigration("001", "partial_first"),
				newTableMigration("002", "partial_second"),
			},
		}
		require.NoError(t, migrator.Migrate(ctx, []MigrationSet{appliedSet}))

		fullSet := MigrationSet{
			PluginID: "partial_plugin",
			Migrations: []Migration{
				newTableMigration("001", "partial_first"),
				newTableMigration("002", "partial_second"),
				newTableMigration("003", "partial_third"),
			},
		}
		require.NoError(t, migrator.RollbackAll(ctx, []MigrationSet{fullSet}))

		assert.False(t, tableExistsInSchema(t, db, "partial_first"))
		assert.False(t, tableExistsInSchema(t, db, "partial_second"))
		assert.False(t, tableExistsInSchema(t, db, "partial_third"))

		applied, err := migrator.ListApplied(ctx, "partial_plugin")
		require.NoError(t, err)
		assert.Empty(t, applied)
	})

	t.Run("stops on Down failure", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db)
		set := MigrationSet{
			PluginID: "rollback_all_failure_plugin",
			Migrations: []Migration{
				newTableMigration("001", "rollback_all_first"),
				{
					Version: "002",
					Up: func(ctx context.Context, tx bun.Tx) error {
						_, err := tx.ExecContext(ctx, "CREATE TABLE rollback_all_second (id TEXT);")
						return err
					},
					Down: func(context.Context, bun.Tx) error {
						return errors.New("boom")
					},
				},
			},
		}

		require.NoError(t, migrator.Migrate(ctx, []MigrationSet{set}))
		err := migrator.RollbackAll(ctx, []MigrationSet{set})
		require.Error(t, err)
		assert.ErrorContains(t, err, "down failed")

		assert.True(t, tableExistsInSchema(t, db, "rollback_all_first"))
		assert.True(t, tableExistsInSchema(t, db, "rollback_all_second"))

		applied, err := migrator.ListApplied(ctx, "rollback_all_failure_plugin")
		require.NoError(t, err)
		assert.Len(t, applied, 2)
	})

	t.Run("requires dependencies", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		migrator := mustNewMigrator(t, db)
		set := MigrationSet{PluginID: "needs_missing", DependsOn: []string{"missing"}}

		err := migrator.RollbackAll(context.Background(), []MigrationSet{set})
		require.Error(t, err)
		assert.ErrorContains(t, err, "has no registered migrations")
	})

	t.Run("returns schema table errors", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		require.NoError(t, db.Close())
		migrator := mustNewMigrator(t, db)

		err := migrator.RollbackAll(context.Background(), []MigrationSet{{PluginID: "closed_plugin"}})
		require.Error(t, err)
		assert.ErrorContains(t, err, "create auth_schema_migrations table")
	})
}

func TestMigrator_latestApplied(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrNoRows when nothing is applied", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db)
		require.NoError(t, migrator.ensureTable(ctx))

		latest, err := migrator.latestApplied(ctx, "missing_plugin")
		assert.Nil(t, latest)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("returns the most recent record for a plugin", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db)
		require.NoError(t, migrator.ensureTable(ctx))

		earlier := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		later := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
		entries := []schemaMigration{
			{PluginID: "ordered_plugin", Version: "001", AppliedAt: earlier},
			{PluginID: "ordered_plugin", Version: "002", AppliedAt: later},
			{PluginID: "other_plugin", Version: "003", AppliedAt: later.Add(time.Hour)},
		}
		_, err := db.NewInsert().Model(&entries).Exec(ctx)
		require.NoError(t, err)

		latest, err := migrator.latestApplied(ctx, "ordered_plugin")
		require.NoError(t, err)
		require.NotNil(t, latest)
		assert.Equal(t, "ordered_plugin", latest.PluginID)
		assert.Equal(t, "002", latest.Version)
		assert.True(t, latest.AppliedAt.Equal(later))
	})

	t.Run("returns query errors", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		require.NoError(t, db.Close())
		migrator := mustNewMigrator(t, db)

		latest, err := migrator.latestApplied(context.Background(), "closed_plugin")
		assert.Nil(t, latest)
		require.Error(t, err)
		assert.NotErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestMigrator_ListApplied(t *testing.T) {
	t.Parallel()

	t.Run("returns an empty list for an empty schema table", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		migrator := mustNewMigrator(t, db)

		applied, err := migrator.ListApplied(context.Background(), "")
		require.NoError(t, err)
		assert.Empty(t, applied)
	})

	t.Run("lists all plugins and filters by plugin", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db)

		alpha := MigrationSet{
			PluginID: "alpha",
			Migrations: []Migration{
				newTableMigration("001", "list_alpha_first"),
				newTableMigration("002", "list_alpha_second"),
			},
		}
		beta := MigrationSet{
			PluginID:   "beta",
			Migrations: []Migration{newTableMigration("001", "list_beta_first")},
		}
		require.NoError(t, migrator.Migrate(ctx, []MigrationSet{beta, alpha}))

		all, err := migrator.ListApplied(ctx, "")
		require.NoError(t, err)
		require.Len(t, all, 3)
		assert.Equal(t, "alpha", all[0].PluginID)
		assert.Equal(t, "001", all[0].Version)
		assert.Equal(t, "alpha", all[1].PluginID)
		assert.Equal(t, "002", all[1].Version)
		assert.Equal(t, "beta", all[2].PluginID)
		assert.Equal(t, "001", all[2].Version)

		filtered, err := migrator.ListApplied(ctx, "alpha")
		require.NoError(t, err)
		require.Len(t, filtered, 2)
		assert.Equal(t, "alpha", filtered[0].PluginID)
		assert.Equal(t, "alpha", filtered[1].PluginID)

		whitespace, err := migrator.ListApplied(ctx, "   ")
		require.NoError(t, err)
		assert.Len(t, whitespace, 3)

		missing, err := migrator.ListApplied(ctx, "missing")
		require.NoError(t, err)
		assert.Empty(t, missing)
	})

	t.Run("returns listing errors", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		require.NoError(t, db.Close())
		migrator := mustNewMigrator(t, db)

		applied, err := migrator.ListApplied(context.Background(), "closed_plugin")
		assert.Nil(t, applied)
		require.Error(t, err)
		assert.ErrorContains(t, err, "create auth_schema_migrations table")
	})
}

func TestMigrator_appliedVersions(t *testing.T) {
	t.Parallel()

	t.Run("rejects an empty plugin ID", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		migrator := mustNewMigrator(t, db)

		applied, err := migrator.appliedVersions(context.Background(), "")
		assert.Nil(t, applied)
		require.Error(t, err)
		assert.ErrorContains(t, err, "plugin ID cannot be empty")
	})

	t.Run("returns records keyed by version", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		ctx := context.Background()
		migrator := mustNewMigrator(t, db)
		set := MigrationSet{
			PluginID: "versions_plugin",
			Migrations: []Migration{
				newTableMigration("001", "versions_first"),
				newTableMigration("002", "versions_second"),
			},
		}
		require.NoError(t, migrator.Migrate(ctx, []MigrationSet{set}))

		applied, err := migrator.appliedVersions(ctx, "versions_plugin")
		require.NoError(t, err)
		require.Len(t, applied, 2)
		assert.Contains(t, applied, "001")
		assert.Contains(t, applied, "002")
		assert.Equal(t, "versions_plugin", applied["001"].PluginID)

		other, err := migrator.appliedVersions(ctx, "other_plugin")
		require.NoError(t, err)
		assert.Empty(t, other)
	})

	t.Run("returns query errors", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		require.NoError(t, db.Close())
		migrator := mustNewMigrator(t, db)

		applied, err := migrator.appliedVersions(context.Background(), "closed_plugin")
		assert.Nil(t, applied)
		require.Error(t, err)
		assert.ErrorContains(t, err, "select migrations for closed_plugin")
	})
}

func TestMigrator_applyUpErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		opts      []MigratorOption
		migration Migration
		wantErr   string
	}{
		{
			name:      "requires an Up function",
			migration: Migration{Version: "001"},
			wantErr:   "migration plugin_a:001 missing Up function",
		},
		{
			name:      "wraps record insert errors",
			opts:      []MigratorOption{withSchemaTable("missing_schema_table")},
			migration: Migration{Version: "001", Up: noopMigration},
			wantErr:   "record migration plugin_a:001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDB(t)
			migrator := mustNewMigrator(t, db, tt.opts...)

			err := migrator.applyUp(context.Background(), "plugin_a", tt.migration)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestMigrator_applyDownErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		opts      []MigratorOption
		migration Migration
		wantErr   string
	}{
		{
			name:      "requires a Down function",
			migration: Migration{Version: "001"},
			wantErr:   "migration plugin_a:001 missing Down function",
		},
		{
			name:      "wraps record delete errors",
			opts:      []MigratorOption{withSchemaTable("missing_schema_table")},
			migration: Migration{Version: "001", Down: noopMigration},
			wantErr:   "remove migration record plugin_a:001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDB(t)
			migrator := mustNewMigrator(t, db, tt.opts...)

			err := migrator.applyDown(context.Background(), "plugin_a", tt.migration)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestSortMigrationSets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		sets    []MigrationSet
		wantErr string
	}{
		{name: "returns nil for no sets", sets: nil},
		{
			name:    "rejects self dependencies",
			sets:    []MigrationSet{{PluginID: "self", DependsOn: []string{"self"}}},
			wantErr: "plugin self cannot depend on itself",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ordered, err := sortMigrationSets(tt.sets)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				assert.Nil(t, ordered)
				return
			}
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestMigrator_DeduplicatesDependencies(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	ctx := context.Background()
	migrator := mustNewMigrator(t, db)

	parent := MigrationSet{
		PluginID:   "dedupe_parent",
		Migrations: []Migration{newTableMigration("001", "dedupe_parent_table")},
	}
	child := MigrationSet{
		PluginID:   "dedupe_child",
		DependsOn:  []string{"dedupe_parent", "dedupe_parent", "", "   "},
		Migrations: []Migration{newTableMigration("001", "dedupe_child_table")},
	}

	require.NoError(t, migrator.Migrate(ctx, []MigrationSet{child, parent}))
	assert.True(t, tableExistsInSchema(t, db, "dedupe_parent_table"))
	assert.True(t, tableExistsInSchema(t, db, "dedupe_child_table"))
}

func TestMigrator_RollbackAllRejectsInvalidDefinition(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	ctx := context.Background()
	migrator := mustNewMigrator(t, db)
	require.NoError(t, migrator.ensureTable(ctx))

	entry := schemaMigration{
		PluginID:  "invalid_definition_plugin",
		Version:   "001",
		AppliedAt: time.Now().UTC(),
	}
	_, err := db.NewInsert().Model(&entry).Exec(ctx)
	require.NoError(t, err)

	set := MigrationSet{
		PluginID:   "invalid_definition_plugin",
		Migrations: []Migration{{Version: "001"}},
	}
	err = migrator.RollbackAll(ctx, []MigrationSet{set})
	require.Error(t, err)
	assert.ErrorContains(t, err, "migration invalid_definition_plugin:001 missing Up function")
}

func TestMigrator_SchemaTableErrors(t *testing.T) {
	t.Parallel()

	const brokenTable = "broken_schema_migrations"

	brokenMigrator := func(t *testing.T) *Migrator {
		t.Helper()

		db := newTestDB(t)
		_, err := db.ExecContext(context.Background(),
			fmt.Sprintf("CREATE TABLE %s (unrelated TEXT);", brokenTable))
		require.NoError(t, err)
		return mustNewMigrator(t, db, withSchemaTable(brokenTable))
	}

	targetSet := MigrationSet{
		PluginID:   "broken_plugin",
		Migrations: []Migration{newTableMigration("001", "broken_plugin_table")},
	}

	t.Run("Migrate wraps table creation errors", func(t *testing.T) {
		t.Parallel()

		db := newTestDB(t)
		require.NoError(t, db.Close())
		migrator := mustNewMigrator(t, db)

		err := migrator.Migrate(context.Background(), []MigrationSet{targetSet})
		require.Error(t, err)
		assert.ErrorContains(t, err, "create auth_schema_migrations table")
	})

	t.Run("Migrate wraps applied version query errors", func(t *testing.T) {
		t.Parallel()

		err := brokenMigrator(t).Migrate(context.Background(), []MigrationSet{targetSet})
		require.Error(t, err)
		assert.ErrorContains(t, err, "select migrations for broken_plugin")
	})

	t.Run("RollbackAll wraps applied version query errors", func(t *testing.T) {
		t.Parallel()

		err := brokenMigrator(t).RollbackAll(context.Background(), []MigrationSet{targetSet})
		require.Error(t, err)
		assert.ErrorContains(t, err, "select migrations for broken_plugin")
	})

	t.Run("RollbackLast wraps latest version query errors", func(t *testing.T) {
		t.Parallel()

		err := brokenMigrator(t).RollbackLast(context.Background(), targetSet)
		require.Error(t, err)
		assert.NotErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("ListApplied wraps listing errors", func(t *testing.T) {
		t.Parallel()

		applied, err := brokenMigrator(t).ListApplied(context.Background(), "")
		assert.Nil(t, applied)
		require.Error(t, err)
		assert.ErrorContains(t, err, "list applied migrations")
	})
}
