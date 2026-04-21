package repositories

import (
	"context"
	"strings"
	"testing"

	plugintests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestBunRolesRepositoryCreateRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		role       *types.Role
		seed       *types.Role
		wantErr    error
		wantID     string
		wantName   string
		wantDesc   *string
		wantWeight int
	}{
		{
			name:       "success",
			role:       &types.Role{ID: "r1", Name: "editor", Description: new("Editor role"), Weight: 10, IsSystem: false},
			wantID:     "r1",
			wantName:   "editor",
			wantDesc:   new("Editor role"),
			wantWeight: 10,
		},
		{
			name:    "duplicate name returns conflict",
			role:    &types.Role{ID: "r2", Name: "editor", Description: new("Duplicate role"), Weight: 10, IsSystem: false},
			seed:    &types.Role{ID: "r1", Name: "editor", Description: new("Original role"), Weight: 10, IsSystem: false},
			wantErr: nil,
		},
		{
			name:    "query error returns wrapped error",
			role:    &types.Role{ID: "r3", Name: "reviewer", Description: new("Reviewer role"), Weight: 10, IsSystem: false},
			wantErr: nil,
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

			if tc.name == "query error returns wrapped error" {
				if err := db.Close(); err != nil {
					t.Fatalf("failed to close db: %v", err)
				}
			}

			err := repo.CreateRole(ctx, tc.role)
			if tc.name == "query error returns wrapped error" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != "sql: database is closed" {
					t.Fatalf("expected direct db error, got %v", err)
				}
				return
			}
			if tc.name == "duplicate name returns conflict" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "UNIQUE constraint failed: access_control_roles.name") {
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

			stored, err := repo.GetRoleByID(ctx, tc.wantID)
			if err != nil {
				t.Fatalf("failed to fetch stored role: %v", err)
			}
			if stored == nil {
				t.Fatal("expected stored role, got nil")
			} else {
				if stored.ID != tc.wantID || stored.Name != tc.wantName {
					t.Fatalf("unexpected stored role: %#v", stored)
				}
				if !stringPtrEqual(stored.Description, tc.wantDesc) {
					t.Fatalf("expected description %#v, got %#v", tc.wantDesc, stored.Description)
				}
				if stored.Weight != tc.wantWeight {
					t.Fatalf("expected weight %d, got %d", tc.wantWeight, stored.Weight)
				}
				if stored.CreatedAt.IsZero() || stored.UpdatedAt.IsZero() {
					t.Fatal("expected timestamps to be populated")
				}
			}
		})
	}
}

func TestBunRolesRepositoryGetAllRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		seedRoles   []*types.Role
		wantIDs     []string
		wantNames   []string
		wantDescs   []*string
		wantWeights []int
		wantErrMsg  string
	}{
		{
			name:        "empty result",
			wantIDs:     []string{},
			wantNames:   []string{},
			wantDescs:   []*string{},
			wantWeights: []int{},
		},
		{
			name: "returns roles ordered by weight",
			seedRoles: []*types.Role{
				{ID: "r2", Name: "viewer", Description: new("Viewer role"), Weight: 10},
				{ID: "r1", Name: "editor", Description: new("Editor role"), Weight: 30},
			},
			wantIDs:     []string{"r1", "r2"},
			wantNames:   []string{"editor", "viewer"},
			wantDescs:   []*string{new("Editor role"), new("Viewer role")},
			wantWeights: []int{30, 10},
		},
		{
			name:       "query error",
			wantErrMsg: "failed to get roles",
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

			if tc.wantErrMsg != "" {
				if err := db.Close(); err != nil {
					t.Fatalf("failed to close db: %v", err)
				}
			}

			roles, err := repo.GetAllRoles(ctx)
			if tc.wantErrMsg != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Fatalf("expected direct db error, got %v", err)
				}
				if roles != nil {
					t.Fatalf("expected nil roles on error, got %#v", roles)
				}
				return
			}
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
				if roles[i].Weight != tc.wantWeights[i] {
					t.Fatalf("unexpected weight at %d: %#v", i, roles[i])
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
		wantWeight int
		wantErrMsg string
	}{
		{
			name:    "not found",
			roleID:  "missing",
			wantNil: true,
		},
		{
			name:       "success",
			roleID:     "r1",
			seedRole:   &types.Role{ID: "r1", Name: "editor", Description: new("Editor role"), IsSystem: true, Weight: 20},
			wantName:   "editor",
			wantDesc:   new("Editor role"),
			wantSystem: true,
			wantWeight: 20,
		},
		{
			name:       "query error",
			roleID:     "r1",
			wantErrMsg: "failed to get role by id",
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

			if tc.wantErrMsg != "" {
				if err := db.Close(); err != nil {
					t.Fatalf("failed to close db: %v", err)
				}
			}

			role, err := repo.GetRoleByID(ctx, tc.roleID)
			if tc.wantErrMsg != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if role != nil {
					t.Fatalf("expected nil role on error, got %#v", role)
				}
				if !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Fatalf("expected direct db error, got %v", err)
				}
				return
			}
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
			} else {
				if role.ID != tc.roleID || role.Name != tc.wantName || role.IsSystem != tc.wantSystem {
					t.Fatalf("unexpected role: %#v", role)
				}
				if !stringPtrEqual(role.Description, tc.wantDesc) {
					t.Fatalf("expected description %#v, got %#v", tc.wantDesc, role.Description)
				}
				if role.Weight != tc.wantWeight {
					t.Fatalf("expected weight %d, got %d", tc.wantWeight, role.Weight)
				}
			}
		})
	}
}

