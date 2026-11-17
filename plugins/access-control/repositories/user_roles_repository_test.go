package repositories

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	plugintests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestBunUserRolesRepositoryGetUserRoles(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	futureExpiry := time.Unix(now.Add(24*time.Hour).Unix(), 0).UTC()
	roleDescription := new("Editor role")
	assignedBy := new("u2")

	tests := []struct {
		name      string
		seed      func(*BunRolesRepository, *BunUserRolesRepository, context.Context)
		userID    string
		wantRoles []types.UserRoleInfo
	}{
		{
			name:      "empty result",
			userID:    "missing-user",
			wantRoles: []types.UserRoleInfo{},
		},
		{
			name:   "returns assigned roles ordered by role name",
			userID: "u1",
			seed: func(rolesRepo *BunRolesRepository, userRolesRepo *BunUserRolesRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "r2", Name: "viewer", Weight: 10}); err != nil {
					panic(err)
				}
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "r1", Name: "editor", Description: roleDescription, Weight: 30}); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, "u1", "r1", assignedBy, &futureExpiry); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, "u1", "r2", nil, nil); err != nil {
					panic(err)
				}
			},
			wantRoles: []types.UserRoleInfo{
				{
					RoleID:           "r1",
					RoleName:         "editor",
					RoleDescription:  roleDescription,
					RoleWeight:       30,
					AssignedByUserID: assignedBy,
					ExpiresAt:        &futureExpiry,
				},
				{
					RoleID:           "r2",
					RoleName:         "viewer",
					RoleDescription:  nil,
					RoleWeight:       10,
					AssignedByUserID: nil,
					ExpiresAt:        nil,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			rolesRepo := NewBunRolesRepository(db)
			userRolesRepo := NewBunUserRolesRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				tc.seed(rolesRepo, userRolesRepo, ctx)
			}

			roles, err := userRolesRepo.GetUserRoles(ctx, tc.userID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if roles == nil {
				t.Fatal("expected roles slice, got nil")
			}
			if len(roles) != len(tc.wantRoles) {
				t.Fatalf("expected %d roles, got %d", len(tc.wantRoles), len(roles))
			}
			for i := range tc.wantRoles {
				if roles[i].RoleID != tc.wantRoles[i].RoleID || roles[i].RoleName != tc.wantRoles[i].RoleName {
					t.Fatalf("unexpected role at %d: %#v", i, roles[i])
				}
				if !reflect.DeepEqual(roles[i].RoleDescription, tc.wantRoles[i].RoleDescription) {
					t.Fatalf("unexpected role description at %d: %#v", i, roles[i])
				}
				if !reflect.DeepEqual(roles[i].AssignedByUserID, tc.wantRoles[i].AssignedByUserID) || !reflect.DeepEqual(roles[i].ExpiresAt, tc.wantRoles[i].ExpiresAt) {
					t.Fatalf("unexpected assignment metadata at %d: %#v", i, roles[i])
				}
				if roles[i].AssignedAt == nil {
					t.Fatalf("expected assigned_at to be populated at %d", i)
				}
			}
		})
	}
}

func TestBunUserRolesRepositoryReplaceUserRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		seed        func(*BunRolesRepository, *BunUserRolesRepository, context.Context)
		userID      string
		roleIDs     []string
		wantRoleIDs []string
	}{
		{
			name:    "replaces all roles",
			userID:  "u1",
			roleIDs: []string{"r2", "r1"},
			seed: func(rolesRepo *BunRolesRepository, userRolesRepo *BunUserRolesRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "r1", Name: "editor", Weight: 10}); err != nil {
					panic(err)
				}
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "r2", Name: "viewer", Weight: 20}); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, "u1", "r1", nil, nil); err != nil {
					panic(err)
				}
			},
			wantRoleIDs: []string{"r2", "r1"},
		},
		{
			name:        "empty list clears roles",
			userID:      "u1",
			roleIDs:     []string{},
			wantRoleIDs: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			rolesRepo := NewBunRolesRepository(db)
			userRolesRepo := NewBunUserRolesRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				tc.seed(rolesRepo, userRolesRepo, ctx)
			}

			if err := userRolesRepo.ReplaceUserRoles(ctx, tc.userID, tc.roleIDs, nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			roles, err := userRolesRepo.GetUserRoles(ctx, tc.userID)
			if err != nil {
				t.Fatalf("failed to fetch roles: %v", err)
			}
			if len(roles) != len(tc.wantRoleIDs) {
				t.Fatalf("expected %d roles, got %d", len(tc.wantRoleIDs), len(roles))
			}
			for i, wantRoleID := range tc.wantRoleIDs {
				if roles[i].RoleID != wantRoleID {
					t.Fatalf("expected role %s at index %d, got %#v", wantRoleID, i, roles[i])
				}
				if roles[i].RoleWeight == 0 {
					t.Fatalf("expected role weight to be populated at index %d", i)
				}
			}
		})
	}
}

