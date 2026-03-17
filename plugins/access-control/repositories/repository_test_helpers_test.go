package repositories

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

func setupRepoDB(t *testing.T) *bun.DB {
	t.Helper()

	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() {
		_ = db.Close()
	})

	ctx := context.Background()

	if _, err := db.NewCreateTable().Model((*models.User)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*types.Role)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create access control roles table: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*types.Permission)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create access control permissions table: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*types.RolePermission)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create access control role permissions table: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*types.UserRole)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create access control user roles table: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, name, email, email_verified, metadata) VALUES ('u1', 'User One', 'u1@example.com', 1, '{}')`); err != nil {
		t.Fatalf("failed to seed user u1: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, name, email, email_verified, metadata) VALUES ('u2', 'User Two', 'u2@example.com', 1, '{}')`); err != nil {
		t.Fatalf("failed to seed user u2: %v", err)
	}

	return db
}
