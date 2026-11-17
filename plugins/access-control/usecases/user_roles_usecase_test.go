package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestUserRolesUseCase(t *testing.T) {
	t.Parallel()

	actor := testActor()
	repoErr := errors.New("repository failed")
	assignedBy := "actor-1"

	tests := []struct {
		name    string
		setup   func(*useCaseMocks)
		run     func(context.Context, *UserRolesUseCase) (any, error)
		wantErr error
		check   func(*testing.T, any)
	}{
		{
			name: "get user roles denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserRolesReadPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *UserRolesUseCase) (any, error) {
				return uc.GetUserRoles(ctx, actor, "user-1")
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "get user roles success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserRolesReadPermission).Return(nil).Once()
				m.userRoles.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{{RoleID: "role-1"}}, nil).Once()
			},
			run: func(ctx context.Context, uc *UserRolesUseCase) (any, error) {
				return uc.GetUserRoles(ctx, actor, "user-1")
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				assertUserRoles(t, result, 1)
			},
		},
		{
			name: "get user roles service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserRolesReadPermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UserRolesUseCase) (any, error) {
				return uc.GetUserRoles(ctx, actor, "")
			},
			wantErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "replace user roles denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserRolesAssignPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *UserRolesUseCase) (any, error) {
				return nil, uc.ReplaceUserRoles(ctx, actor, "user-1", []string{"role-1"}, &assignedBy)
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "replace user roles success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserRolesAssignPermission).Return(nil).Once()
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Weight: 10}, nil).Once()
				m.userRoles.On("GetUserRoles", mock.Anything, "actor-1").Return([]types.UserRoleInfo{{RoleID: "role-9", RoleWeight: 20}}, nil).Once()
				m.userRoles.On("ReplaceUserRoles", mock.Anything, "user-1", []string{"role-1"}, &assignedBy).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UserRolesUseCase) (any, error) {
				return nil, uc.ReplaceUserRoles(ctx, actor, "user-1", []string{"role-1"}, &assignedBy)
			},
		},
		{
			name: "replace user roles service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserRolesAssignPermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UserRolesUseCase) (any, error) {
				return nil, uc.ReplaceUserRoles(ctx, actor, "", []string{"role-1"}, &assignedBy)
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "assign role denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserRolesAssignPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *UserRolesUseCase) (any, error) {
				return nil, uc.AssignRoleToUser(ctx, actor, "user-1", types.AssignUserRoleRequest{RoleID: "role-1"}, &assignedBy)
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "assign role success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserRolesAssignPermission).Return(nil).Once()
				m.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Weight: 10}, nil).Once()
				m.userRoles.On("GetUserRoles", mock.Anything, "actor-1").Return([]types.UserRoleInfo{{RoleID: "role-9", RoleWeight: 20}}, nil).Once()
				m.userRoles.On("AssignUserRole", mock.Anything, "user-1", "role-1", &assignedBy, (*time.Time)(nil)).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UserRolesUseCase) (any, error) {
				return nil, uc.AssignRoleToUser(ctx, actor, "user-1", types.AssignUserRoleRequest{RoleID: "role-1"}, &assignedBy)
			},
		},
		{
			name: "assign role service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserRolesAssignPermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UserRolesUseCase) (any, error) {
				return nil, uc.AssignRoleToUser(ctx, actor, "", types.AssignUserRoleRequest{RoleID: "role-1"}, &assignedBy)
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "remove role denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserRolesRemovePermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *UserRolesUseCase) (any, error) {
				return nil, uc.RemoveRoleFromUser(ctx, actor, "user-1", "role-1")
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "remove role success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserRolesRemovePermission).Return(nil).Once()
				m.userRoles.On("RemoveUserRole", mock.Anything, "user-1", "role-1").Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UserRolesUseCase) (any, error) {
				return nil, uc.RemoveRoleFromUser(ctx, actor, "user-1", "role-1")
			},
		},
		{
			name: "remove role service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserRolesRemovePermission).Return(nil).Once()
			},
			run: func(ctx context.Context, uc *UserRolesUseCase) (any, error) {
				return nil, uc.RemoveRoleFromUser(ctx, actor, "user-1", "")
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "service repository error propagates",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserRolesReadPermission).Return(nil).Once()
				m.userRoles.On("GetUserRoles", mock.Anything, "user-1").Return(([]types.UserRoleInfo)(nil), repoErr).Once()
			},
			run: func(ctx context.Context, uc *UserRolesUseCase) (any, error) {
				return uc.GetUserRoles(ctx, actor, "user-1")
			},
			wantErr: repoErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mocks := newUseCaseMocks()
			tc.setup(mocks)

			result, err := tc.run(context.Background(), mocks.userRolesUseCase())
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
