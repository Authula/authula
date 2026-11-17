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

func TestPermissionsUseCase(t *testing.T) {
	t.Parallel()

	actor := testActor()
	repoErr := errors.New("repository failed")

	tests := []struct {
		name    string
		setup   func(*useCaseMocks)
		run     func(context.Context, *PermissionsUseCase) (any, error)
		wantErr error
		check   func(*testing.T, any)
	}{
		{
			name: "create permission denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsCreatePermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.CreatePermission(ctx, actor, types.CreatePermissionRequest{Key: "users.read"})
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "create permission success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsCreatePermission).Return(nil).Once()
				m.permissions.On("CreatePermission", mock.Anything, mock.MatchedBy(func(permission *types.Permission) bool {
					return permission != nil && permission.ID != "" && permission.Key == "users.read"
				})).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.CreatePermission(ctx, actor, types.CreatePermissionRequest{Key: "users.read"})
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				permission, ok := result.(*types.Permission)
				if !ok || permission == nil || permission.ID == "" || permission.Key != "users.read" {
					t.Fatalf("unexpected permission %#v", result)
				}
			},
		},
		{
			name: "create permission service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsCreatePermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.CreatePermission(ctx, actor, types.CreatePermissionRequest{Key: ""})
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "get all permissions denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsListPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.GetAllPermissions(ctx, actor)
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "get all permissions success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsListPermission).Return(nil).Once()
				m.permissions.On("GetAllPermissions", mock.Anything).Return([]types.Permission{{ID: "perm-1"}}, nil).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.GetAllPermissions(ctx, actor)
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				assertPermissions(t, result, 1)
			},
		},
		{
			name: "get all permissions service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsListPermission).Return(nil).Once()
				m.permissions.On("GetAllPermissions", mock.Anything).Return(([]types.Permission)(nil), repoErr).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.GetAllPermissions(ctx, actor)
			},
			wantErr: repoErr,
		},
		{
			name: "get permission by id denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsReadPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.GetPermissionByID(ctx, actor, "perm-1")
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "get permission by id success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsReadPermission).Return(nil).Once()
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.GetPermissionByID(ctx, actor, "perm-1")
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				permission, ok := result.(*types.Permission)
				if !ok || permission == nil || permission.ID != "perm-1" {
					t.Fatalf("unexpected permission %#v", result)
				}
			},
		},
		{
			name: "get permission by id service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsReadPermission).Return(nil).Once()
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), repoErr).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.GetPermissionByID(ctx, actor, "perm-1")
			},
			wantErr: repoErr,
		},
		{
			name: "get permission by key denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsReadPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.GetPermissionByKey(ctx, actor, "users.read")
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "get permission by key success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsReadPermission).Return(nil).Once()
				m.permissions.On("GetPermissionByKey", mock.Anything, "users.read").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.GetPermissionByKey(ctx, actor, "users.read")
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				permission, ok := result.(*types.Permission)
				if !ok || permission == nil || permission.Key != "users.read" {
					t.Fatalf("unexpected permission %#v", result)
				}
			},
		},
		{
			name: "get permission by key service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsReadPermission).Return(nil).Once()
				m.permissions.On("GetPermissionByKey", mock.Anything, "users.read").Return((*types.Permission)(nil), nil).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.GetPermissionByKey(ctx, actor, "users.read")
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name: "update permission denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsUpdatePermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.UpdatePermission(ctx, actor, "perm-1", types.UpdatePermissionRequest{Description: new("updated")})
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "update permission success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsUpdatePermission).Return(nil).Once()
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				m.permissions.On("UpdatePermission", mock.Anything, "perm-1", new("updated")).Return(true, nil).Once()
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Description: new("updated")}, nil).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.UpdatePermission(ctx, actor, "perm-1", types.UpdatePermissionRequest{Description: new("updated")})
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				permission, ok := result.(*types.Permission)
				if !ok || permission == nil || permission.Description == nil || *permission.Description != "updated" {
					t.Fatalf("unexpected permission %#v", result)
				}
			},
		},
		{
			name: "update permission service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsUpdatePermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return uc.UpdatePermission(ctx, actor, "", types.UpdatePermissionRequest{Description: new("updated")})
			},
			wantErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "delete permission denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsDeletePermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return nil, uc.DeletePermission(ctx, actor, "perm-1")
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "delete permission success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsDeletePermission).Return(nil).Once()
				m.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				m.rolePermissions.On("CountRolesByPermission", mock.Anything, "perm-1").Return(0, nil).Once()
				m.permissions.On("DeletePermission", mock.Anything, "perm-1").Return(true, nil).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return nil, uc.DeletePermission(ctx, actor, "perm-1")
			},
		},
		{
			name: "delete permission service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.PermissionsDeletePermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *PermissionsUseCase) (any, error) {
				return nil, uc.DeletePermission(ctx, actor, "")
			},
			wantErr: coreerrors.ErrBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mocks := newUseCaseMocks()
			tc.setup(mocks)

			result, err := tc.run(context.Background(), mocks.permissionsUseCase())
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
