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
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	adminservices "github.com/Authula/authula/plugins/admin/services"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	admintypes "github.com/Authula/authula/plugins/admin/types"
)

func TestAccountsService_Create(t *testing.T) {
	t.Parallel()

	hashErr := errors.New("hash failed")
	repoErr := errors.New("repo failure")
	createErr := errors.New("create failure")

	tests := []struct {
		name           string
		userID         string
		request        admintypes.CreateAccountRequest
		nilPasswordSvc bool
		setup          func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository, passwordSvc *admintests.MockPasswordService)
		wantErr        error
		wantErrMsg     string
	}{
		{
			name:    "success hashes password",
			userID:  "u1",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1", Password: new("plain")},
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository, passwordSvc *admintests.MockPasswordService) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), nil).Once()
				passwordSvc.On("Hash", "plain").Return("hashed", nil).Once()
				accountRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Account")).Run(func(args mock.Arguments) {
					acc := args.Get(1).(*models.Account)
					if assert.NotNil(t, acc.Password) {
						assert.Equal(t, "hashed", *acc.Password)
					}
				}).Return(&models.Account{ID: "acc-1", UserID: "u1"}, nil).Once()
			},
		},
		{
			name:    "user not found",
			userID:  "u1",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1"},
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository, passwordSvc *admintests.MockPasswordService) {
				userRepo.On("GetByID", mock.Anything, "u1").Return((*models.User)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:    "conflict on existing provider account",
			userID:  "u1",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1"},
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository, passwordSvc *admintests.MockPasswordService) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return(&models.Account{ID: "acc-existing"}, nil).Once()
			},
			wantErr: coreerrors.ErrConflict,
		},
		{
			name:    "password hash error",
			userID:  "u1",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1", Password: new("plain")},
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository, passwordSvc *admintests.MockPasswordService) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), nil).Once()
				passwordSvc.On("Hash", "plain").Return("", hashErr).Once()
			},
			wantErr: hashErr,
		},
		{
			name:    "missing user id",
			userID:  "   ",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1"},
			wantErr: adminconstants.ErrUserIDRequired,
		},
		{
			name:    "missing provider id",
			userID:  "u1",
			request: admintypes.CreateAccountRequest{ProviderID: "", AccountID: "acct-1"},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "missing account id",
			userID:  "u1",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: ""},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "user lookup fails",
			userID:  "u1",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1"},
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository, passwordSvc *admintests.MockPasswordService) {
				userRepo.On("GetByID", mock.Anything, "u1").Return((*models.User)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:    "provider lookup fails",
			userID:  "u1",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1"},
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository, passwordSvc *admintests.MockPasswordService) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:           "password service unavailable",
			userID:         "u1",
			request:        admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1", Password: new("plain")},
			nilPasswordSvc: true,
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository, passwordSvc *admintests.MockPasswordService) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), nil).Once()
			},
			wantErrMsg: "password service unavailable",
		},
		{
			name:    "create fails",
			userID:  "u1",
			request: admintypes.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1"},
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository, passwordSvc *admintests.MockPasswordService) {
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

			svc, accountRepo, userRepo, passwordSvc := admintests.NewAccountsServiceFixture()
			if tt.nilPasswordSvc {
				svc = adminservices.NewAccountsService(accountRepo, userRepo, nil)
			}
			if tt.setup != nil {
				tt.setup(accountRepo, userRepo, passwordSvc)
			}

			created, err := svc.Create(context.Background(), internaltests.TestActor(), tt.userID, tt.request)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, created)
			} else if tt.wantErrMsg != "" {
				assert.ErrorContains(t, err, tt.wantErrMsg)
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

func TestAccountsService_GetByUserID(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo failure")

	tests := []struct {
		name    string
		userID  string
		setup   func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository)
		wantErr error
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetAllByUserID", mock.Anything, "u1").Return([]models.Account{{ID: "a1", UserID: "u1"}}, nil).Once()
			},
		},
		{
			name:   "user not found",
			userID: "u1",
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository) {
				userRepo.On("GetByID", mock.Anything, "u1").Return((*models.User)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:    "missing user id",
			userID:  "   ",
			wantErr: adminconstants.ErrUserIDRequired,
		},
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
			if tt.setup != nil {
				tt.setup(accountRepo, userRepo)
			}

			accounts, err := svc.GetByUserID(context.Background(), internaltests.TestActor(), tt.userID)

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

func TestAccountsService_Update(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo failure")
	updateErr := errors.New("update failure")
	hashErr := errors.New("hash failure")

	tests := []struct {
		name           string
		accountID      string
		request        admintypes.UpdateAccountRequest
		nilPasswordSvc bool
		setup          func(accountRepo *internaltests.MockAccountRepository, passwordSvc *admintests.MockPasswordService)
		wantErr        error
		wantErrMsg     string
	}{
		{
			name:      "success hashes password",
			accountID: "acc-1",
			request:   admintypes.UpdateAccountRequest{Password: new("new-password")},
			setup: func(accountRepo *internaltests.MockAccountRepository, passwordSvc *admintests.MockPasswordService) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1", UserID: "u1"}, nil).Once()
				passwordSvc.On("Hash", "new-password").Return("hashed-new", nil).Once()
				accountRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Account")).Run(func(args mock.Arguments) {
					acc := args.Get(1).(*models.Account)
					if assert.NotNil(t, acc.Password) {
						assert.Equal(t, "hashed-new", *acc.Password)
					}
				}).Return(&models.Account{ID: "acc-1", UserID: "u1", Password: new("hashed-new")}, nil).Once()
			},
		},
		{
			name:      "not found",
			accountID: "acc-1",
			request:   admintypes.UpdateAccountRequest{Scope: new("openid")},
			setup: func(accountRepo *internaltests.MockAccountRepository, passwordSvc *admintests.MockPasswordService) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return((*models.Account)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:      "missing account id",
			accountID: "",
			request:   admintypes.UpdateAccountRequest{Scope: new("openid")},
			wantErr:   coreerrors.ErrBadRequest,
		},
		{
			name:      "nothing to update",
			accountID: "acc-1",
			request:   admintypes.UpdateAccountRequest{},
			wantErr:   coreerrors.ErrBadRequest,
		},
		{
			name:      "lookup fails",
			accountID: "acc-1",
			request:   admintypes.UpdateAccountRequest{Scope: new("openid")},
			setup: func(accountRepo *internaltests.MockAccountRepository, _ *admintests.MockPasswordService) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return((*models.Account)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:      "password hash fails",
			accountID: "acc-1",
			request:   admintypes.UpdateAccountRequest{Password: new("plain")},
			setup: func(accountRepo *internaltests.MockAccountRepository, passwordSvc *admintests.MockPasswordService) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
				passwordSvc.On("Hash", "plain").Return("", hashErr).Once()
			},
			wantErr: hashErr,
		},
		{
			name:           "password service unavailable",
			accountID:      "acc-1",
			request:        admintypes.UpdateAccountRequest{Password: new("plain")},
			nilPasswordSvc: true,
			setup: func(accountRepo *internaltests.MockAccountRepository, _ *admintests.MockPasswordService) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
			},
			wantErrMsg: "password service unavailable",
		},
		{
			name:      "update fails",
			accountID: "acc-1",
			request:   admintypes.UpdateAccountRequest{Scope: new("openid")},
			setup: func(accountRepo *internaltests.MockAccountRepository, _ *admintests.MockPasswordService) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
				accountRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Account")).Return((*models.Account)(nil), updateErr).Once()
			},
			wantErr: updateErr,
		},
		{
			name:      "trims provider id, account id and scope",
			accountID: "acc-1",
			request:   admintypes.UpdateAccountRequest{ProviderID: new("  EMAIL "), AccountID: new(" acct "), Scope: new(" openid ")},
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

			svc, accountRepo, userRepo, passwordSvc := admintests.NewAccountsServiceFixture()
			if tt.nilPasswordSvc {
				svc = adminservices.NewAccountsService(accountRepo, userRepo, nil)
			}
			if tt.setup != nil {
				tt.setup(accountRepo, passwordSvc)
			}

			updated, err := svc.Update(context.Background(), internaltests.TestActor(), tt.accountID, tt.request)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, updated)
			} else if tt.wantErrMsg != "" {
				assert.ErrorContains(t, err, tt.wantErrMsg)
				assert.Nil(t, updated)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updated)
			}

			accountRepo.AssertExpectations(t)
			passwordSvc.AssertExpectations(t)
		})
	}
}

func TestAccountsService_Delete(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo failure")
	deleteErr := errors.New("delete failure")

	tests := []struct {
		name      string
		accountID string
		setup     func(accountRepo *internaltests.MockAccountRepository)
		wantErr   error
	}{
		{
			name:      "success",
			accountID: "acc-1",
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
				accountRepo.On("Delete", mock.Anything, "acc-1").Return(nil).Once()
			},
		},
		{
			name:      "not found",
			accountID: "acc-1",
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return((*models.Account)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:      "missing account id",
			accountID: "",
			wantErr:   coreerrors.ErrBadRequest,
		},
		{
			name:      "lookup fails",
			accountID: "acc-1",
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return((*models.Account)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:      "delete fails",
			accountID: "acc-1",
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
			if tt.setup != nil {
				tt.setup(accountRepo)
			}

			err := svc.Delete(context.Background(), internaltests.TestActor(), tt.accountID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			accountRepo.AssertExpectations(t)
		})
	}
}
