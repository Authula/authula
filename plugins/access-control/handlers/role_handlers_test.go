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

func TestCreateRoleHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	description := new(string)
	*description = "platform administrator"

	tests := []struct {
		name           string
		body           []byte
		setupMock      func(*accesscontroltests.MockRolesRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:           "invalid request body",
			body:           []byte("{invalid json"),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   map[string]string{"message": "invalid request body"},
		},
		{
			name: "service error",
			body: internaltests.MarshalToJSON(t, types.CreateRoleRequest{
				Name:        "Administrator",
				Description: description,
				IsSystem:    true,
			}),
			setupMock: func(m *accesscontroltests.MockRolesRepository) {
				m.On("CreateRole", mock.Anything, mock.MatchedBy(func(role *types.Role) bool {
					return role != nil && role.Name == "Administrator" && role.Description != nil && *role.Description == *description && role.IsSystem && role.ID != ""
				})).Return(constants.ErrUnauthorized).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   map[string]string{"message": "unauthorized"},
		},
		{
			name: "success",
			body: internaltests.MarshalToJSON(t, types.CreateRoleRequest{
				Name:        "Administrator",
				Description: description,
				IsSystem:    true,
			}),
			setupMock: func(m *accesscontroltests.MockRolesRepository) {
				m.On("CreateRole", mock.Anything, mock.MatchedBy(func(role *types.Role) bool {
					return role != nil && role.Name == "Administrator" && role.Description != nil && *role.Description == *description && role.IsSystem && role.ID != ""
				})).Run(func(args mock.Arguments) {
					role := args.Get(1).(*types.Role)
					role.ID = "role-1"
					role.CreatedAt = fixedTime
					role.UpdatedAt = fixedTime
				}).Return(nil).Once()
			},
			expectedStatus: http.StatusCreated,
			expectedBody: types.CreateRoleResponse{
				Role: &types.Role{
					ID:          "role-1",
					Name:        "Administrator",
					Description: description,
					IsSystem:    true,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setupMock != nil {
				tc.setupMock(rolesRepo)
			}

			useCase := newRolesUseCase(rolesRepo, rolePermissionsRepo, userAccessRepo)
			handler := NewCreateRoleHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/roles", tc.body, nil)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusCreated {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				rolesRepo.AssertExpectations(t)
				rolePermissionsRepo.AssertExpectations(t)
				userAccessRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.CreateRoleResponse](t, reqCtx)
			assertCreateRoleResponseEqual(t, payload, tc.expectedBody.(types.CreateRoleResponse))

			rolesRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func TestGetAllRolesHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	description := new(string)
	*description = "platform administrator"

	tests := []struct {
		name           string
		setupMock      func(*accesscontroltests.MockRolesRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name: "service error",
			setupMock: func(m *accesscontroltests.MockRolesRepository) {
				m.On("GetAllRoles", mock.Anything).Return(([]types.Role)(nil), constants.ErrForbidden).Once()
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   map[string]string{"message": "forbidden"},
		},
		{
			name: "success",
			setupMock: func(m *accesscontroltests.MockRolesRepository) {
				m.On("GetAllRoles", mock.Anything).Return([]types.Role{
					{
						ID:          "role-1",
						Name:        "Administrator",
						Description: description,
						IsSystem:    true,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
					{
						ID:          "role-2",
						Name:        "Editor",
						Description: nil,
						IsSystem:    false,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: []types.Role{
				{
					ID:          "role-1",
					Name:        "Administrator",
					Description: description,
					IsSystem:    true,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
				{
					ID:          "role-2",
					Name:        "Editor",
					Description: nil,
					IsSystem:    false,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setupMock != nil {
				tc.setupMock(rolesRepo)
			}

			useCase := newRolesUseCase(rolesRepo, rolePermissionsRepo, userAccessRepo)
			handler := NewGetAllRolesHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/roles", nil, nil)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				rolesRepo.AssertExpectations(t)
				rolePermissionsRepo.AssertExpectations(t)
				userAccessRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[[]types.Role](t, reqCtx)
			assertRolesEqual(t, payload, tc.expectedBody.([]types.Role))

			rolesRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func TestGetRoleByIDHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	description := new(string)
	*description = "platform administrator"
	grantedAt := fixedTime
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
			name:   "service error",
			roleID: "role-404",
			setupMock: func(m *accesscontroltests.MockRolesRepository, _ *accesscontroltests.MockRolePermissionsRepository) {
				m.On("GetRoleByID", mock.Anything, "role-404").Return((*types.Role)(nil), constants.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   map[string]string{"message": "not found"},
		},
		{
			name:   "success",
			roleID: "role-1",
			setupMock: func(m *accesscontroltests.MockRolesRepository, rp *accesscontroltests.MockRolePermissionsRepository) {
				m.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{
					ID:          "role-1",
					Name:        "Administrator",
					Description: description,
					IsSystem:    true,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				}, nil).Once()
				rp.On("GetRolePermissions", mock.Anything, "role-1").Return([]types.UserPermissionInfo{
					{
						PermissionID:          "perm-1",
						PermissionKey:         "users.read",
						PermissionDescription: description,
						GrantedByUserID:       grantedByUserID,
						GrantedAt:             &grantedAt,
					},
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: types.RoleDetails{
				Role: types.Role{
					ID:          "role-1",
					Name:        "Administrator",
					Description: description,
					IsSystem:    true,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
				Permissions: []types.UserPermissionInfo{
					{
						PermissionID:          "perm-1",
						PermissionKey:         "users.read",
						PermissionDescription: description,
						GrantedByUserID:       grantedByUserID,
						GrantedAt:             &grantedAt,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setupMock != nil {
				tc.setupMock(rolesRepo, rolePermissionsRepo)
			}

			useCase := newRolesUseCase(rolesRepo, rolePermissionsRepo, userAccessRepo)
			handler := NewGetRoleByIDHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/roles/"+tc.roleID, nil, nil)
			req.SetPathValue("role_id", tc.roleID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				rolesRepo.AssertExpectations(t)
				rolePermissionsRepo.AssertExpectations(t)
				userAccessRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.RoleDetails](t, reqCtx)
			assertRoleDetailsEqual(t, payload, tc.expectedBody.(types.RoleDetails))

			rolesRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateRoleHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	description := new(string)
	*description = "updated administrator"
	updatedName := "Administrator"

	tests := []struct {
		name           string
		body           []byte
		setupMock      func(*accesscontroltests.MockRolesRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:           "invalid request body",
			body:           []byte("{invalid json"),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   map[string]string{"message": "invalid request body"},
		},
		{
			name: "service error",
			body: internaltests.MarshalToJSON(t, types.UpdateRoleRequest{
				Name:        &updatedName,
				Description: description,
			}),
			setupMock: func(m *accesscontroltests.MockRolesRepository) {
				m.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Administrator", IsSystem: true}, nil).Once()
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   map[string]string{"message": "cannot update system role"},
		},
		{
			name: "success",
			body: internaltests.MarshalToJSON(t, types.UpdateRoleRequest{
				Name:        &updatedName,
				Description: description,
			}),
			setupMock: func(m *accesscontroltests.MockRolesRepository) {
				m.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Administrator", Description: description, IsSystem: false}, nil).Once()
				m.On("UpdateRole", mock.Anything, "role-1", mock.MatchedBy(func(name *string) bool {
					return name != nil && *name == "Administrator"
				}), mock.MatchedBy(func(desc *string) bool {
					return desc != nil && *desc == *description
				})).Return(true, nil).Once()
				m.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{
					ID:          "role-1",
					Name:        "Administrator",
					Description: description,
					IsSystem:    false,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: types.UpdateRoleResponse{
				Role: &types.Role{
					ID:          "role-1",
					Name:        "Administrator",
					Description: description,
					IsSystem:    false,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setupMock != nil {
				tc.setupMock(rolesRepo)
			}

			useCase := newRolesUseCase(rolesRepo, rolePermissionsRepo, userAccessRepo)
			handler := NewUpdateRoleHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/roles/role-1", tc.body, nil)
			req.SetPathValue("role_id", "role-1")

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				rolesRepo.AssertExpectations(t)
				rolePermissionsRepo.AssertExpectations(t)
				userAccessRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.UpdateRoleResponse](t, reqCtx)
			assertUpdateRoleResponseEqual(t, payload, tc.expectedBody.(types.UpdateRoleResponse))

			rolesRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func TestDeleteRoleHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		roleID         string
		setupMock      func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockUserAccessRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:   "service error",
			roleID: "role-1",
			setupMock: func(m *accesscontroltests.MockRolesRepository, _ *accesscontroltests.MockUserAccessRepository) {
				m.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Administrator", IsSystem: true}, nil).Once()
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   map[string]string{"message": "cannot update system role"},
		},
		{
			name:   "success",
			roleID: "role-1",
			setupMock: func(m *accesscontroltests.MockRolesRepository, u *accesscontroltests.MockUserAccessRepository) {
				m.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Administrator", IsSystem: false}, nil).Once()
				u.On("CountUserAssignmentsByRoleID", mock.Anything, "role-1").Return(0, nil).Once()
				m.On("DeleteRole", mock.Anything, "role-1").Return(true, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.DeleteRoleResponse{Message: "deleted role"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setupMock != nil {
				tc.setupMock(rolesRepo, userAccessRepo)
			}

			useCase := newRolesUseCase(rolesRepo, rolePermissionsRepo, userAccessRepo)
			handler := NewDeleteRoleHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/roles/"+tc.roleID, nil, nil)
			req.SetPathValue("role_id", tc.roleID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				rolesRepo.AssertExpectations(t)
				rolePermissionsRepo.AssertExpectations(t)
				userAccessRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.DeleteRoleResponse](t, reqCtx)
			assertDeleteRoleResponseEqual(t, payload, tc.expectedBody.(types.DeleteRoleResponse))

			rolesRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func newRolesUseCase(rolesRepo *accesscontroltests.MockRolesRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository, userAccessRepo *accesscontroltests.MockUserAccessRepository) *usecases.RolesUseCase {
	return usecases.NewRolesUseCase(services.NewRolesService(rolesRepo, rolePermissionsRepo, userAccessRepo))
}

func assertRolesEqual(t *testing.T, got []types.Role, want []types.Role) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d roles, got %d", len(want), len(got))
	}

	for i := range want {
		assertRoleEqual(t, got[i], want[i])
	}
}

func assertRoleEqual(t *testing.T, got types.Role, want types.Role) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("expected id %q, got %q", want.ID, got.ID)
	}
	if got.Name != want.Name {
		t.Fatalf("expected name %q, got %q", want.Name, got.Name)
	}
	if got.IsSystem != want.IsSystem {
		t.Fatalf("expected is_system %v, got %v", want.IsSystem, got.IsSystem)
	}
	if !timesEqual(got.CreatedAt, want.CreatedAt) {
		t.Fatalf("expected created_at %v, got %v", want.CreatedAt, got.CreatedAt)
	}
	if !timesEqual(got.UpdatedAt, want.UpdatedAt) {
		t.Fatalf("expected updated_at %v, got %v", want.UpdatedAt, got.UpdatedAt)
	}
	if !stringsEqualPtr(got.Description, want.Description) {
		t.Fatalf("expected description %#v, got %#v", want.Description, got.Description)
	}
}

func assertRoleDetailsEqual(t *testing.T, got types.RoleDetails, want types.RoleDetails) {
	t.Helper()

	assertRoleEqual(t, got.Role, want.Role)
	if len(got.Permissions) != len(want.Permissions) {
		t.Fatalf("expected %d permissions, got %d", len(want.Permissions), len(got.Permissions))
	}
	for i := range want.Permissions {
		gotPerm := got.Permissions[i]
		wantPerm := want.Permissions[i]
		if gotPerm.PermissionID != wantPerm.PermissionID {
			t.Fatalf("expected permission id %q, got %q", wantPerm.PermissionID, gotPerm.PermissionID)
		}
		if gotPerm.PermissionKey != wantPerm.PermissionKey {
			t.Fatalf("expected permission key %q, got %q", wantPerm.PermissionKey, gotPerm.PermissionKey)
		}
		if !stringsEqualPtr(gotPerm.PermissionDescription, wantPerm.PermissionDescription) {
			t.Fatalf("expected permission description %#v, got %#v", wantPerm.PermissionDescription, gotPerm.PermissionDescription)
		}
		if !stringsEqualPtr(gotPerm.GrantedByUserID, wantPerm.GrantedByUserID) {
			t.Fatalf("expected granted_by_user_id %#v, got %#v", wantPerm.GrantedByUserID, gotPerm.GrantedByUserID)
		}
		if !timesEqualPtr(gotPerm.GrantedAt, wantPerm.GrantedAt) {
			t.Fatalf("expected granted_at %#v, got %#v", wantPerm.GrantedAt, gotPerm.GrantedAt)
		}
	}
}

func assertCreateRoleResponseEqual(t *testing.T, got types.CreateRoleResponse, want types.CreateRoleResponse) {
	t.Helper()

	if got.Role == nil || want.Role == nil {
		if got.Role != want.Role {
			t.Fatalf("expected role %#v, got %#v", want.Role, got.Role)
		}
		return
	}

	assertRoleEqual(t, *got.Role, *want.Role)
}

func assertUpdateRoleResponseEqual(t *testing.T, got types.UpdateRoleResponse, want types.UpdateRoleResponse) {
	t.Helper()

	if got.Role == nil || want.Role == nil {
		if got.Role != want.Role {
			t.Fatalf("expected role %#v, got %#v", want.Role, got.Role)
		}
		return
	}

	assertRoleEqual(t, *got.Role, *want.Role)
}

func assertDeleteRoleResponseEqual(t *testing.T, got types.DeleteRoleResponse, want types.DeleteRoleResponse) {
	t.Helper()

	if got.Message != want.Message {
		t.Fatalf("expected message %q, got %q", want.Message, got.Message)
	}
}

func timesEqualPtr(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Equal(*right)
}
