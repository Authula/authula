package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"

	plugintests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestBunRolePermissionsRepositoryGetRolePermissions(t *testing.T) {
	t.Parallel()

	roleID := uuid.New().String()
	perm1ID := uuid.New().String()
	perm2ID := uuid.New().String()
	user1ID := uuid.New().String()
	user2ID := uuid.New().String()

	tests := []struct {
		name     string
		seed     func(*BunRolesRepository, *BunPermissionsRepository, *BunRolePermissionsRepository, context.Context)
		roleID   string
		wantIDs  []string
		wantKeys []string
		wantNil  bool
	}{
		{
			name:     "empty result",
			roleID:   "00000000-0000-0000-0000-000000000000",
			wantNil:  false,
			wantIDs:  []string{},
			wantKeys: []string{},
		},
		{
			name:   "success",
			roleID: roleID,
			seed: func(rolesRepo *BunRolesRepository, permissionsRepo *BunPermissionsRepository, rolePermissionsRepo *BunRolePermissionsRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: roleID, Name: "editor", Weight: 10}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: perm2ID, Key: "users.write", Description: new("Write users")}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: perm1ID, Key: "users.read", Description: new("Read users")}); err != nil {
					panic(err)
				}
				grantedBy := user2ID
				if err := rolePermissionsRepo.AddRolePermission(ctx, roleID, perm2ID, &grantedBy); err != nil {
					panic(err)
				}
				if err := rolePermissionsRepo.AddRolePermission(ctx, roleID, perm1ID, &grantedBy); err != nil {
					panic(err)
				}
			},
			wantIDs:  []string{perm1ID, perm2ID},
			wantKeys: []string{"users.read", "users.write"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t, user1ID, user2ID)
			rolesRepo := NewBunRolesRepository(db)
			permissionsRepo := NewBunPermissionsRepository(db)
			rolePermissionsRepo := NewBunRolePermissionsRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				tc.seed(rolesRepo, permissionsRepo, rolePermissionsRepo, ctx)
			}

			permissions, err := rolePermissionsRepo.GetRolePermissions(ctx, tc.roleID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if permissions == nil {
				t.Fatal("expected permissions slice, got nil")
			}
			if len(permissions) != len(tc.wantIDs) {
				t.Fatalf("expected %d permissions, got %d", len(tc.wantIDs), len(permissions))
			}
			for i := range tc.wantIDs {
				if permissions[i].PermissionID != tc.wantIDs[i] || permissions[i].PermissionKey != tc.wantKeys[i] {
					t.Fatalf("unexpected permission at %d: %#v", i, permissions[i])
				}
			}
		})
	}
}

func TestBunRolePermissionsRepositoryAddRolePermission(t *testing.T) {
	t.Parallel()

	roleID := uuid.New().String()
	perm1ID := uuid.New().String()
	user1ID := uuid.New().String()
	user2ID := uuid.New().String()

	tests := []struct {
		name            string
		setup           func(*BunRolesRepository, *BunPermissionsRepository, *BunRolePermissionsRepository, context.Context)
		roleID          string
		permissionID    string
		grantedByUserID *string
		wantErr         error
	}{
		{
			name:         "success",
			roleID:       roleID,
			permissionID: perm1ID,
			setup: func(rolesRepo *BunRolesRepository, permissionsRepo *BunPermissionsRepository, rolePermissionsRepo *BunRolePermissionsRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: roleID, Name: "editor", Weight: 10}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: perm1ID, Key: "users.read"}); err != nil {
					panic(err)
				}
			},
		},
		{
			name:         "duplicate grant returns conflict",
			roleID:       roleID,
			permissionID: perm1ID,
			setup: func(rolesRepo *BunRolesRepository, permissionsRepo *BunPermissionsRepository, rolePermissionsRepo *BunRolePermissionsRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: roleID, Name: "editor", Weight: 10}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: perm1ID, Key: "users.read"}); err != nil {
					panic(err)
				}
				if err := rolePermissionsRepo.AddRolePermission(ctx, roleID, perm1ID, nil); err != nil {
					panic(err)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t, user1ID, user2ID)
			rolesRepo := NewBunRolesRepository(db)
			permissionsRepo := NewBunPermissionsRepository(db)
			rolePermissionsRepo := NewBunRolePermissionsRepository(db)
			ctx := context.Background()

			if tc.setup != nil {
				tc.setup(rolesRepo, permissionsRepo, rolePermissionsRepo, ctx)
			}

			err := rolePermissionsRepo.AddRolePermission(ctx, tc.roleID, tc.permissionID, tc.grantedByUserID)
			if tc.name == "duplicate grant returns conflict" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			if tc.wantErr != nil {
				return
			}

			permissions, err := rolePermissionsRepo.GetRolePermissions(ctx, tc.roleID)
			if err != nil {
				t.Fatalf("failed to fetch role permissions: %v", err)
			}
			if len(permissions) != 1 || permissions[0].PermissionID != tc.permissionID {
				t.Fatalf("unexpected permissions: %#v", permissions)
			}
			if permissions[0].GrantedAt == nil {
				t.Fatal("expected granted_at to be populated")
			}
		})
	}
}

