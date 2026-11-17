package usecases_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	admintypes "github.com/Authula/authula/plugins/admin/types"
	"github.com/Authula/authula/plugins/admin/usecases"
)

func TestUsersUseCase_AuthorizerDenies(t *testing.T) {
	t.Parallel()

	denied := coreerrors.ErrForbidden
	ctx := context.Background()

	tests := []struct {
		name  string
		scope string
		call  func(uc usecases.UsersUseCase) error
	}{
		{
			name:  "create",
			scope: adminconstants.UsersCreatePermission,
			call: func(uc usecases.UsersUseCase) error {
				_, err := uc.Create(ctx, internaltests.TestActor(), admintypes.CreateUserRequest{})
				return err
			},
		},
		{
			name:  "get all",
			scope: adminconstants.UsersListPermission,
			call: func(uc usecases.UsersUseCase) error {
				_, err := uc.GetAll(ctx, internaltests.TestActor(), nil, 10)
				return err
			},
		},
		{
			name:  "get by id",
			scope: adminconstants.UsersReadPermission,
			call: func(uc usecases.UsersUseCase) error {
				_, err := uc.GetByID(ctx, internaltests.TestActor(), "u1")
				return err
			},
		},
		{
			name:  "update",
			scope: adminconstants.UsersUpdatePermission,
			call: func(uc usecases.UsersUseCase) error {
				_, err := uc.Update(ctx, internaltests.TestActor(), "u1", admintypes.UpdateUserRequest{})
				return err
			},
		},
		{
			name:  "delete",
			scope: adminconstants.UsersDeletePermission,
			call: func(uc usecases.UsersUseCase) error {
				return uc.Delete(ctx, internaltests.TestActor(), "u1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			authorizer := &internaltests.MockAuthorizer{}
			authorizer.On("AuthorizeScope", mock.Anything, mock.Anything, tt.scope).Return(denied).Once()
			uc := usecases.NewUsersUseCase(nil, authorizer)

			assert.ErrorIs(t, tt.call(uc), denied)
			authorizer.AssertExpectations(t)
		})
	}
}

func TestAccountsUseCase_AuthorizerDenies(t *testing.T) {
	t.Parallel()

	denied := coreerrors.ErrForbidden
	ctx := context.Background()

	tests := []struct {
		name  string
		scope string
		call  func(uc usecases.AccountsUseCase) error
	}{
		{
			name:  "get by id",
			scope: adminconstants.AccountsReadPermission,
			call: func(uc usecases.AccountsUseCase) error {
				_, err := uc.GetByID(ctx, internaltests.TestActor(), "acc-1")
				return err
			},
		},
		{
			name:  "get by user id",
			scope: adminconstants.AccountsListPermission,
			call: func(uc usecases.AccountsUseCase) error {
				_, err := uc.GetByUserID(ctx, internaltests.TestActor(), "u1")
				return err
			},
		},
		{
			name:  "create",
			scope: adminconstants.AccountsCreatePermission,
			call: func(uc usecases.AccountsUseCase) error {
				_, err := uc.Create(ctx, internaltests.TestActor(), "u1", admintypes.CreateAccountRequest{})
				return err
			},
		},
		{
			name:  "update",
			scope: adminconstants.AccountsUpdatePermission,
			call: func(uc usecases.AccountsUseCase) error {
				_, err := uc.Update(ctx, internaltests.TestActor(), "acc-1", admintypes.UpdateAccountRequest{})
				return err
			},
		},
		{
			name:  "delete",
			scope: adminconstants.AccountsDeletePermission,
			call: func(uc usecases.AccountsUseCase) error {
				return uc.Delete(ctx, internaltests.TestActor(), "acc-1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			authorizer := &internaltests.MockAuthorizer{}
			authorizer.On("AuthorizeScope", mock.Anything, mock.Anything, tt.scope).Return(denied).Once()
			uc := usecases.NewAccountsUseCase(nil, authorizer)

			assert.ErrorIs(t, tt.call(uc), denied)
			authorizer.AssertExpectations(t)
		})
	}
}

func TestImpersonationUseCase_AuthorizerDenies(t *testing.T) {
	t.Parallel()

	denied := coreerrors.ErrForbidden
	ctx := context.Background()

	tests := []struct {
		name  string
		scope string
		call  func(uc usecases.ImpersonationUseCase) error
	}{
		{
			name:  "get all",
			scope: adminconstants.ImpersonationsListPermission,
			call: func(uc usecases.ImpersonationUseCase) error {
				_, err := uc.GetAllImpersonations(ctx, internaltests.TestActor())
				return err
			},
		},
		{
			name:  "get by id",
			scope: adminconstants.ImpersonationsReadPermission,
			call: func(uc usecases.ImpersonationUseCase) error {
				_, err := uc.GetImpersonationByID(ctx, internaltests.TestActor(), "imp-1")
				return err
			},
		},
		{
			name:  "start",
			scope: adminconstants.ImpersonationsStartPermission,
			call: func(uc usecases.ImpersonationUseCase) error {
				_, err := uc.StartImpersonation(ctx, internaltests.TestActor(), nil, nil, nil, admintypes.StartImpersonationRequest{}, nil, "", 0)
				return err
			},
		},
		{
			name:  "stop",
			scope: adminconstants.ImpersonationsStopPermission,
			call: func(uc usecases.ImpersonationUseCase) error {
				_, err := uc.StopImpersonation(ctx, internaltests.TestActor(), "u1", "s1", "", admintypes.StopImpersonationRequest{})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			authorizer := &internaltests.MockAuthorizer{}
			authorizer.On("AuthorizeScope", mock.Anything, mock.Anything, tt.scope).Return(denied).Once()
			uc := usecases.NewImpersonationUseCase(nil, nil, authorizer)

			assert.ErrorIs(t, tt.call(uc), denied)
			authorizer.AssertExpectations(t)
		})
	}
}
