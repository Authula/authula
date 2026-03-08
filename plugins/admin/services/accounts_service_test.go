package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	adminconstants "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	admintests "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/tests"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestAccountsService_Create_HashesPassword(t *testing.T) {
	t.Parallel()

	svc, accountRepo, userRepo, passwordSvc := admintests.NewAccountsServiceFixture()
	ctx := context.Background()
	request := admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1", Password: admintests.PtrString(t, "plain")}

	userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
	accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), nil).Once()
	passwordSvc.On("Hash", "plain").Return("hashed", nil).Once()
	accountRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Account")).Run(func(args mock.Arguments) {
		acc := args.Get(1).(*models.Account)
		if assert.NotNil(t, acc.Password) {
			assert.Equal(t, "hashed", *acc.Password)
		}
	}).Return(&models.Account{ID: "acc-1", UserID: "u1"}, nil).Once()

	created, err := svc.Create(ctx, "u1", request)
	assert.NoError(t, err)
	assert.NotNil(t, created)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	passwordSvc.AssertExpectations(t)
}

func TestAccountsService_Create_UserNotFound(t *testing.T) {
	t.Parallel()

	svc, accountRepo, userRepo, passwordSvc := admintests.NewAccountsServiceFixture()
	ctx := context.Background()
	request := admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1"}

	userRepo.On("GetByID", mock.Anything, "u1").Return((*models.User)(nil), nil).Once()

	created, err := svc.Create(ctx, "u1", request)
	assert.ErrorIs(t, err, adminconstants.ErrNotFound)
	assert.Nil(t, created)
	userRepo.AssertExpectations(t)
	accountRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	passwordSvc.AssertNotCalled(t, "Hash", mock.Anything)
}

func TestAccountsService_Create_Conflict(t *testing.T) {
	t.Parallel()

	svc, accountRepo, userRepo, passwordSvc := admintests.NewAccountsServiceFixture()
	ctx := context.Background()
	request := admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1"}

	userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
	accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return(&models.Account{ID: "acc-existing"}, nil).Once()

	created, err := svc.Create(ctx, "u1", request)
	assert.ErrorIs(t, err, adminconstants.ErrConflict)
	assert.Nil(t, created)
	accountRepo.AssertExpectations(t)
	passwordSvc.AssertNotCalled(t, "Hash", mock.Anything)
}

func TestAccountsService_GetByUserID(t *testing.T) {
	t.Parallel()

	svc, accountRepo, userRepo, _ := admintests.NewAccountsServiceFixture()
	ctx := context.Background()

	userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
	accountRepo.On("GetAllByUserID", mock.Anything, "u1").Return([]models.Account{{ID: "a1", UserID: "u1"}}, nil).Once()

	accounts, err := svc.GetByUserID(ctx, "u1")
	assert.NoError(t, err)
	assert.Len(t, accounts, 1)
	accountRepo.AssertExpectations(t)
}

func TestAccountsService_Update_HashesPassword(t *testing.T) {
	t.Parallel()

	svc, accountRepo, _, passwordSvc := admintests.NewAccountsServiceFixture()
	ctx := context.Background()
	plain := "new-password"
	request := admintypes.UpdateAccountRequest{Password: &plain}

	accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1", UserID: "u1"}, nil).Once()
	passwordSvc.On("Hash", "new-password").Return("hashed-new", nil).Once()
	accountRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Account")).Run(func(args mock.Arguments) {
		acc := args.Get(1).(*models.Account)
		if assert.NotNil(t, acc.Password) {
			assert.Equal(t, "hashed-new", *acc.Password)
		}
	}).Return(&models.Account{ID: "acc-1", UserID: "u1", Password: admintests.PtrString(t, "hashed-new")}, nil).Once()

	updated, err := svc.Update(ctx, "acc-1", request)
	assert.NoError(t, err)
	assert.NotNil(t, updated)
	accountRepo.AssertExpectations(t)
	passwordSvc.AssertExpectations(t)
}

func TestAccountsService_Update_NotFound(t *testing.T) {
	t.Parallel()

	svc, accountRepo, _, _ := admintests.NewAccountsServiceFixture()
	ctx := context.Background()

	accountRepo.On("GetByID", mock.Anything, "acc-1").Return((*models.Account)(nil), nil).Once()

	updated, err := svc.Update(ctx, "acc-1", admintypes.UpdateAccountRequest{Scope: admintests.PtrString(t, "openid")})
	assert.ErrorIs(t, err, adminconstants.ErrNotFound)
	assert.Nil(t, updated)
}

func TestAccountsService_Delete(t *testing.T) {
	t.Parallel()

	svc, accountRepo, _, _ := admintests.NewAccountsServiceFixture()
	ctx := context.Background()

	accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
	accountRepo.On("Delete", mock.Anything, "acc-1").Return(nil).Once()

	err := svc.Delete(ctx, "acc-1")
	assert.NoError(t, err)
	accountRepo.AssertExpectations(t)
}

func TestAccountsService_Create_PasswordHashError(t *testing.T) {
	t.Parallel()

	svc, accountRepo, userRepo, passwordSvc := admintests.NewAccountsServiceFixture()
	ctx := context.Background()
	request := admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1", Password: admintests.PtrString(t, "plain")}

	userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
	accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), nil).Once()
	passwordSvc.On("Hash", "plain").Return("", errors.New("hash failed")).Once()

	created, err := svc.Create(ctx, "u1", request)
	assert.Error(t, err)
	assert.Nil(t, created)
	accountRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}
