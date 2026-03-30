package handlers

import (
	"errors"
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

func TestGetAllPermissionsHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	description := new(string)
	*description = "read access"

	tests := []struct {
		name           string
		setupMock      func(*accesscontroltests.MockPermissionsRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name: "service error",
			setupMock: func(m *accesscontroltests.MockPermissionsRepository) {
				m.On("GetAllPermissions", mock.Anything).Return(([]types.Permission)(nil), constants.ErrForbidden).Once()
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   map[string]string{"message": "forbidden"},
		},
		{
			name: "success",
			setupMock: func(m *accesscontroltests.MockPermissionsRepository) {
				m.On("GetAllPermissions", mock.Anything).Return([]types.Permission{
					{
						ID:          "perm-1",
						Key:         "users.read",
						Description: description,
						IsSystem:    false,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
					{
						ID:          "perm-2",
						Key:         "users.write",
						Description: nil,
						IsSystem:    true,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: []types.Permission{
				{
					ID:          "perm-1",
					Key:         "users.read",
					Description: description,
					IsSystem:    false,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
				{
					ID:          "perm-2",
					Key:         "users.write",
					Description: nil,
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

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setupMock != nil {
				tc.setupMock(permissionsRepo)
			}

			useCase := newPermissionsUseCase(permissionsRepo, userAccessRepo)
			handler := NewGetAllPermissionsHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/permissions", nil, nil)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				permissionsRepo.AssertExpectations(t)
				userAccessRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[[]types.Permission](t, reqCtx)
			assertPermissionsEqual(t, payload, tc.expectedBody.([]types.Permission))

			permissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func TestCreatePermissionHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	description := new(string)
	*description = "create access"

	tests := []struct {
		name           string
		body           []byte
		setupMock      func(*accesscontroltests.MockPermissionsRepository)
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
			body: internaltests.MarshalToJSON(t, types.CreatePermissionRequest{
				Key:         "users.create",
				Description: description,
				IsSystem:    false,
			}),
			setupMock: func(m *accesscontroltests.MockPermissionsRepository) {
				m.On("CreatePermission", mock.Anything, mock.MatchedBy(func(permission *types.Permission) bool {
					return permission != nil && permission.Key == "users.create" && permission.Description != nil && *permission.Description == *description && !permission.IsSystem && permission.ID != ""
				})).Return(constants.ErrBadRequest).Once()
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]string{"message": "bad request"},
		},
		{
			name: "success",
			body: internaltests.MarshalToJSON(t, types.CreatePermissionRequest{
				Key:         "users.create",
				Description: description,
				IsSystem:    false,
			}),
			setupMock: func(m *accesscontroltests.MockPermissionsRepository) {
				m.On("CreatePermission", mock.Anything, mock.MatchedBy(func(permission *types.Permission) bool {
					return permission != nil && permission.Key == "users.create" && permission.Description != nil && *permission.Description == *description && !permission.IsSystem && permission.ID != ""
				})).Run(func(args mock.Arguments) {
					permission := args.Get(1).(*types.Permission)
					permission.ID = "perm-1"
					permission.CreatedAt = fixedTime
					permission.UpdatedAt = fixedTime
				}).Return(nil).Once()
			},
			expectedStatus: http.StatusCreated,
			expectedBody: types.CreatePermissionResponse{
				Permission: &types.Permission{
					ID:          "perm-1",
					Key:         "users.create",
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

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setupMock != nil {
				tc.setupMock(permissionsRepo)
			}

			useCase := newPermissionsUseCase(permissionsRepo, userAccessRepo)
			handler := NewCreatePermissionHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/permissions", tc.body, nil)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusCreated {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				permissionsRepo.AssertExpectations(t)
				userAccessRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.CreatePermissionResponse](t, reqCtx)
			assertCreatePermissionResponseEqual(t, payload, tc.expectedBody.(types.CreatePermissionResponse))

			permissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func TestGetPermissionByIDHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	description := new(string)
	*description = "read access"

	tests := []struct {
		name           string
		permissionID   string
		setupMock      func(*accesscontroltests.MockPermissionsRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:         "service error",
			permissionID: "perm-404",
			setupMock: func(m *accesscontroltests.MockPermissionsRepository) {
				m.On("GetPermissionByID", mock.Anything, "perm-404").Return((*types.Permission)(nil), constants.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   map[string]string{"message": "not found"},
		},
		{
			name:         "success",
			permissionID: "perm-1",
			setupMock: func(m *accesscontroltests.MockPermissionsRepository) {
				m.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{
					ID:          "perm-1",
					Key:         "users.read",
					Description: description,
					IsSystem:    false,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: &types.Permission{
				ID:          "perm-1",
				Key:         "users.read",
				Description: description,
				IsSystem:    false,
				CreatedAt:   fixedTime,
				UpdatedAt:   fixedTime,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setupMock != nil {
				tc.setupMock(permissionsRepo)
			}

			useCase := newPermissionsUseCase(permissionsRepo, userAccessRepo)
			handler := NewGetPermissionByIDHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/permissions/"+tc.permissionID, nil, nil)
			req.SetPathValue("permission_id", tc.permissionID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				permissionsRepo.AssertExpectations(t)
				userAccessRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.Permission](t, reqCtx)
			assertPermissionEqual(t, payload, *tc.expectedBody.(*types.Permission))

			permissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func TestUpdatePermissionHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	updatedDescription := "updated read access"
	updatedDescriptionPtr := &updatedDescription
	existingDescription := new(string)
	*existingDescription = "read access"

	tests := []struct {
		name           string
		permissionID   string
		body           []byte
		setupMock      func(*accesscontroltests.MockPermissionsRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:           "invalid request body",
			permissionID:   "perm-1",
			body:           []byte("{invalid json"),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   map[string]string{"message": "invalid request body"},
		},
		{
			name:         "service error",
			permissionID: "perm-1",
			body:         internaltests.MarshalToJSON(t, types.UpdatePermissionRequest{Description: updatedDescriptionPtr}),
			setupMock: func(m *accesscontroltests.MockPermissionsRepository) {
				m.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read", Description: existingDescription, IsSystem: false}, nil).Once()
				m.On("UpdatePermission", mock.Anything, "perm-1", mock.MatchedBy(func(description *string) bool {
					return description != nil && *description == *updatedDescriptionPtr
				})).Return(false, constants.ErrBadRequest).Once()
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]string{"message": "bad request"},
		},
		{
			name:         "success",
			permissionID: "perm-1",
			body:         internaltests.MarshalToJSON(t, types.UpdatePermissionRequest{Description: updatedDescriptionPtr}),
			setupMock: func(m *accesscontroltests.MockPermissionsRepository) {
				m.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read", Description: existingDescription, IsSystem: false}, nil).Once()
				m.On("UpdatePermission", mock.Anything, "perm-1", mock.MatchedBy(func(description *string) bool {
					return description != nil && *description == *updatedDescriptionPtr
				})).Return(true, nil).Once()
				m.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{
					ID:          "perm-1",
					Key:         "users.read",
					Description: updatedDescriptionPtr,
					IsSystem:    false,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: types.UpdatePermissionResponse{
				Permission: &types.Permission{
					ID:          "perm-1",
					Key:         "users.read",
					Description: updatedDescriptionPtr,
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

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setupMock != nil {
				tc.setupMock(permissionsRepo)
			}

			useCase := newPermissionsUseCase(permissionsRepo, userAccessRepo)
			handler := NewUpdatePermissionHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, "/permissions/"+tc.permissionID, tc.body, nil)
			req.SetPathValue("permission_id", tc.permissionID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				permissionsRepo.AssertExpectations(t)
				userAccessRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.UpdatePermissionResponse](t, reqCtx)
			assertUpdatePermissionResponseEqual(t, payload, tc.expectedBody.(types.UpdatePermissionResponse))

			permissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func TestDeletePermissionHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		permissionID   string
		setupMock      func(*accesscontroltests.MockPermissionsRepository, *accesscontroltests.MockUserAccessRepository)
		expectedStatus int
		expectedBody   any
	}{
		{
			name:         "service error",
			permissionID: "perm-1",
			setupMock: func(m *accesscontroltests.MockPermissionsRepository, userAccessRepo *accesscontroltests.MockUserAccessRepository) {
				m.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read", IsSystem: false}, nil).Once()
				userAccessRepo.On("CountRoleAssignmentsByPermissionID", mock.Anything, "perm-1").Return(0, nil).Once()
				m.On("DeletePermission", mock.Anything, "perm-1").Return(false, errors.New("database error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   map[string]string{"message": "database error"},
		},
		{
			name:         "success",
			permissionID: "perm-1",
			setupMock: func(m *accesscontroltests.MockPermissionsRepository, u *accesscontroltests.MockUserAccessRepository) {
				m.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read", IsSystem: false}, nil).Once()
				u.On("CountRoleAssignmentsByPermissionID", mock.Anything, "perm-1").Return(0, nil).Once()
				m.On("DeletePermission", mock.Anything, "perm-1").Return(true, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   types.DeletePermissionResponse{Message: "permission deleted"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setupMock != nil {
				tc.setupMock(permissionsRepo, userAccessRepo)
			}

			useCase := newPermissionsUseCase(permissionsRepo, userAccessRepo)
			handler := NewDeletePermissionHandler(useCase)
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, "/permissions/"+tc.permissionID, nil, nil)
			req.SetPathValue("permission_id", tc.permissionID)

			handler.Handler()(w, req)

			if tc.expectedStatus != http.StatusOK {
				internaltests.AssertErrorMessage(t, reqCtx, tc.expectedStatus, tc.expectedBody.(map[string]string)["message"])
				permissionsRepo.AssertExpectations(t)
				userAccessRepo.AssertExpectations(t)
				return
			}

			if reqCtx.ResponseStatus != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, reqCtx.ResponseStatus)
			}

			payload := internaltests.DecodeResponseJSON[types.DeletePermissionResponse](t, reqCtx)
			assertDeletePermissionResponseEqual(t, payload, tc.expectedBody.(types.DeletePermissionResponse))

			permissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func newPermissionsUseCase(permissionsRepo *accesscontroltests.MockPermissionsRepository, userAccessRepo *accesscontroltests.MockUserAccessRepository) *usecases.PermissionsUseCase {
	return usecases.NewPermissionsUseCase(services.NewPermissionsService(permissionsRepo, userAccessRepo))
}

func assertPermissionsEqual(t *testing.T, got []types.Permission, want []types.Permission) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d permissions, got %d", len(want), len(got))
	}

	for i := range want {
		assertPermissionEqual(t, got[i], want[i])
	}
}

func assertPermissionEqual(t *testing.T, got types.Permission, want types.Permission) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("expected id %q, got %q", want.ID, got.ID)
	}
	if got.Key != want.Key {
		t.Fatalf("expected key %q, got %q", want.Key, got.Key)
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

func assertCreatePermissionResponseEqual(t *testing.T, got types.CreatePermissionResponse, want types.CreatePermissionResponse) {
	t.Helper()

	if got.Permission == nil || want.Permission == nil {
		if got.Permission != want.Permission {
			t.Fatalf("expected permission %#v, got %#v", want.Permission, got.Permission)
		}
		return
	}

	assertPermissionEqual(t, *got.Permission, *want.Permission)
}

func assertUpdatePermissionResponseEqual(t *testing.T, got types.UpdatePermissionResponse, want types.UpdatePermissionResponse) {
	t.Helper()

	if got.Permission == nil || want.Permission == nil {
		if got.Permission != want.Permission {
			t.Fatalf("expected permission %#v, got %#v", want.Permission, got.Permission)
		}
		return
	}

	assertPermissionEqual(t, *got.Permission, *want.Permission)
}

func assertDeletePermissionResponseEqual(t *testing.T, got types.DeletePermissionResponse, want types.DeletePermissionResponse) {
	t.Helper()

	if got.Message != want.Message {
		t.Fatalf("expected message %q, got %q", want.Message, got.Message)
	}
}

func timesEqual(left, right time.Time) bool {
	return left.Equal(right)
}

func stringsEqualPtr(left, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}