func TestBunUserRolesRepositoryAssignUserRole(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	futureExpiry := time.Unix(now.Add(24*time.Hour).Unix(), 0).UTC()

	tests := []struct {
		name      string
		seed      func(*BunRolesRepository, *BunUserRolesRepository, context.Context)
		userID    string
		roleID    string
		expiresAt *time.Time
		wantErr   error
	}{
		{
			name:      "success",
			userID:    "u1",
			roleID:    "r1",
			expiresAt: &futureExpiry,
			seed: func(rolesRepo *BunRolesRepository, userRolesRepo *BunUserRolesRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "r1", Name: "editor", Weight: 10}); err != nil {
					panic(err)
				}
			},
		},
		{
			name:   "duplicate assignment returns conflict",
			userID: "u1",
			roleID: "r1",
			seed: func(rolesRepo *BunRolesRepository, userRolesRepo *BunUserRolesRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "r1", Name: "editor", Weight: 10}); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, "u1", "r1", nil, nil); err != nil {
					panic(err)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			rolesRepo := NewBunRolesRepository(db)
			userRolesRepo := NewBunUserRolesRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				tc.seed(rolesRepo, userRolesRepo, ctx)
			}

			err := userRolesRepo.AssignUserRole(ctx, tc.userID, tc.roleID, nil, tc.expiresAt)
			if tc.name == "duplicate assignment returns conflict" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "UNIQUE constraint failed: access_control_user_roles.user_id, access_control_user_roles.role_id") {
					t.Fatalf("expected raw unique constraint error, got %v", err)
				}
				return
			}
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			if tc.wantErr != nil {
				return
			}

			roles, err := userRolesRepo.GetUserRoles(ctx, tc.userID)
			if err != nil {
				t.Fatalf("failed to fetch assigned role: %v", err)
			}
			if len(roles) != 1 || roles[0].RoleID != tc.roleID || roles[0].RoleName != "editor" {
				t.Fatalf("unexpected roles after assign: %#v", roles)
			}
			if roles[0].RoleWeight != 10 {
				t.Fatalf("expected role weight 10, got %#v", roles[0])
			}
			if roles[0].AssignedAt == nil {
				t.Fatal("expected assigned_at to be populated")
			}
			if tc.expiresAt != nil && !reflect.DeepEqual(roles[0].ExpiresAt, tc.expiresAt) {
				t.Fatalf("unexpected expiry: %#v", roles[0])
			}
		})
	}
}

func TestBunUserRolesRepositoryRemoveUserRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		seed      func(*BunRolesRepository, *BunUserRolesRepository, context.Context)
		userID    string
		roleID    string
		wantRoles []types.UserRoleInfo
	}{
		{
			name:   "success",
			userID: "u1",
			roleID: "r1",
			seed: func(rolesRepo *BunRolesRepository, userRolesRepo *BunUserRolesRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "r1", Name: "editor", Weight: 10}); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, "u1", "r1", nil, nil); err != nil {
					panic(err)
				}
			},
			wantRoles: []types.UserRoleInfo{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			rolesRepo := NewBunRolesRepository(db)
			userRolesRepo := NewBunUserRolesRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				tc.seed(rolesRepo, userRolesRepo, ctx)
			}

			if err := userRolesRepo.RemoveUserRole(ctx, tc.userID, tc.roleID); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			roles, err := userRolesRepo.GetUserRoles(ctx, tc.userID)
			if err != nil {
				t.Fatalf("failed to fetch roles: %v", err)
			}
			if len(roles) != len(tc.wantRoles) {
				t.Fatalf("expected %d roles after remove, got %#v", len(tc.wantRoles), roles)
			}
		})
	}
}

func TestBunUserRolesRepositoryCountUsersByRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		seed       func(*BunRolesRepository, *BunUserRolesRepository, context.Context)
		roleID     string
		wantCount  int
		wantErrMsg string
	}{
		{
			name:      "no assignments",
			roleID:    "role-1",
			wantCount: 0,
		},
		{
			name:   "counts users",
			roleID: "role-1",
			seed: func(rolesRepo *BunRolesRepository, userRolesRepo *BunUserRolesRepository, ctx context.Context) {
				if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "role-1", Name: "editor", Weight: 10}); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, "u1", "role-1", nil, nil); err != nil {
					panic(err)
				}
				if err := userRolesRepo.AssignUserRole(ctx, "u2", "role-1", nil, nil); err != nil {
					panic(err)
				}
			},
			wantCount: 2,
		},
		{
			name:       "query error",
			roleID:     "role-1",
			wantErrMsg: "failed to count users by role",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			rolesRepo := NewBunRolesRepository(db)
			userRolesRepo := NewBunUserRolesRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				tc.seed(rolesRepo, userRolesRepo, ctx)
			}
			if tc.wantErrMsg != "" {
				if err := db.Close(); err != nil {
					t.Fatalf("failed to close db: %v", err)
				}
			}

			count, err := userRolesRepo.CountUsersByRole(ctx, tc.roleID)
			if tc.wantErrMsg != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if count != 0 {
					t.Fatalf("expected 0 count on error, got %d", count)
				}
				if !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErrMsg, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if count != tc.wantCount {
				t.Fatalf("expected count %d, got %d", tc.wantCount, count)
			}
		})
	}
}

func TestBunUserRolesRepositoryQueryErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(context.Context, *BunUserRolesRepository) error
	}{
		{
			name: "get user roles",
			run: func(ctx context.Context, repo *BunUserRolesRepository) error {
				_, err := repo.GetUserRoles(ctx, "u1")
				return err
			},
		},
		{
			name: "replace user roles",
			run: func(ctx context.Context, repo *BunUserRolesRepository) error {
				return repo.ReplaceUserRoles(ctx, "u1", []string{"role-1"}, nil)
			},
		},
		{
			name: "remove user role",
			run: func(ctx context.Context, repo *BunUserRolesRepository) error {
				return repo.RemoveUserRole(ctx, "u1", "role-1")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			repo := NewBunUserRolesRepository(db)
			if err := db.Close(); err != nil {
				t.Fatalf("failed to close db: %v", err)
			}

			if err := tc.run(context.Background(), repo); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestBunUserRolesRepositoryReplaceUserRolesDuplicate(t *testing.T) {
	t.Parallel()

	db := plugintests.SetupRepoDB(t)
	rolesRepo := NewBunRolesRepository(db)
	userRolesRepo := NewBunUserRolesRepository(db)
	ctx := context.Background()

	if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "role-1", Name: "editor", Weight: 10}); err != nil {
		t.Fatalf("failed to seed role: %v", err)
	}
	if err := rolesRepo.CreateRole(ctx, &types.Role{ID: "role-2", Name: "viewer", Weight: 10}); err != nil {
		t.Fatalf("failed to seed role: %v", err)
	}
	if err := userRolesRepo.AssignUserRole(ctx, "u1", "role-2", nil, nil); err != nil {
		t.Fatalf("failed to seed user role: %v", err)
	}

	err := userRolesRepo.ReplaceUserRoles(ctx, "u1", []string{"role-1", "role-1"}, nil)
	if err == nil {
		t.Fatal("expected duplicate insert error, got nil")
	}

	roles, err := userRolesRepo.GetUserRoles(ctx, "u1")
	if err != nil {
		t.Fatalf("failed to fetch user roles: %v", err)
	}
	if len(roles) != 1 || roles[0].RoleID != "role-2" {
		t.Fatalf("expected rollback to preserve existing role, got %#v", roles)
	}
}
