package handlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	adminhandlers "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/handlers"
	admintests "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestCreateAccountHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid body", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _, _ := admintests.NewAccountsUseCaseFixture()
		handler := adminhandlers.NewCreateAccountHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/users/u1/accounts", []byte("{invalid"))
		req.SetPathValue("user_id", "u1")

		handler.Handler()(w, req)
		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, _, accountRepo, userRepo, passwordSvc := admintests.NewAccountsUseCaseFixture()
		handler := adminhandlers.NewCreateAccountHandler(useCase)
		request := types.CreateAccountRequest{ProviderID: "email", AccountID: "acct-1", Password: admintests.PtrString(t, "plain")}

		userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
		accountRepo.On("GetByProviderAndAccountID", mock.Anything, "email", "acct-1").Return((*models.Account)(nil), nil).Once()
		passwordSvc.On("Hash", "plain").Return("hashed", nil).Once()
		accountRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Account")).Return(&models.Account{ID: "acc-1", UserID: "u1"}, nil).Once()

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/admin/users/u1/accounts", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("user_id", "u1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.CreateAccountResponse](t, reqCtx)
		if payload.Account == nil {
			t.Fatalf("expected account in payload")
		}
	})
}

func TestGetAccountByIDHandler(t *testing.T) {
	t.Parallel()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		useCase, _, accountRepo, _, _ := admintests.NewAccountsUseCaseFixture()
		handler := adminhandlers.NewGetAccountByIDHandler(useCase)

		accountRepo.On("GetByID", mock.Anything, "acc-1").Return((*models.Account)(nil), nil).Once()
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/accounts/acc-1", nil)
		req.SetPathValue("id", "acc-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "account not found")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, _, accountRepo, _, _ := admintests.NewAccountsUseCaseFixture()
		handler := adminhandlers.NewGetAccountByIDHandler(useCase)

		accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/accounts/acc-1", nil)
		req.SetPathValue("id", "acc-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.GetAccountByIDResponse](t, reqCtx)
		if payload.Account == nil {
			t.Fatalf("expected account in payload")
		}
	})
}

func TestGetUserAccountsHandler(t *testing.T) {
	t.Parallel()

	useCase, _, accountRepo, userRepo, _ := admintests.NewAccountsUseCaseFixture()
	handler := adminhandlers.NewGetUserAccountsHandler(useCase)

	userRepo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
	accountRepo.On("GetAllByUserID", mock.Anything, "u1").Return([]models.Account{{ID: "a1", UserID: "u1"}}, nil).Once()

	req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/admin/users/u1/accounts", nil)
	req.SetPathValue("user_id", "u1")
	handler.Handler()(w, req)

	if reqCtx.ResponseStatus != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
	}
	payload := internaltests.DecodeResponseJSON[types.UserAccountsResponse](t, reqCtx)
	if len(payload.Accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(payload.Accounts))
	}
}

func TestUpdateAccountHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid body", func(t *testing.T) {
		t.Parallel()

		useCase, _, _, _, _ := admintests.NewAccountsUseCaseFixture()
		handler := adminhandlers.NewUpdateAccountHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPatch, "/admin/accounts/acc-1", []byte("{invalid"))
		req.SetPathValue("id", "acc-1")
		handler.Handler()(w, req)
		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, _, accountRepo, _, _ := admintests.NewAccountsUseCaseFixture()
		handler := adminhandlers.NewUpdateAccountHandler(useCase)
		scope := "openid"

		accountRepo.On("GetByID", mock.Anything, "acc-1").Return((*models.Account)(nil), nil).Once()
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPatch, "/admin/accounts/acc-1", internaltests.MarshalToJSON(t, types.UpdateAccountRequest{Scope: &scope}))
		req.SetPathValue("id", "acc-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
	})
}

func TestDeleteAccountHandler(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, _, accountRepo, _, _ := admintests.NewAccountsUseCaseFixture()
		handler := adminhandlers.NewDeleteAccountHandler(useCase)

		accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
		accountRepo.On("Delete", mock.Anything, "acc-1").Return(constants.ErrBadRequest).Once()
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/admin/accounts/acc-1", nil)
		req.SetPathValue("id", "acc-1")
		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "bad request")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, _, accountRepo, _, _ := admintests.NewAccountsUseCaseFixture()
		handler := adminhandlers.NewDeleteAccountHandler(useCase)

		accountRepo.On("GetByID", mock.Anything, "acc-1").Return(&models.Account{ID: "acc-1"}, nil).Once()
		accountRepo.On("Delete", mock.Anything, "acc-1").Return(nil).Once()
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/admin/accounts/acc-1", nil)
		req.SetPathValue("id", "acc-1")
		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.DeleteAccountResponse](t, reqCtx)
		if payload.Message != "account deleted" {
			t.Fatalf("expected account deleted, got %s", payload.Message)
		}
	})
}
