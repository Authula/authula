package repositories_test

import (
	"context"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestRolePermissionRepository_CreateAndGetAll(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := repositories.NewAdminRepositories(db).RolePermissionRepository()
	ctx := context.Background()

	permission := &admintypes.Permission{ID: "perm-1", Key: "admin.read", IsSystem: false}
	if err := repo.CreatePermission(ctx, permission); err != nil {
		t.Fatalf("failed to create permission: %v", err)
	}

	role := &admintypes.Role{ID: "role-1", Name: "operators", IsSystem: false}
	if err := repo.CreateRole(ctx, role); err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	permissions, err := repo.GetAllPermissions(ctx)
	if err != nil {
		t.Fatalf("failed to get permissions: %v", err)
	}
	if len(permissions) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(permissions))
	}
	if permissions[0].ID != permission.ID {
		t.Fatalf("expected permission id %s, got %s", permission.ID, permissions[0].ID)
	}

	roles, err := repo.GetAllRoles(ctx)
	if err != nil {
		t.Fatalf("failed to get roles: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(roles))
	}
	if roles[0].ID != role.ID {
		t.Fatalf("expected role id %s, got %s", role.ID, roles[0].ID)
	}
}

func TestRolePermissionRepository_UpdatePermissionDescription(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := repositories.NewAdminRepositories(db).RolePermissionRepository()
	ctx := context.Background()

	oldDescription := "old description"
	permission := &admintypes.Permission{ID: "perm-update-1", Key: "admin.update", Description: &oldDescription, IsSystem: false}
	if err := repo.CreatePermission(ctx, permission); err != nil {
		t.Fatalf("failed to create permission: %v", err)
	}

	newDescription := "new description"
	updated, err := repo.UpdatePermissionDescription(ctx, permission.ID, &newDescription)
	if err != nil {
		t.Fatalf("failed to update permission description: %v", err)
	}
	if !updated {
		t.Fatalf("expected update to affect one row")
	}

	fetched, err := repo.GetPermissionByID(ctx, permission.ID)
	if err != nil {
		t.Fatalf("failed to fetch permission: %v", err)
	}
	if fetched == nil {
		t.Fatalf("expected fetched permission")
	}
	if fetched.Description == nil || *fetched.Description != newDescription {
		t.Fatalf("expected updated description %q, got %+v", newDescription, fetched.Description)
	}
}

func TestRolePermissionRepository_CountRoleAssignmentsByPermissionID(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := repositories.NewAdminRepositories(db).RolePermissionRepository()
	ctx := context.Background()

	permission := &admintypes.Permission{ID: "perm-count-1", Key: "admin.count", IsSystem: false}
	if err := repo.CreatePermission(ctx, permission); err != nil {
		t.Fatalf("failed to create permission: %v", err)
	}

	role := &admintypes.Role{ID: "role-1", Name: "role-count", IsSystem: false}
	if err := repo.CreateRole(ctx, role); err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	if err := repo.ReplaceRolePermissions(ctx, role.ID, []string{permission.ID}, nil); err != nil {
		t.Fatalf("failed to assign role permission: %v", err)
	}

	count, err := repo.CountRoleAssignmentsByPermissionID(ctx, permission.ID)
	if err != nil {
		t.Fatalf("failed to count role assignments: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 assignment, got %d", count)
	}
}

func TestRolePermissionRepository_DeletePermission(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := repositories.NewAdminRepositories(db).RolePermissionRepository()
	ctx := context.Background()

	permission := &admintypes.Permission{ID: "perm-delete-1", Key: "admin.delete", IsSystem: false}
	if err := repo.CreatePermission(ctx, permission); err != nil {
		t.Fatalf("failed to create permission: %v", err)
	}

	deleted, err := repo.DeletePermission(ctx, permission.ID)
	if err != nil {
		t.Fatalf("failed to delete permission: %v", err)
	}
	if !deleted {
		t.Fatalf("expected delete to affect one row")
	}

	fetched, err := repo.GetPermissionByID(ctx, permission.ID)
	if err != nil {
		t.Fatalf("failed to get permission by id: %v", err)
	}
	if fetched != nil {
		t.Fatalf("expected permission to be deleted")
	}
}