func TestBunRolePermissionsRepositoryReplaceRolePermissions(t *testing.T) {
	t.Parallel()

	roleID := uuid.New().String()
	perm1ID := uuid.New().String()
	perm2ID := uuid.New().String()
	user1ID := uuid.New().String()
	user2ID := uuid.New().String()

	tests := []struct {
		name          string
		seed          func(*BunRolesRepository, *BunPermissionsRepository, *BunRolePermissionsRepository, context.Context)
		roleID        string
		permissionIDs []string
		wantIDs       []string
	}{
		{
			name:          "success",
			roleID:        roleID,
			permissionIDs: []string{perm2ID, perm1ID},
			seed: func(rolesRepo *BunRolesRepository, permissionsRepo *BunPermissionsRepository, rolePermissionsRepo *BunRolePermissionsRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: roleID, Name: "editor", Weight: 10}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: perm1ID, Key: "users.read"}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: perm2ID, Key: "users.write"}); err != nil {
					panic(err)
				}
				if err := rolePermissionsRepo.AddRolePermission(ctx, roleID, perm2ID, nil); err != nil {
					panic(err)
				}
			},
			wantIDs: []string{perm1ID, perm2ID},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t, user1ID, user2ID)
			rolesRepo := NewBunRolesRepository(db)
			permissionsRepo := NewBunPermissionsRepository(db)
			rolePermissionsRepo := NewBunRolePermissionsRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				tc.seed(rolesRepo, permissionsRepo, rolePermissionsRepo, ctx)
			}

			if err := rolePermissionsRepo.ReplaceRolePermissions(ctx, tc.roleID, tc.permissionIDs, nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			permissions, err := rolePermissionsRepo.GetRolePermissions(ctx, tc.roleID)
			if err != nil {
				t.Fatalf("failed to fetch role permissions: %v", err)
			}
			if len(permissions) != len(tc.wantIDs) {
				t.Fatalf("expected %d permissions, got %d", len(tc.wantIDs), len(permissions))
			}
			for i, wantID := range tc.wantIDs {
				if permissions[i].PermissionID != wantID {
					t.Fatalf("expected permission %s at index %d, got %#v", wantID, i, permissions[i])
				}
			}
		})
	}
}

func TestBunRolePermissionsRepositoryRemoveRolePermission(t *testing.T) {
	t.Parallel()

	roleID := uuid.New().String()
	perm1ID := uuid.New().String()
	user1ID := uuid.New().String()
	user2ID := uuid.New().String()

	tests := []struct {
		name         string
		seed         func(*BunRolesRepository, *BunPermissionsRepository, *BunRolePermissionsRepository, context.Context)
		roleID       string
		permissionID string
		wantExists   bool
	}{
		{
			name:         "success",
			roleID:       roleID,
			permissionID: perm1ID,
			seed: func(rolesRepo *BunRolesRepository, permissionsRepo *BunPermissionsRepository, rolePermissionsRepo *BunRolePermissionsRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: roleID, Name: "editor", Weight: 10}); err != nil {
					panic(err)
				}
				if err := permissionsRepo.CreatePermission(ctx, &types.Permission{ID: perm1ID, Key: "users.read"}); err != nil {
					panic(err)
				}
				if err := rolePermissionsRepo.AddRolePermission(ctx, roleID, perm1ID, nil); err != nil {
					panic(err)
				}
			},
			wantExists: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t, user1ID, user2ID)
			rolesRepo := NewBunRolesRepository(db)
			permissionsRepo := NewBunPermissionsRepository(db)
			rolePermissionsRepo := NewBunRolePermissionsRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				tc.seed(rolesRepo, permissionsRepo, rolePermissionsRepo, ctx)
			}

			if err := rolePermissionsRepo.RemoveRolePermission(ctx, tc.roleID, tc.permissionID); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			permissions, err := rolePermissionsRepo.GetRolePermissions(ctx, tc.roleID)
			if err != nil {
				t.Fatalf("failed to fetch role permissions: %v", err)
			}
			if tc.wantExists {
				if len(permissions) == 0 {
					t.Fatal("expected permission to remain")
				}
				return
			}
			if len(permissions) != 0 {
				t.Fatalf("expected no permissions after remove, got %#v", permissions)
			}
		})
	}
}
