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
		actor := reqCtx.Actor

		var request types.CreateRoleRequest
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

		role, err := h.useCase.CreateRole(ctx, actor, request)
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
		actor := reqCtx.Actor

		roles, err := h.useCase.GetAllRoles(ctx, actor)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, roles)
	}
}

type GetRoleByNameHandler struct {
	useCase *usecases.RolesUseCase
}

func NewGetRoleByNameHandler(useCase *usecases.RolesUseCase) *GetRoleByNameHandler {
	return &GetRoleByNameHandler{useCase: useCase}
}

func (h *GetRoleByNameHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		actor := reqCtx.Actor
		roleName := r.PathValue("role_name")

		role, err := h.useCase.GetRoleByName(ctx, actor, roleName)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, role)
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
		actor := reqCtx.Actor
		roleID := r.PathValue("role_id")

		roleDetails, err := h.useCase.GetRoleByID(ctx, actor, roleID)
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
		actor := reqCtx.Actor
		roleID := r.PathValue("role_id")

		var request types.UpdateRoleRequest
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

		role, err := h.useCase.UpdateRole(ctx, actor, roleID, request)
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
		actor := reqCtx.Actor
		roleID := r.PathValue("role_id")

		if err := h.useCase.DeleteRole(ctx, actor, roleID); err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.DeleteRoleResponse{
			Message: "deleted role",
		})
	}
}
