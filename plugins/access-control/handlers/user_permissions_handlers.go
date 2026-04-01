package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
)

type GetUserPermissionsHandler struct {
	useCase *usecases.UserPermissionsUseCase
}

func NewGetUserPermissionsHandler(useCase *usecases.UserPermissionsUseCase) *GetUserPermissionsHandler {
	return &GetUserPermissionsHandler{useCase: useCase}
}

func (h *GetUserPermissionsHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		userID := r.PathValue("user_id")

		permissions, err := h.useCase.GetUserPermissions(r.Context(), userID)
		if err != nil {
			respondUserHandlerError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GetUserEffectivePermissionsResponse{Permissions: permissions})
	}
}

type CheckUserPermissionsHandler struct {
	useCase *usecases.UserPermissionsUseCase
}

func NewCheckUserPermissionsHandler(useCase *usecases.UserPermissionsUseCase) *CheckUserPermissionsHandler {
	return &CheckUserPermissionsHandler{useCase: useCase}
}

func (h *CheckUserPermissionsHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		userID := r.PathValue("user_id")

		var payload types.CheckUserPermissionsRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		allowed, err := h.useCase.HasPermissions(r.Context(), userID, payload.PermissionKeys)
		if err != nil {
			respondUserHandlerError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.CheckUserPermissionsResponse{HasPermissions: allowed})
	}
}
