package handlers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/Authula/authula/internal/tests"
	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/services"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
)

func TestGetUserPermissionsHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		userID         string
		setupMock      func(*accesscontroltests.MockUserPermissionsRepository)
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
			name:   "success",
			userID: "u1",
			setupMock: func(m *accesscontroltests.MockUserPermissionsRepository) {
				m.On("GetUserPermissions", mock.Anything, "u1").Return([]types.UserPermissionInfo{{PermissionID: "perm-1", PermissionKey: "users.read"}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.GetUserEffectivePermissionsResponse{Permissions: []types.UserPermissionInfo{{PermissionID: "perm-1", PermissionKey: "users.read"}}},
		},
		{
			name:   "repo error",
			userID: "u1",
			setupMock: func(m *accesscontroltests.MockUserPermissionsRepository) {
				m.On("GetUserPermissions", mock.Anything, "u1").Return(([]types.UserPermissionInfo)(nil), accesscontrolconstants.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   map[string]string{"message": "not found"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &accesscontroltests.MockUserPermissionsRepository{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}

			handler := NewGetUserPermissionsHandler(newUserPermissionsUseCase(repo))
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/users/"+tc.userID+"/permissions", nil, nil)
			req.SetPathValue("user_id", tc.userID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				repo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.GetUserEffectivePermissionsResponse](t, reqCtx)
			assertUserPermissionInfosEqual(t, payload.Permissions, tc.expectedBody.(types.GetUserEffectivePermissionsResponse).Permissions)
			repo.AssertExpectations(t)
		})
	}
}

func TestCheckUserPermissionsHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		userID         string
		body           []byte
		setupMock      func(*accesscontroltests.MockUserPermissionsRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:           "invalid request body",
			userID:         "u1",
			body:           []byte("{invalid json"),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   map[string]string{"message": "invalid request body"},
		},
		{
			name:   "success",
			userID: "u1",
			body:   internaltests.MarshalToJSON(t, types.CheckUserPermissionsRequest{PermissionKeys: []string{"users.read", "users.write"}}),
			setupMock: func(m *accesscontroltests.MockUserPermissionsRepository) {
				m.On("HasPermissions", mock.Anything, "u1", []string{"users.read", "users.write"}).Return(true, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.CheckUserPermissionsResponse{HasPermissions: true},
		},
		{
			name:   "repo error",
			userID: "u1",
			body:   internaltests.MarshalToJSON(t, types.CheckUserPermissionsRequest{PermissionKeys: []string{"users.read"}}),
			setupMock: func(m *accesscontroltests.MockUserPermissionsRepository) {
				m.On("HasPermissions", mock.Anything, "u1", []string{"users.read"}).Return(false, accesscontrolconstants.ErrForbidden).Once()
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   map[string]string{"message": "forbidden"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &accesscontroltests.MockUserPermissionsRepository{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}

			handler := NewCheckUserPermissionsHandler(newUserPermissionsUseCase(repo))
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/users/"+tc.userID+"/permissions/check", tc.body, nil)
			req.SetPathValue("user_id", tc.userID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				repo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.CheckUserPermissionsResponse](t, reqCtx)
			if payload != tc.expectedBody.(types.CheckUserPermissionsResponse) {
				t.Fatalf("unexpected response: %#v", payload)
			}
			repo.AssertExpectations(t)
		})
	}
}

func newUserPermissionsUseCase(repo *accesscontroltests.MockUserPermissionsRepository) *usecases.UserPermissionsUseCase {
	return usecases.NewUserPermissionsUseCase(services.NewUserPermissionsService(repo))
}
