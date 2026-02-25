package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/uptrace/bun"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	migrationsmodule "github.com/GoBetterAuth/go-better-auth/v2/migrations"
	adminplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type repositoryTestLogger struct{}

func (l repositoryTestLogger) Debug(msg string, args ...any) {}
func (l repositoryTestLogger) Info(msg string, args ...any)  {}
func (l repositoryTestLogger) Warn(msg string, args ...any)  {}
func (l repositoryTestLogger) Error(msg string, args ...any) {}

func newRepositoryTestDB(t *testing.T) *bun.DB {
	t.Helper()
	ctx := context.Background()

	db := internaltests.NewSQLiteIntegrationDB(t)

	migrator, err := migrationsmodule.NewMigrator(db, repositoryTestLogger{})
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	coreSet, err := migrationsmodule.CoreMigrationSet("sqlite")
	if err != nil {
		t.Fatalf("failed to load core migration set: %v", err)
	}

	plugin := adminplugin.New(admintypes.AdminPluginConfig{})
	adminSet := migrationsmodule.MigrationSet{
		PluginID:   plugin.Metadata().ID,
		DependsOn:  plugin.DependsOn(),
		Migrations: plugin.Migrations("sqlite"),
	}

	if err := migrator.Migrate(ctx, []migrationsmodule.MigrationSet{coreSet, adminSet}); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

func seedRepositoryUser(t *testing.T, db bun.IDB, id, email string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `INSERT INTO users (id, name, email, email_verified) VALUES (?, ?, ?, ?)`, id, "Repository User", email, false)
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func seedRepositorySession(t *testing.T, db bun.IDB, id, userID string) {
	t.Helper()
	expiresAt := time.Now().UTC().Add(30 * time.Minute)
	_, err := db.ExecContext(context.Background(), `INSERT INTO sessions (id, user_id, token, expires_at) VALUES (?, ?, ?, ?)`, id, userID, "token-"+id, expiresAt)
	if err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}
}
