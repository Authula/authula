package handlers

import (
	"net/http"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

type GetAllPermissionsHandler struct {
	useCase usecases.RolePermissionUseCase
}

func NewGetAllPermissionsHandler(useCase usecases.RolePermissionUseCase) *GetAllPermissionsHandler {
	return &GetAllPermissionsHandler{useCase: useCase}
}

func (h *GetAllPermissionsHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		permissions, err := h.useCase.GetAllPermissions(r.Context())
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"data": permissions})
	}
}

type CreatePermissionHandler struct {
	useCase usecases.RolePermissionUseCase
}

func NewCreatePermissionHandler(useCase usecases.RolePermissionUseCase) *CreatePermissionHandler {
	return &CreatePermissionHandler{useCase: useCase}
}

func (h *CreatePermissionHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var payload types.CreatePermissionRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		permission, err := h.useCase.CreatePermission(r.Context(), payload)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusCreated, map[string]any{"message": "permission created", "data": permission})
	}
}

type UpdatePermissionHandler struct {
	useCase usecases.RolePermissionUseCase
}

func NewUpdatePermissionHandler(useCase usecases.RolePermissionUseCase) *UpdatePermissionHandler {
	return &UpdatePermissionHandler{useCase: useCase}
}

func (h *UpdatePermissionHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		permissionID := r.PathValue("permission_id")

		var payload types.UpdatePermissionRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		permission, err := h.useCase.UpdatePermission(r.Context(), permissionID, payload)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"message": "permission updated", "data": permission})
	}
}

type DeletePermissionHandler struct {
	useCase usecases.RolePermissionUseCase
}

func NewDeletePermissionHandler(useCase usecases.RolePermissionUseCase) *DeletePermissionHandler {
	return &DeletePermissionHandler{useCase: useCase}
}

func (h *DeletePermissionHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		permissionID := r.PathValue("permission_id")

		if err := h.useCase.DeletePermission(r.Context(), permissionID); err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"message": "permission deleted"})
	}
}

type GetAllRolesHandler struct {
	useCase usecases.RolePermissionUseCase
}

func NewGetAllRolesHandler(useCase usecases.RolePermissionUseCase) *GetAllRolesHandler {
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

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"data": roles})
	}
}

type GetRoleByIDHandler struct {
	useCase usecases.RolePermissionUseCase
}

func NewGetRoleByIDHandler(useCase usecases.RolePermissionUseCase) *GetRoleByIDHandler {
	return &GetRoleByIDHandler{useCase: useCase}
}

func (h *GetRoleByIDHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		roleID := r.PathValue("role_id")

		role, err := h.useCase.GetRoleByID(r.Context(), roleID)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"data": role})
	}
}

type CreateRoleHandler struct {
	useCase usecases.RolePermissionUseCase
}

func NewCreateRoleHandler(useCase usecases.RolePermissionUseCase) *CreateRoleHandler {
	return &CreateRoleHandler{useCase: useCase}
}

func (h *CreateRoleHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		var payload types.CreateRoleRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		role, err := h.useCase.CreateRole(r.Context(), payload)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusCreated, map[string]any{"message": "role created", "data": role})
	}
}

type UpdateRoleHandler struct {
	useCase usecases.RolePermissionUseCase
}

func NewUpdateRoleHandler(useCase usecases.RolePermissionUseCase) *UpdateRoleHandler {
	return &UpdateRoleHandler{useCase: useCase}
}

func (h *UpdateRoleHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		roleID := r.PathValue("role_id")

		var payload types.UpdateRoleRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		role, err := h.useCase.UpdateRole(r.Context(), roleID, payload)
		if err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"message": "role updated", "data": role})
	}
}

type DeleteRoleHandler struct {
	useCase usecases.RolePermissionUseCase
}

func NewDeleteRoleHandler(useCase usecases.RolePermissionUseCase) *DeleteRoleHandler {
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

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"message": "role deleted"})
	}
}

type ReplaceRolePermissionsHandler struct {
	useCase usecases.RolePermissionUseCase
}

func NewReplaceRolePermissionsHandler(useCase usecases.RolePermissionUseCase) *ReplaceRolePermissionsHandler {
	return &ReplaceRolePermissionsHandler{useCase: useCase}
}

func (h *ReplaceRolePermissionsHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		roleID := r.PathValue("role_id")

		var payload types.ReplaceRolePermissionsRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{"message": "invalid request body"})
			reqCtx.Handled = true
			return
		}

		if err := h.useCase.ReplaceRolePermissions(r.Context(), roleID, payload.PermissionIDs, rolePermissionActorUserID(reqCtx)); err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"message": "role permissions replaced"})
	}
}

type AddRolePermissionHandler struct {
	useCase usecases.RolePermissionUseCase
}

func NewAddRolePermissionHandler(useCase usecases.RolePermissionUseCase) *AddRolePermissionHandler {
	return &AddRolePermissionHandler{useCase: useCase}
}

func (h *AddRolePermissionHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		roleID := r.PathValue("role_id")
		permissionID := r.URL.Query().Get("permission_id")

		if permissionID == "" {
			var payload struct {
				PermissionID string `json:"permission_id"`
			}
			if err := util.ParseJSON(r, &payload); err == nil {
				permissionID = payload.PermissionID
			}
		}

		if err := h.useCase.AddPermissionToRole(r.Context(), roleID, permissionID, rolePermissionActorUserID(reqCtx)); err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"message": "permission assigned to role"})
	}
}

type RemoveRolePermissionHandler struct {
	useCase usecases.RolePermissionUseCase
}

func NewRemoveRolePermissionHandler(useCase usecases.RolePermissionUseCase) *RemoveRolePermissionHandler {
	return &RemoveRolePermissionHandler{useCase: useCase}
}

func (h *RemoveRolePermissionHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)
		roleID := r.PathValue("role_id")
		permissionID := r.PathValue("permission_id")

		if err := h.useCase.RemovePermissionFromRole(r.Context(), roleID, permissionID); err != nil {
			respondRolePermissionError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{"message": "permission removed from role"})
	}
}

func rolePermissionActorUserID(reqCtx *models.RequestContext) *string {
	if reqCtx == nil || reqCtx.UserID == nil || *reqCtx.UserID == "" {
		return nil
	}
	return reqCtx.UserID
}

func respondRolePermissionError(reqCtx *models.RequestContext, err error) {
	if reqCtx == nil {
		return
	}

	message := "internal server error"
	if err != nil {
		message = err.Error()
	}

	reqCtx.SetJSONResponse(mapRolePermissionErrorStatus(err), map[string]any{"message": message})
	reqCtx.Handled = true
}

func mapRolePermissionErrorStatus(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	message := strings.ToLower(strings.TrimSpace(err.Error()))

	switch {
	case strings.Contains(message, "unauthorized"):
		return http.StatusUnauthorized
	case strings.Contains(message, "forbidden"):
		return http.StatusForbidden
	case strings.Contains(message, "not found"):
		return http.StatusNotFound
	case strings.Contains(message, "in use"), strings.Contains(message, "conflict"):
		return http.StatusConflict
	case rolePermissionIsBadRequestMessage(message):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func rolePermissionIsBadRequestMessage(message string) bool {
	markers := []string{"required", "invalid", "cannot", "exceeds", "you can only", "no active"}
	for _, marker := range markers {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}
