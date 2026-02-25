package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

// Stub implementations
type userRolesUseCaseStub struct {
	getUserRolesFn                func(ctx context.Context, userID string) ([]types.UserRoleInfo, error)
	getUserEffectivePermissionsFn func(ctx context.Context, userID string) ([]types.UserPermissionInfo, error)
}

func (s userRolesUseCaseStub) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	return s.getUserRolesFn(ctx, userID)
}

func (s userRolesUseCaseStub) GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	return s.getUserEffectivePermissionsFn(ctx, userID)
}

func (s userRolesUseCaseStub) HasPermissions(ctx context.Context, userID string, requiredPermissions []string) (bool, error) {
	return true, nil
}

func (s userRolesUseCaseStub) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
	return nil, nil
}

func (s userRolesUseCaseStub) GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error) {
	return nil, nil
}

func (s userRolesUseCaseStub) GetUserAuthorizationProfile(ctx context.Context, userID string) (*types.UserAuthorizationProfile, error) {
	return nil, nil
}

type userRolesRolePermissionUseCaseStub struct {
	replaceUserRolesFn   func(ctx context.Context, userID string, roleIDs []string, actorUserID *string) error
	assignRoleToUserFn   func(ctx context.Context, userID string, request types.AssignUserRoleRequest, actorUserID *string) error
	removeRoleFromUserFn func(ctx context.Context, userID string, roleID string) error
}

func (s userRolesRolePermissionUseCaseStub) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, actorUserID *string) error {
	return s.replaceUserRolesFn(ctx, userID, roleIDs, actorUserID)
}

func (s userRolesRolePermissionUseCaseStub) AssignRoleToUser(ctx context.Context, userID string, request types.AssignUserRoleRequest, actorUserID *string) error {
	return s.assignRoleToUserFn(ctx, userID, request, actorUserID)
}

func (s userRolesRolePermissionUseCaseStub) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	return s.removeRoleFromUserFn(ctx, userID, roleID)
}

// Stub methods for unused interface methods (minimal implementations)
func (s userRolesRolePermissionUseCaseStub) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	return nil, nil
}

func (s userRolesRolePermissionUseCaseStub) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return nil, nil
}

func (s userRolesRolePermissionUseCaseStub) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	return nil, nil
}

func (s userRolesRolePermissionUseCaseStub) DeletePermission(ctx context.Context, permissionID string) error {
	return nil
}

func (s userRolesRolePermissionUseCaseStub) CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) {
	return nil, nil
}

func (s userRolesRolePermissionUseCaseStub) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return nil, nil
}

func (s userRolesRolePermissionUseCaseStub) GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error) {
	return nil, nil
}

func (s userRolesRolePermissionUseCaseStub) UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	return nil, nil
}

func (s userRolesRolePermissionUseCaseStub) DeleteRole(ctx context.Context, roleID string) error {
	return nil
}

func (s userRolesRolePermissionUseCaseStub) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	return nil
}

func (s userRolesRolePermissionUseCaseStub) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	return nil
}

func (s userRolesRolePermissionUseCaseStub) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return nil
}

