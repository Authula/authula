package repositories_test

import (
	"context"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestGetUserWithRolesByID_UserWithoutRoles(t *testing.T) {
	db := newRepositoryTestDB(t)
	repos := repositories.NewAdminRepositories(db)
	repo := repos.UserAccessRepository()
	seedRepositoryUser(t, db, "repo-user-roles-empty", "repo-user-roles-empty@example.com")

	result, err := repo.GetUserWithRolesByID(context.Background(), "repo-user-roles-empty")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatalf("expected user result")
	}
	if result.User.ID != "repo-user-roles-empty" {
		t.Fatalf("expected user id repo-user-roles-empty, got %s", result.User.ID)
	}
	if len(result.Roles) != 0 {
		t.Fatalf("expected no roles, got %d", len(result.Roles))
	}
}

func TestGetUserWithPermissionsByID_UserWithoutPermissions(t *testing.T) {
	db := newRepositoryTestDB(t)
	repos := repositories.NewAdminRepositories(db)
	repo := repos.UserAccessRepository()
	seedRepositoryUser(t, db, "repo-user-perms-empty", "repo-user-perms-empty@example.com")

	result, err := repo.GetUserWithPermissionsByID(context.Background(), "repo-user-perms-empty")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatalf("expected user result")
	}
	if result.User.ID != "repo-user-perms-empty" {
		t.Fatalf("expected user id repo-user-perms-empty, got %s", result.User.ID)
	}
	if len(result.Permissions) != 0 {
		t.Fatalf("expected no permissions, got %d", len(result.Permissions))
	}
}

func TestGetUserWithRolesByID_NonExistentUser(t *testing.T) {
	db := newRepositoryTestDB(t)
	repos := repositories.NewAdminRepositories(db)
	repo := repos.UserAccessRepository()

	result, err := repo.GetUserWithRolesByID(context.Background(), "repo-missing-user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for missing user")
	}
}

func TestGetUserWithPermissionsByID_NonExistentUser(t *testing.T) {
	db := newRepositoryTestDB(t)
	repos := repositories.NewAdminRepositories(db)
	repo := repos.UserAccessRepository()

	result, err := repo.GetUserWithPermissionsByID(context.Background(), "repo-missing-user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for missing user")
	}
}

func TestGetUserWithPermissionsByID_DeduplicatesAcrossRoles(t *testing.T) {
	db := newRepositoryTestDB(t)
	repos := repositories.NewAdminRepositories(db)
	rolePermissionRepo := repos.RolePermissionRepository()
	userAccessRepo := repos.UserAccessRepository()
	ctx := context.Background()

	seedRepositoryUser(t, db, "repo-user-dedupe", "repo-user-dedupe@example.com")

	permission := &admintypes.Permission{ID: "perm-dedupe", Key: "admin.read", IsSystem: false}
	if err := rolePermissionRepo.CreatePermission(ctx, permission); err != nil {
		t.Fatalf("failed to create permission: %v", err)
	}

	roleA := &admintypes.Role{ID: "role-dedupe-a", Name: "role-a", IsSystem: false}
	if err := rolePermissionRepo.CreateRole(ctx, roleA); err != nil {
		t.Fatalf("failed to create role A: %v", err)
	}
	roleB := &admintypes.Role{ID: "role-dedupe-b", Name: "role-b", IsSystem: false}
	if err := rolePermissionRepo.CreateRole(ctx, roleB); err != nil {
		t.Fatalf("failed to create role B: %v", err)
	}

	if err := rolePermissionRepo.ReplaceRolePermissions(ctx, roleA.ID, []string{permission.ID}, nil); err != nil {
		t.Fatalf("failed to assign role A permission: %v", err)
	}
	if err := rolePermissionRepo.ReplaceRolePermissions(ctx, roleB.ID, []string{permission.ID}, nil); err != nil {
		t.Fatalf("failed to assign role B permission: %v", err)
	}

	if err := rolePermissionRepo.ReplaceUserRoles(ctx, "repo-user-dedupe", []string{roleA.ID, roleB.ID}, nil); err != nil {
		t.Fatalf("failed to assign user roles: %v", err)
	}

	result, err := userAccessRepo.GetUserWithPermissionsByID(ctx, "repo-user-dedupe")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatalf("expected user result")
	}
	if len(result.Permissions) != 1 {
		t.Fatalf("expected exactly 1 deduplicated permission, got %d", len(result.Permissions))
	}
	if result.Permissions[0].PermissionID != permission.ID {
		t.Fatalf("expected permission id %s, got %s", permission.ID, result.Permissions[0].PermissionID)
	}
}
