package repositories

import (
	"context"
	"testing"

	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
	plugintests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestBunRolesRepositoryCreateRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		role     *types.Role
		seed     *types.Role
		wantErr  error
		wantID   string
		wantName string
		wantDesc *string
	}{
		{
			name:     "success",
			role:     &types.Role{ID: "r1", Name: "editor", Description: new("Editor role"), IsSystem: false},
			wantID:   "r1",
			wantName: "editor",
			wantDesc: new("Editor role"),
		},
		{
			name:    "duplicate name returns conflict",
			role:    &types.Role{ID: "r2", Name: "editor", Description: new("Duplicate role"), IsSystem: false},
			seed:    &types.Role{ID: "r1", Name: "editor", Description: new("Original role"), IsSystem: false},
			wantErr: accesscontrolconstants.ErrConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			repo := NewBunRolesRepository(db)
			ctx := context.Background()

			if tc.seed != nil {
				if err := repo.CreateRole(ctx, tc.seed); err != nil {
					t.Fatalf("failed to seed role: %v", err)
				}
			}

			err := repo.CreateRole(ctx, tc.role)
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			if tc.wantErr != nil {
				return
			}

			stored, err := repo.GetRoleByID(ctx, tc.wantID)
			if err != nil {
				t.Fatalf("failed to fetch stored role: %v", err)
			}
			if stored == nil {
				t.Fatal("expected stored role, got nil")
			}
			if stored.ID != tc.wantID || stored.Name != tc.wantName {
				t.Fatalf("unexpected stored role: %#v", stored)
			}
			if !stringPtrEqual(stored.Description, tc.wantDesc) {
				t.Fatalf("expected description %#v, got %#v", tc.wantDesc, stored.Description)
			}
			if stored.CreatedAt.IsZero() || stored.UpdatedAt.IsZero() {
				t.Fatal("expected timestamps to be populated")
			}
		})
	}
}

func TestBunRolesRepositoryGetAllRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		seedRoles []*types.Role
		wantIDs   []string
		wantNames []string
		wantDescs []*string
	}{
		{
			name:      "empty result",
			wantIDs:   []string{},
			wantNames: []string{},
			wantDescs: []*string{},
		},
		{
			name: "returns roles ordered by creation time",
			seedRoles: []*types.Role{
				{ID: "r2", Name: "viewer", Description: new("Viewer role")},
				{ID: "r1", Name: "editor", Description: new("Editor role")},
			},
			wantIDs:   []string{"r2", "r1"},
			wantNames: []string{"viewer", "editor"},
			wantDescs: []*string{new("Viewer role"), new("Editor role")},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			repo := NewBunRolesRepository(db)
			ctx := context.Background()

			for _, role := range tc.seedRoles {
				if err := repo.CreateRole(ctx, role); err != nil {
					t.Fatalf("failed to seed role %s: %v", role.ID, err)
				}
			}

			roles, err := repo.GetAllRoles(ctx)
			if err != nil {
				t.Fatalf("failed to get roles: %v", err)
			}
			if roles == nil {
				t.Fatal("expected roles slice, got nil")
			}
			if len(roles) != len(tc.wantIDs) {
				t.Fatalf("expected %d roles, got %d", len(tc.wantIDs), len(roles))
			}
			for i := range tc.wantIDs {
				if roles[i].ID != tc.wantIDs[i] || roles[i].Name != tc.wantNames[i] {
					t.Fatalf("unexpected role at %d: %#v", i, roles[i])
				}
				if !stringPtrEqual(roles[i].Description, tc.wantDescs[i]) {
					t.Fatalf("unexpected description at %d: %#v", i, roles[i])
				}
				if roles[i].CreatedAt.IsZero() || roles[i].UpdatedAt.IsZero() {
					t.Fatalf("expected timestamps to be populated at %d", i)
				}
			}
		})
	}
}

func TestBunRolesRepositoryGetRoleByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		roleID     string
		seedRole   *types.Role
		wantNil    bool
		wantName   string
		wantDesc   *string
		wantSystem bool
	}{
		{
			name:    "not found",
			roleID:  "missing",
			wantNil: true,
		},
		{
			name:       "success",
			roleID:     "r1",
			seedRole:   &types.Role{ID: "r1", Name: "editor", Description: new("Editor role"), IsSystem: true},
			wantName:   "editor",
			wantDesc:   new("Editor role"),
			wantSystem: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			repo := NewBunRolesRepository(db)
			ctx := context.Background()

			if tc.seedRole != nil {
				if err := repo.CreateRole(ctx, tc.seedRole); err != nil {
					t.Fatalf("failed to seed role: %v", err)
				}
			}

			role, err := repo.GetRoleByID(ctx, tc.roleID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil {
				if role != nil {
					t.Fatalf("expected nil role, got %#v", role)
				}
				return
			}
			if role == nil {
				t.Fatal("expected role, got nil")
			}
			if role.ID != tc.roleID || role.Name != tc.wantName || role.IsSystem != tc.wantSystem {
				t.Fatalf("unexpected role: %#v", role)
			}
			if !stringPtrEqual(role.Description, tc.wantDesc) {
				t.Fatalf("expected description %#v, got %#v", tc.wantDesc, role.Description)
			}
		})
	}
}

func TestBunRolesRepositoryUpdateRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		seedRole    *types.Role
		roleID      string
		nameValue   *string
		description *string
		wantUpdated bool
		wantName    *string
		wantDesc    *string
	}{
		{
			name:        "missing role",
			roleID:      "missing",
			nameValue:   new("updated"),
			description: new("updated description"),
			wantUpdated: false,
		},
		{
			name:        "update name only",
			seedRole:    &types.Role{ID: "r1", Name: "editor", Description: new("Editor role"), IsSystem: false},
			roleID:      "r1",
			nameValue:   new("editor-updated"),
			wantUpdated: true,
			wantName:    new("editor-updated"),
			wantDesc:    new("Editor role"),
		},
		{
			name:        "update description only",
			seedRole:    &types.Role{ID: "r2", Name: "viewer", Description: new("Viewer role"), IsSystem: false},
			roleID:      "r2",
			description: new("Viewer role updated"),
			wantUpdated: true,
			wantName:    new("viewer"),
			wantDesc:    new("Viewer role updated"),
		},
		{
			name:        "update name and description",
			seedRole:    &types.Role{ID: "r3", Name: "author", Description: new("Author role"), IsSystem: false},
			roleID:      "r3",
			nameValue:   new("author-updated"),
			description: new("Author role updated"),
			wantUpdated: true,
			wantName:    new("author-updated"),
			wantDesc:    new("Author role updated"),
		},
		{
			name:        "update with no fields still updates timestamp",
			seedRole:    &types.Role{ID: "r4", Name: "reviewer", Description: new("Reviewer role"), IsSystem: false},
			roleID:      "r4",
			wantUpdated: true,
			wantName:    new("reviewer"),
			wantDesc:    new("Reviewer role"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			repo := NewBunRolesRepository(db)
			ctx := context.Background()

			if tc.seedRole != nil {
				if err := repo.CreateRole(ctx, tc.seedRole); err != nil {
					t.Fatalf("failed to seed role: %v", err)
				}
			}

			updated, err := repo.UpdateRole(ctx, tc.roleID, tc.nameValue, tc.description)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if updated != tc.wantUpdated {
				t.Fatalf("expected updated=%v, got %v", tc.wantUpdated, updated)
			}

			if !tc.wantUpdated {
				return
			}

			role, err := repo.GetRoleByID(ctx, tc.roleID)
			if err != nil {
				t.Fatalf("failed to fetch updated role: %v", err)
			}
			if role == nil {
				t.Fatal("expected updated role, got nil")
			}
			if role.Name != derefOrEmpty(tc.wantName) {
				t.Fatalf("expected name %q, got %q", derefOrEmpty(tc.wantName), role.Name)
			}
			if !stringPtrEqual(role.Description, tc.wantDesc) {
				t.Fatalf("expected description %#v, got %#v", tc.wantDesc, role.Description)
			}
			if role.UpdatedAt.IsZero() {
				t.Fatal("expected updated_at to be populated")
			}
		})
	}
}

func TestBunRolesRepositoryDeleteRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		seedRole    *types.Role
		roleID      string
		wantDeleted bool
	}{
		{
			name:        "missing role",
			roleID:      "missing",
			wantDeleted: false,
		},
		{
			name:        "success",
			seedRole:    &types.Role{ID: "r1", Name: "editor", Description: new("Editor role"), IsSystem: false},
			roleID:      "r1",
			wantDeleted: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			repo := NewBunRolesRepository(db)
			ctx := context.Background()

			if tc.seedRole != nil {
				if err := repo.CreateRole(ctx, tc.seedRole); err != nil {
					t.Fatalf("failed to seed role: %v", err)
				}
			}

			deleted, err := repo.DeleteRole(ctx, tc.roleID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if deleted != tc.wantDeleted {
				t.Fatalf("expected deleted=%v, got %v", tc.wantDeleted, deleted)
			}

			if !tc.wantDeleted {
				return
			}

			role, err := repo.GetRoleByID(ctx, tc.roleID)
			if err != nil {
				t.Fatalf("failed to verify role deletion: %v", err)
			}
			if role != nil {
				t.Fatalf("expected deleted role to be absent, got %#v", role)
			}
		})
	}
}

func stringPtrEqual(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func derefOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
