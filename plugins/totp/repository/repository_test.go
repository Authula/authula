package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/v2/migrations"
	totpplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/totp"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/repository"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

type testLogger struct{}

func (testLogger) Debug(msg string, args ...any) {}
func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}

func newTestTOTPDB(t *testing.T) *bun.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	migrator, err := migrations.NewMigrator(db, testLogger{})
	require.NoError(t, err)

	coreSet, err := migrations.CoreMigrationSet("sqlite")
	require.NoError(t, err)
	totpSet := totpplugin.MigrationSet("sqlite")

	err = migrator.Migrate(ctx, []migrations.MigrationSet{coreSet, totpSet})
	require.NoError(t, err)

	return db
}

func TestCompareAndSwapBackupCodes(t *testing.T) {
	db := newTestTOTPDB(t)
	repo := repository.NewTOTPRepository(db)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `INSERT INTO users (id, name, email) VALUES (?, ?, ?)`, "user-1", "User One", "user1@example.com")
	require.NoError(t, err)

	initial := `["h1","h2"]`
	_, err = repo.Create(ctx, "user-1", "encrypted-secret", initial)
	require.NoError(t, err)

	updated, err := repo.CompareAndSwapBackupCodes(ctx, "user-1", initial, `["h2"]`)
	require.NoError(t, err)
	require.True(t, updated)

	record, err := repo.GetByUserID(ctx, "user-1")
	require.NoError(t, err)
	require.Equal(t, `["h2"]`, record.BackupCodes)

	updated, err = repo.CompareAndSwapBackupCodes(ctx, "user-1", initial, `[]`)
	require.NoError(t, err)
	require.False(t, updated)

	record, err = repo.GetByUserID(ctx, "user-1")
	require.NoError(t, err)
	require.Equal(t, `["h2"]`, record.BackupCodes)
}
