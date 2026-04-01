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

func TestGetUserRolesHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	description := new(string)
	*description = "editor role"
	assignedByUserID := new(string)
	*assignedByUserID = "user-2"

	tests := []struct {
		name           string
		userID         string
		setupMock      func(*accesscontroltests.MockUserRolesRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:           "blank user id",
			userID:         "",
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   map[string]string{"message": "unprocessable entity"},
		},
		{
			name:   "use case error",
			userID: "user-404",
			setupMock: func(m *accesscontroltests.MockUserRolesRepository) {
				m.On("GetUserRoles", mock.Anything, "user-404").Return(([]types.UserRoleInfo)(nil), constants.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   map[string]string{"message": "not found"},
		},
		{
			name:   "success",
			userID: "user-1",
			setupMock: func(m *accesscontroltests.MockUserRolesRepository) {
				m.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{{
					RoleID:           "role-1",
					RoleName:         "Editor",
					RoleDescription:  description,
					AssignedByUserID: assignedByUserID,
					AssignedAt:       &fixedTime,
					ExpiresAt:        &fixedTime,
				}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: []types.UserRoleInfo{{
				RoleID:           "role-1",
				RoleName:         "Editor",
				RoleDescription:  description,
				AssignedByUserID: assignedByUserID,
				AssignedAt:       &fixedTime,
				ExpiresAt:        &fixedTime,
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			rolesRepo := &accesscontroltests.MockRolesRepository{}
			if tc.setupMock != nil {
				tc.setupMock(userRolesRepo)
			}

			useCase := newUserRolesUseCase(rolesRepo, userRolesRepo)
			handler := NewGetUserRolesHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/users/"+tc.userID+"/roles", nil, nil)
			req.SetPathValue("user_id", tc.userID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				userRolesRepo.AssertExpectations(t)
				rolesRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[[]types.UserRoleInfo](t, reqCtx)
			assertUserRoleInfosEqual(t, payload, tc.expectedBody.([]types.UserRoleInfo))

			userRolesRepo.AssertExpectations(t)
			rolesRepo.AssertExpectations(t)
		})
	}
}

func TestReplaceUserRolesHandler(t *testing.T) {
	t.Parallel()

	actorUserID := "user-1"
	actorUserIDPtr := &actorUserID

	tests := []struct {
		name           string
		userID         string
		body           []byte
		userIDPtr      *string
		setupMock      func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockUserRolesRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:           "invalid request body",
			userID:         "user-1",
			body:           []byte("{invalid json"),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   map[string]string{"message": "invalid request body"},
		},
		{
			name:           "blank user id",
			userID:         "",
			body:           internaltests.MarshalToJSON(t, types.ReplaceUserRolesRequest{RoleIDs: []string{"role-1"}}),
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]string{"message": "bad request"},
		},
		{
			name:   "success",
			userID: "user-1",
			body:   internaltests.MarshalToJSON(t, types.ReplaceUserRolesRequest{RoleIDs: []string{"role-1", "role-2"}}),
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Editor"}, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-2").Return(&types.Role{ID: "role-2", Name: "Viewer"}, nil).Once()
				userRolesRepo.On("ReplaceUserRoles", mock.Anything, "user-1", []string{"role-1", "role-2"}, (*string)(nil)).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.ReplaceUserRolesResponse{Message: "user roles replaced"},
		},
		{
			name:      "success with actor user id",
			userID:    "user-1",
			userIDPtr: actorUserIDPtr,
			body:      internaltests.MarshalToJSON(t, types.ReplaceUserRolesRequest{RoleIDs: []string{"role-3", "role-4"}}),
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-3").Return(&types.Role{ID: "role-3", Name: "Reviewer"}, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-4").Return(&types.Role{ID: "role-4", Name: "Commenter"}, nil).Once()
				userRolesRepo.On("ReplaceUserRoles", mock.Anything, "user-1", []string{"role-3", "role-4"}, actorUserIDPtr).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.ReplaceUserRolesResponse{Message: "user roles replaced"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			if tc.setupMock != nil {
				tc.setupMock(rolesRepo, userRolesRepo)
			}

			useCase := newUserRolesUseCase(rolesRepo, userRolesRepo)
			handler := NewReplaceUserRolesHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/users/"+tc.userID+"/roles", tc.body, tc.userIDPtr)
			req.SetPathValue("user_id", tc.userID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				rolesRepo.AssertExpectations(t)
				userRolesRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.ReplaceUserRolesResponse](t, reqCtx)
			assertReplaceUserRolesResponseEqual(t, payload, tc.expectedBody.(types.ReplaceUserRolesResponse))

			rolesRepo.AssertExpectations(t)
			userRolesRepo.AssertExpectations(t)
		})
	}
}

func TestAssignUserRoleHandler(t *testing.T) {
	t.Parallel()

	futureTime := time.Date(2030, 3, 29, 13, 0, 0, 0, time.UTC)
	actorUserID := "user-1"
	actorUserIDPtr := &actorUserID

	tests := []struct {
		name           string
		userID         string
		body           []byte
		userIDPtr      *string
		setupMock      func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockUserRolesRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:           "invalid request body",
			userID:         "user-1",
			body:           []byte("{invalid json"),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   map[string]string{"message": "invalid request body"},
		},
		{
			name:           "blank user id",
			userID:         "",
			body:           internaltests.MarshalToJSON(t, types.AssignUserRoleRequest{RoleID: "role-1"}),
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]string{"message": "bad request"},
		},
		{
			name:   "use case error",
			userID: "user-1",
			body:   internaltests.MarshalToJSON(t, types.AssignUserRoleRequest{RoleID: "role-1"}),
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Editor"}, nil).Once()
				userRolesRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", (*string)(nil), (*time.Time)(nil)).Return(constants.ErrUnauthorized).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   map[string]string{"message": "unauthorized"},
		},
		{
			name:   "success",
			userID: "user-1",
			body:   internaltests.MarshalToJSON(t, types.AssignUserRoleRequest{RoleID: "role-1", ExpiresAt: &futureTime}),
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Editor"}, nil).Once()
				userRolesRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", (*string)(nil), &futureTime).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.AssignUserRoleResponse{Message: "role assigned"},
		},
		{
			name:      "success with actor user id",
			userID:    "user-1",
			userIDPtr: actorUserIDPtr,
			body:      internaltests.MarshalToJSON(t, types.AssignUserRoleRequest{RoleID: "role-2"}),
			setupMock: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-2").Return(&types.Role{ID: "role-2", Name: "Reviewer"}, nil).Once()
				userRolesRepo.On("AssignUserRole", mock.Anything, "user-1", "role-2", actorUserIDPtr, (*time.Time)(nil)).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.AssignUserRoleResponse{Message: "role assigned"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			if tc.setupMock != nil {
				tc.setupMock(rolesRepo, userRolesRepo)
			}

			useCase := newUserRolesUseCase(rolesRepo, userRolesRepo)
			handler := NewAssignUserRoleHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/users/"+tc.userID+"/roles", tc.body, tc.userIDPtr)
			req.SetPathValue("user_id", tc.userID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				rolesRepo.AssertExpectations(t)
				userRolesRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.AssignUserRoleResponse](t, reqCtx)
			assertAssignUserRoleResponseEqual(t, payload, tc.expectedBody.(types.AssignUserRoleResponse))

			rolesRepo.AssertExpectations(t)
			userRolesRepo.AssertExpectations(t)
		})
	}
}

func TestRemoveUserRoleHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		userID         string
		roleID         string
		setupMock      func(*accesscontroltests.MockUserRolesRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:           "blank role id",
			userID:         "user-1",
			roleID:         "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]string{"message": "bad request"},
		},
		{
			name:   "use case error",
			userID: "user-1",
			roleID: "role-1",
			setupMock: func(m *accesscontroltests.MockUserRolesRepository) {
				m.On("RemoveUserRole", mock.Anything, "user-1", "role-1").Return(constants.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   map[string]string{"message": "not found"},
		},
		{
			name:   "success",
			userID: "user-1",
			roleID: "role-1",
			setupMock: func(m *accesscontroltests.MockUserRolesRepository) {
				m.On("RemoveUserRole", mock.Anything, "user-1", "role-1").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.RemoveUserRoleResponse{Message: "role removed"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			if tc.setupMock != nil {
				tc.setupMock(userRolesRepo)
			}

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			useCase := newUserRolesUseCase(rolesRepo, userRolesRepo)
			handler := NewRemoveUserRoleHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/users/"+tc.userID+"/roles/"+tc.roleID, nil, nil)
			req.SetPathValue("user_id", tc.userID)
			req.SetPathValue("role_id", tc.roleID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				userRolesRepo.AssertExpectations(t)
				rolesRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.RemoveUserRoleResponse](t, reqCtx)
			assertRemoveUserRoleResponseEqual(t, payload, tc.expectedBody.(types.RemoveUserRoleResponse))

			userRolesRepo.AssertExpectations(t)
			rolesRepo.AssertExpectations(t)
		})
	}
}

func newUserRolesUseCase(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) *usecases.UserRolesUseCase {
	return usecases.NewUserRolesUseCase(services.NewUserRolesService(userRolesRepo, rolesRepo))
}

func assertUserRoleInfosEqual(t *testing.T, got []types.UserRoleInfo, want []types.UserRoleInfo) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d roles, got %d", len(want), len(got))
	}

	for i := range want {
		assertUserRoleInfoEqual(t, got[i], want[i])
	}
}

func assertUserRoleInfoEqual(t *testing.T, got types.UserRoleInfo, want types.UserRoleInfo) {
	t.Helper()

	if got.RoleID != want.RoleID || got.RoleName != want.RoleName {
		t.Fatalf("unexpected role info: %#v", got)
	}
	if !stringsEqualPtr(got.RoleDescription, want.RoleDescription) {
		t.Fatalf("unexpected role description: %#v", got)
	}
	if !stringsEqualPtr(got.AssignedByUserID, want.AssignedByUserID) {
		t.Fatalf("unexpected assigned_by_user_id: %#v", got)
	}
	if !timesEqualPtr(got.AssignedAt, want.AssignedAt) {
		t.Fatalf("unexpected assigned_at: %#v", got)
	}
	if !timesEqualPtr(got.ExpiresAt, want.ExpiresAt) {
		t.Fatalf("unexpected expires_at: %#v", got)
	}
}

func assertReplaceUserRolesResponseEqual(t *testing.T, got types.ReplaceUserRolesResponse, want types.ReplaceUserRolesResponse) {
	t.Helper()
	if got.Message != want.Message {
		t.Fatalf("expected message %q, got %q", want.Message, got.Message)
	}
}

func assertAssignUserRoleResponseEqual(t *testing.T, got types.AssignUserRoleResponse, want types.AssignUserRoleResponse) {
	t.Helper()
	if got.Message != want.Message {
		t.Fatalf("expected message %q, got %q", want.Message, got.Message)
	}
}

func assertRemoveUserRoleResponseEqual(t *testing.T, got types.RemoveUserRoleResponse, want types.RemoveUserRoleResponse) {
	t.Helper()
	if got.Message != want.Message {
		t.Fatalf("expected message %q, got %q", want.Message, got.Message)
	}
}
