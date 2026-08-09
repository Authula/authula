package usecases_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	admintypes "github.com/Authula/authula/plugins/admin/types"
)

func TestAccountsUseCase_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		request admintypes.CreateAccountRequest
		setup   func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository, passwordSvc *admintests.MockPasswordService)
		wantErr error
	}{
		{
			name:    "missing user id",
			userID:  "",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "a1"},
			wantErr: adminconstants.ErrUserIDRequired,
		},
		{
			name:    "missing provider id",
			userID:  "u1",
			request: admintypes.CreateAccountRequest{ProviderID: "", AccountID: "a1"},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "missing account id",
			userID:  "u1",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: ""},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "trims and normalizes",
			userID:  "u1",
			request: admintypes.CreateAccountRequest{ProviderID: "  EMAIL ", AccountID: "  acct-1  ", Password: new("secret")},
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository, passwordSvc *admintests.MockPasswordService) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), nil).Once()
				passwordSvc.On("Hash", "secret").Return("hashed-secret", nil).Once()
				accountRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Account")).Run(func(args mock.Arguments) {
					acc := args.Get(1).(*models.Account)
					assert.Equal(t, "email", acc.ProviderID)
					assert.Equal(t, "acct-1", acc.AccountID)
				}).Return(&models.Account{ID: "acc-1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, accountRepo, userRepo, passwordSvc := admintests.NewAccountsUseCaseFixture()
			if tt.setup != nil {
				tt.setup(accountRepo, userRepo, passwordSvc)
			}

			created, err := useCase.Create(context.Background(), internaltests.TestActor(), tt.userID, tt.request)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, created)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, created)
			}

			accountRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
			passwordSvc.AssertExpectations(t)
		})
	}
}

func TestAccountsUseCase_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		accountID string
		setup     func(accountRepo *internaltests.MockAccountRepository)
		wantErr   error
	}{
		{
			name:      "missing account id",
			accountID: "   ",
			wantErr:   coreerrors.ErrBadRequest,
		},
		{
			name:      "success",
			accountID: "acc-1",
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, accountRepo, _, _ := admintests.NewAccountsUseCaseFixture()
			if tt.setup != nil {
				tt.setup(accountRepo)
			}

			account, err := useCase.GetByID(context.Background(), internaltests.TestActor(), tt.accountID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, account)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "acc-1", account.ID)
			}
			accountRepo.AssertExpectations(t)
		})
	}
}

func TestAccountsUseCase_GetByUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository)
		wantErr error
	}{
		{
			name:    "missing user id",
			userID:  "   ",
			wantErr: adminconstants.ErrUserIDRequired,
		},
		{
			name:   "success",
			userID: "u1",
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetAllByUserID", mock.Anything, "u1").Return([]models.Account{{ID: "a1"}}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, accountRepo, userRepo, _ := admintests.NewAccountsUseCaseFixture()
			if tt.setup != nil {
				tt.setup(accountRepo, userRepo)
			}

			accounts, err := useCase.GetByUserID(context.Background(), internaltests.TestActor(), tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, accounts)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, accounts)
			}

			accountRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
		})
	}
}

func TestAccountsUseCase_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		accountID string
		request   admintypes.UpdateAccountRequest
		setup     func(accountRepo *internaltests.MockAccountRepository)
		wantErr   error
	}{
		{
			name:      "missing account id",
			accountID: "",
			request:   admintypes.UpdateAccountRequest{Scope: new("x")},
			wantErr:   coreerrors.ErrBadRequest,
		},
		{
			name:      "nothing to update",
			accountID: "acc-1",
			request:   admintypes.UpdateAccountRequest{},
			wantErr:   coreerrors.ErrBadRequest,
		},
		{
			name:      "success",
			accountID: "acc-1",
			request:   admintypes.UpdateAccountRequest{Scope: new("openid")},
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
				accountRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Account")).Return(&models.Account{ID: "acc-1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, accountRepo, _, _ := admintests.NewAccountsUseCaseFixture()
			if tt.setup != nil {
				tt.setup(accountRepo)
			}

			updated, err := useCase.Update(context.Background(), internaltests.TestActor(), tt.accountID, tt.request)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, updated)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updated)
			}
			accountRepo.AssertExpectations(t)
		})
	}
}

func TestAccountsUseCase_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		accountID string
		setup     func(accountRepo *internaltests.MockAccountRepository)
		wantErr   error
	}{
		{
			name:      "missing account id",
			accountID: "",
			wantErr:   coreerrors.ErrBadRequest,
		},
		{
			name:      "success",
			accountID: "acc-1",
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
				accountRepo.On("Delete", mock.Anything, "acc-1").Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, accountRepo, _, _ := admintests.NewAccountsUseCaseFixture()
			if tt.setup != nil {
				tt.setup(accountRepo)
			}

			err := useCase.Delete(context.Background(), internaltests.TestActor(), tt.accountID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			accountRepo.AssertExpectations(t)
		})
	}
}
