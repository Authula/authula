package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
)

type GetRolePermissionsHandler struct {
	useCase *usecases.RolePermissionsUseCase
}

func NewGetRolePermissionsHandler(useCase *usecases.RolePermissionsUseCase) *GetRolePermissionsHandler {
	return &GetRolePermissionsHandler{useCase: useCase}
}

func (h *GetRolePermissionsHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		roleID := r.PathValue("role_id")

		permissions, err := h.useCase.GetRolePermissions(r.Context(), roleID)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, permissions)
	}
}

type AddRolePermissionHandler struct {
	useCase *usecases.RolePermissionsUseCase
}

func NewAddRolePermissionHandler(useCase *usecases.RolePermissionsUseCase) *AddRolePermissionHandler {
	return &AddRolePermissionHandler{useCase: useCase}
}

func (h *AddRolePermissionHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		roleID := r.PathValue("role_id")

		var request types.AddRolePermissionRequest
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

		if err := h.useCase.AddPermissionToRole(ctx, roleID, request.PermissionID, rolePermissionActorUserID(reqCtx)); err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.AddRolePermissionResponse{
			Message: "permission assigned to role",
		})
	}
}

type ReplaceRolePermissionsHandler struct {
	useCase *usecases.RolePermissionsUseCase
}

func NewReplaceRolePermissionsHandler(useCase *usecases.RolePermissionsUseCase) *ReplaceRolePermissionsHandler {
	return &ReplaceRolePermissionsHandler{useCase: useCase}
}

func (h *ReplaceRolePermissionsHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		roleID := r.PathValue("role_id")

		var request types.ReplaceRolePermissionsRequest
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

		if err := h.useCase.ReplaceRolePermissions(ctx, roleID, request.PermissionIDs, rolePermissionActorUserID(reqCtx)); err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.ReplaceRolePermissionResponse{
			Message: "role permissions replaced",
		})
	}
}

type RemoveRolePermissionHandler struct {
	useCase *usecases.RolePermissionsUseCase
}

func NewRemoveRolePermissionHandler(useCase *usecases.RolePermissionsUseCase) *RemoveRolePermissionHandler {
	return &RemoveRolePermissionHandler{useCase: useCase}
}

func (h *RemoveRolePermissionHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		roleID := r.PathValue("role_id")
		permissionID := r.PathValue("permission_id")

		if err := h.useCase.RemovePermissionFromRole(ctx, roleID, permissionID); err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.RemoveRolePermissionResponse{
			Message: "permission removed from role",
		})
	}
}
