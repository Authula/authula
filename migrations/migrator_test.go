package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	_ "github.com/mattn/go-sqlite3"
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
	sqlDB, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
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
	require.NoError(t, err)

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
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO test_entities (id) VALUES (?)`, "abc")
	require.NoError(t, err)

	err = migrator.RollbackAll(ctx, []MigrationSet{set})
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO test_entities (id) VALUES (?)`, "def")
	require.Error(t, err)
}

func TestMigrator_DependencyOrdering(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	ctx := context.Background()

	migrator, err := NewMigrator(db, testLogger{t})
	require.NoError(t, err)

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
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO parent_entities (id) VALUES (?)`, "parent-1")
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO child_entities (id, parent_id) VALUES (?, ?)`, "child-1", "parent-1")
	require.NoError(t, err)

	err = migrator.RollbackAll(ctx, []MigrationSet{parent, child})
	require.NoError(t, err)
}

func TestMigrator_DetectsCycles(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	ctx := context.Background()

	migrator, err := NewMigrator(db, testLogger{t})
	require.NoError(t, err)

	setA := MigrationSet{PluginID: "a", DependsOn: []string{"b"}}
	setB := MigrationSet{PluginID: "b", DependsOn: []string{"a"}}

	err = migrator.Migrate(ctx, []MigrationSet{setA, setB})
	require.Error(t, err)
}
