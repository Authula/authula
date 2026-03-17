package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/usecases"
)

type GetUserRolesHandler struct {
	useCase usecases.UserRolesUseCase
}

func NewGetUserRolesHandler(useCase usecases.UserRolesUseCase) *GetUserRolesHandler {
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
	useCase usecases.RolePermissionUseCase
}

func NewReplaceUserRolesHandler(useCase usecases.RolePermissionUseCase) *ReplaceUserRolesHandler {
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
	useCase usecases.RolePermissionUseCase
}

func NewAssignUserRoleHandler(useCase usecases.RolePermissionUseCase) *AssignUserRoleHandler {
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
	useCase usecases.RolePermissionUseCase
}

func NewRemoveUserRoleHandler(useCase usecases.RolePermissionUseCase) *RemoveUserRoleHandler {
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

type GetUserEffectivePermissionsHandler struct {
	useCase usecases.UserRolesUseCase
}

func NewGetUserEffectivePermissionsHandler(useCase usecases.UserRolesUseCase) *GetUserEffectivePermissionsHandler {
	return &GetUserEffectivePermissionsHandler{
		useCase: useCase,
	}
}

func (h *GetUserEffectivePermissionsHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		userID := r.PathValue("user_id")

		permissions, err := h.useCase.GetUserEffectivePermissions(r.Context(), userID)
		if err != nil {
			respondUserHandlerError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GetUserEffectivePermissionsResponse{Permissions: permissions})
	}
}

func userActorUserID(reqCtx *models.RequestContext) *string {
	if reqCtx == nil || reqCtx.UserID == nil || *reqCtx.UserID == "" {
		return nil
	}
	return reqCtx.UserID
}

func respondUserHandlerError(reqCtx *models.RequestContext, err error) {
	if reqCtx == nil {
		return
	}

	reqCtx.SetJSONResponse(mapUserHandlerErrorStatus(err), map[string]any{"message": mapHttpErrorMessage(err)})
	reqCtx.Handled = true
}

func mapUserHandlerErrorStatus(err error) int {
	return mapHttpErrorStatus(err)
}