func TestGetUserRolesHandler(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		setupStub      func() userRolesUseCaseStub
		expectedStatus int
		expectedBody   map[string]any // for partial matching
		assertBody     func(t *testing.T, body string)
	}{
		{
			name:   "Success with multiple roles",
			userID: "u-123",
			setupStub: func() userRolesUseCaseStub {
				return userRolesUseCaseStub{
					getUserRolesFn: func(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
						if userID != "u-123" {
							t.Fatalf("expected userID u-123, got %s", userID)
						}
						return []types.UserRoleInfo{
							{RoleID: "r-1", RoleName: "Admin"},
							{RoleID: "r-2", RoleName: "Editor"},
						}, nil
					},
					getUserEffectivePermissionsFn: func(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
						return nil, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, `"role_id":"r-1"`) || !strings.Contains(body, `"role_id":"r-2"`) {
					t.Fatalf("response body missing expected role IDs: %s", body)
				}
				if !strings.Contains(body, `"role_name":"Admin"`) || !strings.Contains(body, `"role_name":"Editor"`) {
					t.Fatalf("response body missing expected role names: %s", body)
				}
			},
		},
		{
			name:   "Empty role list",
			userID: "u-456",
			setupStub: func() userRolesUseCaseStub {
				return userRolesUseCaseStub{
					getUserRolesFn: func(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
						return []types.UserRoleInfo{}, nil
					},
					getUserEffectivePermissionsFn: func(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
						return nil, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, `"data":[]`) {
					t.Fatalf("response body should contain empty data array: %s", body)
				}
			},
		},
		{
			name:   "User not found",
			userID: "u-missing",
			setupStub: func() userRolesUseCaseStub {
				return userRolesUseCaseStub{
					getUserRolesFn: func(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
						return nil, errors.New("user not found")
					},
					getUserEffectivePermissionsFn: func(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
						return nil, nil
					},
				}
			},
			expectedStatus: http.StatusNotFound,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "user not found") {
					t.Fatalf("response body missing expected error message: %s", body)
				}
			},
		},
		{
			name:   "Internal server error",
			userID: "u-789",
			setupStub: func() userRolesUseCaseStub {
				return userRolesUseCaseStub{
					getUserRolesFn: func(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
						return nil, errors.New("database connection failed")
					},
					getUserEffectivePermissionsFn: func(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
						return nil, nil
					},
				}
			},
			expectedStatus: http.StatusInternalServerError,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "database connection failed") {
					t.Fatalf("response body missing expected error message: %s", body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := tt.setupStub()
			h := NewGetUserRolesHandler(stub)
			req := httptest.NewRequest(http.MethodGet, "/admin/users/"+tt.userID+"/roles", nil)
			req.SetPathValue("user_id", tt.userID)
			req, rc := withReqCtx(req)
			w := httptest.NewRecorder()

			h.Handler().ServeHTTP(w, req)

			if rc.ResponseStatus != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rc.ResponseStatus)
			}

			body := string(rc.ResponseBody)
			if tt.assertBody != nil {
				tt.assertBody(t, body)
			}
		})
	}
}

func TestReplaceUserRolesHandler(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		payload        string
		setupStub      func() userRolesRolePermissionUseCaseStub
		expectedStatus int
		assertBody     func(t *testing.T, body string)
	}{
		{
			name:    "Success replacing roles",
			userID:  "u-123",
			payload: `{"role_ids":["r-1","r-2"]}`,
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					replaceUserRolesFn: func(ctx context.Context, userID string, roleIDs []string, actorUserID *string) error {
						if userID != "u-123" {
							t.Fatalf("expected userID u-123, got %s", userID)
						}
						if len(roleIDs) != 2 || roleIDs[0] != "r-1" || roleIDs[1] != "r-2" {
							t.Fatalf("expected roles [r-1, r-2], got %v", roleIDs)
						}
						return nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "user roles replaced") {
					t.Fatalf("response body missing expected message: %s", body)
				}
			},
		},
		{
			name:    "Empty role list",
			userID:  "u-123",
			payload: `{"role_ids":[]}`,
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					replaceUserRolesFn: func(ctx context.Context, userID string, roleIDs []string, actorUserID *string) error {
						if len(roleIDs) != 0 {
							t.Fatalf("expected empty roles, got %v", roleIDs)
						}
						return nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "user roles replaced") {
					t.Fatalf("response body missing expected message: %s", body)
				}
			},
		},
		{
			name:    "Bad JSON request",
			userID:  "u-123",
			payload: `{invalid json`,
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					replaceUserRolesFn: func(ctx context.Context, userID string, roleIDs []string, actorUserID *string) error {
						t.Fatal("should not reach usecase")
						return nil
					},
				}
			},
			expectedStatus: http.StatusUnprocessableEntity,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "invalid request body") {
					t.Fatalf("response body missing expected error message: %s", body)
				}
			},
		},
		{
			name:    "User not found",
			userID:  "u-missing",
			payload: `{"role_ids":["r-1"]}`,
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					replaceUserRolesFn: func(ctx context.Context, userID string, roleIDs []string, actorUserID *string) error {
						return errors.New("user not found")
					},
				}
			},
			expectedStatus: http.StatusNotFound,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "user not found") {
					t.Fatalf("response body missing expected error message: %s", body)
				}
			},
		},
		{
			name:    "Unauthorized to modify roles",
			userID:  "u-123",
			payload: `{"role_ids":["r-1"]}`,
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					replaceUserRolesFn: func(ctx context.Context, userID string, roleIDs []string, actorUserID *string) error {
						return errors.New("unauthorized to modify roles")
					},
				}
			},
			expectedStatus: http.StatusUnauthorized,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "unauthorized") {
					t.Fatalf("response body missing expected error message: %s", body)
				}
			},
		},
		{
			name:    "Forbidden - insufficient permissions",
			userID:  "u-123",
			payload: `{"role_ids":["r-1"]}`,
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					replaceUserRolesFn: func(ctx context.Context, userID string, roleIDs []string, actorUserID *string) error {
						return errors.New("forbidden to replace roles")
					},
				}
			},
			expectedStatus: http.StatusForbidden,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "forbidden") {
					t.Fatalf("response body missing expected error message: %s", body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := tt.setupStub()
			h := NewReplaceUserRolesHandler(stub)
			req := httptest.NewRequest(http.MethodPut, "/admin/users/"+tt.userID+"/roles", strings.NewReader(tt.payload))
			req.SetPathValue("user_id", tt.userID)
			req, rc := withReqCtx(req)
			w := httptest.NewRecorder()

			h.Handler().ServeHTTP(w, req)

			if rc.ResponseStatus != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rc.ResponseStatus)
			}

			body := string(rc.ResponseBody)
			if tt.assertBody != nil {
				tt.assertBody(t, body)
			}
		})
	}
}

