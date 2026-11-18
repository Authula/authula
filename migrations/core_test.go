package migrations

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

func TestCoreMigrationSet_ProviderSelection(t *testing.T) {
	t.Parallel()

	expectedVersions := []string{
		"20260126000000_core_initial",
		"20260127000000_core_add_metadata",
	}

	for _, provider := range []string{"sqlite", "postgres", "mysql"} {
		t.Run(provider, func(t *testing.T) {
			t.Parallel()

			set, err := CoreMigrationSet(provider)
			require.NoError(t, err)
			assert.Equal(t, CorePluginID, set.PluginID)
			require.Len(t, set.Migrations, len(expectedVersions))

			versions := make([]string, 0, len(set.Migrations))
			for i, migration := range set.Migrations {
				assert.Equal(t, expectedVersions[i], migration.Version)
				assert.NotNil(t, migration.Up)
				assert.NotNil(t, migration.Down)
				versions = append(versions, migration.Version)
			}

			assert.True(t, sort.StringsAreSorted(versions), "versions should be ordered")
			assert.Equal(t, expectedVersions, versions)

			repeated, err := CoreMigrationSet(provider)
			require.NoError(t, err)
			require.Len(t, repeated.Migrations, len(expectedVersions))
			for i, migration := range repeated.Migrations {
				assert.Equal(t, expectedVersions[i], migration.Version, "versions should be stable across calls")
			}
		})
	}
}

func TestCoreMigrationSet_UnsupportedProvider(t *testing.T) {
	t.Parallel()

	for _, provider := range []string{"oracle", "", "sqlite3", "PostgreSQL"} {
		t.Run("provider_"+provider, func(t *testing.T) {
			t.Parallel()

			set, err := CoreMigrationSet(provider)
			require.Error(t, err)
			assert.EqualError(t, err, "unsupported database provider: "+provider)
			assert.Equal(t, MigrationSet{}, set)
		})
	}
}

func TestCoreSQLiteMigrations_ExecuteAndRollback(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	ctx := context.Background()

	migrator, err := NewMigrator(db, testLogger{t})
	require.NoError(t, err)

	set, err := CoreMigrationSet("sqlite")
	require.NoError(t, err)

	require.NoError(t, migrator.Migrate(ctx, []MigrationSet{set}))

	for _, table := range []string{"users", "accounts", "sessions", "verifications"} {
		assert.Equal(t, 1, countSQLiteObjects(t, db, "table", table), "table %s should exist", table)
	}

	for _, index := range []string{
		"idx_accounts_user_id",
		"idx_sessions_user_id",
		"idx_sessions_expires_at",
		"idx_verifications_user_id",
		"idx_verifications_identifier",
		"idx_verifications_type",
		"idx_verifications_expires_at",
	} {
		assert.Equal(t, 1, countSQLiteObjects(t, db, "index", index), "index %s should exist", index)
	}

	assert.Equal(t, 1, countSQLiteColumns(t, db, "users", "metadata"))

	_, err = db.ExecContext(ctx, `INSERT INTO users (id, name, email) VALUES (?, ?, ?)`, "user-1", "User One", "user@example.com")
	require.NoError(t, err)

	var metadata string
	require.NoError(t, db.NewRaw(`SELECT metadata FROM users WHERE id = ?`, "user-1").Scan(ctx, &metadata))
	assert.JSONEq(t, `{}`, metadata)

	require.NoError(t, migrator.RollbackAll(ctx, []MigrationSet{set}))

	for _, table := range []string{"users", "accounts", "sessions", "verifications"} {
		assert.Equal(t, 0, countSQLiteObjects(t, db, "table", table), "table %s should be dropped", table)
	}
}

func TestCorePostgresMigrations_Statements(t *testing.T) {
	t.Parallel()

	migration := corePostgresInitial()

	up := recordedStatements(t, migration.Up)
	assert.Len(t, up, 22)
	assertAllStatementsNonEmpty(t, up)
	for _, marker := range []string{
		"CREATE EXTENSION IF NOT EXISTS pgcrypto",
		"LANGUAGE plpgsql",
		"gen_random_uuid()",
		"CREATE TRIGGER update_users_updated_at_trigger",
		"cleanup_expired_records_fn",
		"CREATE INDEX IF NOT EXISTS idx_sessions_user_id",
	} {
		assertStatementsContain(t, up, marker)
	}
	for _, statement := range up {
		assert.NotContains(t, statement, "PRAGMA", "postgres migrations must not use SQLite syntax")
		assert.NotContains(t, statement, "AUTO_INCREMENT", "postgres migrations must not use MySQL syntax")
	}

	down := recordedStatements(t, migration.Down)
	assert.Len(t, down, 10)
	assertAllStatementsNonEmpty(t, down)
	assertStatementsContain(t, down, "DROP TABLE IF EXISTS verifications CASCADE")
	assertStatementsContain(t, down, "DROP FUNCTION IF EXISTS core_set_updated_at_fn()")

	metadata := corePostgresAddMetadata()
	metadataUp := recordedStatements(t, metadata.Up)
	assert.Len(t, metadataUp, 1)
	assert.Contains(t, metadataUp[0], "JSONB")

	metadataDown := recordedStatements(t, metadata.Down)
	assert.Len(t, metadataDown, 1)
	assert.Contains(t, metadataDown[0], "ALTER TABLE users DROP COLUMN metadata")
}

