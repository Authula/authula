package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"

	plugintests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestBunUserPermissionsRepositoryGetUserPermissions(t *testing.T) {
	t.Parallel()

	description := new(string)
	*description = "Read users"

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	role1ID := uuid.New().String()
	role2ID := uuid.New().String()
	perm1ID := uuid.New().String()
	perm2ID := uuid.New().String()

	tests := []struct {
		name      string
		seed      func(*BunRolesRepository, *BunPermissionsRepository, *BunRolePermissionsRepository, *BunUserRolesRepository, context.Context)
		userID    string
		wantEmpty bool
		wantKeys  []string
		wantSrcs  []int
	}{
		{
			name:      "empty result",
			userID:    "00000000-0000-0000-0000-000000000000",
			wantEmpty: true,
		},
		{
			name:   "aggregates permissions across roles and ignores expired roles",
			userID: user1ID,
			seed: func(rolesRepo *BunRolesRepository, permissionsRepo *BunPermissionsRepository, rolePermissionsRepo *BunRolePermissionsRepository, userRolesRepo *BunUserRolesRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: role1ID, Name: "editor", Weight: 10}); err != nil {
					panic(err)
				}
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: role2ID, Name: "viewer", Weight: 10}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: perm1ID, Key: "users.read", Description: description}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: perm2ID, Key: "users.write"}); err != nil {
					panic(err)
				}
				grantedBy := user2ID
				if err := rolePermissionsRepo.AddRolePermission(ctx, role1ID, perm1ID, &grantedBy); err != nil {
					panic(err)
				}
				if err := rolePermissionsRepo.AddRolePermission(ctx, role2ID, perm1ID, nil); err != nil {
					panic(err)
				}
				if err := rolePermissionsRepo.AddRolePermission(ctx, role2ID, perm2ID, &grantedBy); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, user1ID, role1ID, &grantedBy, nil); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, user1ID, role2ID, nil, nil); err != nil {
					panic(err)
				}
			},
			wantKeys: []string{"users.read", "users.write"},
			wantSrcs: []int{2, 1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t, user1ID, user2ID)
			rolesRepo := NewBunRolesRepository(db)
			permissionsRepo := NewBunPermissionsRepository(db)
			rolePermissionsRepo := NewBunRolePermissionsRepository(db)
			userRolesRepo := NewBunUserRolesRepository(db)
			repo := NewBunUserPermissionsRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				tc.seed(rolesRepo, permissionsRepo, rolePermissionsRepo, userRolesRepo, ctx)
			}

			permissions, err := repo.GetUserPermissions(ctx, tc.userID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if permissions == nil {
				t.Fatal("expected permissions slice, got nil")
			}
			if tc.wantEmpty {
				if len(permissions) != 0 {
					t.Fatalf("expected no permissions, got %#v", permissions)
				}
				return
			}
			if len(permissions) != len(tc.wantKeys) {
				t.Fatalf("expected %d permissions, got %d", len(tc.wantKeys), len(permissions))
			}
			for i := range tc.wantKeys {
				if permissions[i].PermissionKey != tc.wantKeys[i] {
					t.Fatalf("unexpected permission key at %d: %#v", i, permissions[i])
				}
				if permissions[i].PermissionKey == "users.read" && !sameStringPtr(permissions[i].PermissionDescription, description) {
					t.Fatalf("unexpected permission description at %d: %#v", i, permissions[i])
				}
				if permissions[i].GrantedAt == nil {
					t.Fatalf("expected granted_at to be populated at %d", i)
				}
				if len(permissions[i].Sources) != tc.wantSrcs[i] {
					t.Fatalf("expected %d sources at %d, got %#v", tc.wantSrcs[i], i, permissions[i])
				}
			}
		})
	}
}

func TestBunUserPermissionsRepositoryHasPermissions(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()

	tests := []struct {
		name           string
		seed           func(*BunRolesRepository, *BunPermissionsRepository, *BunRolePermissionsRepository, *BunUserRolesRepository, context.Context)
		userID         string
		permissionKeys []string
		wantHasPerms   bool
	}{
		{
			name:           "empty permission list returns true",
			userID:         user1ID,
			permissionKeys: []string{},
			wantHasPerms:   true,
		},
		{
			name:           "missing permission returns false",
			userID:         user1ID,
			permissionKeys: []string{"users.read", "users.delete"},
			seed: func(rolesRepo *BunRolesRepository, permissionsRepo *BunPermissionsRepository, rolePermissionsRepo *BunRolePermissionsRepository, userRolesRepo *BunUserRolesRepository, ctx context.Context) {
				roleID := uuid.New().String()
				permID := uuid.New().String()
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: roleID, Name: "editor", Weight: 10}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: permID, Key: "users.read"}); err != nil {
					panic(err)
				}
				if err := rolePermissionsRepo.AddRolePermission(ctx, roleID, permID, nil); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, user1ID, roleID, nil, nil); err != nil {
					panic(err)
				}
			},
			wantHasPerms: false,
		},
		{
			name:           "success",
			userID:         user1ID,
			permissionKeys: []string{"users.read"},
			seed: func(rolesRepo *BunRolesRepository, permissionsRepo *BunPermissionsRepository, rolePermissionsRepo *BunRolePermissionsRepository, userRolesRepo *BunUserRolesRepository, ctx context.Context) {
				roleID := uuid.New().String()
				permID := uuid.New().String()
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: roleID, Name: "editor", Weight: 10}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: permID, Key: "users.read"}); err != nil {
					panic(err)
				}
				grantedBy := user2ID
				if err := rolePermissionsRepo.AddRolePermission(ctx, roleID, permID, &grantedBy); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, user1ID, roleID, nil, nil); err != nil {
					panic(err)
				}
			},
			wantHasPerms: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t, user1ID, user2ID)
			rolesRepo := NewBunRolesRepository(db)
			permissionsRepo := NewBunPermissionsRepository(db)
			rolePermissionsRepo := NewBunRolePermissionsRepository(db)
			userRolesRepo := NewBunUserRolesRepository(db)
			repo := NewBunUserPermissionsRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				tc.seed(rolesRepo, permissionsRepo, rolePermissionsRepo, userRolesRepo, ctx)
			}

			hasPermissions, err := repo.HasPermissions(ctx, tc.userID, tc.permissionKeys)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hasPermissions != tc.wantHasPerms {
				t.Fatalf("expected hasPermissions=%v, got %v", tc.wantHasPerms, hasPermissions)
			}
		})
	}
}

func sameStringPtr(got, want *string) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return *got == *want
}
