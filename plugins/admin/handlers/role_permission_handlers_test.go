package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestGetAllRolesHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("GetAllRoles", mock.Anything).Return(nil, errors.New("internal error")).Once()
		handler := NewGetAllRolesHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/roles", nil)

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusInternalServerError, "internal error")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("GetAllRoles", mock.Anything).Return([]types.Role{{ID: "role-1", Name: "admin"}}, nil).Once()
		handler := NewGetAllRolesHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/roles", nil)

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["roles"]; !ok {
			t.Fatalf("expected roles key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestGetRoleByIDHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("GetRoleByID", mock.Anything, "role-1").Return(nil, errors.New("not found")).Once()
		handler := NewGetRoleByIDHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/roles/role-1", nil)
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("GetRoleByID", mock.Anything, "role-1").Return(&types.RoleDetails{Role: types.Role{ID: "role-1", Name: "admin"}}, nil).Once()
		handler := NewGetRoleByIDHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/roles/role-1", nil)
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestCreateRoleHandler(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		handler := NewCreateRoleHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/roles", []byte("{invalid"))

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		request := types.CreateRoleRequest{Name: "admin"}
		useCase.On("CreateRole", mock.Anything, request).Return(nil, errors.New("invalid role name")).Once()
		handler := NewCreateRoleHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/roles", mustJSON(t, request))

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "invalid role name")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		request := types.CreateRoleRequest{Name: "admin"}
		useCase.On("CreateRole", mock.Anything, request).Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
		handler := NewCreateRoleHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/roles", mustJSON(t, request))

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "role created" {
			t.Fatalf("expected role created message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}

func TestUpdateRoleHandler(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		handler := NewUpdateRoleHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPatch, "/admin/roles/role-1", []byte("{invalid"))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		name := "ops"
		request := types.UpdateRoleRequest{Name: &name}
		useCase.On("UpdateRole", mock.Anything, "role-1", request).Return(nil, errors.New("forbidden")).Once()
		handler := NewUpdateRoleHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPatch, "/admin/roles/role-1", mustJSON(t, request))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		name := "ops"
		request := types.UpdateRoleRequest{Name: &name}
		useCase.On("UpdateRole", mock.Anything, "role-1", request).Return(&types.Role{ID: "role-1", Name: name}, nil).Once()
		handler := NewUpdateRoleHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPatch, "/admin/roles/role-1", mustJSON(t, request))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "role updated" {
			t.Fatalf("expected role updated message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}

func TestDeleteRoleHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("DeleteRole", mock.Anything, "role-1").Return(errors.New("in use")).Once()
		handler := NewDeleteRoleHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/roles/role-1", nil)
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusConflict, "in use")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("DeleteRole", mock.Anything, "role-1").Return(nil).Once()
		handler := NewDeleteRoleHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/roles/role-1", nil)
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "role deleted" {
			t.Fatalf("expected role deleted message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}

func TestGetAllPermissionsHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("GetAllPermissions", mock.Anything).Return(nil, errors.New("unauthorized")).Once()
		handler := NewGetAllPermissionsHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/permissions", nil)

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusUnauthorized, "unauthorized")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("GetAllPermissions", mock.Anything).Return([]types.Permission{{ID: "perm-1", Key: "admin.read"}}, nil).Once()
		handler := NewGetAllPermissionsHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/permissions", nil)

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestCreatePermissionHandler(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		handler := NewCreatePermissionHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/permissions", []byte("{invalid"))

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		request := types.CreatePermissionRequest{Key: "admin.read"}
		useCase.On("CreatePermission", mock.Anything, request).Return(nil, errors.New("invalid permission key")).Once()
		handler := NewCreatePermissionHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/permissions", mustJSON(t, request))

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "invalid permission key")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		request := types.CreatePermissionRequest{Key: "admin.read"}
		useCase.On("CreatePermission", mock.Anything, request).Return(&types.Permission{ID: "perm-1", Key: "admin.read"}, nil).Once()
		handler := NewCreatePermissionHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/permissions", mustJSON(t, request))

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if _, ok := payload["data"]; !ok {
			t.Fatalf("expected data key, got %v", payload)
		}
		useCase.AssertExpectations(t)
	})
}

func TestUpdatePermissionHandler(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		handler := NewUpdatePermissionHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPatch, "/admin/permissions/perm-1", []byte("{invalid"))
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		desc := "updated"
		request := types.UpdatePermissionRequest{Description: &desc}
		useCase.On("UpdatePermission", mock.Anything, "perm-1", request).Return(nil, errors.New("forbidden")).Once()
		handler := NewUpdatePermissionHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPatch, "/admin/permissions/perm-1", mustJSON(t, request))
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		desc := "updated"
		request := types.UpdatePermissionRequest{Description: &desc}
		useCase.On("UpdatePermission", mock.Anything, "perm-1", request).Return(&types.Permission{ID: "perm-1", Key: "admin.read", Description: &desc}, nil).Once()
		handler := NewUpdatePermissionHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPatch, "/admin/permissions/perm-1", mustJSON(t, request))
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "permission updated" {
			t.Fatalf("expected permission updated message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}

func TestDeletePermissionHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("DeletePermission", mock.Anything, "perm-1").Return(errors.New("not found")).Once()
		handler := NewDeletePermissionHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/permissions/perm-1", nil)
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("DeletePermission", mock.Anything, "perm-1").Return(nil).Once()
		handler := NewDeletePermissionHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/permissions/perm-1", nil)
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "permission deleted" {
			t.Fatalf("expected permission deleted message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}

func TestAddRolePermissionHandler(t *testing.T) {
	t.Run("uses query param", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		actorID := "actor-1"
		useCase.On("AddPermissionToRole", mock.Anything, "role-1", "perm-1", &actorID).Return(nil).Once()
		handler := NewAddRolePermissionHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/roles/role-1/permissions?permission_id=perm-1", nil)
		req.SetPathValue("role_id", "role-1")
		reqCtx.UserID = &actorID

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "permission assigned to role" {
			t.Fatalf("expected success message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})

	t.Run("falls back to json body", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("AddPermissionToRole", mock.Anything, "role-1", "perm-2", (*string)(nil)).Return(nil).Once()
		handler := NewAddRolePermissionHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/roles/role-1/permissions", mustJSON(t, map[string]string{"permission_id": "perm-2"}))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		useCase.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("AddPermissionToRole", mock.Anything, "role-1", "perm-1", (*string)(nil)).Return(errors.New("forbidden")).Once()
		handler := NewAddRolePermissionHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/roles/role-1/permissions?permission_id=perm-1", nil)
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		useCase.AssertExpectations(t)
	})
}

func TestReplaceRolePermissionsHandler(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		handler := NewReplaceRolePermissionsHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPut, "/admin/roles/role-1/permissions", []byte("{invalid"))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		actorID := "actor-1"
		request := types.ReplaceRolePermissionsRequest{PermissionIDs: []string{"perm-1", "perm-2"}}
		useCase.On("ReplaceRolePermissions", mock.Anything, "role-1", request.PermissionIDs, &actorID).Return(errors.New("cannot replace")).Once()
		handler := NewReplaceRolePermissionsHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPut, "/admin/roles/role-1/permissions", mustJSON(t, request))
		req.SetPathValue("role_id", "role-1")
		reqCtx.UserID = &actorID

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "cannot replace")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		request := types.ReplaceRolePermissionsRequest{PermissionIDs: []string{"perm-1"}}
		useCase.On("ReplaceRolePermissions", mock.Anything, "role-1", request.PermissionIDs, (*string)(nil)).Return(nil).Once()
		handler := NewReplaceRolePermissionsHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPut, "/admin/roles/role-1/permissions", mustJSON(t, request))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "role permissions replaced" {
			t.Fatalf("expected role permissions replaced message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}

func TestRemoveRolePermissionHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("RemovePermissionFromRole", mock.Anything, "role-1", "perm-1").Return(errors.New("not found")).Once()
		handler := NewRemoveRolePermissionHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/roles/role-1/permissions/perm-1", nil)
		req.SetPathValue("role_id", "role-1")
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("RemovePermissionFromRole", mock.Anything, "role-1", "perm-1").Return(nil).Once()
		handler := NewRemoveRolePermissionHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/roles/role-1/permissions/perm-1", nil)
		req.SetPathValue("role_id", "role-1")
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "permission removed from role" {
			t.Fatalf("expected permission removed message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}
