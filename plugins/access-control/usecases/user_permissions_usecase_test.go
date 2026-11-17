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

func TestUserPermissionsUseCase(t *testing.T) {
	t.Parallel()

	actor := testActor()
	repoErr := errors.New("repository failed")

	tests := []struct {
		name    string
		setup   func(*useCaseMocks)
		run     func(context.Context, *UserPermissionsUseCase) (any, error)
		wantErr error
		check   func(*testing.T, any)
	}{
		{
			name:  "self permissions forbidden for other user",
			setup: func(m *useCaseMocks) {},
			run: func(ctx context.Context, uc *UserPermissionsUseCase) (any, error) {
				return uc.GetSelfUserPermissions(ctx, actor, "other-user")
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "self permissions success",
			setup: func(m *useCaseMocks) {
				m.userPermissions.On("GetUserPermissions", mock.Anything, actor.ID).Return([]types.UserPermissionInfo{{PermissionID: "perm-1"}}, nil).Once()
			},
			run: func(ctx context.Context, uc *UserPermissionsUseCase) (any, error) {
				return uc.GetSelfUserPermissions(ctx, actor, actor.ID)
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				assertRolePermissions(t, result, 1)
			},
		},
		{
			name: "self permissions service error",
			setup: func(m *useCaseMocks) {
				m.userPermissions.On("GetUserPermissions", mock.Anything, actor.ID).Return(([]types.UserPermissionInfo)(nil), repoErr).Once()
			},
			run: func(ctx context.Context, uc *UserPermissionsUseCase) (any, error) {
				return uc.GetSelfUserPermissions(ctx, actor, actor.ID)
			},
			wantErr: repoErr,
		},
		{
			name: "get user permissions denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserPermissionsReadPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *UserPermissionsUseCase) (any, error) {
				return uc.GetUserPermissions(ctx, actor, "user-1")
			},
			wantErr: coreerrors.ErrForbidden,
		},
		{
			name: "get user permissions success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserPermissionsReadPermission).Return(nil).Once()
				m.userPermissions.On("GetUserPermissions", mock.Anything, "user-1").Return([]types.UserPermissionInfo{{PermissionID: "perm-1"}}, nil).Once()
			},
			run: func(ctx context.Context, uc *UserPermissionsUseCase) (any, error) {
				return uc.GetUserPermissions(ctx, actor, "user-1")
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				assertRolePermissions(t, result, 1)
			},
		},
		{
			name: "get user permissions service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserPermissionsReadPermission).Return(nil).Once()
				m.userPermissions.On("GetUserPermissions", mock.Anything, "user-1").Return(([]types.UserPermissionInfo)(nil), repoErr).Once()
			},
			run: func(ctx context.Context, uc *UserPermissionsUseCase) (any, error) {
				return uc.GetUserPermissions(ctx, actor, "user-1")
			},
			wantErr: repoErr,
		},
		{
			name: "has permissions denied",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserPermissionsCheckPermission).Return(coreerrors.ErrForbidden).Once()
			},
			run: func(ctx context.Context, uc *UserPermissionsUseCase) (any, error) {
				return uc.HasPermissions(ctx, actor, "user-1", []string{"users.read"})
			},
			wantErr: coreerrors.ErrForbidden,
			check: func(t *testing.T, result any) {
				t.Helper()
				assertBool(t, result, false)
			},
		},
		{
			name: "has permissions success",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserPermissionsCheckPermission).Return(nil).Once()
				m.userPermissions.On("HasPermissions", mock.Anything, "user-1", []string{"users.read"}).Return(true, nil).Once()
			},
			run: func(ctx context.Context, uc *UserPermissionsUseCase) (any, error) {
				return uc.HasPermissions(ctx, actor, "user-1", []string{"users.read"})
			},
			check: func(t *testing.T, result any) {
				t.Helper()
				assertBool(t, result, true)
			},
		},
		{
			name: "has permissions service error",
			setup: func(m *useCaseMocks) {
				m.authorizer.On("AuthorizeScope", mock.Anything, actor, constants.UserPermissionsCheckPermission).Return(nil).Once()
				m.userPermissions.On("HasPermissions", mock.Anything, "user-1", []string{"users.read"}).Return(false, repoErr).Once()
			},
			run: func(ctx context.Context, uc *UserPermissionsUseCase) (any, error) {
				return uc.HasPermissions(ctx, actor, "user-1", []string{"users.read"})
			},
			wantErr: repoErr,
			check: func(t *testing.T, result any) {
				t.Helper()
				assertBool(t, result, false)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mocks := newUseCaseMocks()
			tc.setup(mocks)

			result, err := tc.run(context.Background(), mocks.userPermissionsUseCase())
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil && !isNilResult(result) && tc.check == nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if tc.check != nil {
				tc.check(t, result)
			}

			mocks.assertExpectations(t)
		})
	}
}
