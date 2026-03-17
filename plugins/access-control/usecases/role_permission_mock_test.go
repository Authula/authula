package usecases

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

type MockRolePermissionRepository struct {
	mock.Mock
}

type MockRolePermissionService = MockRolePermissionRepository

func (m *MockRolePermissionRepository) CreateRole(ctx context.Context, role *types.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRolePermissionRepository) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	args := m.Called(ctx)
	if roles := args.Get(0); roles != nil {
		return roles.([]types.Role), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRolePermissionRepository) GetRoleByID(ctx context.Context, roleID string) (*types.Role, error) {
	args := m.Called(ctx, roleID)
	if role := args.Get(0); role != nil {
		return role.(*types.Role), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRolePermissionRepository) UpdateRole(ctx context.Context, roleID string, name *string, description *string) (bool, error) {
	args := m.Called(ctx, roleID, name, description)
	return args.Bool(0), args.Error(1)
}

func (m *MockRolePermissionRepository) DeleteRole(ctx context.Context, roleID string) (bool, error) {
	args := m.Called(ctx, roleID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRolePermissionRepository) CountUserAssignmentsByRoleID(ctx context.Context, roleID string) (int, error) {
	args := m.Called(ctx, roleID)
	return args.Int(0), args.Error(1)
}

func (m *MockRolePermissionRepository) GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error) {
	args := m.Called(ctx, roleID)
	if perms := args.Get(0); perms != nil {
		return perms.([]types.UserPermissionInfo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRolePermissionRepository) CreatePermission(ctx context.Context, permission *types.Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *MockRolePermissionRepository) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	args := m.Called(ctx)
	if perms := args.Get(0); perms != nil {
		return perms.([]types.Permission), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRolePermissionRepository) GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error) {
	args := m.Called(ctx, permissionID)
	if perm := args.Get(0); perm != nil {
		return perm.(*types.Permission), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRolePermissionRepository) UpdatePermission(ctx context.Context, permissionID string, description *string) (bool, error) {
	args := m.Called(ctx, permissionID, description)
	return args.Bool(0), args.Error(1)
}

func (m *MockRolePermissionRepository) DeletePermission(ctx context.Context, permissionID string) (bool, error) {
	args := m.Called(ctx, permissionID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRolePermissionRepository) CountRoleAssignmentsByPermissionID(ctx context.Context, permissionID string) (int, error) {
	args := m.Called(ctx, permissionID)
	return args.Int(0), args.Error(1)
}

func (m *MockRolePermissionRepository) AddRolePermission(ctx context.Context, roleID, permissionID string, grantedByUserID *string) error {
	args := m.Called(ctx, roleID, permissionID, grantedByUserID)
	return args.Error(0)
}

func (m *MockRolePermissionRepository) RemoveRolePermission(ctx context.Context, roleID, permissionID string) error {
	args := m.Called(ctx, roleID, permissionID)
	return args.Error(0)
}

func (m *MockRolePermissionRepository) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	args := m.Called(ctx, roleID, permissionIDs, grantedByUserID)
	return args.Error(0)
}

func (m *MockRolePermissionRepository) AssignUserRole(ctx context.Context, userID, roleID string, assignedByUserID *string, expiresAt *time.Time) error {
	args := m.Called(ctx, userID, roleID, assignedByUserID, expiresAt)
	return args.Error(0)
}

func (m *MockRolePermissionRepository) RemoveUserRole(ctx context.Context, userID, roleID string) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockRolePermissionRepository) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	args := m.Called(ctx, userID, roleIDs, assignedByUserID)
	return args.Error(0)
}
