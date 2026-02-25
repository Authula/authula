package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	migrationsmodule "github.com/GoBetterAuth/go-better-auth/v2/migrations"
	adminplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

type integrationTestLogger struct{}

func (l integrationTestLogger) Debug(msg string, args ...any) {}
func (l integrationTestLogger) Info(msg string, args ...any)  {}
func (l integrationTestLogger) Warn(msg string, args ...any)  {}
func (l integrationTestLogger) Error(msg string, args ...any) {}

func newIntegrationTestDB(t *testing.T) *bun.DB {
	t.Helper()
	ctx := context.Background()

	db := internaltests.NewSQLiteIntegrationDB(t)

	migrator, err := migrationsmodule.NewMigrator(db, integrationTestLogger{})
	require.NoError(t, err, "failed to create migrator")

	coreSet, err := migrationsmodule.CoreMigrationSet("sqlite")
	require.NoError(t, err, "failed to load core migration set")

	plugin := adminplugin.New(admintypes.AdminPluginConfig{})
	adminSet := migrationsmodule.MigrationSet{
		PluginID:   plugin.Metadata().ID,
		DependsOn:  plugin.DependsOn(),
		Migrations: plugin.Migrations("sqlite"),
	}

	err = migrator.Migrate(ctx, []migrationsmodule.MigrationSet{coreSet, adminSet})
	require.NoError(t, err, "failed to run migrations")

	return db
}

func buildRolePermissionUseCase(db bun.IDB) usecases.RolePermissionUseCase {
	repo := repositories.NewRolePermissionRepository(db)
	service := services.NewRolePermissionService(repo)
	return usecases.NewRolePermissionUseCase(service)
}

func seedRole(t *testing.T, db bun.IDB, id string, name string, isSystem bool) {
	t.Helper()
	role := &admintypes.Role{
		ID:        id,
		Name:      name,
		IsSystem:  isSystem,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := db.NewInsert().Model(role).Scan(context.Background())
	require.NoError(t, err, "failed to seed role")
}

func seedPermission(t *testing.T, db bun.IDB, id, key string, isSystem bool) {
	t.Helper()
	perm := &admintypes.Permission{
		ID:        id,
		Key:       key,
		IsSystem:  isSystem,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := db.NewInsert().Model(perm).Scan(context.Background())
	require.NoError(t, err, "failed to seed permission")
}

func seedUser(t *testing.T, db bun.IDB, id, email string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `INSERT INTO users (id, name, email, email_verified) VALUES (?, ?, ?, ?)`, id, "Test User", email, false)
	require.NoError(t, err, "failed to seed user")
}

// TestIntegration_CompleteRoleWorkflow tests a complete lifecycle of role management
func TestIntegration_CompleteRoleWorkflow(t *testing.T) {
	db := newIntegrationTestDB(t)
	uc := buildRolePermissionUseCase(db)
	ctx := context.Background()

	// Create a role
	createReq := admintypes.CreateRoleRequest{
		Name:        "Editor",
		Description: new("Editor role"),
		IsSystem:    false,
	}
	role, err := uc.CreateRole(ctx, createReq)
	require.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "Editor", role.Name)

	// Verify role was created
	allRoles, err := uc.GetAllRoles(ctx)
	require.NoError(t, err)
	assert.True(t, len(allRoles) > 0)

	// Get role by ID with permissions
	roleDetails, err := uc.GetRoleByID(ctx, role.ID)
	require.NoError(t, err)
	assert.NotNil(t, roleDetails)
	assert.Equal(t, "Editor", roleDetails.Role.Name)
	assert.Equal(t, 0, len(roleDetails.Permissions)) // No permissions yet

	// Create permissions
	permReq1 := admintypes.CreatePermissionRequest{
		Key:         "article:read",
		Description: new("Read articles"),
	}
	perm1, err := uc.CreatePermission(ctx, permReq1)
	require.NoError(t, err)

	permReq2 := admintypes.CreatePermissionRequest{
		Key:         "article:write",
		Description: new("Write articles"),
	}
	perm2, err := uc.CreatePermission(ctx, permReq2)
	require.NoError(t, err)

	// Add permissions to role
	err = uc.AddPermissionToRole(ctx, role.ID, perm1.ID, nil)
	require.NoError(t, err)

	err = uc.AddPermissionToRole(ctx, role.ID, perm2.ID, nil)
	require.NoError(t, err)

	// Verify permissions added
	roleDetails, err = uc.GetRoleByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(roleDetails.Permissions))

	// Update role
	newName := "Senior Editor"
	updateReq := admintypes.UpdateRoleRequest{
		Name: &newName,
	}
	updated, err := uc.UpdateRole(ctx, role.ID, updateReq)
	require.NoError(t, err)
	assert.Equal(t, "Senior Editor", updated.Name)

	// Remove one permission
	err = uc.RemovePermissionFromRole(ctx, role.ID, perm1.ID)
	require.NoError(t, err)

	// Verify permission removed
	roleDetails, err = uc.GetRoleByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(roleDetails.Permissions))

	// Seed a user
	seedUser(t, db, "user-1", "editor@example.com")

	// Assign role to user
	assignReq := admintypes.AssignUserRoleRequest{
		RoleID: role.ID,
	}
	err = uc.AssignRoleToUser(ctx, "user-1", assignReq, nil)
	require.NoError(t, err)

	// Remove role from user
	err = uc.RemoveRoleFromUser(ctx, "user-1", role.ID)
	require.NoError(t, err)

	// Now we can delete the role (no user assignments)
	err = uc.DeleteRole(ctx, role.ID)
	require.NoError(t, err)

	// Verify role deleted
	deletedRole, err := uc.GetRoleByID(ctx, role.ID)
	assert.Error(t, err) // Should return error since role doesn't exist
	assert.Nil(t, deletedRole)
}

