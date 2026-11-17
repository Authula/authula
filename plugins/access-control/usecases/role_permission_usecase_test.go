package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestRolePermissionsUseCase(t *testing.T) {
	t.Parallel()

	actor := testActor()
	repoErr := errors.New("repository failed")
	grantedBy := "actor-1"

	tests := []struct {
		name    string
		setup   func(*useCaseMocks)
		run     func(context.Context, *RolePermissionsUseCase) (any, error)
		wantErr error
		check   func(*testing.T, any)
	}{
		{
			name: "get role permissions denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.RolePermissionsReadPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *RolePermissionsUseCase) (any, error) {
				return uc.GetRolePermissions(ctx, actor, "role-1")
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "get role permissions success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.RolePermissionsReadPermission).Return(nil).Once()
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				m.rolePermissions.On("GetRolePermissions", mock.Anything, "role-1").Return([]types.UserPermissionInfo{{PermissionID: "perm-1"}}, nil).Once()
			},
			run: func(ctx context.Context, uc *RolePermissionsUseCase) (any, error) {
				return uc.GetRolePermissions(ctx, actor, "role-1")
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				assertRolePermissions(t, result, 1)
			},
		},
		{
			name: "get role permissions service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.RolePermissionsReadPermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *RolePermissionsUseCase) (any, error) {
				return uc.GetRolePermissions(ctx, actor, "")
			},
			wantErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "add permission denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.RolePermissionsAssignPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *RolePermissionsUseCase) (any, error) {
				return nil, uc.AddPermissionToRole(ctx, actor, "role-1", "perm-1", &grantedBy)
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "add permission success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.RolePermissionsAssignPermission).Return(nil).Once()
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				m.rolePermissions.On("AddRolePermission", mock.Anything, "role-1", "perm-1", &grantedBy).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *RolePermissionsUseCase) (any, error) {
				return nil, uc.AddPermissionToRole(ctx, actor, "role-1", "perm-1", &grantedBy)
			},
		},
		{
			name: "add permission service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.RolePermissionsAssignPermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *RolePermissionsUseCase) (any, error) {
				return nil, uc.AddPermissionToRole(ctx, actor, "", "perm-1", &grantedBy)
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "remove permission denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.RolePermissionsRemovePermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *RolePermissionsUseCase) (any, error) {
				return nil, uc.RemovePermissionFromRole(ctx, actor, "role-1", "perm-1")
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "remove permission success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.RolePermissionsRemovePermission).Return(nil).Once()
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				m.rolePermissions.On("RemoveRolePermission", mock.Anything, "role-1", "perm-1").Return(nil).Once()
			},
			run: func(ctx context.Context, uc *RolePermissionsUseCase) (any, error) {
				return nil, uc.RemovePermissionFromRole(ctx, actor, "role-1", "perm-1")
			},
		},
		{
			name: "remove permission service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.RolePermissionsRemovePermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *RolePermissionsUseCase) (any, error) {
				return nil, uc.RemovePermissionFromRole(ctx, actor, "role-1", "")
			},
			wantErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "replace permissions denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.RolePermissionsAssignPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *RolePermissionsUseCase) (any, error) {
				return nil, uc.ReplaceRolePermissions(ctx, actor, "role-1", []string{"perm-1"}, &grantedBy)
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "replace permissions success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.RolePermissionsAssignPermission).Return(nil).Once()
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				m.rolePermissions.On("ReplaceRolePermissions", mock.Anything, "role-1", []string{"perm-1"}, &grantedBy).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *RolePermissionsUseCase) (any, error) {
				return nil, uc.ReplaceRolePermissions(ctx, actor, "role-1", []string{"perm-1"}, &grantedBy)
			},
		},
		{
			name: "replace permissions service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.RolePermissionsAssignPermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *RolePermissionsUseCase) (any, error) {
				return nil, uc.ReplaceRolePermissions(ctx, actor, "", []string{"perm-1"}, &grantedBy)
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "service repository error propagates",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.RolePermissionsReadPermission).Return(nil).Once()
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), repoErr).Once()
			},
			run: func(ctx context.Context, uc *RolePermissionsUseCase) (any, error) {
				return uc.GetRolePermissions(ctx, actor, "role-1")
			},
			wantErr: repoErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mocks := newUseCaseMocks()
			tc.setup(mocks)

			result, err := tc.run(context.Background(), mocks.rolePermissionsUseCase())
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil && !isNilResult(result) {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if tc.check != nil {
				tc.check(t, result)
			}

			mocks.assertExpectations(t)
		})
	}
}
