package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	accesscontrolconstants "github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

func TestCreateRoleHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid request body", func(t *testing.T) {
		t.Parallel()

		useCase, _ := tests.NewRolePermissionUseCaseFixture()
		handler := NewCreateRoleHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/access-control/roles", []byte("{invalid"))

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("CreateRole", mock.Anything, mock.AnythingOfType("*types.Role")).Return(accesscontrolconstants.ErrConflict).Once()
		handler := NewCreateRoleHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/access-control/roles", internaltests.MarshalToJSON(t, types.CreateRoleRequest{Name: "admin"}))

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusConflict, "conflict")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("CreateRole", mock.Anything, mock.AnythingOfType("*types.Role")).Return(nil).Once()
		handler := NewCreateRoleHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/access-control/roles", internaltests.MarshalToJSON(t, types.CreateRoleRequest{Name: "admin"}))

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.CreateRoleResponse](t, reqCtx)
		if payload.Role == nil {
			t.Fatal("expected role key, got nil")
		}
		repo.AssertExpectations(t)
	})
}

func TestGetAllRolesHandler(t *testing.T) {
	t.Parallel()

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetAllRoles", mock.Anything).Return(([]types.Role)(nil), errors.New("internal error")).Once()
		handler := NewGetAllRolesHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/access-control/roles", nil)

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusInternalServerError, "internal error")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetAllRoles", mock.Anything).Return([]types.Role{{ID: "role-1", Name: "admin"}}, nil).Once()
		handler := NewGetAllRolesHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/access-control/roles", nil)

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[[]types.Role](t, reqCtx)
		if payload == nil {
			t.Fatalf("expected roles key, got %v", payload)
		}
		repo.AssertExpectations(t)
	})
}

func TestGetRoleByIDHandler(t *testing.T) {
	t.Parallel()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), nil).Once()
		handler := NewGetRoleByIDHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/access-control/roles/role-1", nil)
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
		repo.On("GetRolePermissions", mock.Anything, "role-1").Return([]types.UserPermissionInfo{{PermissionID: "perm-1", PermissionKey: "users.read"}}, nil).Once()
		handler := NewGetRoleByIDHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/access-control/roles/role-1", nil)
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[*types.RoleDetails](t, reqCtx)
		if payload == nil {
			t.Fatalf("expected role details, got %v", payload)
		}
		repo.AssertExpectations(t)
	})
}

func TestUpdateRoleHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid payload", func(t *testing.T) {
		t.Parallel()

		useCase, _ := tests.NewRolePermissionUseCaseFixture()
		handler := NewUpdateRoleHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPatch, "/access-control/roles/role-1", []byte("{invalid"))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		name := "new-admin"
		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), nil).Once()
		handler := NewUpdateRoleHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPatch, "/access-control/roles/role-1", internaltests.MarshalToJSON(t, types.UpdateRoleRequest{Name: &name}))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		repo.AssertExpectations(t)
	})

	t.Run("cannot update system role", func(t *testing.T) {
		t.Parallel()

		name := "new-admin"
		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin", IsSystem: true}, nil).Once()
		handler := NewUpdateRoleHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPatch, "/access-control/roles/role-1", internaltests.MarshalToJSON(t, types.UpdateRoleRequest{Name: &name}))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusForbidden, "cannot update system role")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		name := "new-admin"
		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
		repo.On("UpdateRole", mock.Anything, "role-1", &name, (*string)(nil)).Return(true, nil).Once()
		repo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: name}, nil).Once()
		handler := NewUpdateRoleHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPatch, "/access-control/roles/role-1", internaltests.MarshalToJSON(t, types.UpdateRoleRequest{Name: &name}))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.UpdateRoleResponse](t, reqCtx)
		if payload.Role == nil {
			t.Fatalf("expected role key, got %v", payload)
		}
		repo.AssertExpectations(t)
	})
}