func TestBunRolesRepositoryGetRoleByName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		roleName   string
		seedRole   *types.Role
		wantNil    bool
		wantName   string
		wantDesc   *string
		wantSystem bool
		wantWeight int
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:     "not found",
			roleName: "missing",
			wantNil:  true,
		},
		{
			name:       "success",
			roleName:   "editor",
			seedRole:   &types.Role{ID: "r1", Name: "editor", Description: new("Editor role"), Weight: 20, IsSystem: true},
			wantName:   "editor",
			wantDesc:   new("Editor role"),
			wantSystem: true,
			wantWeight: 20,
		},
		{
			name:       "query error",
			roleName:   "editor",
			seedRole:   &types.Role{ID: "r1", Name: "editor", Description: new("Editor role"), Weight: 10, IsSystem: false},
			wantErr:    true,
			wantErrMsg: "failed to get role by name",
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

			if tc.wantErr {
				if err := db.Close(); err != nil {
					t.Fatalf("failed to close db: %v", err)
				}
			}

			role, err := repo.GetRoleByName(ctx, tc.roleName)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if role != nil {
					t.Fatalf("expected nil role on error, got %#v", role)
				}
				if !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Fatalf("expected direct db error, got %v", err)
				}
				return
			}
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
			} else {
				if role.Name != tc.wantName || role.IsSystem != tc.wantSystem {
					t.Fatalf("unexpected role: %#v", role)
				}
				if !stringPtrEqual(role.Description, tc.wantDesc) {
					t.Fatalf("expected description %#v, got %#v", tc.wantDesc, role.Description)
				}
				if role.Weight != tc.wantWeight {
					t.Fatalf("expected weight %d, got %d", tc.wantWeight, role.Weight)
				}
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
		weight      *int
		wantUpdated bool
		wantName    *string
		wantDesc    *string
		wantWeight  *int
		wantErrMsg  string
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
			seedRole:    &types.Role{ID: "r1", Name: "editor", Description: new("Editor role"), IsSystem: false, Weight: 10},
			roleID:      "r1",
			nameValue:   new("editor-updated"),
			wantUpdated: true,
			wantName:    new("editor-updated"),
			wantDesc:    new("Editor role"),
			wantWeight:  new(10),
		},
		{
			name:        "update description only",
			seedRole:    &types.Role{ID: "r2", Name: "viewer", Description: new("Viewer role"), IsSystem: false, Weight: 10},
			roleID:      "r2",
			description: new("Viewer role updated"),
			wantUpdated: true,
			wantName:    new("viewer"),
			wantDesc:    new("Viewer role updated"),
			wantWeight:  new(10),
		},
		{
			name:        "update name and description",
			seedRole:    &types.Role{ID: "r3", Name: "author", Description: new("Author role"), IsSystem: false, Weight: 10},
			roleID:      "r3",
			nameValue:   new("author-updated"),
			description: new("Author role updated"),
			wantUpdated: true,
			wantName:    new("author-updated"),
			wantDesc:    new("Author role updated"),
			wantWeight:  new(10),
		},
		{
			name:        "update with no fields still updates timestamp",
			seedRole:    &types.Role{ID: "r4", Name: "reviewer", Description: new("Reviewer role"), IsSystem: false, Weight: 10},
			roleID:      "r4",
			wantUpdated: true,
			wantName:    new("reviewer"),
			wantDesc:    new("Reviewer role"),
			wantWeight:  new(10),
		},
		{
			name:       "query error",
			roleID:     "r5",
			nameValue:  new("updated"),
			wantErrMsg: "sql: database is closed",
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

			if tc.wantErrMsg != "" {
				if err := db.Close(); err != nil {
					t.Fatalf("failed to close db: %v", err)
				}
			}

			updated, err := repo.UpdateRole(ctx, tc.roleID, tc.nameValue, tc.description, tc.weight)
			if tc.wantErrMsg != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if updated {
					t.Fatal("expected updated=false on error")
				}
				if !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Fatalf("expected direct db error, got %v", err)
				}
				return
			}
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
			} else {
				if role.Name != derefOrEmpty(tc.wantName) {
					t.Fatalf("expected name %q, got %q", derefOrEmpty(tc.wantName), role.Name)
				}
				if !stringPtrEqual(role.Description, tc.wantDesc) {
					t.Fatalf("expected description %#v, got %#v", tc.wantDesc, role.Description)
				}
				if tc.wantWeight != nil && role.Weight != *tc.wantWeight {
					t.Fatalf("expected weight %d, got %d", *tc.wantWeight, role.Weight)
				}
				if role.UpdatedAt.IsZero() {
					t.Fatal("expected updated_at to be populated")
				}
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
		wantErrMsg  string
	}{
		{
			name:        "missing role",
			roleID:      "missing",
			wantDeleted: false,
		},
		{
			name:        "success",
			seedRole:    &types.Role{ID: "r1", Name: "editor", Description: new("Editor role"), Weight: 10, IsSystem: false},
			roleID:      "r1",
			wantDeleted: true,
		},
		{
			name:       "query error",
			roleID:     "r5",
			wantErrMsg: "sql: database is closed",
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

			if tc.wantErrMsg != "" {
				if err := db.Close(); err != nil {
					t.Fatalf("failed to close db: %v", err)
				}
			}

			deleted, err := repo.DeleteRole(ctx, tc.roleID)
			if tc.wantErrMsg != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if deleted {
					t.Fatal("expected deleted=false on error")
				}
				if !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Fatalf("expected direct db error, got %v", err)
				}
				return
			}
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
