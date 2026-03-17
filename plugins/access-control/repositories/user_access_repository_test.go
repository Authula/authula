package repositories

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

func TestBunUserAccessRepositoryGetUserRolesIncludesExpiredWithMetadata(t *testing.T) {
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

	assignerID := "u2"
	if err := rpRepo.AssignUserRole(ctx, "u1", "r-active", &assignerID, nil); err != nil {
		t.Fatalf("failed to assign active role: %v", err)
	}
	if err := rpRepo.AssignUserRole(ctx, "u1", "r-expired", &assignerID, internaltests.PtrTime(time.Now().UTC().Add(-1*time.Hour))); err != nil {
		t.Fatalf("failed to assign expired role: %v", err)
	}

	roles, err := uaRepo.GetUserRoles(ctx, "u1")
	if err != nil {
		t.Fatalf("failed to get user roles: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("expected 2 roles including expired, got %d", len(roles))
	}
	if roles[0].RoleID != "r-active" {
		t.Fatalf("expected role sorted by name, got %s", roles[0].RoleID)
	}
	if roles[0].AssignedByUserID == nil || *roles[0].AssignedByUserID != assignerID {
		t.Fatalf("expected assigned_by_user_id=%s", assignerID)
	}
	if roles[0].AssignedAt == nil {
		t.Fatal("expected assigned_at to be populated")
	}
	if roles[1].RoleID != "r-expired" {
		t.Fatalf("expected expired role included, got %s", roles[1].RoleID)
	}
	if roles[1].ExpiresAt == nil {
		t.Fatal("expected expired role to include expires_at")
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
	synctest.Test(t, func(t *testing.T) {
		db := setupRepoDB(t)
		rpRepo := NewBunRolePermissionRepository(db)
		uaRepo := NewBunUserAccessRepository(db)
		ctx := context.Background()

		if err := rpRepo.CreateRole(ctx, &types.Role{ID: "r1", Name: "editor"}); err != nil {
			t.Fatalf("failed to create role: %v", err)
		}
		if err := rpRepo.CreateRole(ctx, &types.Role{ID: "r2", Name: "viewer"}); err != nil {
			t.Fatalf("failed to create second role: %v", err)
		}
		if err := rpRepo.CreatePermission(ctx, &types.Permission{ID: "p1", Key: "posts.read", Description: internaltests.PtrString("Read posts")}); err != nil {
			t.Fatalf("failed to create permission: %v", err)
		}
		grantedBy := "u2"
		if err := rpRepo.AddRolePermission(ctx, "r1", "p1", &grantedBy); err != nil {
			t.Fatalf("failed to add role permission: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
		if err := rpRepo.AddRolePermission(ctx, "r2", "p1", &grantedBy); err != nil {
			t.Fatalf("failed to add second role permission: %v", err)
		}
		if err := rpRepo.AssignUserRole(ctx, "u1", "r1", nil, nil); err != nil {
			t.Fatalf("failed to assign role: %v", err)
		}
		if err := rpRepo.AssignUserRole(ctx, "u1", "r2", nil, nil); err != nil {
			t.Fatalf("failed to assign second role: %v", err)
		}

		perms, err := uaRepo.GetUserEffectivePermissions(ctx, "u1")
		if err != nil {
			t.Fatalf("failed to get effective permissions: %v", err)
		}
		if len(perms) != 1 {
			t.Fatalf("expected 1 deduplicated permission, got %d", len(perms))
		}
		if perms[0].PermissionKey != "posts.read" {
			t.Fatalf("expected posts.read, got %s", perms[0].PermissionKey)
		}
		if perms[0].PermissionDescription == nil || *perms[0].PermissionDescription != "Read posts" {
			t.Fatal("expected permission description to be populated")
		}
		if len(perms[0].Sources) != 2 {
			t.Fatalf("expected 2 permission sources, got %d", len(perms[0].Sources))
		}
		if perms[0].Sources[0].RoleName != "editor" || perms[0].Sources[1].RoleName != "viewer" {
			t.Fatalf("expected deterministic source ordering by role_name, got %s then %s", perms[0].Sources[0].RoleName, perms[0].Sources[1].RoleName)
		}
		if perms[0].Sources[0].GrantedByUserID == nil || *perms[0].Sources[0].GrantedByUserID != grantedBy {
			t.Fatal("expected source granted_by_user_id to be populated")
		}
		if perms[0].Sources[0].GrantedAt == nil || perms[0].Sources[1].GrantedAt == nil {
			t.Fatal("expected source granted_at timestamps to be populated")
		}
	})
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