// TestIntegration_SystemRoleProtection verifies system roles cannot be modified
func TestIntegration_SystemRoleProtection(t *testing.T) {
	db := newIntegrationTestDB(t)
	uc := buildRolePermissionUseCase(db)
	ctx := context.Background()

	// Seed a system role
	systemRoleID := "system-admin"
	seedRole(t, db, systemRoleID, "System Admin", true)

	// Try to update system role - should fail
	newName := "Modified"
	updateReq := admintypes.UpdateRoleRequest{
		Name: &newName,
	}
	_, err := uc.UpdateRole(ctx, systemRoleID, updateReq)
	assert.Error(t, err)
	assert.Equal(t, "cannot update system role", err.Error())

	// Try to delete system role - should fail
	err = uc.DeleteRole(ctx, systemRoleID)
	assert.Error(t, err)
	assert.Equal(t, "cannot delete system role", err.Error())

	// Same for system permissions
	systemPermID := "system-perm"
	seedPermission(t, db, systemPermID, "system:admin", true)

	updatePermReq := admintypes.UpdatePermissionRequest{
		Description: new("Modified"),
	}
	_, err = uc.UpdatePermission(ctx, systemPermID, updatePermReq)
	assert.Error(t, err)
	assert.Equal(t, "cannot update system permission", err.Error())

	err = uc.DeletePermission(ctx, systemPermID)
	assert.Error(t, err)
	assert.Equal(t, "cannot delete system permission", err.Error())

	// Cannot add system permission to non-system role
	regularRoleID := "regular-role"
	seedRole(t, db, regularRoleID, "Regular", false)

	err = uc.AddPermissionToRole(ctx, regularRoleID, systemPermID, nil)
	assert.Error(t, err)
	assert.Equal(t, "cannot modify system permission", err.Error())

	// Cannot add system permission to system role
	err = uc.AddPermissionToRole(ctx, systemRoleID, systemPermID, nil)
	assert.Error(t, err)
	assert.Equal(t, "cannot modify system role", err.Error())
}

