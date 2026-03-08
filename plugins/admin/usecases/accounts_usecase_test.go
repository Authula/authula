package usecases_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	admintests "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/tests"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestAccountsUseCase_Create_Validation(t *testing.T) {
	t.Parallel()

	useCase, _, _, _, _ := admintests.NewAccountsUseCaseFixture()

	_, err := useCase.Create(context.Background(), "", admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "a1"})
	assert.ErrorIs(t, err, constants.ErrUserIDRequired)

	_, err = useCase.Create(context.Background(), "u1", admintypes.CreateAccountRequest{ProviderID: "", AccountID: "a1"})
	assert.ErrorIs(t, err, constants.ErrBadRequest)

	_, err = useCase.Create(context.Background(), "u1", admintypes.CreateAccountRequest{ProviderID: "email", AccountID: ""})
	assert.ErrorIs(t, err, constants.ErrBadRequest)
}

func TestAccountsUseCase_Create_TrimsAndNormalizes(t *testing.T) {
	t.Parallel()

	useCase, _, accountRepo, userRepo, passwordSvc := admintests.NewAccountsUseCaseFixture()
	userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
	accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), nil).Once()
	passwordSvc.On("Hash", "secret").Return("hashed-secret", nil).Once()
	accountRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Account")).Run(func(args mock.Arguments) {
		acc := args.Get(1).(*models.Account)
		assert.Equal(t, "email", acc.ProviderID)
		assert.Equal(t, "acct-1", acc.AccountID)
	}).Return(&models.Account{ID: "acc-1"}, nil).Once()

	_, err := useCase.Create(context.Background(), "u1", admintypes.CreateAccountRequest{
		ProviderID: "  EMAIL ",
		AccountID:  "  acct-1  ",
		Password:   admintests.PtrString(t, "secret"),
	})
	assert.NoError(t, err)
	accountRepo.AssertExpectations(t)
}

func TestAccountsUseCase_GetByID_Validation(t *testing.T) {
	t.Parallel()

	useCase, _, _, _, _ := admintests.NewAccountsUseCaseFixture()
	_, err := useCase.GetByID(context.Background(), "   ")
	assert.ErrorIs(t, err, constants.ErrBadRequest)
}

func TestAccountsUseCase_GetByUserID_Validation(t *testing.T) {
	t.Parallel()

	useCase, _, _, _, _ := admintests.NewAccountsUseCaseFixture()
	_, err := useCase.GetByUserID(context.Background(), "   ")
	assert.ErrorIs(t, err, constants.ErrUserIDRequired)
}

func TestAccountsUseCase_Update_Validation(t *testing.T) {
	t.Parallel()

	useCase, _, _, _, _ := admintests.NewAccountsUseCaseFixture()

	_, err := useCase.Update(context.Background(), "", admintypes.UpdateAccountRequest{Scope: admintests.PtrString(t, "x")})
	assert.ErrorIs(t, err, constants.ErrBadRequest)

	_, err = useCase.Update(context.Background(), "acc-1", admintypes.UpdateAccountRequest{})
	assert.ErrorIs(t, err, constants.ErrBadRequest)
}

func TestAccountsUseCase_Delete_Validation(t *testing.T) {
	t.Parallel()

	useCase, _, _, _, _ := admintests.NewAccountsUseCaseFixture()
	err := useCase.Delete(context.Background(), "")
	assert.ErrorIs(t, err, constants.ErrBadRequest)
}