func TestAssignUserRoleHandler(t *testing.T) {
	expiresAt := time.Now().Add(24 * time.Hour)
	expiresAtStr := expiresAt.Format(time.RFC3339)

	tests := []struct {
		name           string
		userID         string
		payload        string
		setupStub      func() userRolesRolePermissionUseCaseStub
		expectedStatus int
		assertBody     func(t *testing.T, body string)
	}{
		{
			name:    "Success assigning role without expiration",
			userID:  "u-123",
			payload: `{"role_id":"r-1"}`,
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					assignRoleToUserFn: func(ctx context.Context, userID string, request types.AssignUserRoleRequest, actorUserID *string) error {
						if userID != "u-123" {
							t.Fatalf("expected userID u-123, got %s", userID)
						}
						if request.RoleID != "r-1" {
							t.Fatalf("expected roleID r-1, got %s", request.RoleID)
						}
						if request.ExpiresAt != nil {
							t.Fatal("expected ExpiresAt to be nil")
						}
						return nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "role assigned") {
					t.Fatalf("response body missing expected message: %s", body)
				}
			},
		},
		{
			name:    "Success assigning role with expiration date",
			userID:  "u-123",
			payload: `{"role_id":"r-1","expires_at":"` + expiresAtStr + `"}`,
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					assignRoleToUserFn: func(ctx context.Context, userID string, request types.AssignUserRoleRequest, actorUserID *string) error {
						if request.ExpiresAt == nil {
							t.Fatal("expected ExpiresAt to be set")
						}
						return nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "role assigned") {
					t.Fatalf("response body missing expected message: %s", body)
				}
			},
		},
		{
			name:    "Bad JSON request",
			userID:  "u-123",
			payload: `{incomplete`,
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					assignRoleToUserFn: func(ctx context.Context, userID string, request types.AssignUserRoleRequest, actorUserID *string) error {
						t.Fatal("should not reach usecase")
						return nil
					},
				}
			},
			expectedStatus: http.StatusUnprocessableEntity,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "invalid request body") {
					t.Fatalf("response body missing expected error message: %s", body)
				}
			},
		},
		{
			name:    "Role not found",
			userID:  "u-123",
			payload: `{"role_id":"r-missing"}`,
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					assignRoleToUserFn: func(ctx context.Context, userID string, request types.AssignUserRoleRequest, actorUserID *string) error {
						return errors.New("role not found")
					},
				}
			},
			expectedStatus: http.StatusNotFound,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "role not found") {
					t.Fatalf("response body missing expected error message: %s", body)
				}
			},
		},
		{
			name:    "Invalid expiration date in the past",
			userID:  "u-123",
			payload: `{"role_id":"r-1","expires_at":"2020-01-01T00:00:00Z"}`,
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					assignRoleToUserFn: func(ctx context.Context, userID string, request types.AssignUserRoleRequest, actorUserID *string) error {
						return errors.New("expires_at cannot be in the past")
					},
				}
			},
			expectedStatus: http.StatusBadRequest,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "expires") {
					t.Fatalf("response body missing expected error message: %s", body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := tt.setupStub()
			h := NewAssignUserRoleHandler(stub)
			req := httptest.NewRequest(http.MethodPost, "/admin/users/"+tt.userID+"/roles", strings.NewReader(tt.payload))
			req.SetPathValue("user_id", tt.userID)
			req, rc := withReqCtx(req)
			w := httptest.NewRecorder()

			h.Handler().ServeHTTP(w, req)

			if rc.ResponseStatus != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rc.ResponseStatus)
			}

			body := string(rc.ResponseBody)
			if tt.assertBody != nil {
				tt.assertBody(t, body)
			}
		})
	}
}

