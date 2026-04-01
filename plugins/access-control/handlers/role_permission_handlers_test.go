package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/services"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
)

func TestGetRolePermissionsHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	description := new(string)
	*description = "read access"
	grantedByUserID := new(string)
	*grantedByUserID = "user-1"

	tests := []struct {
		name           string
		roleID         string
		setupMock      func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockRolePermissionsRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:           "blank role id",
			roleID:         "",
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   map[string]string{"message": "unprocessable entity"},
		},
		{
			name:   "use case error",
			roleID: "role-404",
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, _ *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-404").Return((*types.Role)(nil), constants.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   map[string]string{"message": "not found"},
		},
		{
			name:   "success",
			roleID: "role-1",
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Administrator"}, nil).Once()
				rolePermissionsRepo.On("GetRolePermissions", mock.Anything, "role-1").Return([]types.UserPermissionInfo{{
					PermissionID:          "perm-1",
					PermissionKey:         "users.read",
					PermissionDescription: description,
					GrantedByUserID:       grantedByUserID,
					GrantedAt:             &fixedTime,
				}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: []types.UserPermissionInfo{{
				PermissionID:          "perm-1",
				PermissionKey:         "users.read",
				PermissionDescription: description,
				GrantedByUserID:       grantedByUserID,
				GrantedAt:             &fixedTime,
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			if tc.setupMock != nil {
				tc.setupMock(rolesRepo, rolePermissionsRepo)
			}

			useCase := newRolePermissionsUseCase(rolesRepo, permissionsRepo, rolePermissionsRepo)
			handler := NewGetRolePermissionsHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/roles/"+tc.roleID+"/permissions", nil, nil)
			req.SetPathValue("role_id", tc.roleID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				rolesRepo.AssertExpectations(t)
				permissionsRepo.AssertExpectations(t)
				rolePermissionsRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[[]types.UserPermissionInfo](t, reqCtx)
			assertUserPermissionInfosEqual(t, payload, tc.expectedBody.([]types.UserPermissionInfo))

			rolesRepo.AssertExpectations(t)
			permissionsRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
		})
	}
}

func TestAddRolePermissionHandler(t *testing.T) {
	t.Parallel()

	grantedByUserID := "user-1"
	actorUserID := &grantedByUserID

	tests := []struct {
		name           string
		roleID         string
		body           []byte
		userID         *string
		setupMock      func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockPermissionsRepository, *accesscontroltests.MockRolePermissionsRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:           "invalid request body",
			roleID:         "role-1",
			body:           []byte("{invalid json"),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   map[string]string{"message": "invalid request body"},
		},
		{
			name:   "use case error",
			roleID: "role-1",
			body:   internaltests.MarshalToJSON(t, types.AddRolePermissionRequest{PermissionID: "perm-1"}),
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Administrator"}, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				rolePermissionsRepo.On("AddRolePermission", mock.Anything, "role-1", "perm-1", (*string)(nil)).Return(constants.ErrUnauthorized).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   map[string]string{"message": "unauthorized"},
		},
		{
			name:   "success",
			roleID: "role-1",
			body:   internaltests.MarshalToJSON(t, types.AddRolePermissionRequest{PermissionID: "perm-1"}),
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Administrator"}, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				rolePermissionsRepo.On("AddRolePermission", mock.Anything, "role-1", "perm-1", (*string)(nil)).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.AddRolePermissionResponse{Message: "permission assigned to role"},
		},
		{
			name:   "success with actor user id",
			roleID: "role-1",
			body:   internaltests.MarshalToJSON(t, types.AddRolePermissionRequest{PermissionID: "perm-2"}),
			userID: actorUserID,
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Administrator"}, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-2").Return(&types.Permission{ID: "perm-2", Key: "users.write"}, nil).Once()
				rolePermissionsRepo.On("AddRolePermission", mock.Anything, "role-1", "perm-2", actorUserID).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.AddRolePermissionResponse{Message: "permission assigned to role"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			if tc.setupMock != nil {
				tc.setupMock(rolesRepo, permissionsRepo, rolePermissionsRepo)
			}

			useCase := newRolePermissionsUseCase(rolesRepo, permissionsRepo, rolePermissionsRepo)
			handler := NewAddRolePermissionHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/roles/"+tc.roleID+"/permissions", tc.body, tc.userID)
			req.SetPathValue("role_id", tc.roleID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				rolesRepo.AssertExpectations(t)
				permissionsRepo.AssertExpectations(t)
				rolePermissionsRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.AddRolePermissionResponse](t, reqCtx)
			assertAddRolePermissionResponseEqual(t, payload, tc.expectedBody.(types.AddRolePermissionResponse))

			rolesRepo.AssertExpectations(t)
			permissionsRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
		})
	}
}