func TestDeleteRoleHandler(t *testing.T) {
	t.Parallel()

	t.Run("conflict when assigned", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
		repo.On("CountUserAssignmentsByRoleID", mock.Anything, "role-1").Return(1, nil).Once()
		handler := NewDeleteRoleHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/access-control/roles/role-1", nil)
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusConflict, "conflict")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
		repo.On("CountUserAssignmentsByRoleID", mock.Anything, "role-1").Return(0, nil).Once()
		repo.On("DeleteRole", mock.Anything, "role-1").Return(true, nil).Once()
		handler := NewDeleteRoleHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/access-control/roles/role-1", nil)
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.DeleteRoleResponse](t, reqCtx)
		if payload.Message != "deleted role" {
			t.Fatalf("expected deleted role message, got %v", payload.Message)
		}
		repo.AssertExpectations(t)
	})
}

func TestGetAllPermissionsHandler(t *testing.T) {
	t.Parallel()

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetAllPermissions", mock.Anything).Return(([]types.Permission)(nil), accesscontrolconstants.ErrForbidden).Once()
		handler := NewGetAllPermissionsHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/access-control/permissions", nil)

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetAllPermissions", mock.Anything).Return([]types.Permission{{ID: "perm-1", Key: "users.read"}}, nil).Once()
		handler := NewGetAllPermissionsHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/access-control/permissions", nil)

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[[]types.Permission](t, reqCtx)
		if payload == nil {
			t.Fatalf("expected permissions key, got %v", payload)
		}
		repo.AssertExpectations(t)
	})
}

func TestCreatePermissionHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid payload", func(t *testing.T) {
		t.Parallel()

		useCase, _ := tests.NewRolePermissionUseCaseFixture()
		handler := NewCreatePermissionHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/access-control/permissions", []byte("{invalid"))

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("CreatePermission", mock.Anything, mock.AnythingOfType("*types.Permission")).Return(accesscontrolconstants.ErrBadRequest).Once()
		handler := NewCreatePermissionHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/access-control/permissions", internaltests.MarshalToJSON(t, types.CreatePermissionRequest{Key: "users.read"}))

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "bad request")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("CreatePermission", mock.Anything, mock.AnythingOfType("*types.Permission")).Return(nil).Once()
		handler := NewCreatePermissionHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/access-control/permissions", internaltests.MarshalToJSON(t, types.CreatePermissionRequest{Key: "users.read"}))

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.CreatePermissionResponse](t, reqCtx)
		if payload.Permission == nil {
			t.Fatalf("expected permission key, got %v", payload)
		}
		repo.AssertExpectations(t)
	})
}

func TestUpdatePermissionHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid payload", func(t *testing.T) {
		t.Parallel()

		useCase, _ := tests.NewRolePermissionUseCaseFixture()
		handler := NewUpdatePermissionHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/access-control/permissions/perm-1", []byte("{invalid"))
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		desc := "updated"
		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), nil).Once()
		handler := NewUpdatePermissionHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/access-control/permissions/perm-1", internaltests.MarshalToJSON(t, types.UpdatePermissionRequest{Description: &desc}))
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		desc := "updated"
		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
		repo.On("UpdatePermission", mock.Anything, "perm-1", &desc).Return(true, nil).Once()
		repo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read", Description: &desc}, nil).Once()
		handler := NewUpdatePermissionHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/access-control/permissions/perm-1", internaltests.MarshalToJSON(t, types.UpdatePermissionRequest{Description: &desc}))
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[types.UpdatePermissionResponse](t, reqCtx)
		if payload.Permission == nil {
			t.Fatalf("expected permission key, got %v", payload)
		}
		repo.AssertExpectations(t)
	})
}

func TestDeletePermissionHandler(t *testing.T) {
	t.Parallel()

	t.Run("in use conflict", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
		repo.On("CountRoleAssignmentsByPermissionID", mock.Anything, "perm-1").Return(1, nil).Once()
		handler := NewDeletePermissionHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/access-control/permissions/perm-1", nil)
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusConflict, "conflict")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "admin.read"}, nil).Once()
		repo.On("CountRoleAssignmentsByPermissionID", mock.Anything, "perm-1").Return(0, nil).Once()
		repo.On("DeletePermission", mock.Anything, "perm-1").Return(true, nil).Once()
		handler := NewDeletePermissionHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/access-control/permissions/perm-1", nil)
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[map[string]any](t, reqCtx)
		if payload["message"] != "permission deleted" {
			t.Fatalf("expected permission deleted message, got %v", payload["message"])
		}
		repo.AssertExpectations(t)
	})
}

func TestAddRolePermissionHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid request body", func(t *testing.T) {
		t.Parallel()

		useCase, _ := tests.NewRolePermissionUseCaseFixture()
		handler := NewAddRolePermissionHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/access-control/roles/role-1/permissions", []byte("{invalid"))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), nil).Once()
		handler := NewAddRolePermissionHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/access-control/roles/role-1/permissions", internaltests.MarshalToJSON(t, types.AddRolePermissionRequest{PermissionID: "perm-1"}))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		actorID := "actor-1"
		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
		repo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
		repo.On("AddRolePermission", mock.Anything, "role-1", "perm-1", &actorID).Return(nil).Once()
		handler := NewAddRolePermissionHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/access-control/roles/role-1/permissions", internaltests.MarshalToJSON(t, types.AddRolePermissionRequest{PermissionID: "perm-1"}))
		req.SetPathValue("role_id", "role-1")
		reqCtx.UserID = &actorID

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[map[string]any](t, reqCtx)
		if payload["message"] != "permission assigned to role" {
			t.Fatalf("expected permission assigned to role message, got %v", payload["message"])
		}
		repo.AssertExpectations(t)
	})
}

func TestReplaceRolePermissionsHandler(t *testing.T) {
	t.Parallel()

	t.Run("invalid payload", func(t *testing.T) {
		t.Parallel()

		useCase, _ := tests.NewRolePermissionUseCaseFixture()
		handler := NewReplaceRolePermissionsHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/access-control/roles/role-1/permissions", []byte("{invalid"))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
	})

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		actorID := "actor-1"
		request := types.ReplaceRolePermissionsRequest{PermissionIDs: []string{"perm-1"}}
		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("ReplaceRolePermissions", mock.Anything, "role-1", request.PermissionIDs, &actorID).Return(accesscontrolconstants.ErrForbidden).Once()
		handler := NewReplaceRolePermissionsHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/access-control/roles/role-1/permissions", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("role_id", "role-1")
		reqCtx.UserID = &actorID

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusForbidden, "forbidden")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		request := types.ReplaceRolePermissionsRequest{PermissionIDs: []string{"perm-1", "perm-1", "perm-2"}}
		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("ReplaceRolePermissions", mock.Anything, "role-1", []string{"perm-1", "perm-2"}, (*string)(nil)).Return(nil).Once()
		handler := NewReplaceRolePermissionsHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/access-control/roles/role-1/permissions", internaltests.MarshalToJSON(t, request))
		req.SetPathValue("role_id", "role-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[map[string]any](t, reqCtx)
		if payload["message"] != "role permissions replaced" {
			t.Fatalf("expected role permissions replaced message, got %v", payload["message"])
		}
		repo.AssertExpectations(t)
	})
}

func TestRemoveRolePermissionHandler(t *testing.T) {
	t.Parallel()

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
		repo.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), nil).Once()
		handler := NewRemoveRolePermissionHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/access-control/roles/role-1/permissions/perm-1", nil)
		req.SetPathValue("role_id", "role-1")
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		internaltests.AssertErrorMessage(t, reqCtx, http.StatusNotFound, "not found")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := tests.NewRolePermissionUseCaseFixture()
		repo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
		repo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "admin.read"}, nil).Once()
		repo.On("RemoveRolePermission", mock.Anything, "role-1", "perm-1").Return(nil).Once()
		handler := NewRemoveRolePermissionHandler(useCase)
		req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/access-control/roles/role-1/permissions/perm-1", nil)
		req.SetPathValue("role_id", "role-1")
		req.SetPathValue("permission_id", "perm-1")

		handler.Handler()(w, req)

		if reqCtx.ResponseStatus != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, reqCtx.ResponseStatus)
		}
		payload := internaltests.DecodeResponseJSON[map[string]any](t, reqCtx)
		if payload["message"] != "permission removed from role" {
			t.Fatalf("expected permission removed from role message, got %v", payload["message"])
		}
		repo.AssertExpectations(t)
	})
}
