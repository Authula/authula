package repositories

import (
	"context"
	"testing"
	"time"

	plugintests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestBunUserAccessRepositoryCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		seed                func(*BunRolesRepository, *BunPermissionsRepository, *BunRolePermissionsRepository, *BunUserRolesRepository, context.Context)
		roleID              string
		permissionID        string
		wantRoleCount       int
		wantPermissionCount int
	}{
		{
			name:                "empty counts",
			roleID:              "role-missing",
			permissionID:        "perm-missing",
			wantRoleCount:       0,
			wantPermissionCount: 0,
		},
		{
			name:         "counts assigned records",
			roleID:       "role-1",
			permissionID: "perm-1",
			seed: func(rolesRepo *BunRolesRepository, permissionsRepo *BunPermissionsRepository, rolePermissionsRepo *BunRolePermissionsRepository, userRolesRepo *BunUserRolesRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "role-1", Name: "editor"}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: "perm-1", Key: "users.read"}); err != nil {
					panic(err)
				}
				if err := rolePermissionsRepo.AddRolePermission(ctx, "role-1", "perm-1", nil); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, "u1", "role-1", nil, nil); err != nil {
					panic(err)
				}
			},
			wantRoleCount:       1,
			wantPermissionCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			rolesRepo := NewBunRolesRepository(db)
			permissionsRepo := NewBunPermissionsRepository(db)
			rolePermissionsRepo := NewBunRolePermissionsRepository(db)
			userRolesRepo := NewBunUserRolesRepository(db)
			repo := NewBunUserAccessRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				tc.seed(rolesRepo, permissionsRepo, rolePermissionsRepo, userRolesRepo, ctx)
			}

			roleCount, err := repo.CountUserAssignmentsByRoleID(ctx, tc.roleID)
			if err != nil {
				t.Fatalf("failed to count role assignments: %v", err)
			}
			permissionCount, err := repo.CountRoleAssignmentsByPermissionID(ctx, tc.permissionID)
			if err != nil {
				t.Fatalf("failed to count permission assignments: %v", err)
			}

			if roleCount != tc.wantRoleCount {
				t.Fatalf("expected role count %d, got %d", tc.wantRoleCount, roleCount)
			}
			if permissionCount != tc.wantPermissionCount {
				t.Fatalf("expected permission count %d, got %d", tc.wantPermissionCount, permissionCount)
			}
		})
	}
}

func TestBunUserAccessRepositoryGetUserEffectivePermissions(t *testing.T) {
	t.Parallel()

	activeUntil := time.Now().UTC().Add(1 * time.Hour)
	expiredAt := time.Now().UTC().Add(-1 * time.Hour)
	description := new("Read users")
	grantedBy := "u2"

	tests := []struct {
		name        string
		seed        func(*BunRolesRepository, *BunPermissionsRepository, *BunRolePermissionsRepository, *BunUserRolesRepository, context.Context)
		userID      string
		wantKeys    []string
		wantSources int
	}{
		{
			name:     "empty result",
			userID:   "missing-user",
			wantKeys: []string{},
		},
		{
			name:   "aggregates active permissions and ignores expired roles",
			userID: "u1",
			seed: func(rolesRepo *BunRolesRepository, permissionsRepo *BunPermissionsRepository, rolePermissionsRepo *BunRolePermissionsRepository, userRolesRepo *BunUserRolesRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "r1", Name: "editor"}); err != nil {
					panic(err)
				}
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "r2", Name: "viewer"}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: "p1", Key: "posts.read", Description: description}); err != nil {
					panic(err)
				}
				if err := rolePermissionsRepo.AddRolePermission(ctx, "r1", "p1", &grantedBy); err != nil {
					panic(err)
				}
				if err := rolePermissionsRepo.AddRolePermission(ctx, "r2", "p1", &grantedBy); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, "u1", "r1", nil, &activeUntil); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, "u1", "r2", nil, &expiredAt); err != nil {
					panic(err)
				}
			},
			wantKeys:    []string{"posts.read"},
			wantSources: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			rolesRepo := NewBunRolesRepository(db)
			permissionsRepo := NewBunPermissionsRepository(db)
			rolePermissionsRepo := NewBunRolePermissionsRepository(db)
			userRolesRepo := NewBunUserRolesRepository(db)
			repo := NewBunUserAccessRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				tc.seed(rolesRepo, permissionsRepo, rolePermissionsRepo, userRolesRepo, ctx)
			}

			permissions, err := repo.GetUserEffectivePermissions(ctx, tc.userID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if permissions == nil {
				t.Fatal("expected permissions slice, got nil")
			}
			if len(permissions) != len(tc.wantKeys) {
				t.Fatalf("expected %d permissions, got %d", len(tc.wantKeys), len(permissions))
			}
			for i, wantKey := range tc.wantKeys {
				if permissions[i].PermissionKey != wantKey {
					t.Fatalf("expected permission key %s at index %d, got %#v", wantKey, i, permissions[i])
				}
				if len(permissions[i].Sources) != tc.wantSources {
					t.Fatalf("expected %d sources, got %d", tc.wantSources, len(permissions[i].Sources))
				}
				if permissions[i].PermissionDescription == nil || *permissions[i].PermissionDescription != "Read users" {
					t.Fatalf("expected permission description to be populated, got %#v", permissions[i].PermissionDescription)
				}
			}
		})
	}
}

