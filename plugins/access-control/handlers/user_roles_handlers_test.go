package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	accesscontrolconstants "github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

func TestGetUserRolesHandler(t *testing.T) {
	t.Parallel()

	t.Run("missing user id", func(t *testing.T) {
		t.Parallel()

		useCase, _ := tests.NewUserRolesUseCaseFixture()
		handler := NewGetUserRolesHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/access-control/users//roles", nil)
		req.SetPathValue("user_id", "   ")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "unprocessable entity")
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, accessRepo := tests.NewUserRolesUseCaseFixture()
		accessRepo.On("GetUserRoles", mock.Anything, "user-1").Return(([]types.UserRoleInfo)(nil), accesscontrolconstants.ErrNotFound).Once()
		handler := NewGetUserRolesHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/access-control/users/user-1/roles", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		accessRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, accessRepo := tests.NewUserRolesUseCaseFixture()
		expiresAt := time.Now().UTC().Add(time.Hour)
		accessRepo.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{{RoleID: "role-1", RoleName: "admin", ExpiresAt: &expiresAt}}, nil).Once()
		handler := NewGetUserRolesHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/access-control/users/user-1/roles", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[[]types.UserRoleInfo](t, reqCtx)
		if len(payload) != 1 {
			t.Fatalf("expected 1 role, got %d", len(payload))
		}
		accessRepo.AssertExpectations(t)
	})
}

func TestReplaceUserRolesHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		useCase, _ := tests.NewRolePermissionUseCaseFixture()
		handler := NewReplaceUserRolesHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/access-control/users/user-1/roles", []byte("{invalid"))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, roleRepo := tests.NewRolePermissionUseCaseFixture()
		request := types.ReplaceUserRolesRequest{RoleIDs: []string{"role-1"}}
		actorID := "actor-1"
		roleRepo.On("ReplaceUserRoles", mock.Anything, "user-1", request.RoleIDs, &actorID).Return(accesscontrolconstants.ErrForbidden).Once()
		handler := NewReplaceUserRolesHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/access-control/users/user-1/roles", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("user_id", "user-1")
		reqCtx.UserID = &actorID

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		roleRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, roleRepo := tests.NewRolePermissionUseCaseFixture()
		request := types.ReplaceUserRolesRequest{RoleIDs: []string{"role-1", "role-2"}}
		roleRepo.On("ReplaceUserRoles", mock.Anything, "user-1", request.RoleIDs, (*string)(nil)).Return(nil).Once()
		handler := NewReplaceUserRolesHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/access-control/users/user-1/roles", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.ReplaceUserRolesResponse](t, reqCtx)
		if payload.Message != "user roles replaced" {
			t.Fatalf("expected message user roles replaced, got %v", payload.Message)
		}
		roleRepo.AssertExpectations(t)
	})
}

func TestAssignUserRoleHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		useCase, _ := tests.NewRolePermissionUseCaseFixture()
		handler := NewAssignUserRoleHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/access-control/users/user-1/roles", []byte("{invalid"))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, roleRepo := tests.NewRolePermissionUseCaseFixture()
		request := types.AssignUserRoleRequest{RoleID: "role-1"}
		actorID := "actor-1"
		roleRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", &actorID, (*time.Time)(nil)).Return(accesscontrolconstants.ErrBadRequest).Once()
		handler := NewAssignUserRoleHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/access-control/users/user-1/roles", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("user_id", "user-1")
		reqCtx.UserID = &actorID

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "bad request")
		roleRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, roleRepo := tests.NewRolePermissionUseCaseFixture()
		request := types.AssignUserRoleRequest{RoleID: "role-1"}
		roleRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", (*string)(nil), (*time.Time)(nil)).Return(nil).Once()
		handler := NewAssignUserRoleHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/access-control/users/user-1/roles", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.AssignUserRoleResponse](t, reqCtx)
		if payload.Message != "role assigned" {
			t.Fatalf("expected role assigned message, got %v", payload.Message)
		}
		roleRepo.AssertExpectations(t)
	})
}

func TestRemoveUserRoleHandler(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, roleRepo := tests.NewRolePermissionUseCaseFixture()
		roleRepo.On("RemoveUserRole", mock.Anything, "user-1", "role-1").Return(accesscontrolconstants.ErrNotFound).Once()
		handler := NewRemoveUserRoleHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/access-control/users/user-1/roles/role-1", nil)
		req.SetPathValue("user_id", "user-1")
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		roleRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, roleRepo := tests.NewRolePermissionUseCaseFixture()
		roleRepo.On("RemoveUserRole", mock.Anything, "user-1", "role-1").Return(nil).Once()
		handler := NewRemoveUserRoleHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/access-control/users/user-1/roles/role-1", nil)
		req.SetPathValue("user_id", "user-1")
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.RemoveUserRoleResponse](t, reqCtx)
		if payload.Message != "role removed" {
			t.Fatalf("expected role removed message, got %v", payload.Message)
		}
		roleRepo.AssertExpectations(t)
	})
}

func TestGetUserEffectivePermissionsHandler(t *testing.T) {
	t.Parallel()

	t.Run("missing user id", func(t *testing.T) {
		t.Parallel()

		useCase, _ := tests.NewUserRolesUseCaseFixture()
		handler := NewGetUserEffectivePermissionsHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/access-control/users//permissions", nil)
		req.SetPathValue("user_id", "")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "unprocessable entity")
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		useCase, accessRepo := tests.NewUserRolesUseCaseFixture()
		accessRepo.On("GetUserEffectivePermissions", mock.Anything, "user-1").Return(([]types.UserPermissionInfo)(nil), accesscontrolconstants.ErrUnauthorized).Once()
		handler := NewGetUserEffectivePermissionsHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/access-control/users/user-1/permissions", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnauthorized, "unauthorized")
		accessRepo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, accessRepo := tests.NewUserRolesUseCaseFixture()
		accessRepo.On("GetUserEffectivePermissions", mock.Anything, "user-1").Return([]types.UserPermissionInfo{{PermissionID: "perm-1", PermissionKey: "admin.read"}}, nil).Once()
		handler := NewGetUserEffectivePermissionsHandler(useCase)

		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/access-control/users/user-1/permissions", nil)
		req.SetPathValue("user_id", "user-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.GetUserEffectivePermissionsResponse](t, reqCtx)
		if len(payload.Permissions) != 1 {
			t.Fatalf("expected 1 permission, got %d", len(payload.Permissions))
		}
		accessRepo.AssertExpectations(t)
	})
}