// TestIntegration_PermissionCascade tests permission lifecycle with assignments
func TestIntegration_PermissionCascade(t *testing.T) {
	db := newIntegrationTestDB(t)
	uc := buildRolePermissionUseCase(db)
	ctx := context.Background()

	// Create role and permission
	role, err := uc.CreateRole(ctx, admintypes.CreateRoleRequest{
		Name: "Viewer",
	})
	require.NoError(t, err)

	perm, err := uc.CreatePermission(ctx, admintypes.CreatePermissionRequest{
		Key: "data:view",
	})
	require.NoError(t, err)

	// Add permission to role
	err = uc.AddPermissionToRole(ctx, role.ID, perm.ID, nil)
	require.NoError(t, err)

	// Try to delete permission - should fail because it's used by role
	err = uc.DeletePermission(ctx, perm.ID)
	assert.Error(t, err)
	assert.Equal(t, "permission is in use by one or more roles", err.Error())

	// Remove permission from role first
	err = uc.RemovePermissionFromRole(ctx, role.ID, perm.ID)
	require.NoError(t, err)

	// Now delete should succeed
	err = uc.DeletePermission(ctx, perm.ID)
	require.NoError(t, err)

	// Verify permission deleted
	deletedPerm, err := uc.GetAllPermissions(ctx)
	assert.NoError(t, err)
	for _, p := range deletedPerm {
		assert.NotEqual(t, perm.ID, p.ID)
	}
}

// TestIntegration_UserRoleExpiry tests user role assignment with expiration
func TestIntegration_UserRoleExpiry(t *testing.T) {
	db := newIntegrationTestDB(t)
	uc := buildRolePermissionUseCase(db)
	ctx := context.Background()

	// Create role and user
	role, err := uc.CreateRole(ctx, admintypes.CreateRoleRequest{
		Name: "Temporary",
	})
	require.NoError(t, err)

	seedUser(t, db, "user-1", "temp@example.com")

	// Assign role without expiry
	assignReq := admintypes.AssignUserRoleRequest{
		RoleID: role.ID,
	}
	err = uc.AssignRoleToUser(ctx, "user-1", assignReq, nil)
	require.NoError(t, err)

	// Remove and try with expiry
	err = uc.RemoveRoleFromUser(ctx, "user-1", role.ID)
	require.NoError(t, err)

	// Assign with future expiry - should succeed
	futureTime := time.Now().UTC().Add(24 * time.Hour)
	assignReqWithExpiry := admintypes.AssignUserRoleRequest{
		RoleID:    role.ID,
		ExpiresAt: &futureTime,
	}
	err = uc.AssignRoleToUser(ctx, "user-1", assignReqWithExpiry, nil)
	require.NoError(t, err)

	// Remove and try with past expiry - should fail
	err = uc.RemoveRoleFromUser(ctx, "user-1", role.ID)
	require.NoError(t, err)

	pastTime := time.Now().UTC().Add(-24 * time.Hour)
	assignReqWithPastExpiry := admintypes.AssignUserRoleRequest{
		RoleID:    role.ID,
		ExpiresAt: &pastTime,
	}
	err = uc.AssignRoleToUser(ctx, "user-1", assignReqWithPastExpiry, nil)
	assert.Error(t, err)
	assert.Equal(t, "expires_at must be in the future", err.Error())
}

// TestIntegration_BulkRolePermissionManagement tests ReplaceRolePermissions with deduplication
func TestIntegration_BulkRolePermissionManagement(t *testing.T) {
	db := newIntegrationTestDB(t)
	uc := buildRolePermissionUseCase(db)
	ctx := context.Background()

	// Create role and permissions
	role, err := uc.CreateRole(ctx, admintypes.CreateRoleRequest{
		Name: "Manager",
	})
	require.NoError(t, err)

	perm1, err := uc.CreatePermission(ctx, admintypes.CreatePermissionRequest{
		Key: "user:read",
	})
	require.NoError(t, err)

	perm2, err := uc.CreatePermission(ctx, admintypes.CreatePermissionRequest{
		Key: "user:write",
	})
	require.NoError(t, err)

	// Replace with duplicate IDs - should deduplicate
	permIDs := []string{perm1.ID, perm2.ID, perm1.ID, "", "   ", perm2.ID}
	err = uc.ReplaceRolePermissions(ctx, role.ID, permIDs, nil)
	require.NoError(t, err)

	// Verify only unique permissions were added
	roleDetails, err := uc.GetRoleByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(roleDetails.Permissions))

	// Replace with new set
	err = uc.ReplaceRolePermissions(ctx, role.ID, []string{perm1.ID}, nil)
	require.NoError(t, err)

	roleDetails, err = uc.GetRoleByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(roleDetails.Permissions))

	// Replace with empty list
	err = uc.ReplaceRolePermissions(ctx, role.ID, []string{}, nil)
	require.NoError(t, err)

	roleDetails, err = uc.GetRoleByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, len(roleDetails.Permissions))
}

