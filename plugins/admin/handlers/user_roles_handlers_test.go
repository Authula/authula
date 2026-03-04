package handlers

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestGetUserRolesHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockUserRolesUseCase{}
		useCase.On("GetUserRoles", mock.Anything, "user-1").Return(nil, errors.New("not found")).Once()
		handler := NewGetUserRolesHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/users/user-1/roles", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockUserRolesUseCase{}
		expiresAt := time.Now().UTC().Add(time.Hour)
		useCase.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{{RoleID: "role-1", RoleName: "admin", ExpiresAt: &expiresAt}}, nil).Once()
		handler := NewGetUserRolesHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/users/user-1/roles", nil)
		req.SetPathValue("user_id", "user-1")

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

func TestReplaceUserRolesHandler(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		handler := NewReplaceUserRolesHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPut, "/admin/users/user-1/roles", []byte("{invalid"))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
		useCase.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		request := types.ReplaceUserRolesRequest{RoleIDs: []string{"role-1"}}
		actorID := "actor-1"
		useCase.On("ReplaceUserRoles", mock.Anything, "user-1", request.RoleIDs, &actorID).Return(errors.New("forbidden")).Once()
		handler := NewReplaceUserRolesHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPut, "/admin/users/user-1/roles", mustJSON(t, request))
		req.SetPathValue("user_id", "user-1")
		reqCtx.UserID = &actorID

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		request := types.ReplaceUserRolesRequest{RoleIDs: []string{"role-1", "role-2"}}
		useCase.On("ReplaceUserRoles", mock.Anything, "user-1", request.RoleIDs, (*string)(nil)).Return(nil).Once()
		handler := NewReplaceUserRolesHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPut, "/admin/users/user-1/roles", mustJSON(t, request))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "user roles replaced" {
			t.Fatalf("expected message user roles replaced, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}

func TestAssignUserRoleHandler(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		handler := NewAssignUserRoleHandler(useCase)
		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/users/user-1/roles", []byte("{invalid"))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
		useCase.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		request := types.AssignUserRoleRequest{RoleID: "role-1"}
		actorID := "actor-1"
		useCase.On("AssignRoleToUser", mock.Anything, "user-1", request, &actorID).Return(errors.New("required field")).Once()
		handler := NewAssignUserRoleHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/users/user-1/roles", mustJSON(t, request))
		req.SetPathValue("user_id", "user-1")
		reqCtx.UserID = &actorID

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusBadRequest, "required field")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		request := types.AssignUserRoleRequest{RoleID: "role-1"}
		useCase.On("AssignRoleToUser", mock.Anything, "user-1", request, (*string)(nil)).Return(nil).Once()
		handler := NewAssignUserRoleHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodPost, "/admin/users/user-1/roles", mustJSON(t, request))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "role assigned" {
			t.Fatalf("expected role assigned message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}

func TestRemoveUserRoleHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("RemoveRoleFromUser", mock.Anything, "user-1", "role-1").Return(errors.New("not found")).Once()
		handler := NewRemoveUserRoleHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/users/user-1/roles/role-1", nil)
		req.SetPathValue("user_id", "user-1")
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockRolePermissionUseCase{}
		useCase.On("RemoveRoleFromUser", mock.Anything, "user-1", "role-1").Return(nil).Once()
		handler := NewRemoveUserRoleHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodDelete, "/admin/users/user-1/roles/role-1", nil)
		req.SetPathValue("user_id", "user-1")
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := decodeResponseJSON(t, reqCtx)
		if payload["message"] != "role removed" {
			t.Fatalf("expected role removed message, got %v", payload["message"])
		}
		useCase.AssertExpectations(t)
	})
}

func TestGetUserEffectivePermissionsHandler(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		useCase := &mockUserRolesUseCase{}
		useCase.On("GetUserEffectivePermissions", mock.Anything, "user-1").Return(nil, errors.New("unauthorized")).Once()
		handler := NewGetUserEffectivePermissionsHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/users/user-1/permissions", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		assertErrorMessage(t, reqCtx, http.StatusUnauthorized, "unauthorized")
		useCase.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		useCase := &mockUserRolesUseCase{}
		useCase.On("GetUserEffectivePermissions", mock.Anything, "user-1").Return([]types.UserPermissionInfo{{PermissionID: "perm-1", PermissionKey: "admin.read"}}, nil).Once()
		handler := NewGetUserEffectivePermissionsHandler(useCase)

		req, w, reqCtx := newAdminHandlerRequest(t, http.MethodGet, "/admin/users/user-1/permissions", nil)
		req.SetPathValue("user_id", "user-1")

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
