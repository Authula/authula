package handlers

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
	"github.com/Authula/authula/util"
)

type CreatePermissionHandler struct {
	useCase *usecases.PermissionsUseCase
}

func NewCreatePermissionHandler(useCase *usecases.PermissionsUseCase) *CreatePermissionHandler {
	return &CreatePermissionHandler{useCase: useCase}
}

func (h *CreatePermissionHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		var request types.CreatePermissionRequest
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

		permission, err := h.useCase.CreatePermission(ctx, actor, request)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusCreated, &types.CreatePermissionResponse{
			Permission: permission,
		})
	}
}

type GetAllPermissionsHandler struct {
	useCase *usecases.PermissionsUseCase
}

func NewGetAllPermissionsHandler(useCase *usecases.PermissionsUseCase) *GetAllPermissionsHandler {
	return &GetAllPermissionsHandler{useCase: useCase}
}

func (h *GetAllPermissionsHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor

		permissions, err := h.useCase.GetAllPermissions(ctx, actor)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, permissions)
	}
}

type GetPermissionByIDHandler struct {
	useCase *usecases.PermissionsUseCase
}

func NewGetPermissionByIDHandler(useCase *usecases.PermissionsUseCase) *GetPermissionByIDHandler {
	return &GetPermissionByIDHandler{useCase: useCase}
}

func (h *GetPermissionByIDHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor
		permissionID := r.PathValue("permission_id")

		permission, err := h.useCase.GetPermissionByID(ctx, actor, permissionID)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, permission)
	}
}

type GetPermissionByKeyHandler struct {
	useCase *usecases.PermissionsUseCase
}

func NewGetPermissionByKeyHandler(useCase *usecases.PermissionsUseCase) *GetPermissionByKeyHandler {
	return &GetPermissionByKeyHandler{useCase: useCase}
}

func (h *GetPermissionByKeyHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor
		permissionKey := r.PathValue("permission_key")

		permission, err := h.useCase.GetPermissionByKey(ctx, actor, permissionKey)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, permission)
	}
}

type UpdatePermissionHandler struct {
	useCase *usecases.PermissionsUseCase
}

func NewUpdatePermissionHandler(useCase *usecases.PermissionsUseCase) *UpdatePermissionHandler {
	return &UpdatePermissionHandler{useCase: useCase}
}

func (h *UpdatePermissionHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor
		permissionID := r.PathValue("permission_id")

		var request types.UpdatePermissionRequest
		if err := util.ParseJSON(r, &request); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		permission, err := h.useCase.UpdatePermission(ctx, actor, permissionID, request)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.UpdatePermissionResponse{
			Permission: permission,
		})
	}
}

type DeletePermissionHandler struct {
	useCase *usecases.PermissionsUseCase
}

func NewDeletePermissionHandler(useCase *usecases.PermissionsUseCase) *DeletePermissionHandler {
	return &DeletePermissionHandler{useCase: useCase}
}

func (h *DeletePermissionHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor
		permissionID := r.PathValue("permission_id")

		if err := h.useCase.DeletePermission(ctx, actor, permissionID); err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.DeletePermissionResponse{
			Message: "permission deleted",
		})
	}
}
