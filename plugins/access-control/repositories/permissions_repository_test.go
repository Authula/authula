package repositories

import (
	"context"
	"strings"
	"testing"

	plugintests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestBunPermissionsRepositoryCreatePermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		permission      *types.Permission
		wantErr         error
		wantID          string
		wantKey         string
		wantDescription *string
	}{
		{
			name:            "success",
			permission:      &types.Permission{ID: "p1", Key: "users.read", Description: new("Read users"), IsSystem: false},
			wantID:          "p1",
			wantKey:         "users.read",
			wantDescription: new("Read users"),
		},
		{
			name:       "duplicate key returns conflict",
			permission: &types.Permission{ID: "p2", Key: "users.read", Description: new("Duplicate"), IsSystem: false},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			repo := NewBunPermissionsRepository(db)
			ctx := context.Background()

			if tc.name == "duplicate key returns conflict" {
				if err := repo.CreatePermission(ctx, &types.Permission{ID: "p1", Key: "users.read", Description: new("Read users"), IsSystem: false}); err != nil {
					t.Fatalf("failed to seed permission: %v", err)
				}
			}

			err := repo.CreatePermission(ctx, tc.permission)
			if tc.name == "duplicate key returns conflict" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "UNIQUE constraint failed: access_control_permissions.key") {
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

			stored, err := repo.GetPermissionByID(ctx, tc.wantID)
			if err != nil {
				t.Fatalf("failed to fetch stored permission: %v", err)
			}
			if stored == nil {
				t.Fatal("expected stored permission, got nil")
			}
			if stored.ID != tc.wantID || stored.Key != tc.wantKey {
				t.Fatalf("unexpected stored permission: %#v", stored)
			}
			if tc.wantDescription != nil {
				if stored.Description == nil || *stored.Description != *tc.wantDescription {
					t.Fatalf("expected description %q, got %#v", *tc.wantDescription, stored.Description)
				}
			}
			if stored.CreatedAt.IsZero() || stored.UpdatedAt.IsZero() {
				t.Fatal("expected timestamps to be populated")
			}
		})
	}
}

func TestBunPermissionsRepositoryGetAllPermissions(t *testing.T) {
	t.Parallel()

	db := plugintests.SetupRepoDB(t)
	repo := NewBunPermissionsRepository(db)
	ctx := context.Background()

	if err := repo.CreatePermission(ctx, &types.Permission{ID: "p2", Key: "users.write"}); err != nil {
		t.Fatalf("failed to seed permission p2: %v", err)
	}
	if err := repo.CreatePermission(ctx, &types.Permission{ID: "p1", Key: "users.read"}); err != nil {
		t.Fatalf("failed to seed permission p1: %v", err)
	}

	permissions, err := repo.GetAllPermissions(ctx)
	if err != nil {
		t.Fatalf("failed to get permissions: %v", err)
	}
	if len(permissions) != 2 {
		t.Fatalf("expected 2 permissions, got %d", len(permissions))
	}
	if permissions[0].ID != "p2" || permissions[1].ID != "p1" {
		t.Fatalf("expected permissions ordered by creation time, got %#v", permissions)
	}
}

func TestBunPermissionsRepositoryGetPermissionByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		permissionID   string
		seedPermission *types.Permission
		wantNil        bool
	}{
		{
			name:         "not found",
			permissionID: "missing",
			wantNil:      true,
		},
		{
			name:           "success",
			permissionID:   "p1",
			seedPermission: &types.Permission{ID: "p1", Key: "users.read", Description: new("Read users"), IsSystem: false},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			repo := NewBunPermissionsRepository(db)
			ctx := context.Background()

			if tc.seedPermission != nil {
				if err := repo.CreatePermission(ctx, tc.seedPermission); err != nil {
					t.Fatalf("failed to seed permission: %v", err)
				}
			}

			permission, err := repo.GetPermissionByID(ctx, tc.permissionID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil {
				if permission != nil {
					t.Fatalf("expected nil permission, got %#v", permission)
				}
				return
			}
			if permission == nil || permission.ID != tc.permissionID {
				t.Fatalf("unexpected permission: %#v", permission)
			}
		})
	}
}

func TestBunPermissionsRepositoryUpdatePermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		seedPermission *types.Permission
		permissionID   string
		description    *string
		wantUpdated    bool
	}{
		{
			name:         "missing permission",
			permissionID: "missing",
			description:  new("updated"),
			wantUpdated:  false,
		},
		{
			name:           "success",
			seedPermission: &types.Permission{ID: "p1", Key: "users.read", Description: new("Read users"), IsSystem: false},
			permissionID:   "p1",
			description:    new("updated"),
			wantUpdated:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			repo := NewBunPermissionsRepository(db)
			ctx := context.Background()

			if tc.seedPermission != nil {
				if err := repo.CreatePermission(ctx, tc.seedPermission); err != nil {
					t.Fatalf("failed to seed permission: %v", err)
				}
			}

			updated, err := repo.UpdatePermission(ctx, tc.permissionID, tc.description)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if updated != tc.wantUpdated {
				t.Fatalf("expected updated=%v, got %v", tc.wantUpdated, updated)
			}

			if tc.wantUpdated {
				permission, err := repo.GetPermissionByID(ctx, tc.permissionID)
				if err != nil {
					t.Fatalf("failed to fetch permission: %v", err)
				}
				if permission == nil || permission.Description == nil || *permission.Description != *tc.description {
					t.Fatalf("unexpected permission after update: %#v", permission)
				}
			}
		})
	}
}

func TestBunPermissionsRepositoryDeletePermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		seedPermission *types.Permission
		permissionID   string
		wantDeleted    bool
	}{
		{
			name:         "missing permission",
			permissionID: "missing",
			wantDeleted:  false,
		},
		{
			name:           "success",
			seedPermission: &types.Permission{ID: "p1", Key: "users.read", Description: new("Read users"), IsSystem: false},
			permissionID:   "p1",
			wantDeleted:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			repo := NewBunPermissionsRepository(db)
			ctx := context.Background()

			if tc.seedPermission != nil {
				if err := repo.CreatePermission(ctx, tc.seedPermission); err != nil {
					t.Fatalf("failed to seed permission: %v", err)
				}
			}

			deleted, err := repo.DeletePermission(ctx, tc.permissionID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if deleted != tc.wantDeleted {
				t.Fatalf("expected deleted=%v, got %v", tc.wantDeleted, deleted)
			}
		})
	}
}
