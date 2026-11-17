package usecases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
)

func newAggregateUseCases(mocks *useCaseMocks) *UseCases {
	authorizer := &internaltests.NoopAuthorizer{}

	return NewAccessControlUseCases(
		NewRolesUseCase(services.NewRolesService(mocks.roles, mocks.rolePermissions, mocks.userRoles), authorizer),
		NewPermissionsUseCase(services.NewPermissionsService(mocks.permissions, mocks.rolePermissions), authorizer),
		NewRolePermissionsUseCase(services.NewRolePermissionsService(mocks.roles, mocks.permissions, mocks.rolePermissions), authorizer),
		NewUserRolesUseCase(services.NewUserRolesService(mocks.userRoles, mocks.roles), authorizer),
		NewUserPermissionsUseCase(services.NewUserPermissionsService(mocks.userPermissions), authorizer),
	)
}

func TestNewAccessControlUseCasesAccessors(t *testing.T) {
	t.Parallel()

	authorizer := &internaltests.NoopAuthorizer{}
	roles := NewRolesUseCase(services.NewRolesService(nil, nil, nil), authorizer)
	permissions := NewPermissionsUseCase(services.NewPermissionsService(nil, nil), authorizer)
	rolePermissions := NewRolePermissionsUseCase(services.NewRolePermissionsService(nil, nil, nil), authorizer)
	userRoles := NewUserRolesUseCase(services.NewUserRolesService(nil, nil), authorizer)
	userPermissions := NewUserPermissionsUseCase(services.NewUserPermissionsService(nil), authorizer)

	useCases := NewAccessControlUseCases(roles, permissions, rolePermissions, userRoles, userPermissions)

	if useCases.RolesUseCase() != roles {
		t.Fatal("expected roles use case to be returned")
	}
	if useCases.PermissionsUseCase() != permissions {
		t.Fatal("expected permissions use case to be returned")
	}
	if useCases.RolePermissionsUseCase() != rolePermissions {
		t.Fatal("expected role permissions use case to be returned")
	}
	if useCases.UserRolesUseCase() != userRoles {
		t.Fatal("expected user roles use case to be returned")
	}
	if useCases.UserPermissionsUseCase() != userPermissions {
		t.Fatal("expected user permissions use case to be returned")
	}
}

