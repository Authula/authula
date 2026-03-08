package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

type CreateAccountHandler struct {
	useCase usecases.AccountsUseCase
}

func NewCreateAccountHandler(useCase usecases.AccountsUseCase) *CreateAccountHandler {
	return &CreateAccountHandler{useCase: useCase}
}

func (h *CreateAccountHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		userID := r.PathValue("user_id")

		var payload types.CreateAccountRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		account, err := h.useCase.Create(r.Context(), userID, payload)
		if err != nil {
			respondAccountsError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusCreated, &types.CreateAccountResponse{Account: account})
	}
}

type GetAccountByIDHandler struct {
	useCase usecases.AccountsUseCase
}

func NewGetAccountByIDHandler(useCase usecases.AccountsUseCase) *GetAccountByIDHandler {
	return &GetAccountByIDHandler{useCase: useCase}
}

func (h *GetAccountByIDHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		accountID := r.PathValue("id")

		account, err := h.useCase.GetByID(r.Context(), accountID)
		if err != nil {
			respondAccountsError(reqCtx, err)
			return
		}
		if account == nil {
			reqCtx.SetJSONResponse(http.StatusNotFound, map[string]any{"message": "account not found"})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GetAccountByIDResponse{Account: account})
	}
}

type GetUserAccountsHandler struct {
	useCase usecases.AccountsUseCase
}

func NewGetUserAccountsHandler(useCase usecases.AccountsUseCase) *GetUserAccountsHandler {
	return &GetUserAccountsHandler{useCase: useCase}
}

func (h *GetUserAccountsHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		userID := r.PathValue("user_id")

		accounts, err := h.useCase.GetByUserID(r.Context(), userID)
		if err != nil {
			respondAccountsError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.UserAccountsResponse{Accounts: accounts})
	}
}

type UpdateAccountHandler struct {
	useCase usecases.AccountsUseCase
}

func NewUpdateAccountHandler(useCase usecases.AccountsUseCase) *UpdateAccountHandler {
	return &UpdateAccountHandler{useCase: useCase}
}

func (h *UpdateAccountHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		accountID := r.PathValue("id")

		var payload types.UpdateAccountRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		account, err := h.useCase.Update(r.Context(), accountID, payload)
		if err != nil {
			respondAccountsError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.UpdateAccountResponse{Account: account})
	}
}

type DeleteAccountHandler struct {
	useCase usecases.AccountsUseCase
}

func NewDeleteAccountHandler(useCase usecases.AccountsUseCase) *DeleteAccountHandler {
	return &DeleteAccountHandler{useCase: useCase}
}

func (h *DeleteAccountHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		accountID := r.PathValue("id")

		if err := h.useCase.Delete(r.Context(), accountID); err != nil {
			respondAccountsError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.DeleteAccountResponse{Message: "account deleted"})
	}
}

func respondAccountsError(reqCtx *models.RequestContext, err error) {
	if reqCtx == nil {
		return
	}

	reqCtx.SetJSONResponse(mapAccountsErrorStatus(err), map[string]any{
		"message": mapAdminHttpErrorMessage(err),
	})
	reqCtx.Handled = true
}

func mapAccountsErrorStatus(err error) int {
	return mapAdminHttpErrorStatus(err)
}