func TestRemoveUserRoleHandler(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		roleID         string
		setupStub      func() userRolesRolePermissionUseCaseStub
		expectedStatus int
		assertBody     func(t *testing.T, body string)
	}{
		{
			name:   "Success removing role",
			userID: "u-123",
			roleID: "r-1",
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					removeRoleFromUserFn: func(ctx context.Context, userID string, roleID string) error {
						if userID != "u-123" {
							t.Fatalf("expected userID u-123, got %s", userID)
						}
						if roleID != "r-1" {
							t.Fatalf("expected roleID r-1, got %s", roleID)
						}
						return nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "role removed") {
					t.Fatalf("response body missing expected message: %s", body)
				}
			},
		},
		{
			name:   "Role assignment not found",
			userID: "u-123",
			roleID: "r-missing",
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					removeRoleFromUserFn: func(ctx context.Context, userID string, roleID string) error {
						return errors.New("role assignment not found")
					},
				}
			},
			expectedStatus: http.StatusNotFound,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "role assignment not found") {
					t.Fatalf("response body missing expected error message: %s", body)
				}
			},
		},
		{
			name:   "Forbidden - cannot remove role",
			userID: "u-123",
			roleID: "r-1",
			setupStub: func() userRolesRolePermissionUseCaseStub {
				return userRolesRolePermissionUseCaseStub{
					removeRoleFromUserFn: func(ctx context.Context, userID string, roleID string) error {
						return errors.New("forbidden to remove this role")
					},
				}
			},
			expectedStatus: http.StatusForbidden,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "forbidden") {
					t.Fatalf("response body missing expected error message: %s", body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := tt.setupStub()
			h := NewRemoveUserRoleHandler(stub)
			req := httptest.NewRequest(http.MethodDelete, "/admin/users/"+tt.userID+"/roles/"+tt.roleID, nil)
			req.SetPathValue("user_id", tt.userID)
			req.SetPathValue("role_id", tt.roleID)
			req, rc := withReqCtx(req)
			w := httptest.NewRecorder()

			h.Handler().ServeHTTP(w, req)

			if rc.ResponseStatus != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rc.ResponseStatus)
			}

			body := string(rc.ResponseBody)
			if tt.assertBody != nil {
				tt.assertBody(t, body)
			}
		})
	}
}

