package tests

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/mock"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	accesscontrolmigrations "github.com/Authula/authula/plugins/access-control/migrationset"
	"github.com/Authula/authula/plugins/access-control/types"
)

type MockRolesRepository struct {
	mock.Mock
}

func (m *MockRolesRepository) CreateRole(ctx context.Context, role *types.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRolesRepository) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Role), args.Error(1)
}

func (m *MockRolesRepository) GetRoleByID(ctx context.Context, roleID string) (*types.Role, error) {
	args := m.Called(ctx, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Role), args.Error(1)
}

func (m *MockRolesRepository) GetRoleByName(ctx context.Context, roleName string) (*types.Role, error) {
	args := m.Called(ctx, roleName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Role), args.Error(1)
}

func (m *MockRolesRepository) UpdateRole(ctx context.Context, roleID string, name *string, description *string, weight *int) (bool, error) {
	args := m.Called(ctx, roleID, name, description, weight)
	return args.Bool(0), args.Error(1)
}

func (m *MockRolesRepository) DeleteRole(ctx context.Context, roleID string) (bool, error) {
	args := m.Called(ctx, roleID)
	return args.Bool(0), args.Error(1)
}

type MockPermissionsRepository struct {
	mock.Mock
}

func (m *MockPermissionsRepository) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Permission), args.Error(1)
}

func (m *MockPermissionsRepository) GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error) {
	args := m.Called(ctx, permissionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Permission), args.Error(1)
}

func (m *MockPermissionsRepository) GetPermissionByKey(ctx context.Context, permissionKey string) (*types.Permission, error) {
	args := m.Called(ctx, permissionKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Permission), args.Error(1)
}

func (m *MockPermissionsRepository) CreatePermission(ctx context.Context, permission *types.Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *MockPermissionsRepository) UpdatePermission(ctx context.Context, permissionID string, description *string) (bool, error) {
	args := m.Called(ctx, permissionID, description)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionsRepository) DeletePermission(ctx context.Context, permissionID string) (bool, error) {
	args := m.Called(ctx, permissionID)
	return args.Bool(0), args.Error(1)
}

type MockRolePermissionsRepository struct {
	mock.Mock
}

func (m *MockRolePermissionsRepository) GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error) {
	args := m.Called(ctx, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.UserPermissionInfo), args.Error(1)
}

func (m *MockRolePermissionsRepository) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	args := m.Called(ctx, roleID, permissionIDs, grantedByUserID)
	return args.Error(0)
}

func (m *MockRolePermissionsRepository) AddRolePermission(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	args := m.Called(ctx, roleID, permissionID, grantedByUserID)
	return args.Error(0)
}

func (m *MockRolePermissionsRepository) RemoveRolePermission(ctx context.Context, roleID string, permissionID string) error {
	args := m.Called(ctx, roleID, permissionID)
	return args.Error(0)
}

func (m *MockRolePermissionsRepository) CountRolesByPermission(ctx context.Context, permissionID string) (int, error) {
	args := m.Called(ctx, permissionID)
	return args.Int(0), args.Error(1)
}

type MockUserRolesRepository struct {
	mock.Mock
}

func (m *MockUserRolesRepository) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.UserRoleInfo), args.Error(1)
}

func (m *MockUserRolesRepository) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	args := m.Called(ctx, userID, roleIDs, assignedByUserID)
	return args.Error(0)
}

func (m *MockUserRolesRepository) AssignUserRole(ctx context.Context, userID string, roleID string, assignedByUserID *string, expiresAt *time.Time) error {
	args := m.Called(ctx, userID, roleID, assignedByUserID, expiresAt)
	return args.Error(0)
}

func (m *MockUserRolesRepository) RemoveUserRole(ctx context.Context, userID string, roleID string) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockUserRolesRepository) CountUsersByRole(ctx context.Context, roleID string) (int, error) {
	args := m.Called(ctx, roleID)
	return args.Int(0), args.Error(1)
}

type MockUserPermissionsRepository struct {
	mock.Mock
}

func (m *MockUserPermissionsRepository) GetUserPermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.UserPermissionInfo), args.Error(1)
}

func (m *MockUserPermissionsRepository) HasPermissions(ctx context.Context, userID string, permissionKeys []string) (bool, error) {
	args := m.Called(ctx, userID, permissionKeys)
	return args.Bool(0), args.Error(1)
}

func SetupRepoDB(t *testing.T) *bun.DB {
	t.Helper()

	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() {
		_ = db.Close()
	})

	ctx := context.Background()

	migrator, err := migrations.NewMigrator(db, &internaltests.MockLogger{})
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	coreSet, err := migrations.CoreMigrationSet("sqlite")
	if err != nil {
		t.Fatalf("failed to build core migration set: %v", err)
	}

	accessControlSet := migrations.MigrationSet{
		PluginID:   models.PluginAccessControl.String(),
		DependsOn:  []string{migrations.CorePluginID},
		Migrations: accesscontrolmigrations.ForProvider("sqlite"),
	}

	if err := migrator.Migrate(ctx, []migrations.MigrationSet{coreSet, accessControlSet}); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, name, email, email_verified, metadata) VALUES ('u1', 'User One', 'u1@example.com', 1, '{}')`); err != nil {
		t.Fatalf("failed to seed user u1: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, name, email, email_verified, metadata) VALUES ('u2', 'User Two', 'u2@example.com', 1, '{}')`); err != nil {
		t.Fatalf("failed to seed user u2: %v", err)
	}

	return db
}
