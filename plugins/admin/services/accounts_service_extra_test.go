package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminservices "github.com/Authula/authula/plugins/admin/services"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	admintypes "github.com/Authula/authula/plugins/admin/types"
)

func TestAccountsService_GetByID(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo failure")

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
			name:      "repo error",
			accountID: "acc-1",
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return((*models.Account)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:      "success trims id",
			accountID: " acc-1 ",
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, accountRepo, _, _ := admintests.NewAccountsServiceFixture()
			if tt.setup != nil {
				tt.setup(accountRepo)
			}

			account, err := svc.GetByID(context.Background(), internaltests.TestActor(), tt.accountID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, account)
			} else {
				require.NoError(t, err)
				require.NotNil(t, account)
				assert.Equal(t, "acc-1", account.ID)
			}

			accountRepo.AssertExpectations(t)
		})
	}
}

func TestAccountsService_GetByUserID_Errors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo failure")

	tests := []struct {
		name    string
		userID  string
		setup   func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository)
		wantErr error
	}{
		{
			name:   "user lookup fails",
			userID: "u1",
			setup: func(_ *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository) {
				userRepo.On("GetByID", mock.Anything, "u1").Return((*models.User)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:   "accounts lookup fails",
			userID: "u1",
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetAllByUserID", mock.Anything, "u1").Return(([]models.Account)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, accountRepo, userRepo, _ := admintests.NewAccountsServiceFixture()
			tt.setup(accountRepo, userRepo)

			accounts, err := svc.GetByUserID(context.Background(), internaltests.TestActor(), tt.userID)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, accounts)

			accountRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
		})
	}
}

func TestAccountsService_Create_Errors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo failure")
	createErr := errors.New("create failure")

	tests := []struct {
		name            string
		nilPasswordSvc  bool
		setup           func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository)
		request         admintypes.CreateAccountRequest
		wantErr         error
		wantUnavailable bool
	}{
		{
			name:    "user lookup fails",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1"},
			setup: func(_ *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository) {
				userRepo.On("GetByID", mock.Anything, "u1").Return((*models.User)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:    "provider lookup fails",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1"},
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:           "password service unavailable",
			nilPasswordSvc: true,
			request:        admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1", Password: new("plain")},
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), nil).Once()
			},
			wantUnavailable: true,
		},
		{
			name:    "create fails",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1"},
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), nil).Once()
				accountRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Account")).Return((*models.Account)(nil), createErr).Once()
			},
			wantErr: createErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			accountRepo := &internaltests.MockAccountRepository{}
			userRepo := &internaltests.MockUserRepository{}
			var passwordSvc *admintests.MockPasswordService
			var svc *adminservices.AccountsService
			if tt.nilPasswordSvc {
				svc = adminservices.NewAccountsService(accountRepo, userRepo, nil)
			} else {
				passwordSvc = &admintests.MockPasswordService{}
				svc = adminservices.NewAccountsService(accountRepo, userRepo, passwordSvc)
			}
			if tt.setup != nil {
				tt.setup(accountRepo, userRepo)
			}

			account, err := svc.Create(context.Background(), internaltests.TestActor(), "u1", tt.request)
			if tt.wantUnavailable {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "password service unavailable")
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
			assert.Nil(t, account)

			accountRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
			if passwordSvc != nil {
				passwordSvc.AssertExpectations(t)
			}
		})
	}
}

func TestAccountsService_Update_Errors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo failure")
	updateErr := errors.New("update failure")
	hashErr := errors.New("hash failure")

	tests := []struct {
		name            string
		nilPasswordSvc  bool
		request         admintypes.UpdateAccountRequest
		setup           func(accountRepo *internaltests.MockAccountRepository, passwordSvc *admintests.MockPasswordService)
		wantErr         error
		wantUnavailable bool
	}{
		{
			name:    "lookup fails",
			request: admintypes.UpdateAccountRequest{Scope: new("openid")},
			setup: func(accountRepo *internaltests.MockAccountRepository, _ *admintests.MockPasswordService) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return((*models.Account)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:    "password hash fails",
			request: admintypes.UpdateAccountRequest{Password: new("plain")},
			setup: func(accountRepo *internaltests.MockAccountRepository, passwordSvc *admintests.MockPasswordService) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
				passwordSvc.On("Hash", "plain").Return("", hashErr).Once()
			},
			wantErr: hashErr,
		},
		{
			name:           "password service unavailable",
			nilPasswordSvc: true,
			request:        admintypes.UpdateAccountRequest{Password: new("plain")},
			setup: func(accountRepo *internaltests.MockAccountRepository, _ *admintests.MockPasswordService) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
			},
			wantUnavailable: true,
		},
		{
			name:    "update fails",
			request: admintypes.UpdateAccountRequest{Scope: new("openid")},
			setup: func(accountRepo *internaltests.MockAccountRepository, _ *admintests.MockPasswordService) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
				accountRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Account")).Return((*models.Account)(nil), updateErr).Once()
			},
			wantErr: updateErr,
		},
		{
			name:    "trims provider id, account id and scope",
			request: admintypes.UpdateAccountRequest{ProviderID: new("  EMAIL "), AccountID: new(" acct "), Scope: new(" openid ")},
			setup: func(accountRepo *internaltests.MockAccountRepository, _ *admintests.MockPasswordService) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
				accountRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Account) bool {
					return a.ProviderID == "email" &&
						a.AccountID == "acct" &&
						a.Scope != nil && *a.Scope == "openid"
				})).Return(&models.Account{ID: "acc-1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			accountRepo := &internaltests.MockAccountRepository{}
			userRepo := &internaltests.MockUserRepository{}
			var passwordSvc *admintests.MockPasswordService
			var svc *adminservices.AccountsService
			if tt.nilPasswordSvc {
				svc = adminservices.NewAccountsService(accountRepo, userRepo, nil)
			} else {
				passwordSvc = &admintests.MockPasswordService{}
				svc = adminservices.NewAccountsService(accountRepo, userRepo, passwordSvc)
			}
			if tt.setup != nil {
				tt.setup(accountRepo, passwordSvc)
			}

			account, err := svc.Update(context.Background(), internaltests.TestActor(), "acc-1", tt.request)
			if tt.wantUnavailable {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "password service unavailable")
				assert.Nil(t, account)
			} else if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, account)
			} else {
				require.NoError(t, err)
				require.NotNil(t, account)
			}

			accountRepo.AssertExpectations(t)
			if passwordSvc != nil {
				passwordSvc.AssertExpectations(t)
			}
		})
	}
}

func TestAccountsService_Delete_Errors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo failure")
	deleteErr := errors.New("delete failure")

	tests := []struct {
		name    string
		setup   func(accountRepo *internaltests.MockAccountRepository)
		wantErr error
	}{
		{
			name: "lookup fails",
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return((*models.Account)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name: "delete fails",
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
				accountRepo.On("Delete", mock.Anything, "acc-1").Return(deleteErr).Once()
			},
			wantErr: deleteErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, accountRepo, _, _ := admintests.NewAccountsServiceFixture()
			tt.setup(accountRepo)

			err := svc.Delete(context.Background(), internaltests.TestActor(), "acc-1")
			assert.ErrorIs(t, err, tt.wantErr)

			accountRepo.AssertExpectations(t)
		})
	}
}
