package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
)

type CreateRoleHandler struct {
	useCase *usecases.RolesUseCase
}

func NewCreateRoleHandler(useCase *usecases.RolesUseCase) *CreateRoleHandler {
	return &CreateRoleHandler{useCase: useCase}
}

func (h *CreateRoleHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var payload types.CreateRoleRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		role, err := h.useCase.CreateRole(r.Context(), payload)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusCreated, &types.CreateRoleResponse{
			Role: role,
		})
	}
}

type GetAllRolesHandler struct {
	useCase *usecases.RolesUseCase
}

func NewGetAllRolesHandler(useCase *usecases.RolesUseCase) *GetAllRolesHandler {
	return &GetAllRolesHandler{useCase: useCase}
}

func (h *GetAllRolesHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		roles, err := h.useCase.GetAllRoles(r.Context())
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, roles)
	}
}

type GetRoleByIDHandler struct {
	useCase *usecases.RolesUseCase
}

func NewGetRoleByIDHandler(useCase *usecases.RolesUseCase) *GetRoleByIDHandler {
	return &GetRoleByIDHandler{useCase: useCase}
}

func (h *GetRoleByIDHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		roleID := r.PathValue("role_id")

		roleDetails, err := h.useCase.GetRoleByID(r.Context(), roleID)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, roleDetails)
	}
}

type UpdateRoleHandler struct {
	useCase *usecases.RolesUseCase
}

func NewUpdateRoleHandler(useCase *usecases.RolesUseCase) *UpdateRoleHandler {
	return &UpdateRoleHandler{useCase: useCase}
}

func (h *UpdateRoleHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		roleID := r.PathValue("role_id")

		var payload types.UpdateRoleRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		role, err := h.useCase.UpdateRole(r.Context(), roleID, payload)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.UpdateRoleResponse{
			Role: role,
		})
	}
}

type DeleteRoleHandler struct {
	useCase *usecases.RolesUseCase
}

func NewDeleteRoleHandler(useCase *usecases.RolesUseCase) *DeleteRoleHandler {
	return &DeleteRoleHandler{useCase: useCase}
}

func (h *DeleteRoleHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		roleID := r.PathValue("role_id")

		if err := h.useCase.DeleteRole(r.Context(), roleID); err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.DeleteRoleResponse{
			Message: "deleted role",
		})
	}
}