func TestBunUserAccessRepositoryGetUserWithPermissionsByID(t *testing.T) {
	t.Parallel()

	permissionDescription := new("Read users")

	tests := []struct {
		name               string
		seed               func(*BunRolesRepository, *BunPermissionsRepository, *BunRolePermissionsRepository, *BunUserRolesRepository, context.Context)
		userID             string
		wantNil            bool
		wantPermissionKeys []string
	}{
		{
			name:    "not found",
			userID:  "missing-user",
			wantNil: true,
		},
		{
			name:   "success",
			userID: "u1",
			seed: func(rolesRepo *BunRolesRepository, permissionsRepo *BunPermissionsRepository, rolePermissionsRepo *BunRolePermissionsRepository, userRolesRepo *BunUserRolesRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "r1", Name: "editor"}); err != nil {
					panic(err)
				}
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "r2", Name: "viewer"}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: "p1", Key: "posts.read", Description: permissionDescription}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: "p2", Key: "posts.write"}); err != nil {
					panic(err)
				}
				if err := rolePermissionsRepo.AddRolePermission(ctx, "r1", "p1", nil); err != nil {
					panic(err)
				}
				if err := rolePermissionsRepo.AddRolePermission(ctx, "r2", "p2", nil); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, "u1", "r1", nil, nil); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, "u1", "r2", nil, nil); err != nil {
					panic(err)
				}
			},
			wantPermissionKeys: []string{"posts.read", "posts.write"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			rolesRepo := NewBunRolesRepository(db)
			permissionsRepo := NewBunPermissionsRepository(db)
			rolePermissionsRepo := NewBunRolePermissionsRepository(db)
			userRolesRepo := NewBunUserRolesRepository(db)
			repo := NewBunUserAccessRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				tc.seed(rolesRepo, permissionsRepo, rolePermissionsRepo, userRolesRepo, ctx)
			}

			userWithPermissions, err := repo.GetUserWithPermissionsByID(ctx, tc.userID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil {
				if userWithPermissions != nil {
					t.Fatalf("expected nil user, got %#v", userWithPermissions)
				}
				return
			}
			if userWithPermissions == nil {
				t.Fatal("expected user, got nil")
			}
			if userWithPermissions.User.ID != tc.userID {
				t.Fatalf("expected user ID %s, got %s", tc.userID, userWithPermissions.User.ID)
			}
			if len(userWithPermissions.Permissions) != len(tc.wantPermissionKeys) {
				t.Fatalf("expected %d permissions, got %d", len(tc.wantPermissionKeys), len(userWithPermissions.Permissions))
			}
			for i, wantKey := range tc.wantPermissionKeys {
				if userWithPermissions.Permissions[i].PermissionKey != wantKey {
					t.Fatalf("expected permission key %s at index %d, got %#v", wantKey, i, userWithPermissions.Permissions[i])
				}
			}
		})
	}
}