func TestReplaceRolePermissionsHandler(t *testing.T) {
	t.Parallel()

	actorUserID := "user-1"
	actorUserIDPtr := &actorUserID

	tests := []struct {
		name           string
		roleID         string
		body           []byte
		userID         *string
		setupMock      func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockPermissionsRepository, *accesscontroltests.MockRolePermissionsRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:           "invalid request body",
			roleID:         "role-1",
			body:           []byte("{invalid json"),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   map[string]string{"message": "invalid request body"},
		},
		{
			name:   "use case error",
			roleID: "role-1",
			body:   internaltests.MarshalToJSON(t, types.ReplaceRolePermissionsRequest{PermissionIDs: []string{"perm-1", "perm-2"}}),
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Administrator"}, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-2").Return(&types.Permission{ID: "perm-2", Key: "users.write"}, nil).Once()
				rolePermissionsRepo.On("ReplaceRolePermissions", mock.Anything, "role-1", []string{"perm-1", "perm-2"}, (*string)(nil)).Return(constants.ErrConflict).Once()
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   map[string]string{"message": "conflict"},
		},
		{
			name:   "success",
			roleID: "role-1",
			body:   internaltests.MarshalToJSON(t, types.ReplaceRolePermissionsRequest{PermissionIDs: []string{"perm-1", "perm-2"}}),
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Administrator"}, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-2").Return(&types.Permission{ID: "perm-2", Key: "users.write"}, nil).Once()
				rolePermissionsRepo.On("ReplaceRolePermissions", mock.Anything, "role-1", []string{"perm-1", "perm-2"}, (*string)(nil)).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.ReplaceRolePermissionResponse{Message: "role permissions replaced"},
		},
		{
			name:   "success with actor user id",
			roleID: "role-1",
			body:   internaltests.MarshalToJSON(t, types.ReplaceRolePermissionsRequest{PermissionIDs: []string{"perm-3", "perm-4"}}),
			userID: actorUserIDPtr,
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Administrator"}, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-3").Return(&types.Permission{ID: "perm-3", Key: "users.create"}, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-4").Return(&types.Permission{ID: "perm-4", Key: "users.update"}, nil).Once()
				rolePermissionsRepo.On("ReplaceRolePermissions", mock.Anything, "role-1", []string{"perm-3", "perm-4"}, actorUserIDPtr).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.ReplaceRolePermissionResponse{Message: "role permissions replaced"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			if tc.setupMock != nil {
				tc.setupMock(rolesRepo, permissionsRepo, rolePermissionsRepo)
			}

			useCase := newRolePermissionsUseCase(rolesRepo, permissionsRepo, rolePermissionsRepo)
			handler := NewReplaceRolePermissionsHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/roles/"+tc.roleID+"/permissions", tc.body, tc.userID)
			req.SetPathValue("role_id", tc.roleID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				rolesRepo.AssertExpectations(t)
				permissionsRepo.AssertExpectations(t)
				rolePermissionsRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.ReplaceRolePermissionResponse](t, reqCtx)
			assertReplaceRolePermissionResponseEqual(t, payload, tc.expectedBody.(types.ReplaceRolePermissionResponse))

			rolesRepo.AssertExpectations(t)
			permissionsRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
		})
	}
}

func TestRemoveRolePermissionHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		roleID         string
		permissionID   string
		setupMock      func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockPermissionsRepository, *accesscontroltests.MockRolePermissionsRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:         "use case error",
			roleID:       "role-1",
			permissionID: "perm-1",
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Administrator"}, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				rolePermissionsRepo.On("RemoveRolePermission", mock.Anything, "role-1", "perm-1").Return(constants.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   map[string]string{"message": "not found"},
		},
		{
			name:         "success",
			roleID:       "role-1",
			permissionID: "perm-1",
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Administrator"}, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				rolePermissionsRepo.On("RemoveRolePermission", mock.Anything, "role-1", "perm-1").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.RemoveRolePermissionResponse{Message: "permission removed from role"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			if tc.setupMock != nil {
				tc.setupMock(rolesRepo, permissionsRepo, rolePermissionsRepo)
			}

			useCase := newRolePermissionsUseCase(rolesRepo, permissionsRepo, rolePermissionsRepo)
			handler := NewRemoveRolePermissionHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/roles/"+tc.roleID+"/permissions/"+tc.permissionID, nil, nil)
			req.SetPathValue("role_id", tc.roleID)
			req.SetPathValue("permission_id", tc.permissionID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				rolesRepo.AssertExpectations(t)
				permissionsRepo.AssertExpectations(t)
				rolePermissionsRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.RemoveRolePermissionResponse](t, reqCtx)
			assertRemoveRolePermissionResponseEqual(t, payload, tc.expectedBody.(types.RemoveRolePermissionResponse))

			rolesRepo.AssertExpectations(t)
			permissionsRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
		})
	}
}

func newRolePermissionsUseCase(rolesRepo *accesscontroltests.MockRolesRepository, permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) *usecases.RolePermissionsUseCase {
	return usecases.NewRolePermissionsUseCase(services.NewRolePermissionsService(rolesRepo, permissionsRepo, rolePermissionsRepo))
}

func assertUserPermissionInfosEqual(t *testing.T, got []types.UserPermissionInfo, want []types.UserPermissionInfo) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d permissions, got %d", len(want), len(got))
	}

	for i := range want {
		assertUserPermissionInfoEqual(t, got[i], want[i])
	}
}

func assertUserPermissionInfoEqual(t *testing.T, got types.UserPermissionInfo, want types.UserPermissionInfo) {
	t.Helper()

	if got.PermissionID != want.PermissionID {
		t.Fatalf("expected permission id %q, got %q", want.PermissionID, got.PermissionID)
	}
	if got.PermissionKey != want.PermissionKey {
		t.Fatalf("expected permission key %q, got %q", want.PermissionKey, got.PermissionKey)
	}
	if !stringsEqualPtr(got.PermissionDescription, want.PermissionDescription) {
		t.Fatalf("expected permission description %#v, got %#v", want.PermissionDescription, got.PermissionDescription)
	}
	if !stringsEqualPtr(got.GrantedByUserID, want.GrantedByUserID) {
		t.Fatalf("expected granted_by_user_id %#v, got %#v", want.GrantedByUserID, got.GrantedByUserID)
	}
	if !timesEqualPtr(got.GrantedAt, want.GrantedAt) {
		t.Fatalf("expected granted_at %#v, got %#v", want.GrantedAt, got.GrantedAt)
	}
}

func assertAddRolePermissionResponseEqual(t *testing.T, got types.AddRolePermissionResponse, want types.AddRolePermissionResponse) {
	t.Helper()
	if got.Message != want.Message {
		t.Fatalf("expected message %q, got %q", want.Message, got.Message)
	}
}

func assertReplaceRolePermissionResponseEqual(t *testing.T, got types.ReplaceRolePermissionResponse, want types.ReplaceRolePermissionResponse) {
	t.Helper()
	if got.Message != want.Message {
		t.Fatalf("expected message %q, got %q", want.Message, got.Message)
	}
}

func assertRemoveRolePermissionResponseEqual(t *testing.T, got types.RemoveRolePermissionResponse, want types.RemoveRolePermissionResponse) {
	t.Helper()
	if got.Message != want.Message {
		t.Fatalf("expected message %q, got %q", want.Message, got.Message)
	}
}
