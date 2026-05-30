package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
)

type GetUserRolesHandler struct {
	useCase *usecases.UserRolesUseCase
}

func NewGetUserRolesHandler(useCase *usecases.UserRolesUseCase) *GetUserRolesHandler {
	return &GetUserRolesHandler{
		useCase: useCase,
	}
}

func (h *GetUserRolesHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		userID := r.PathValue("user_id")

		roles, err := h.useCase.GetUserRoles(r.Context(), userID)
		if err != nil {
			respondUserHandlerError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, roles)
	}
}

type ReplaceUserRolesHandler struct {
	useCase *usecases.UserRolesUseCase
}

func NewReplaceUserRolesHandler(useCase *usecases.UserRolesUseCase) *ReplaceUserRolesHandler {
	return &ReplaceUserRolesHandler{
		useCase: useCase,
	}
}

func (h *ReplaceUserRolesHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		userID := r.PathValue("user_id")

		var request types.ReplaceUserRolesRequest
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

		if err := h.useCase.ReplaceUserRoles(ctx, userID, request.RoleIDs, userActorUserID(reqCtx)); err != nil {
			respondUserHandlerError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.ReplaceUserRolesResponse{Message: "user roles replaced"})
	}
}

type AssignUserRoleHandler struct {
	useCase *usecases.UserRolesUseCase
}

func NewAssignUserRoleHandler(useCase *usecases.UserRolesUseCase) *AssignUserRoleHandler {
	return &AssignUserRoleHandler{useCase: useCase}
}

func (h *AssignUserRoleHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		userID := r.PathValue("user_id")

		var request types.AssignUserRoleRequest
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

		if err := h.useCase.AssignRoleToUser(ctx, userID, request, userActorUserID(reqCtx)); err != nil {
			respondUserHandlerError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.AssignUserRoleResponse{Message: "role assigned"})
	}
}

type RemoveUserRoleHandler struct {
	useCase *usecases.UserRolesUseCase
}

func NewRemoveUserRoleHandler(useCase *usecases.UserRolesUseCase) *RemoveUserRoleHandler {
	return &RemoveUserRoleHandler{useCase: useCase}
}

func (h *RemoveUserRoleHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		userID := r.PathValue("user_id")
		roleID := r.PathValue("role_id")

		if err := h.useCase.RemoveRoleFromUser(ctx, userID, roleID); err != nil {
			respondUserHandlerError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.RemoveUserRoleResponse{Message: "role removed"})
	}
}