func TestGetUserEffectivePermissionsHandler(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		setupStub      func() userRolesUseCaseStub
		expectedStatus int
		assertBody     func(t *testing.T, body string)
	}{
		{
			name:   "Success with multiple permissions",
			userID: "u-123",
			setupStub: func() userRolesUseCaseStub {
				return userRolesUseCaseStub{
					getUserRolesFn: func(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
						return nil, nil
					},
					getUserEffectivePermissionsFn: func(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
						if userID != "u-123" {
							t.Fatalf("expected userID u-123, got %s", userID)
						}
						return []types.UserPermissionInfo{
							{PermissionID: "p-1", PermissionKey: "read:users"},
							{PermissionID: "p-2", PermissionKey: "write:users"},
							{PermissionID: "p-3", PermissionKey: "delete:users"},
						}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			assertBody: func(t *testing.T, body string) {
				expectedKeys := []string{"read:users", "write:users", "delete:users"}
				for _, key := range expectedKeys {
					if !strings.Contains(body, `"permission_key":"`+key+`"`) {
						t.Fatalf("response body missing expected permission key: %s, body: %s", key, body)
					}
				}
				// Also verify permission IDs are present
				if !strings.Contains(body, `"permission_id":"p-1"`) {
					t.Fatalf("response body missing expected permission ID p-1: %s", body)
				}
			},
		},
		{
			name:   "Empty permission list",
			userID: "u-456",
			setupStub: func() userRolesUseCaseStub {
				return userRolesUseCaseStub{
					getUserRolesFn: func(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
						return nil, nil
					},
					getUserEffectivePermissionsFn: func(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
						return []types.UserPermissionInfo{}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, `"data":[]`) {
					t.Fatalf("response body should contain empty data array: %s", body)
				}
			},
		},
		{
			name:   "User not found",
			userID: "u-missing",
			setupStub: func() userRolesUseCaseStub {
				return userRolesUseCaseStub{
					getUserRolesFn: func(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
						return nil, nil
					},
					getUserEffectivePermissionsFn: func(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
						return nil, errors.New("user not found")
					},
				}
			},
			expectedStatus: http.StatusNotFound,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "user not found") {
					t.Fatalf("response body missing expected error message: %s", body)
				}
			},
		},
		{
			name:   "Internal server error",
			userID: "u-789",
			setupStub: func() userRolesUseCaseStub {
				return userRolesUseCaseStub{
					getUserRolesFn: func(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
						return nil, nil
					},
					getUserEffectivePermissionsFn: func(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
						return nil, errors.New("database error while fetching permissions")
					},
				}
			},
			expectedStatus: http.StatusInternalServerError,
			assertBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "database error") {
					t.Fatalf("response body missing expected error message: %s", body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := tt.setupStub()
			h := NewGetUserEffectivePermissionsHandler(stub)
			req := httptest.NewRequest(http.MethodGet, "/admin/users/"+tt.userID+"/permissions", nil)
			req.SetPathValue("user_id", tt.userID)
			req, rc := withReqCtx(req)
			w := httptest.NewRecorder()

			h.Handler().ServeHTTP(w, req)

			if rc.ResponseStatus != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rc.ResponseStatus)
			}

			body := string(rc.ResponseBody)
			if tt.assertBody != nil {
				tt.assertBody(t, body)
			}
		})
	}
}

func TestResponseBodyParseable(t *testing.T) {
	stub := userRolesUseCaseStub{
		getUserRolesFn: func(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
			return []types.UserRoleInfo{
				{RoleID: "r-1", RoleName: "Admin"},
			}, nil
		},
		getUserEffectivePermissionsFn: func(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
			return nil, nil
		},
	}

	h := NewGetUserRolesHandler(stub)
	req := httptest.NewRequest(http.MethodGet, "/admin/users/u-123/roles", nil)
	req.SetPathValue("user_id", "u-123")
	req, rc := withReqCtx(req)
	w := httptest.NewRecorder()

	h.Handler().ServeHTTP(w, req)

	var result map[string]any
	err := json.Unmarshal(rc.ResponseBody, &result)
	if err != nil {
		t.Fatalf("response body is not valid JSON: %v, body: %s", err, string(rc.ResponseBody))
	}

	if _, ok := result["data"]; !ok {
		t.Fatalf("response body missing 'data' field")
	}
}
