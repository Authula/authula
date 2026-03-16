package tests

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/usecases"
)

type mockUserAccessRepository struct {
	mock.Mock
}

func (m *mockUserAccessRepository) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.UserRoleInfo), args.Error(1)
}

func (m *mockUserAccessRepository) GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.UserPermissionInfo), args.Error(1)
}

func (m *mockUserAccessRepository) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.UserWithRoles), args.Error(1)
}

func (m *mockUserAccessRepository) GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.UserWithPermissions), args.Error(1)
}

func NewUserRolesUseCaseFixture() (usecases.UserRolesUseCase, *mockUserAccessRepository) {
	repo := &mockUserAccessRepository{}
	service := services.NewUserAccessService(repo)
	return usecases.NewUserRolesUseCase(service), repo
}

type mockRolePermissionRepository struct {
	mock.Mock
}

func (m *mockRolePermissionRepository) CreateRole(ctx context.Context, role *types.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *mockRolePermissionRepository) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Role), args.Error(1)
}

func (m *mockRolePermissionRepository) GetRoleByID(ctx context.Context, roleID string) (*types.Role, error) {
	args := m.Called(ctx, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Role), args.Error(1)
}

func (m *mockRolePermissionRepository) UpdateRole(ctx context.Context, roleID string, name *string, description *string) (bool, error) {
	args := m.Called(ctx, roleID, name, description)
	return args.Bool(0), args.Error(1)
}

func (m *mockRolePermissionRepository) DeleteRole(ctx context.Context, roleID string) (bool, error) {
	args := m.Called(ctx, roleID)
	return args.Bool(0), args.Error(1)
}

func (m *mockRolePermissionRepository) CountUserAssignmentsByRoleID(ctx context.Context, roleID string) (int, error) {
	args := m.Called(ctx, roleID)
	return args.Int(0), args.Error(1)
}

func (m *mockRolePermissionRepository) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Permission), args.Error(1)
}

func (m *mockRolePermissionRepository) GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error) {
	args := m.Called(ctx, permissionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Permission), args.Error(1)
}

func (m *mockRolePermissionRepository) CreatePermission(ctx context.Context, permission *types.Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *mockRolePermissionRepository) UpdatePermission(ctx context.Context, permissionID string, description *string) (bool, error) {
	args := m.Called(ctx, permissionID, description)
	return args.Bool(0), args.Error(1)
}

func (m *mockRolePermissionRepository) DeletePermission(ctx context.Context, permissionID string) (bool, error) {
	args := m.Called(ctx, permissionID)
	return args.Bool(0), args.Error(1)
}

func (m *mockRolePermissionRepository) CountRoleAssignmentsByPermissionID(ctx context.Context, permissionID string) (int, error) {
	args := m.Called(ctx, permissionID)
	return args.Int(0), args.Error(1)
}

func (m *mockRolePermissionRepository) GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error) {
	args := m.Called(ctx, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.UserPermissionInfo), args.Error(1)
}

func (m *mockRolePermissionRepository) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	args := m.Called(ctx, roleID, permissionIDs, grantedByUserID)
	return args.Error(0)
}

func (m *mockRolePermissionRepository) AddRolePermission(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	args := m.Called(ctx, roleID, permissionID, grantedByUserID)
	return args.Error(0)
}

func (m *mockRolePermissionRepository) RemoveRolePermission(ctx context.Context, roleID string, permissionID string) error {
	args := m.Called(ctx, roleID, permissionID)
	return args.Error(0)
}

func (m *mockRolePermissionRepository) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	args := m.Called(ctx, userID, roleIDs, assignedByUserID)
	return args.Error(0)
}

func (m *mockRolePermissionRepository) AssignUserRole(ctx context.Context, userID string, roleID string, assignedByUserID *string, expiresAt *time.Time) error {
	args := m.Called(ctx, userID, roleID, assignedByUserID, expiresAt)
	return args.Error(0)
}

func (m *mockRolePermissionRepository) RemoveUserRole(ctx context.Context, userID string, roleID string) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func NewRolePermissionUseCaseFixture() (usecases.RolePermissionUseCase, *mockRolePermissionRepository) {
	repo := &mockRolePermissionRepository{}
	service := services.NewRolePermissionService(repo)
	return usecases.NewRolePermissionUseCase(service), repo
}
