package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/Authula/authula/internal/tests"
	authmodels "github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/constants"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestGetUserEffectivePermissionsHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	description := new(string)
	*description = "read access"

	tests := []struct {
		name           string
		userID         string
		setupMock      func(*accesscontroltests.MockUserAccessRepository)
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
			userID: "user-1",
			setupMock: func(m *accesscontroltests.MockUserAccessRepository) {
				m.On("GetUserEffectivePermissions", mock.Anything, "user-1").Return(([]types.UserPermissionInfo)(nil), constants.ErrUnauthorized).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   map[string]string{"message": "unauthorized"},
		},
		{
			name:   "success",
			userID: "user-1",
			setupMock: func(m *accesscontroltests.MockUserAccessRepository) {
				m.On("GetUserEffectivePermissions", mock.Anything, "user-1").Return([]types.UserPermissionInfo{{
					PermissionID:          "perm-1",
					PermissionKey:         "users.read",
					PermissionDescription: description,
					GrantedAt:             &fixedTime,
				}}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: &types.GetUserEffectivePermissionsResponse{Permissions: []types.UserPermissionInfo{{
				PermissionID:          "perm-1",
				PermissionKey:         "users.read",
				PermissionDescription: description,
				GrantedAt:             &fixedTime,
			}}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setupMock != nil {
				tc.setupMock(userAccessRepo)
			}

			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			useCase := newUserAccessUseCase(userRolesRepo, userAccessRepo)
			handler := NewGetUserEffectivePermissionsHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/users/"+tc.userID+"/permissions", nil, nil)
			req.SetPathValue("user_id", tc.userID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				userAccessRepo.AssertExpectations(t)
				userRolesRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.GetUserEffectivePermissionsResponse](t, reqCtx)
			assertGetUserEffectivePermissionsResponseEqual(t, payload, *tc.expectedBody.(*types.GetUserEffectivePermissionsResponse))

			userAccessRepo.AssertExpectations(t)
			userRolesRepo.AssertExpectations(t)
		})
	}
}

func TestGetUserAuthorizationProfileHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	roleDescription := new(string)
	*roleDescription = "profile role"
	permissionDescription := new(string)
	*permissionDescription = "profile access"

	tests := []struct {
		name           string
		userID         string
		setupMock      func(*accesscontroltests.MockUserRolesRepository, *accesscontroltests.MockUserAccessRepository)
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
			setupMock: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, _ *accesscontroltests.MockUserAccessRepository) {
				userRolesRepo.On("GetUserWithRolesByID", mock.Anything, "user-404").Return((*types.UserWithRoles)(nil), constants.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   map[string]string{"message": "not found"},
		},
		{
			name:   "success",
			userID: "user-1",
			setupMock: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, userAccessRepo *accesscontroltests.MockUserAccessRepository) {
				userRolesRepo.On("GetUserWithRolesByID", mock.Anything, "user-1").Return(&types.UserWithRoles{
					User: authmodels.User{
						ID:            "user-1",
						Name:          "Pat",
						Email:         "pat@example.com",
						EmailVerified: true,
						Metadata:      json.RawMessage("null"),
						CreatedAt:     fixedTime,
						UpdatedAt:     fixedTime,
					},
					Roles: []types.UserRoleInfo{{
						RoleID:           "role-1",
						RoleName:         "Editor",
						RoleDescription:  roleDescription,
						AssignedByUserID: nil,
						AssignedAt:       &fixedTime,
						ExpiresAt:        nil,
					}},
				}, nil).Once()
				userAccessRepo.On("GetUserWithPermissionsByID", mock.Anything, "user-1").Return(&types.UserWithPermissions{
					User: authmodels.User{
						ID:            "user-1",
						Name:          "Pat",
						Email:         "pat@example.com",
						EmailVerified: true,
						Metadata:      json.RawMessage("null"),
						CreatedAt:     fixedTime,
						UpdatedAt:     fixedTime,
					},
					Permissions: []types.UserPermissionInfo{{
						PermissionID:          "perm-1",
						PermissionKey:         "users.read",
						PermissionDescription: permissionDescription,
						GrantedAt:             &fixedTime,
					}},
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: &types.UserAuthorizationProfile{
				User: authmodels.User{
					ID:            "user-1",
					Name:          "Pat",
					Email:         "pat@example.com",
					EmailVerified: true,
					Metadata:      json.RawMessage("null"),
					CreatedAt:     fixedTime,
					UpdatedAt:     fixedTime,
				},
				Roles: []types.UserRoleInfo{{
					RoleID:           "role-1",
					RoleName:         "Editor",
					RoleDescription:  roleDescription,
					AssignedByUserID: nil,
					AssignedAt:       &fixedTime,
					ExpiresAt:        nil,
				}},
				Permissions: []types.UserPermissionInfo{{
					PermissionID:          "perm-1",
					PermissionKey:         "users.read",
					PermissionDescription: permissionDescription,
					GrantedAt:             &fixedTime,
				}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setupMock != nil {
				tc.setupMock(userRolesRepo, userAccessRepo)
			}

			useCase := newUserAccessUseCase(userRolesRepo, userAccessRepo)
			handler := NewGetUserAuthorizationProfileHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/users/"+tc.userID+"/authorization-profile", nil, nil)
			req.SetPathValue("user_id", tc.userID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				userRolesRepo.AssertExpectations(t)
				userAccessRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.UserAuthorizationProfile](t, reqCtx)
			assertUserAuthorizationProfileEqual(t, payload, *tc.expectedBody.(*types.UserAuthorizationProfile))

			userRolesRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}
