package handlers

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
	"github.com/Authula/authula/util"
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
		actor := reqCtx.Actor
		userID := r.PathValue("user_id")

		permissions, err := h.useCase.GetUserPermissions(ctx, actor, userID)
		if err != nil {
			respondUserHandlerError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GetUserPermissionsResponse{Permissions: permissions})
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
		actor := reqCtx.Actor
		userID := r.PathValue("user_id")

		var request types.CheckUserPermissionsRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}
		if err := request.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		allowed, err := h.useCase.HasPermissions(ctx, actor, userID, request.PermissionKeys)
		if err != nil {
			respondUserHandlerError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.CheckUserPermissionsResponse{HasPermissions: allowed})
	}
}
