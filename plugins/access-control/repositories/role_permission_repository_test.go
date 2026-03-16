package repositories

import (
	"context"
	"testing"
	"time"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

func TestBunRolePermissionRepositoryRoleCRUDAndAssignmentCount(t *testing.T) {
	db := setupRepoDB(t)
	repo := NewBunRolePermissionRepository(db)
	ctx := context.Background()

	role := &types.Role{ID: "r1", Name: "admin"}
	if err := repo.CreateRole(ctx, role); err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	if err := repo.AssignUserRole(ctx, "u1", "r1", nil, nil); err != nil {
		t.Fatalf("failed to assign user role: %v", err)
	}

	count, err := repo.CountUserAssignmentsByRoleID(ctx, "r1")
	if err != nil {
		t.Fatalf("failed to count assignments: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 assignment, got %d", count)
	}

	deleted, err := repo.DeleteRole(ctx, "r1")
	if err != nil {
		t.Fatalf("failed to delete role: %v", err)
	}
	if !deleted {
		t.Fatal("expected role to be deleted")
	}
}

func TestBunRolePermissionRepositoryReplaceRolePermissions(t *testing.T) {
	db := setupRepoDB(t)
	repo := NewBunRolePermissionRepository(db)
	ctx := context.Background()

	if err := repo.CreateRole(ctx, &types.Role{ID: "r1", Name: "editor"}); err != nil {
		t.Fatalf("failed to create role: %v", err)
	}
	if err := repo.CreatePermission(ctx, &types.Permission{ID: "p1", Key: "posts.read"}); err != nil {
		t.Fatalf("failed to create permission p1: %v", err)
	}
	if err := repo.CreatePermission(ctx, &types.Permission{ID: "p2", Key: "posts.write"}); err != nil {
		t.Fatalf("failed to create permission p2: %v", err)
	}

	if err := repo.ReplaceRolePermissions(ctx, "r1", []string{"p1", "p2"}, nil); err != nil {
		t.Fatalf("failed to replace role permissions: %v", err)
	}

	permissions, err := repo.GetRolePermissions(ctx, "r1")
	if err != nil {
		t.Fatalf("failed to get role permissions: %v", err)
	}
	if len(permissions) != 2 {
		t.Fatalf("expected 2 permissions, got %d", len(permissions))
	}
}

func TestBunRolePermissionRepositoryReplaceUserRoles(t *testing.T) {
	db := setupRepoDB(t)
	repo := NewBunRolePermissionRepository(db)
	ctx := context.Background()

	if err := repo.CreateRole(ctx, &types.Role{ID: "r1", Name: "role-1"}); err != nil {
		t.Fatalf("failed to create role r1: %v", err)
	}
	if err := repo.CreateRole(ctx, &types.Role{ID: "r2", Name: "role-2"}); err != nil {
		t.Fatalf("failed to create role r2: %v", err)
	}

	if err := repo.ReplaceUserRoles(ctx, "u1", []string{"r1", "r2"}, nil); err != nil {
		t.Fatalf("failed to replace user roles: %v", err)
	}

	if err := repo.AssignUserRole(ctx, "u1", "r1", nil, internaltests.PtrTime(time.Now().UTC().Add(1*time.Hour))); err == nil {
		t.Fatal("expected duplicate assignment to fail due primary key")
	}
}
