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

type GetUserWithRolesHandler struct {
	useCase *usecases.UserRolesUseCase
}

func NewGetUserWithRolesHandler(useCase *usecases.UserRolesUseCase) *GetUserWithRolesHandler {
	return &GetUserWithRolesHandler{useCase: useCase}
}

func (h *GetUserWithRolesHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		userID := r.PathValue("user_id")

		userWithRoles, err := h.useCase.GetUserWithRolesByID(r.Context(), userID)
		if err != nil {
			respondUserHandlerError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, userWithRoles)
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

		var payload types.ReplaceUserRolesRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		if err := h.useCase.ReplaceUserRoles(r.Context(), userID, payload.RoleIDs, userActorUserID(reqCtx)); err != nil {
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

		var payload types.AssignUserRoleRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		if err := h.useCase.AssignRoleToUser(r.Context(), userID, payload, userActorUserID(reqCtx)); err != nil {
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

		if err := h.useCase.RemoveRoleFromUser(r.Context(), userID, roleID); err != nil {
			respondUserHandlerError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.RemoveUserRoleResponse{Message: "role removed"})
	}
}