func TestUseCasesDelegation(t *testing.T) {
	t.Parallel()

	actor := testActor()
	grantedBy := "actor-1"

	tests := []struct {
		name  string
		setup func(*useCaseMocks)
		run   func(context.Context, *UseCases) (any, error)
	}{
		{
			name: "create role",
			setup: func(m *useCaseMocks) {
				m.roles.On("CreateRole", mock.Anything, mock.AnythingOfType("*types.Role")).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.CreateRole(ctx, actor, types.CreateRoleRequest{Name: "editor"})
			},
		},
		{
			name: "get all roles",
			setup: func(m *useCaseMocks) {
				m.roles.On("GetAllRoles", mock.Anything).Return([]types.Role{}, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.GetAllRoles(ctx, actor)
			},
		},
		{
			name: "get role by name",
			setup: func(m *useCaseMocks) {
				m.roles.On("GetRoleByName", mock.Anything, "editor").Return(&types.Role{ID: "role-1"}, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.GetRoleByName(ctx, actor, "editor")
			},
		},
		{
			name: "get role by id",
			setup: func(m *useCaseMocks) {
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				m.rolePermissions.On("GetRolePermissions", mock.Anything, "role-1").Return([]types.UserPermissionInfo{}, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.GetRoleByID(ctx, actor, "role-1")
			},
		},
		{
			name: "update role",
			setup: func(m *useCaseMocks) {
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				m.roles.On("UpdateRole", mock.Anything, "role-1", new("editor"), (*string)(nil), (*int)(nil)).Return(true, nil).Once()
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "editor"}, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.UpdateRole(ctx, actor, "role-1", types.UpdateRoleRequest{Name: new("editor")})
			},
		},
		{
			name: "delete role",
			setup: func(m *useCaseMocks) {
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				m.userRoles.On("CountUsersByRole", mock.Anything, "role-1").Return(0, nil).Once()
				m.roles.On("DeleteRole", mock.Anything, "role-1").Return(true, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return nil, uc.DeleteRole(ctx, actor, "role-1")
			},
		},
		{
			name: "create permission",
			setup: func(m *useCaseMocks) {
				m.permissions.On("CreatePermission", mock.Anything, mock.AnythingOfType("*types.Permission")).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.CreatePermission(ctx, actor, types.CreatePermissionRequest{Key: "users.read"})
			},
		},
		{
			name: "get all permissions",
			setup: func(m *useCaseMocks) {
				m.permissions.On("GetAllPermissions", mock.Anything).Return([]types.Permission{}, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.GetAllPermissions(ctx, actor)
			},
		},
		{
			name: "get permission by id",
			setup: func(m *useCaseMocks) {
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.GetPermissionByID(ctx, actor, "perm-1")
			},
		},
		{
			name: "get permission by key",
			setup: func(m *useCaseMocks) {
				m.permissions.On("GetPermissionByKey", mock.Anything, "users.read").Return(&types.Permission{ID: "perm-1"}, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.GetPermissionByKey(ctx, actor, "users.read")
			},
		},
		{
			name: "update permission",
			setup: func(m *useCaseMocks) {
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				m.permissions.On("UpdatePermission", mock.Anything, "perm-1", new("updated")).Return(true, nil).Once()
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.UpdatePermission(ctx, actor, "perm-1", types.UpdatePermissionRequest{Description: new("updated")})
			},
		},
		{
			name: "delete permission",
			setup: func(m *useCaseMocks) {
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				m.rolePermissions.On("CountRolesByPermission", mock.Anything, "perm-1").Return(0, nil).Once()
				m.permissions.On("DeletePermission", mock.Anything, "perm-1").Return(true, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return nil, uc.DeletePermission(ctx, actor, "perm-1")
			},
		},
		{
			name: "get role permissions",
			setup: func(m *useCaseMocks) {
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				m.rolePermissions.On("GetRolePermissions", mock.Anything, "role-1").Return([]types.UserPermissionInfo{}, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.GetRolePermissions(ctx, actor, "role-1")
			},
		},
		{
			name: "add permission to role",
			setup: func(m *useCaseMocks) {
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				m.rolePermissions.On("AddRolePermission", mock.Anything, "role-1", "perm-1", &grantedBy).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return nil, uc.AddPermissionToRole(ctx, actor, "role-1", "perm-1", &grantedBy)
			},
		},
		{
			name: "remove permission from role",
			setup: func(m *useCaseMocks) {
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				m.rolePermissions.On("RemoveRolePermission", mock.Anything, "role-1", "perm-1").Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return nil, uc.RemovePermissionFromRole(ctx, actor, "role-1", "perm-1")
			},
		},
		{
			name: "replace role permissions",
			setup: func(m *useCaseMocks) {
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				m.rolePermissions.On("ReplaceRolePermissions", mock.Anything, "role-1", []string{"perm-1"}, &grantedBy).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return nil, uc.ReplaceRolePermissions(ctx, actor, "role-1", []string{"perm-1"}, &grantedBy)
			},
		},
		{
			name: "get user roles",
			setup: func(m *useCaseMocks) {
				m.userRoles.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{}, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.GetUserRoles(ctx, actor, "user-1")
			},
		},
		{
			name: "replace user roles",
			setup: func(m *useCaseMocks) {
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Weight: 10}, nil).Once()
				m.userRoles.On("GetUserRoles", mock.Anything, "actor-1").Return([]types.UserRoleInfo{{RoleID: "role-9", RoleWeight: 20}}, nil).Once()
				m.userRoles.On("ReplaceUserRoles", mock.Anything, "user-1", []string{"role-1"}, &grantedBy).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return nil, uc.ReplaceUserRoles(ctx, actor, "user-1", []string{"role-1"}, &grantedBy)
			},
		},
		{
			name: "assign role to user",
			setup: func(m *useCaseMocks) {
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Weight: 10}, nil).Once()
				m.userRoles.On("GetUserRoles", mock.Anything, "actor-1").Return([]types.UserRoleInfo{{RoleID: "role-9", RoleWeight: 20}}, nil).Once()
				m.userRoles.On("AssignUserRole", mock.Anything, "user-1", "role-1", &grantedBy, mock.Anything).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return nil, uc.AssignRoleToUser(ctx, actor, "user-1", types.AssignUserRoleRequest{RoleID: "role-1"}, &grantedBy)
			},
		},
		{
			name: "remove role from user",
			setup: func(m *useCaseMocks) {
				m.userRoles.On("RemoveUserRole", mock.Anything, "user-1", "role-1").Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return nil, uc.RemoveRoleFromUser(ctx, actor, "user-1", "role-1")
			},
		},
		{
			name: "get self user permissions",
			setup: func(m *useCaseMocks) {
				m.userPermissions.On("GetUserPermissions", mock.Anything, actor.ID).Return([]types.UserPermissionInfo{}, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.GetSelfUserPermissions(ctx, actor, actor.ID)
			},
		},
		{
			name: "get user permissions",
			setup: func(m *useCaseMocks) {
				m.userPermissions.On("GetUserPermissions", mock.Anything, "user-1").Return([]types.UserPermissionInfo{}, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.GetUserPermissions(ctx, actor, "user-1")
			},
		},
		{
			name: "has permissions",
			setup: func(m *useCaseMocks) {
				m.userPermissions.On("HasPermissions", mock.Anything, "user-1", []string{"users.read"}).Return(true, nil).Once()
			},
			run: func(ctx context.Context, uc *UseCases) (any, error) {
				return uc.HasPermissions(ctx, actor, "user-1", []string{"users.read"})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mocks := newUseCaseMocks()
			tc.setup(mocks)

			result, err := tc.run(context.Background(), newAggregateUseCases(mocks))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			_ = result

			mocks.assertExpectations(t)
		})
	}
}
