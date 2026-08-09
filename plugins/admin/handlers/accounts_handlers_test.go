package handlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminhandlers "github.com/Authula/authula/plugins/admin/handlers"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	"github.com/Authula/authula/plugins/admin/types"
)

func TestCreateAccountHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            []byte
		setup           func(*testing.T, *internaltests.MockAccountRepository, *internaltests.MockUserRepository, *admintests.MockPasswordService)
		expectedStatus  int
		expectedMessage string
		expectAccount   bool
	}{
		{
			name:            "invalid body",
			body:            []byte("{invalid"),
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "invalid character 'i' looking for beginning of object key string",
		},
		{
			name: "success",
			setup: func(t *testing.T, accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository, passwordSvc *admintests.MockPasswordService) {
				t.Helper()
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), nil).Once()
				passwordSvc.On("Hash", "plain").Return("hashed", nil).Once()
				accountRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Account")).Return(&models.Account{ID: "acc-1", UserID: "u1"}, nil).Once()
			},
			expectedStatus: http.StatusCreated,
			expectAccount:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, accountRepo, userRepo, passwordSvc := admintests.NewAccountsUseCaseFixture()
			handler := adminhandlers.NewCreateAccountHandler(useCase)
			request := types.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1", Password: new("plain")}
			body := tc.body
			if body == nil {
				body = internaltests.MarshalToJSON(t, request)
			}
			if tc.setup != nil {
				tc.setup(t, accountRepo, userRepo, passwordSvc)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/users/u1/accounts", body, nil)
			req.SetPathValue("user_id", "u1")
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.expectAccount {
				payload := internaltests.DecodeResponseJSON[types.CreateAccountResponse](t, reqCtx)
				if payload.Account == nil {
					t.Fatalf("expected account in payload")
				}
			}

			accountRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
			passwordSvc.AssertExpectations(t)
		})
	}
}

func TestGetAccountByIDHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setup           func(*internaltests.MockAccountRepository)
		expectedStatus  int
		expectedMessage string
		expectAccount   bool
	}{
		{
			name: "not found",
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return((*models.Account)(nil), nil).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "account not found",
		},
		{
			name: "success",
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectAccount:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, accountRepo, _, _ := admintests.NewAccountsUseCaseFixture()
			handler := adminhandlers.NewGetAccountByIDHandler(useCase)
			if tc.setup != nil {
				tc.setup(accountRepo)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/accounts/acc-1", nil, nil)
			req.SetPathValue("id", "acc-1")
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.expectAccount {
				payload := internaltests.DecodeResponseJSON[types.GetAccountByIDResponse](t, reqCtx)
				if payload.Account == nil {
					t.Fatalf("expected account in payload")
				}
			}

			accountRepo.AssertExpectations(t)
		})
	}
}

func TestGetUserAccountsHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setup           func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository)
		expectedStatus  int
		expectedMessage string
	}{
		{
			name: "error",
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetAllByUserID", mock.Anything, "u1").Return(nil, coreerrors.ErrNotFound).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name: "success",
			setup: func(accountRepo *internaltests.MockAccountRepository, userRepo *internaltests.MockUserRepository) {
				userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				accountRepo.On("GetAllByUserID", mock.Anything, "u1").Return([]models.Account{{ID: "a1", UserID: "u1"}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, accountRepo, userRepo, _ := admintests.NewAccountsUseCaseFixture()
			tc.setup(accountRepo, userRepo)
			handler := adminhandlers.NewGetUserAccountsHandler(useCase)

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/users/u1/accounts", nil, nil)
			req.SetPathValue("user_id", "u1")
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}

			accountRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateAccountHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            []byte
		setup           func(*internaltests.MockAccountRepository)
		expectedStatus  int
		expectedMessage string
	}{
		{
			name:            "invalid body",
			body:            []byte("{invalid"),
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "invalid character 'i' looking for beginning of object key string",
		},
		{
			name: "not found",
			body: func() []byte {
				scope := "openid"
				return internaltests.MarshalToJSON(t, types.UpdateAccountRequest{Scope: &scope})
			}(),
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return((*models.Account)(nil), nil).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, accountRepo, _, _ := admintests.NewAccountsUseCaseFixture()
			handler := adminhandlers.NewUpdateAccountHandler(useCase)
			if tc.setup != nil {
				tc.setup(accountRepo)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPatch, "/admin/accounts/acc-1", tc.body, nil)
			req.SetPathValue("id", "acc-1")
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}

			accountRepo.AssertExpectations(t)
		})
	}
}

func TestDeleteAccountHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		setup           func(*internaltests.MockAccountRepository)
		expectedStatus  int
		expectedMessage string
		expectedBody    string
	}{
		{
			name: "error",
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
				accountRepo.On("Delete", mock.Anything, "acc-1").Return(coreerrors.ErrBadRequest).Once()
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "bad request",
		},
		{
			name: "success",
			setup: func(accountRepo *internaltests.MockAccountRepository) {
				accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
				accountRepo.On("Delete", mock.Anything, "acc-1").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "account deleted",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, _, accountRepo, _, _ := admintests.NewAccountsUseCaseFixture()
			handler := adminhandlers.NewDeleteAccountHandler(useCase)
			if tc.setup != nil {
				tc.setup(accountRepo)
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/admin/accounts/acc-1", nil, nil)
			req.SetPathValue("id", "acc-1")
			handler.Handler()(w, req)

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}
			if tc.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedMessage)
			}
			if tc.expectedBody != "" {
				payload := internaltests.DecodeResponseJSON[types.DeleteAccountResponse](t, reqCtx)
				if payload.Message != tc.expectedBody {
					t.Fatalf("expected %s, got %s", tc.expectedBody, payload.Message)
				}
			}

			accountRepo.AssertExpectations(t)
		})
	}
}
