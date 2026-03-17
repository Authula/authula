package repositories

import (
	"context"
	"testing"
	"time"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

func TestBunUserAccessRepositoryGetUserRolesFiltersExpired(t *testing.T) {
	db := setupRepoDB(t)
	rpRepo := NewBunRolePermissionRepository(db)
	uaRepo := NewBunUserAccessRepository(db)
	ctx := context.Background()

	if err := rpRepo.CreateRole(ctx, &types.Role{ID: "r-active", Name: "active"}); err != nil {
		t.Fatalf("failed to create active role: %v", err)
	}
	if err := rpRepo.CreateRole(ctx, &types.Role{ID: "r-expired", Name: "expired"}); err != nil {
		t.Fatalf("failed to create expired role: %v", err)
	}

	if err := rpRepo.AssignUserRole(ctx, "u1", "r-active", nil, nil); err != nil {
		t.Fatalf("failed to assign active role: %v", err)
	}
	if err := rpRepo.AssignUserRole(ctx, "u1", "r-expired", nil, internaltests.PtrTime(time.Now().UTC().Add(-1*time.Hour))); err != nil {
		t.Fatalf("failed to assign expired role: %v", err)
	}

	roles, err := uaRepo.GetUserRoles(ctx, "u1")
	if err != nil {
		t.Fatalf("failed to get user roles: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("expected 1 active role, got %d", len(roles))
	}
	if roles[0].RoleID != "r-active" {
		t.Fatalf("expected active role, got %s", roles[0].RoleID)
	}
}

func TestBunUserAccessRepositoryGetUserRolesReturnsEmptyArrayWhenNoRoles(t *testing.T) {
	db := setupRepoDB(t)
	uaRepo := NewBunUserAccessRepository(db)

	roles, err := uaRepo.GetUserRoles(context.Background(), "missing-user")
	if err != nil {
		t.Fatalf("failed to get user roles: %v", err)
	}
	if roles == nil {
		t.Fatal("expected empty roles slice, got nil")
	}
	if len(roles) != 0 {
		t.Fatalf("expected 0 roles, got %d", len(roles))
	}
}

func TestBunUserAccessRepositoryGetUserEffectivePermissions(t *testing.T) {
	db := setupRepoDB(t)
	rpRepo := NewBunRolePermissionRepository(db)
	uaRepo := NewBunUserAccessRepository(db)
	ctx := context.Background()

	if err := rpRepo.CreateRole(ctx, &types.Role{ID: "r1", Name: "editor"}); err != nil {
		t.Fatalf("failed to create role: %v", err)
	}
	if err := rpRepo.CreatePermission(ctx, &types.Permission{ID: "p1", Key: "posts.read"}); err != nil {
		t.Fatalf("failed to create permission: %v", err)
	}
	if err := rpRepo.AddRolePermission(ctx, "r1", "p1", nil); err != nil {
		t.Fatalf("failed to add role permission: %v", err)
	}
	if err := rpRepo.AssignUserRole(ctx, "u1", "r1", nil, nil); err != nil {
		t.Fatalf("failed to assign role: %v", err)
	}

	perms, err := uaRepo.GetUserEffectivePermissions(ctx, "u1")
	if err != nil {
		t.Fatalf("failed to get effective permissions: %v", err)
	}
	if len(perms) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(perms))
	}
	if perms[0].PermissionKey != "posts.read" {
		t.Fatalf("expected posts.read, got %s", perms[0].PermissionKey)
	}
}

func TestBunUserAccessRepositoryGetUserEffectivePermissionsReturnsEmptyArrayWhenNoPermissions(t *testing.T) {
	db := setupRepoDB(t)
	uaRepo := NewBunUserAccessRepository(db)

	perms, err := uaRepo.GetUserEffectivePermissions(context.Background(), "missing-user")
	if err != nil {
		t.Fatalf("failed to get effective permissions: %v", err)
	}
	if perms == nil {
		t.Fatal("expected empty permissions slice, got nil")
	}
	if len(perms) != 0 {
		t.Fatalf("expected 0 permissions, got %d", len(perms))
	}
}