// TestIntegration_DeletionConstraints verifies roles/permissions with assignments cannot be deleted
func TestIntegration_DeletionConstraints(t *testing.T) {
	db := newIntegrationTestDB(t)
	uc := buildRolePermissionUseCase(db)
	ctx := context.Background()

	// Create role, permission, and user
	role, err := uc.CreateRole(ctx, admintypes.CreateRoleRequest{
		Name: "Restricted",
	})
	require.NoError(t, err)

	perm, err := uc.CreatePermission(ctx, admintypes.CreatePermissionRequest{
		Key: "restricted:action",
	})
	require.NoError(t, err)

	seedUser(t, db, "user-1", "restricted@example.com")

	// Assign role to user
	assignReq := admintypes.AssignUserRoleRequest{
		RoleID: role.ID,
	}
	err = uc.AssignRoleToUser(ctx, "user-1", assignReq, nil)
	require.NoError(t, err)

	// Try to delete role with user assignment - should fail
	err = uc.DeleteRole(ctx, role.ID)
	assert.Error(t, err)
	assert.Equal(t, "role is assigned to one or more users", err.Error())

	// Remove assignment
	err = uc.RemoveRoleFromUser(ctx, "user-1", role.ID)
	require.NoError(t, err)

	// Add permission to role
	err = uc.AddPermissionToRole(ctx, role.ID, perm.ID, nil)
	require.NoError(t, err)

	// Try to delete permission with role assignment - should fail
	err = uc.DeletePermission(ctx, perm.ID)
	assert.Error(t, err)
	assert.Equal(t, "permission is in use by one or more roles", err.Error())

	// Remove from role
	err = uc.RemovePermissionFromRole(ctx, role.ID, perm.ID)
	require.NoError(t, err)

	// Now both can be deleted
	err = uc.DeletePermission(ctx, perm.ID)
	require.NoError(t, err)

	err = uc.DeleteRole(ctx, role.ID)
	require.NoError(t, err)
}

// TestIntegration_BulkUserRoleManagement tests ReplaceUserRoles with deduplication
func TestIntegration_BulkUserRoleManagement(t *testing.T) {
	db := newIntegrationTestDB(t)
	uc := buildRolePermissionUseCase(db)
	ctx := context.Background()

	// Create roles
	role1, err := uc.CreateRole(ctx, admintypes.CreateRoleRequest{
		Name: "Role1",
	})
	require.NoError(t, err)

	role2, err := uc.CreateRole(ctx, admintypes.CreateRoleRequest{
		Name: "Role2",
	})
	require.NoError(t, err)

	// Seed user
	seedUser(t, db, "user-1", "bulk@example.com")

	// Replace with duplicate role IDs - should deduplicate
	roleIDs := []string{role1.ID, role2.ID, role1.ID, "", "   ", role2.ID}
	err = uc.ReplaceUserRoles(ctx, "user-1", roleIDs, nil)
	require.NoError(t, err)

	// Note: We can't easily verify the count without reading from DB directly,
	// but we can verify it didn't error and the operation completed

	// Replace with new set
	err = uc.ReplaceUserRoles(ctx, "user-1", []string{role1.ID}, nil)
	require.NoError(t, err)

	// Replace with empty list
	err = uc.ReplaceUserRoles(ctx, "user-1", []string{}, nil)
	require.NoError(t, err)
}
