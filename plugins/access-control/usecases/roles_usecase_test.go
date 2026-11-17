package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestRolesUseCase(t *testing.T) {
	t.Parallel()

	actor := testActor()
	repoErr := errors.New("repository failed")

	tests := []struct {
		name    string
		setup   func(*useCaseMocks)
		run     func(context.Context, *RolesUseCase) (any, error)
		wantErr error
		check   func(*testing.T, any)
	}{
		{
			name: "create role denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesCreatePermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.CreateRole(ctx, actor, types.CreateRoleRequest{Name: "editor"})
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "create role success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesCreatePermission).Return(nil).Once()
				m.roles.On("CreateRole", mock.Anything, mock.MatchedBy(func(role *types.Role) bool {
					return role != nil && role.ID != "" && role.Name == "editor"
				})).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.CreateRole(ctx, actor, types.CreateRoleRequest{Name: "editor"})
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				role, ok := result.(*types.Role)
				if !ok || role == nil || role.ID == "" || role.Name != "editor" {
					t.Fatalf("unexpected role %#v", result)
				}
			},
		},
		{
			name: "create role service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesCreatePermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.CreateRole(ctx, actor, types.CreateRoleRequest{Name: ""})
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "get all roles denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesListPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.GetAllRoles(ctx, actor)
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "get all roles success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesListPermission).Return(nil).Once()
				m.roles.On("GetAllRoles", mock.Anything).Return([]types.Role{{ID: "role-1"}}, nil).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.GetAllRoles(ctx, actor)
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				roles, ok := result.([]types.Role)
				if !ok || len(roles) != 1 || roles[0].ID != "role-1" {
					t.Fatalf("unexpected roles %#v", result)
				}
			},
		},
		{
			name: "get all roles service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesListPermission).Return(nil).Once()
				m.roles.On("GetAllRoles", mock.Anything).Return(([]types.Role)(nil), repoErr).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.GetAllRoles(ctx, actor)
			},
			wantErr: repoErr,
		},
		{
			name: "get role by name denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesReadPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.GetRoleByName(ctx, actor, "admin")
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "get role by name success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesReadPermission).Return(nil).Once()
				m.roles.On("GetRoleByName", mock.Anything, "admin").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.GetRoleByName(ctx, actor, "admin")
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				assertRole(t, result, "role-1")
			},
		},
		{
			name: "get role by name service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesReadPermission).Return(nil).Once()
				m.roles.On("GetRoleByName", mock.Anything, "admin").Return((*types.Role)(nil), repoErr).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.GetRoleByName(ctx, actor, "admin")
			},
			wantErr: repoErr,
		},
		{
			name: "get role by id denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesReadPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.GetRoleByID(ctx, actor, "role-1")
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "get role by id success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesReadPermission).Return(nil).Once()
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				m.rolePermissions.On("GetRolePermissions", mock.Anything, "role-1").Return([]types.UserPermissionInfo{{PermissionID: "perm-1"}}, nil).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.GetRoleByID(ctx, actor, "role-1")
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				details, ok := result.(*types.RoleDetails)
				if !ok || details == nil || details.Role.ID != "role-1" || len(details.Permissions) != 1 {
					t.Fatalf("unexpected role details %#v", result)
				}
			},
		},
		{
			name: "get role by id service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesReadPermission).Return(nil).Once()
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), repoErr).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.GetRoleByID(ctx, actor, "role-1")
			},
			wantErr: repoErr,
		},
		{
			name: "update role denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesUpdatePermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.UpdateRole(ctx, actor, "role-1", types.UpdateRoleRequest{Name: new("editor")})
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "update role success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesUpdatePermission).Return(nil).Once()
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "viewer"}, nil).Once()
				m.roles.On("UpdateRole", mock.Anything, "role-1", new("editor"), (*string)(nil), (*int)(nil)).Return(true, nil).Once()
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "editor"}, nil).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.UpdateRole(ctx, actor, "role-1", types.UpdateRoleRequest{Name: new("editor")})
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				assertRole(t, result, "role-1")
			},
		},
		{
			name: "update role service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesUpdatePermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return uc.UpdateRole(ctx, actor, "", types.UpdateRoleRequest{Name: new("editor")})
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "delete role denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesDeletePermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return nil, uc.DeleteRole(ctx, actor, "role-1")
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "delete role success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesDeletePermission).Return(nil).Once()
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				m.userRoles.On("CountUsersByRole", mock.Anything, "role-1").Return(0, nil).Once()
				m.roles.On("DeleteRole", mock.Anything, "role-1").Return(true, nil).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return nil, uc.DeleteRole(ctx, actor, "role-1")
			},
		},
		{
			name: "delete role service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, accesscontrolconstants.RolesDeletePermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *RolesUseCase) (any, error) {
				return nil, uc.DeleteRole(ctx, actor, "")
			},
			wantErr: coreerrors.ErrBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mocks := newUseCaseMocks()
			tc.setup(mocks)

			result, err := tc.run(context.Background(), mocks.rolesUseCase())
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
