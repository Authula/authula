package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

func setupRolePermissionServiceDB(t *testing.T) *bun.DB {
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
	stmts := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			email_verified BOOLEAN NOT NULL DEFAULT 0,
			image TEXT,
			metadata BLOB,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE access_control_roles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			is_system BOOLEAN NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE access_control_permissions (
			id TEXT PRIMARY KEY,
			key TEXT NOT NULL UNIQUE,
			description TEXT,
			is_system BOOLEAN NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE access_control_role_permissions (
			role_id TEXT NOT NULL,
			permission_id TEXT NOT NULL,
			granted_by_user_id TEXT,
			granted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (role_id, permission_id),
			FOREIGN KEY (role_id) REFERENCES access_control_roles(id) ON DELETE CASCADE,
			FOREIGN KEY (permission_id) REFERENCES access_control_permissions(id) ON DELETE CASCADE,
			FOREIGN KEY (granted_by_user_id) REFERENCES users(id) ON DELETE SET NULL
		);`,
		`CREATE TABLE access_control_user_roles (
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			assigned_by_user_id TEXT,
			assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			PRIMARY KEY (user_id, role_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (role_id) REFERENCES access_control_roles(id) ON DELETE CASCADE,
			FOREIGN KEY (assigned_by_user_id) REFERENCES users(id) ON DELETE SET NULL
		);`,
	}

	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("failed to exec statement %q: %v", stmt, err)
		}
	}

	return db
}

func TestRolePermissionServiceCreateRoleTrimsName(t *testing.T) {
	db := setupRolePermissionServiceDB(t)
	svc := NewRolePermissionService(repositories.NewBunRolePermissionRepository(db))

	role, err := svc.CreateRole(context.Background(), types.CreateRoleRequest{Name: "  admin  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role.Name != "admin" {
		t.Fatalf("expected trimmed role name, got %q", role.Name)
	}
}

func TestRolePermissionServiceAssignRoleToUserRejectsPastExpiry(t *testing.T) {
	db := setupRolePermissionServiceDB(t)
	svc := NewRolePermissionService(repositories.NewBunRolePermissionRepository(db))

	err := svc.AssignRoleToUser(
		context.Background(),
		"user-1",
		types.AssignUserRoleRequest{RoleID: "role-1", ExpiresAt: ptrTime(time.Now().UTC().Add(-1 * time.Hour))},
		nil,
	)
	if !errors.Is(err, constants.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

func TestRolePermissionServiceDeleteRoleReturnsConflictWhenAssigned(t *testing.T) {
	db := setupRolePermissionServiceDB(t)
	repo := repositories.NewBunRolePermissionRepository(db)
	svc := NewRolePermissionService(repo)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, name, email, email_verified, metadata) VALUES ('u1', 'User', 'u1@example.com', 1, '{}')`); err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	role, err := svc.CreateRole(ctx, types.CreateRoleRequest{Name: "role-for-delete"})
	if err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	if err := svc.AssignRoleToUser(ctx, "u1", types.AssignUserRoleRequest{RoleID: role.ID}, nil); err != nil {
		t.Fatalf("failed to assign role: %v", err)
	}

	err = svc.DeleteRole(ctx, role.ID)
	if !errors.Is(err, constants.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