func TestCoreMySQLMigrations_Statements(t *testing.T) {
	t.Parallel()

	migration := coreMySQLInitial()

	up := recordedStatements(t, migration.Up)
	assert.Len(t, up, 10)
	assertAllStatementsNonEmpty(t, up)
	for _, marker := range []string{
		"BINARY(16) NOT NULL PRIMARY KEY",
		"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE PROCEDURE cleanup_sessions()",
		"CREATE PROCEDURE cleanup_all_expired()",
		"ON UPDATE CURRENT_TIMESTAMP",
	} {
		assertStatementsContain(t, up, marker)
	}
	for _, statement := range up {
		assert.NotContains(t, statement, "PRAGMA", "mysql migrations must not use SQLite syntax")
		assert.NotContains(t, statement, "gen_random_uuid()", "mysql migrations must not use Postgres syntax")
	}

	down := recordedStatements(t, migration.Down)
	assert.Len(t, down, 7)
	assertAllStatementsNonEmpty(t, down)
	assertStatementsContain(t, down, "DROP PROCEDURE IF EXISTS cleanup_all_expired")
	assertStatementsContain(t, down, "DROP TABLE IF EXISTS verifications")

	metadata := coreMySQLAddMetadata()
	metadataUp := recordedStatements(t, metadata.Up)
	assert.Len(t, metadataUp, 1)
	assert.Contains(t, metadataUp[0], "ADD COLUMN metadata JSON")

	metadataDown := recordedStatements(t, metadata.Down)
	assert.Len(t, metadataDown, 1)
	assert.Contains(t, metadataDown[0], "ALTER TABLE users DROP COLUMN metadata")
}

func countSQLiteObjects(t *testing.T, db *bun.DB, objectType, name string) int {
	t.Helper()

	var count int
	err := db.NewRaw(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = ? AND name = ?`,
		objectType,
		name,
	).Scan(context.Background(), &count)
	require.NoError(t, err)

	return count
}

func countSQLiteColumns(t *testing.T, db *bun.DB, table, column string) int {
	t.Helper()

	var count int
	err := db.NewRaw(
		`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`,
		table,
		column,
	).Scan(context.Background(), &count)
	require.NoError(t, err)

	return count
}

func assertAllStatementsNonEmpty(t *testing.T, statements []string) {
	t.Helper()

	require.NotEmpty(t, statements)
	for i, statement := range statements {
		assert.NotEmptyf(t, strings.TrimSpace(statement), "statement %d should not be blank", i)
	}
}

func assertStatementsContain(t *testing.T, statements []string, marker string) {
	t.Helper()

	assert.Truef(t, strings.Contains(strings.Join(statements, "\n"), marker), "expected statements to contain %q", marker)
}

// statementRecorder collects every statement sent through the recording driver.
type statementRecorder struct {
	mu         sync.Mutex
	statements []string
}

func (r *statementRecorder) record(statement string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.statements = append(r.statements, statement)
}

func (r *statementRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.statements...)
}

var recordingDriverCounter atomic.Uint64

type recordingDriver struct {
	recorder *statementRecorder
}

func (d *recordingDriver) Open(string) (driver.Conn, error) {
	return &recordingConn{recorder: d.recorder}, nil
}

type recordingConn struct {
	recorder *statementRecorder
}

func (c *recordingConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("recording driver does not support prepared statements")
}

func (c *recordingConn) Close() error { return nil }

func (c *recordingConn) Begin() (driver.Tx, error) { return recordingTx{}, nil }

func (c *recordingConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.recorder.record(query)
	return driver.RowsAffected(0), nil
}

type recordingTx struct{}

func (recordingTx) Commit() error   { return nil }
func (recordingTx) Rollback() error { return nil }

// recordedStatements runs a migration function against a recording driver.
// The driver stores the SQL but never sends it to a database engine, so
// Postgres and MySQL statements can be inspected without a live server.
func recordedStatements(t *testing.T, run func(ctx context.Context, tx bun.Tx) error) []string {
	t.Helper()

	recorder := &statementRecorder{}
	name := fmt.Sprintf("migrations_recording_%d", recordingDriverCounter.Add(1))
	sql.Register(name, &recordingDriver{recorder: recorder})

	sqlDB, err := sql.Open(name, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() {
		_ = db.Close()
	})

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	require.NoError(t, run(ctx, tx))
	return recorder.snapshot()
}
