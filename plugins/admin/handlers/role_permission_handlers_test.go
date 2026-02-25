package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type rolePermissionUseCaseStub struct {
	createPermissionFn         func(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error)
	getAllPermissionsFn        func(ctx context.Context) ([]types.Permission, error)
	updatePermissionFn         func(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error)
	deletePermissionFn         func(ctx context.Context, permissionID string) error
	createRoleFn               func(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error)
	getAllRolesFn              func(ctx context.Context) ([]types.Role, error)
	getRoleByIDFn              func(ctx context.Context, roleID string) (*types.RoleDetails, error)
	updateRoleFn               func(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error)
	deleteRoleFn               func(ctx context.Context, roleID string) error
	addPermissionToRoleFn      func(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error
	removePermissionFromRoleFn func(ctx context.Context, roleID string, permissionID string) error
	replaceRolePermissionsFn   func(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error
	assignRoleToUserFn         func(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error
	removeRoleFromUserFn       func(ctx context.Context, userID string, roleID string) error
	replaceUserRolesFn         func(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error
}

func (s rolePermissionUseCaseStub) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	return s.createPermissionFn(ctx, req)
}

func (s rolePermissionUseCaseStub) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return s.getAllPermissionsFn(ctx)
}

func (s rolePermissionUseCaseStub) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return s.updatePermissionFn(ctx, permissionID, req)
}

func (s rolePermissionUseCaseStub) DeletePermission(ctx context.Context, permissionID string) error {
	return s.deletePermissionFn(ctx, permissionID)
}

func (s rolePermissionUseCaseStub) CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) {
	return s.createRoleFn(ctx, req)
}

func (s rolePermissionUseCaseStub) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return s.getAllRolesFn(ctx)
}

func (s rolePermissionUseCaseStub) GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error) {
	return s.getRoleByIDFn(ctx, roleID)
}

func (s rolePermissionUseCaseStub) UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	return s.updateRoleFn(ctx, roleID, req)
}

func (s rolePermissionUseCaseStub) DeleteRole(ctx context.Context, roleID string) error {
	return s.deleteRoleFn(ctx, roleID)
}

func (s rolePermissionUseCaseStub) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	return s.addPermissionToRoleFn(ctx, roleID, permissionID, grantedByUserID)
}

func (s rolePermissionUseCaseStub) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	return s.removePermissionFromRoleFn(ctx, roleID, permissionID)
}

func (s rolePermissionUseCaseStub) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return s.replaceRolePermissionsFn(ctx, roleID, permissionIDs, grantedByUserID)
}

func (s rolePermissionUseCaseStub) AssignRoleToUser(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	return s.assignRoleToUserFn(ctx, userID, req, assignedByUserID)
}

func (s rolePermissionUseCaseStub) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	return s.removeRoleFromUserFn(ctx, userID, roleID)
}

func (s rolePermissionUseCaseStub) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	return s.replaceUserRolesFn(ctx, userID, roleIDs, assignedByUserID)
}

func TestUpdatePermissionHandler_BadJSON(t *testing.T) {
	stub := rolePermissionUseCaseStub{
		createPermissionFn: func(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
			return nil, nil
		},
		getAllPermissionsFn: func(ctx context.Context) ([]types.Permission, error) { return nil, nil },
		updatePermissionFn: func(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
			return nil, nil
		},
		deletePermissionFn: func(ctx context.Context, permissionID string) error { return nil },
		createRoleFn:       func(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) { return nil, nil },
		getAllRolesFn:      func(ctx context.Context) ([]types.Role, error) { return nil, nil },
		getRoleByIDFn:      func(ctx context.Context, roleID string) (*types.RoleDetails, error) { return nil, nil },
		updateRoleFn: func(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
			return nil, nil
		},
		deleteRoleFn: func(ctx context.Context, roleID string) error { return nil },
		addPermissionToRoleFn: func(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
			return nil
		},
		removePermissionFromRoleFn: func(ctx context.Context, roleID string, permissionID string) error { return nil },
		replaceRolePermissionsFn: func(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
			return nil
		},
		assignRoleToUserFn: func(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
			return nil
		},
		removeRoleFromUserFn: func(ctx context.Context, userID string, roleID string) error { return nil },
		replaceUserRolesFn:   func(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error { return nil },
	}

	h := NewUpdatePermissionHandler(stub)
	req := httptest.NewRequest(http.MethodPatch, "/admin/permissions/p-1", strings.NewReader("{"))
	req.SetPathValue("id", "p-1")
	req, rc := withReqCtx(req)
	w := httptest.NewRecorder()

	h.Handler().ServeHTTP(w, req)

	if rc.ResponseStatus != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rc.ResponseStatus)
	}
}

func TestDeletePermissionHandler_Conflict(t *testing.T) {
	stub := rolePermissionUseCaseStub{
		createPermissionFn: func(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
			return nil, nil
		},
		getAllPermissionsFn: func(ctx context.Context) ([]types.Permission, error) { return nil, nil },
		updatePermissionFn: func(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
			return nil, nil
		},
		deletePermissionFn: func(ctx context.Context, permissionID string) error {
			return errors.New("permission is in use by one or more roles")
		},
		createRoleFn:  func(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) { return nil, nil },
		getAllRolesFn: func(ctx context.Context) ([]types.Role, error) { return nil, nil },
		getRoleByIDFn: func(ctx context.Context, roleID string) (*types.RoleDetails, error) { return nil, nil },
		updateRoleFn: func(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
			return nil, nil
		},
		deleteRoleFn: func(ctx context.Context, roleID string) error { return nil },
		addPermissionToRoleFn: func(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
			return nil
		},
		removePermissionFromRoleFn: func(ctx context.Context, roleID string, permissionID string) error { return nil },
		replaceRolePermissionsFn: func(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
			return nil
		},
		assignRoleToUserFn: func(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
			return nil
		},
		removeRoleFromUserFn: func(ctx context.Context, userID string, roleID string) error { return nil },
		replaceUserRolesFn:   func(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error { return nil },
	}

	h := NewDeletePermissionHandler(stub)
	req := httptest.NewRequest(http.MethodDelete, "/admin/permissions/p-in-use", nil)
	req.SetPathValue("id", "p-in-use")
	req, rc := withReqCtx(req)
	w := httptest.NewRecorder()

	h.Handler().ServeHTTP(w, req)

	if rc.ResponseStatus != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rc.ResponseStatus)
	}
}
