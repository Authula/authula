package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

// Tests

// TestCreateRole tests the CreateRole method with table-driven test cases
func TestCreateRole(t *testing.T) {
	testCases := []struct {
		name          string
		req           types.CreateRoleRequest
		mockErr       error
		expectedError string
	}{
		{
			name: "success with all fields",
			req: types.CreateRoleRequest{
				Name:        "Admin",
				Description: new("Administrator role"),
				IsSystem:    true,
			},
			expectedError: "",
		},
		{
			name: "success with minimal fields",
			req: types.CreateRoleRequest{
				Name: "User",
			},
			expectedError: "",
		},
		{
			name:          "empty name",
			req:           types.CreateRoleRequest{Name: ""},
			expectedError: "role name is required",
		},
		{
			name:          "whitespace only name",
			req:           types.CreateRoleRequest{Name: "   "},
			expectedError: "role name is required",
		},
		{
			name:          "service error",
			req:           types.CreateRoleRequest{Name: "Test"},
			mockErr:       errors.New("database error"),
			expectedError: "database error",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("CreateRole", mock.Anything, mock.Anything).Return(tt.mockErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			role, err := uc.CreateRole(context.Background(), tt.req)

			if tt.expectedError == "" {
				assert.NoError(t, err)
				assert.NotNil(t, role)
				assert.Equal(t, tt.req.Name, role.Name)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
				assert.Nil(t, role)
			}
		})
	}
}

// TestGetAllRoles tests the GetAllRoles method
func TestGetAllRoles(t *testing.T) {
	testCases := []struct {
		name           string
		mockResult     []types.Role
		mockErr        error
		expectedError  string
		expectedLength int
	}{
		{
			name: "success with roles",
			mockResult: []types.Role{
				{ID: "1", Name: "Admin"},
				{ID: "2", Name: "User"},
			},
			expectedLength: 2,
		},
		{
			name:           "empty list",
			mockResult:     []types.Role{},
			expectedLength: 0,
		},
		{
			name:          "service error",
			mockErr:       errors.New("database error"),
			expectedError: "database error",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("GetAllRoles", mock.Anything).Return(tt.mockResult, tt.mockErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			roles, err := uc.GetAllRoles(context.Background())

			if tt.expectedError == "" {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedLength, len(roles))
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestGetRoleByID tests the GetRoleByID method
func TestGetRoleByID(t *testing.T) {
	testCases := []struct {
		name            string
		roleID          string
		mockRoleResult  *types.Role
		mockRoleErr     error
		mockPermErr     error
		mockPermissions []types.UserPermissionInfo
		expectedError   string
		expectedHasRole bool
	}{
		{
			name:   "success with permissions",
			roleID: "role-1",
			mockRoleResult: &types.Role{
				ID:   "role-1",
				Name: "Admin",
			},
			mockPermissions: []types.UserPermissionInfo{
				{PermissionID: "perm-1", PermissionKey: "admin:read"},
			},
			expectedHasRole: true,
		},
		{
			name:          "empty roleID",
			roleID:        "",
			expectedError: "role_id is required",
		},
		{
			name:          "whitespace roleID",
			roleID:        "   ",
			expectedError: "role_id is required",
		},
		{
			name:           "role not found",
			roleID:         "nonexistent",
			mockRoleResult: nil,
			expectedError:  "role not found",
		},
		{
			name:          "service error on role fetch",
			roleID:        "role-1",
			mockRoleErr:   errors.New("db error"),
			expectedError: "db error",
		},
		{
			name:   "service error on permissions fetch",
			roleID: "role-1",
			mockRoleResult: &types.Role{
				ID:   "role-1",
				Name: "Admin",
			},
			mockPermErr:   errors.New("perm db error"),
			expectedError: "perm db error",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("GetRoleByID", mock.Anything, mock.Anything).Return(tt.mockRoleResult, tt.mockRoleErr).Maybe()
			mockService.On("GetRolePermissions", mock.Anything, mock.Anything).Return(tt.mockPermissions, tt.mockPermErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			result, err := uc.GetRoleByID(context.Background(), tt.roleID)

			if tt.expectedError == "" {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, len(tt.mockPermissions), len(result.Permissions))
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
				assert.Nil(t, result)
			}
		})
	}
}

// TestUpdateRole tests the UpdateRole method
func TestUpdateRole(t *testing.T) {
	testCases := []struct {
		name             string
		roleID           string
		req              types.UpdateRoleRequest
		mockRole         *types.Role
		mockGetErr       error
		mockUpdateErr    error
		expectedError    string
		shouldCallUpdate bool
	}{
		{
			name:   "update name only",
			roleID: "role-1",
			req: types.UpdateRoleRequest{
				Name: new("NewAdmin"),
			},
			mockRole:         &types.Role{ID: "role-1", Name: "Admin", IsSystem: false},
			shouldCallUpdate: true,
		},
		{
			name:   "update description only",
			roleID: "role-1",
			req: types.UpdateRoleRequest{
				Description: new("New description"),
			},
			mockRole:         &types.Role{ID: "role-1", Name: "Admin", IsSystem: false},
			shouldCallUpdate: true,
		},
		{
			name:   "update both",
			roleID: "role-1",
			req: types.UpdateRoleRequest{
				Name:        new("NewAdmin"),
				Description: new("New description"),
			},
			mockRole:         &types.Role{ID: "role-1", Name: "Admin", IsSystem: false},
			shouldCallUpdate: true,
		},
		{
			name:          "empty roleID",
			roleID:        "",
			expectedError: "role_id is required",
		},
		{
			name:   "whitespace roleID",
			roleID: "   ",
			req: types.UpdateRoleRequest{
				Name: new("Admin"),
			},
			expectedError: "role_id is required",
		},
		{
			name:          "no fields provided",
			roleID:        "role-1",
			req:           types.UpdateRoleRequest{},
			expectedError: "at least one field is required",
		},
		{
			name:   "empty name",
			roleID: "role-1",
			req: types.UpdateRoleRequest{
				Name: new("   "),
			},
			mockRole:      &types.Role{ID: "role-1", Name: "Admin", IsSystem: false},
			expectedError: "name is required",
		},
		{
			name:          "role not found",
			roleID:        "role-1",
			req:           types.UpdateRoleRequest{Name: new("Admin")},
			mockRole:      nil,
			expectedError: "role not found",
		},
		{
			name:   "system role protection",
			roleID: "role-1",
			req: types.UpdateRoleRequest{
				Name: new("NewAdmin"),
			},
			mockRole:      &types.Role{ID: "role-1", Name: "Admin", IsSystem: true},
			expectedError: "cannot update system role",
		},
		{
			name:          "service error on get",
			roleID:        "role-1",
			req:           types.UpdateRoleRequest{Name: new("Admin")},
			mockGetErr:    errors.New("db error"),
			expectedError: "db error",
		},
		{
			name:   "service error on update",
			roleID: "role-1",
			req: types.UpdateRoleRequest{
				Name: new("NewAdmin"),
			},
			mockRole:         &types.Role{ID: "role-1", Name: "Admin", IsSystem: false},
			mockUpdateErr:    errors.New("update failed"),
			expectedError:    "update failed",
			shouldCallUpdate: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("GetRoleByID", mock.Anything, mock.Anything).Return(tt.mockRole, tt.mockGetErr).Maybe()
			mockService.On("UpdateRole", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, tt.mockUpdateErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			role, err := uc.UpdateRole(context.Background(), tt.roleID, tt.req)

			if tt.expectedError == "" {
				assert.NoError(t, err)
				assert.NotNil(t, role)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
				assert.Nil(t, role)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestDeleteRole tests the DeleteRole method
func TestDeleteRole(t *testing.T) {
	testCases := []struct {
		name             string
		roleID           string
		mockRole         *types.Role
		mockGetErr       error
		mockCountResult  int
		mockCountErr     error
		mockDeleteErr    error
		expectedError    string
		shouldCallDelete bool
	}{
		{
			name:             "success",
			roleID:           "role-1",
			mockRole:         &types.Role{ID: "role-1", Name: "User", IsSystem: false},
			mockCountResult:  0,
			shouldCallDelete: true,
		},
		{
			name:          "empty roleID",
			roleID:        "",
			expectedError: "role_id is required",
		},
		{
			name:          "whitespace roleID",
			roleID:        "   ",
			expectedError: "role_id is required",
		},
		{
			name:          "role not found",
			roleID:        "role-1",
			mockRole:      nil,
			expectedError: "role not found",
		},
		{
			name:          "system role protection",
			roleID:        "role-1",
			mockRole:      &types.Role{ID: "role-1", Name: "Admin", IsSystem: true},
			expectedError: "cannot delete system role",
		},
		{
			name:          "service error on get",
			roleID:        "role-1",
			mockGetErr:    errors.New("db error"),
			expectedError: "db error",
		},
		{
			name:            "has user assignments",
			roleID:          "role-1",
			mockRole:        &types.Role{ID: "role-1", Name: "User", IsSystem: false},
			mockCountResult: 5,
			expectedError:   "role is assigned to one or more users",
		},
		{
			name:          "service error on count",
			roleID:        "role-1",
			mockRole:      &types.Role{ID: "role-1", Name: "User", IsSystem: false},
			mockCountErr:  errors.New("count error"),
			expectedError: "count error",
		},
		{
			name:             "service error on delete",
			roleID:           "role-1",
			mockRole:         &types.Role{ID: "role-1", Name: "User", IsSystem: false},
			mockCountResult:  0,
			mockDeleteErr:    errors.New("delete failed"),
			expectedError:    "delete failed",
			shouldCallDelete: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("GetRoleByID", mock.Anything, mock.Anything).Return(tt.mockRole, tt.mockGetErr).Maybe()
			mockService.On("CountUserAssignmentsByRoleID", mock.Anything, mock.Anything).Return(tt.mockCountResult, tt.mockCountErr).Maybe()
			mockService.On("DeleteRole", mock.Anything, mock.Anything).Return(true, tt.mockDeleteErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			err := uc.DeleteRole(context.Background(), tt.roleID)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestCreatePermission tests the CreatePermission method
func TestCreatePermission(t *testing.T) {
	testCases := []struct {
		name          string
		req           types.CreatePermissionRequest
		mockErr       error
		expectedError string
	}{
		{
			name: "success with all fields",
			req: types.CreatePermissionRequest{
				Key:         "admin:write",
				Description: new("Admin write access"),
				IsSystem:    true,
			},
			expectedError: "",
		},
		{
			name: "success with minimal fields",
			req: types.CreatePermissionRequest{
				Key: "user:read",
			},
			expectedError: "",
		},
		{
			name:          "empty key",
			req:           types.CreatePermissionRequest{Key: ""},
			expectedError: "permission key is required",
		},
		{
			name:          "whitespace only key",
			req:           types.CreatePermissionRequest{Key: "   "},
			expectedError: "permission key is required",
		},
		{
			name:          "service error",
			req:           types.CreatePermissionRequest{Key: "admin:read"},
			mockErr:       errors.New("database error"),
			expectedError: "database error",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("CreatePermission", mock.Anything, mock.Anything).Return(tt.mockErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			perm, err := uc.CreatePermission(context.Background(), tt.req)

			if tt.expectedError == "" {
				assert.NoError(t, err)
				assert.NotNil(t, perm)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
				assert.Nil(t, perm)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestGetAllPermissions tests the GetAllPermissions method
func TestGetAllPermissions(t *testing.T) {
	testCases := []struct {
		name           string
		mockResult     []types.Permission
		mockErr        error
		expectedError  string
		expectedLength int
	}{
		{
			name: "success with permissions",
			mockResult: []types.Permission{
				{ID: "1", Key: "admin:read"},
				{ID: "2", Key: "user:read"},
			},
			expectedLength: 2,
		},
		{
			name:           "empty list",
			mockResult:     []types.Permission{},
			expectedLength: 0,
		},
		{
			name:          "service error",
			mockErr:       errors.New("database error"),
			expectedError: "database error",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("GetAllPermissions", mock.Anything).Return(tt.mockResult, tt.mockErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			perms, err := uc.GetAllPermissions(context.Background())

			if tt.expectedError == "" {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedLength, len(perms))
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestUpdatePermission tests the UpdatePermission method
func TestUpdatePermission(t *testing.T) {
	testCases := []struct {
		name             string
		permissionID     string
		req              types.UpdatePermissionRequest
		mockPerm         *types.Permission
		mockGetErr       error
		mockUpdateErr    error
		expectedError    string
		shouldCallUpdate bool
	}{
		{
			name:         "success",
			permissionID: "perm-1",
			req: types.UpdatePermissionRequest{
				Description: new("New description"),
			},
			mockPerm:         &types.Permission{ID: "perm-1", Key: "admin:read", IsSystem: false},
			shouldCallUpdate: true,
		},
		{
			name:          "empty permissionID",
			permissionID:  "",
			expectedError: "permission_id is required",
		},
		{
			name:          "whitespace permissionID",
			permissionID:  "   ",
			expectedError: "permission_id is required",
		},
		{
			name:         "nil description",
			permissionID: "perm-1",
			req: types.UpdatePermissionRequest{
				Description: nil,
			},
			expectedError: "description is required",
		},
		{
			name:         "empty description",
			permissionID: "perm-1",
			req: types.UpdatePermissionRequest{
				Description: new("   "),
			},
			expectedError: "description is required",
		},
		{
			name:          "permission not found",
			permissionID:  "perm-1",
			req:           types.UpdatePermissionRequest{Description: new("New")},
			mockPerm:      nil,
			expectedError: "permission not found",
		},
		{
			name:         "system permission protection",
			permissionID: "perm-1",
			req: types.UpdatePermissionRequest{
				Description: new("New"),
			},
			mockPerm:      &types.Permission{ID: "perm-1", Key: "admin:read", IsSystem: true},
			expectedError: "cannot update system permission",
		},
		{
			name:          "service error on get",
			permissionID:  "perm-1",
			req:           types.UpdatePermissionRequest{Description: new("New")},
			mockGetErr:    errors.New("db error"),
			expectedError: "db error",
		},
		{
			name:         "service error on update",
			permissionID: "perm-1",
			req: types.UpdatePermissionRequest{
				Description: new("New"),
			},
			mockPerm:         &types.Permission{ID: "perm-1", Key: "admin:read", IsSystem: false},
			mockUpdateErr:    errors.New("update failed"),
			expectedError:    "update failed",
			shouldCallUpdate: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(tt.mockPerm, tt.mockGetErr).Maybe()
			mockService.On("UpdatePermissionDescription", mock.Anything, mock.Anything, mock.Anything).Return(true, tt.mockUpdateErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			perm, err := uc.UpdatePermission(context.Background(), tt.permissionID, tt.req)

			if tt.expectedError == "" {
				assert.NoError(t, err)
				assert.NotNil(t, perm)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
				assert.Nil(t, perm)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestDeletePermission tests the DeletePermission method
func TestDeletePermission(t *testing.T) {
	testCases := []struct {
		name             string
		permissionID     string
		mockPerm         *types.Permission
		mockGetErr       error
		mockCountResult  int
		mockCountErr     error
		mockDeleteErr    error
		expectedError    string
		shouldCallDelete bool
	}{
		{
			name:             "success",
			permissionID:     "perm-1",
			mockPerm:         &types.Permission{ID: "perm-1", Key: "user:read", IsSystem: false},
			mockCountResult:  0,
			shouldCallDelete: true,
		},
		{
			name:          "empty permissionID",
			permissionID:  "",
			expectedError: "permission_id is required",
		},
		{
			name:          "whitespace permissionID",
			permissionID:  "   ",
			expectedError: "permission_id is required",
		},
		{
			name:          "permission not found",
			permissionID:  "perm-1",
			mockPerm:      nil,
			expectedError: "permission not found",
		},
		{
			name:          "system permission protection",
			permissionID:  "perm-1",
			mockPerm:      &types.Permission{ID: "perm-1", Key: "admin:read", IsSystem: true},
			expectedError: "cannot delete system permission",
		},
		{
			name:          "service error on get",
			permissionID:  "perm-1",
			mockGetErr:    errors.New("db error"),
			expectedError: "db error",
		},
		{
			name:            "has role assignments",
			permissionID:    "perm-1",
			mockPerm:        &types.Permission{ID: "perm-1", Key: "user:read", IsSystem: false},
			mockCountResult: 3,
			expectedError:   "permission is in use by one or more roles",
		},
		{
			name:          "service error on count",
			permissionID:  "perm-1",
			mockPerm:      &types.Permission{ID: "perm-1", Key: "user:read", IsSystem: false},
			mockCountErr:  errors.New("count error"),
			expectedError: "count error",
		},
		{
			name:             "service error on delete",
			permissionID:     "perm-1",
			mockPerm:         &types.Permission{ID: "perm-1", Key: "user:read", IsSystem: false},
			mockCountResult:  0,
			mockDeleteErr:    errors.New("delete failed"),
			expectedError:    "delete failed",
			shouldCallDelete: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(tt.mockPerm, tt.mockGetErr).Maybe()
			mockService.On("CountRoleAssignmentsByPermissionID", mock.Anything, mock.Anything).Return(tt.mockCountResult, tt.mockCountErr).Maybe()
			mockService.On("DeletePermission", mock.Anything, mock.Anything).Return(true, tt.mockDeleteErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			err := uc.DeletePermission(context.Background(), tt.permissionID)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestAddPermissionToRole tests the AddPermissionToRole method
func TestAddPermissionToRole(t *testing.T) {
	testCases := []struct {
		name              string
		roleID            string
		permissionID      string
		mockRole          *types.Role
		mockRoleErr       error
		mockPerm          *types.Permission
		mockPermErr       error
		mockAddErr        error
		expectedError     string
		shouldCallAddRole bool
	}{
		{
			name:              "success",
			roleID:            "role-1",
			permissionID:      "perm-1",
			mockRole:          &types.Role{ID: "role-1", Name: "User", IsSystem: false},
			mockPerm:          &types.Permission{ID: "perm-1", Key: "user:read", IsSystem: false},
			shouldCallAddRole: true,
		},
		{
			name:          "empty roleID",
			roleID:        "",
			permissionID:  "perm-1",
			expectedError: "role_id is required",
		},
		{
			name:          "empty permissionID",
			roleID:        "role-1",
			permissionID:  "",
			expectedError: "permission_id is required",
		},
		{
			name:          "whitespace roleID",
			roleID:        "   ",
			permissionID:  "perm-1",
			expectedError: "role_id is required",
		},
		{
			name:          "whitespace permissionID",
			roleID:        "role-1",
			permissionID:  "   ",
			expectedError: "permission_id is required",
		},
		{
			name:          "role not found",
			roleID:        "role-1",
			permissionID:  "perm-1",
			mockRole:      nil,
			expectedError: "role not found",
		},
		{
			name:          "permission not found",
			roleID:        "role-1",
			permissionID:  "perm-1",
			mockRole:      &types.Role{ID: "role-1", Name: "User", IsSystem: false},
			mockPerm:      nil,
			expectedError: "permission not found",
		},
		{
			name:          "system role protection",
			roleID:        "role-1",
			permissionID:  "perm-1",
			mockRole:      &types.Role{ID: "role-1", Name: "Admin", IsSystem: true},
			mockPerm:      &types.Permission{ID: "perm-1", Key: "admin:read", IsSystem: false},
			expectedError: "cannot modify system role",
		},
		{
			name:          "system permission protection",
			roleID:        "role-1",
			permissionID:  "perm-1",
			mockRole:      &types.Role{ID: "role-1", Name: "User", IsSystem: false},
			mockPerm:      &types.Permission{ID: "perm-1", Key: "admin:read", IsSystem: true},
			expectedError: "cannot modify system permission",
		},
		{
			name:              "service error on add",
			roleID:            "role-1",
			permissionID:      "perm-1",
			mockRole:          &types.Role{ID: "role-1", Name: "User", IsSystem: false},
			mockPerm:          &types.Permission{ID: "perm-1", Key: "user:read", IsSystem: false},
			mockAddErr:        errors.New("add failed"),
			expectedError:     "add failed",
			shouldCallAddRole: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("GetRoleByID", mock.Anything, mock.Anything).Return(tt.mockRole, tt.mockRoleErr).Maybe()
			mockService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(tt.mockPerm, tt.mockPermErr).Maybe()
			mockService.On("AddRolePermission", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tt.mockAddErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			err := uc.AddPermissionToRole(context.Background(), tt.roleID, tt.permissionID, nil)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestRemovePermissionFromRole tests the RemovePermissionFromRole method
func TestRemovePermissionFromRole(t *testing.T) {
	testCases := []struct {
		name                 string
		roleID               string
		permissionID         string
		mockRole             *types.Role
		mockRoleErr          error
		mockPerm             *types.Permission
		mockPermErr          error
		mockRemoveErr        error
		expectedError        string
		shouldCallRemoveRole bool
	}{
		{
			name:                 "success",
			roleID:               "role-1",
			permissionID:         "perm-1",
			mockRole:             &types.Role{ID: "role-1", Name: "User", IsSystem: false},
			mockPerm:             &types.Permission{ID: "perm-1", Key: "user:read", IsSystem: false},
			shouldCallRemoveRole: true,
		},
		{
			name:          "empty roleID",
			roleID:        "",
			permissionID:  "perm-1",
			expectedError: "role_id is required",
		},
		{
			name:          "empty permissionID",
			roleID:        "role-1",
			permissionID:  "",
			expectedError: "permission_id is required",
		},
		{
			name:          "system role protection",
			roleID:        "role-1",
			permissionID:  "perm-1",
			mockRole:      &types.Role{ID: "role-1", Name: "Admin", IsSystem: true},
			mockPerm:      &types.Permission{ID: "perm-1", Key: "admin:read", IsSystem: false},
			expectedError: "cannot modify system role",
		},
		{
			name:          "system permission protection",
			roleID:        "role-1",
			permissionID:  "perm-1",
			mockRole:      &types.Role{ID: "role-1", Name: "User", IsSystem: false},
			mockPerm:      &types.Permission{ID: "perm-1", Key: "admin:read", IsSystem: true},
			expectedError: "cannot modify system permission",
		},
		{
			name:                 "service error on remove",
			roleID:               "role-1",
			permissionID:         "perm-1",
			mockRole:             &types.Role{ID: "role-1", Name: "User", IsSystem: false},
			mockPerm:             &types.Permission{ID: "perm-1", Key: "user:read", IsSystem: false},
			mockRemoveErr:        errors.New("remove failed"),
			expectedError:        "remove failed",
			shouldCallRemoveRole: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("GetRoleByID", mock.Anything, mock.Anything).Return(tt.mockRole, tt.mockRoleErr).Maybe()
			mockService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(tt.mockPerm, tt.mockPermErr).Maybe()
			mockService.On("RemoveRolePermission", mock.Anything, mock.Anything, mock.Anything).Return(tt.mockRemoveErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			err := uc.RemovePermissionFromRole(context.Background(), tt.roleID, tt.permissionID)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestReplaceRolePermissions tests the ReplaceRolePermissions method
func TestReplaceRolePermissions(t *testing.T) {
	testCases := []struct {
		name              string
		roleID            string
		permissionIDs     []string
		mockReplaceErr    error
		expectedError     string
		shouldCallReplace bool
	}{
		{
			name:              "success with permissions",
			roleID:            "role-1",
			permissionIDs:     []string{"perm-1", "perm-2"},
			shouldCallReplace: true,
		},
		{
			name:              "success with empty list",
			roleID:            "role-1",
			permissionIDs:     []string{},
			shouldCallReplace: true,
		},
		{
			name:          "empty roleID",
			roleID:        "",
			permissionIDs: []string{"perm-1"},
			expectedError: "role_id is required",
		},
		{
			name:          "whitespace roleID",
			roleID:        "   ",
			permissionIDs: []string{"perm-1"},
			expectedError: "role_id is required",
		},
		{
			name:              "deduplication",
			roleID:            "role-1",
			permissionIDs:     []string{"perm-1", "perm-1", "perm-2"},
			shouldCallReplace: true,
		},
		{
			name:              "filters empty strings",
			roleID:            "role-1",
			permissionIDs:     []string{"perm-1", "", "   ", "perm-2"},
			shouldCallReplace: true,
		},
		{
			name:              "service error",
			roleID:            "role-1",
			permissionIDs:     []string{"perm-1"},
			mockReplaceErr:    errors.New("db error"),
			expectedError:     "db error",
			shouldCallReplace: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("ReplaceRolePermissions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tt.mockReplaceErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			err := uc.ReplaceRolePermissions(context.Background(), tt.roleID, tt.permissionIDs, nil)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestReplaceUserRoles tests the ReplaceUserRoles method
func TestReplaceUserRoles(t *testing.T) {
	testCases := []struct {
		name              string
		userID            string
		roleIDs           []string
		mockReplaceErr    error
		expectedError     string
		shouldCallReplace bool
	}{
		{
			name:              "success with roles",
			userID:            "user-1",
			roleIDs:           []string{"role-1", "role-2"},
			shouldCallReplace: true,
		},
		{
			name:              "success with empty list",
			userID:            "user-1",
			roleIDs:           []string{},
			shouldCallReplace: true,
		},
		{
			name:          "empty userID",
			userID:        "",
			roleIDs:       []string{"role-1"},
			expectedError: "user_id is required",
		},
		{
			name:          "whitespace userID",
			userID:        "   ",
			roleIDs:       []string{"role-1"},
			expectedError: "user_id is required",
		},
		{
			name:              "deduplication",
			userID:            "user-1",
			roleIDs:           []string{"role-1", "role-1", "role-2"},
			shouldCallReplace: true,
		},
		{
			name:              "filters empty strings",
			userID:            "user-1",
			roleIDs:           []string{"role-1", "", "   ", "role-2"},
			shouldCallReplace: true,
		},
		{
			name:              "service error",
			userID:            "user-1",
			roleIDs:           []string{"role-1"},
			mockReplaceErr:    errors.New("db error"),
			expectedError:     "db error",
			shouldCallReplace: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("ReplaceUserRoles", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tt.mockReplaceErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			err := uc.ReplaceUserRoles(context.Background(), tt.userID, tt.roleIDs, nil)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestAssignRoleToUser tests the AssignRoleToUser method
func TestAssignRoleToUser(t *testing.T) {
	futureTime := time.Now().UTC().Add(24 * time.Hour)
	pastTime := time.Now().UTC().Add(-24 * time.Hour)

	testCases := []struct {
		name             string
		userID           string
		req              types.AssignUserRoleRequest
		mockAssignErr    error
		expectedError    string
		shouldCallAssign bool
	}{
		{
			name:   "success without expiry",
			userID: "user-1",
			req: types.AssignUserRoleRequest{
				RoleID: "role-1",
			},
			shouldCallAssign: true,
		},
		{
			name:   "success with future expiry",
			userID: "user-1",
			req: types.AssignUserRoleRequest{
				RoleID:    "role-1",
				ExpiresAt: &futureTime,
			},
			shouldCallAssign: true,
		},
		{
			name:          "empty userID",
			userID:        "",
			req:           types.AssignUserRoleRequest{RoleID: "role-1"},
			expectedError: "user_id is required",
		},
		{
			name:          "whitespace userID",
			userID:        "   ",
			req:           types.AssignUserRoleRequest{RoleID: "role-1"},
			expectedError: "user_id is required",
		},
		{
			name:   "empty roleID",
			userID: "user-1",
			req: types.AssignUserRoleRequest{
				RoleID: "",
			},
			expectedError: "role_id is required",
		},
		{
			name:   "past expiry date",
			userID: "user-1",
			req: types.AssignUserRoleRequest{
				RoleID:    "role-1",
				ExpiresAt: &pastTime,
			},
			expectedError: "expires_at must be in the future",
		},
		{
			name:   "service error on assign",
			userID: "user-1",
			req: types.AssignUserRoleRequest{
				RoleID: "role-1",
			},
			mockAssignErr:    errors.New("assign failed"),
			expectedError:    "assign failed",
			shouldCallAssign: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("AssignUserRole", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tt.mockAssignErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			err := uc.AssignRoleToUser(context.Background(), tt.userID, tt.req, nil)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestRemoveRoleFromUser tests the RemoveRoleFromUser method
func TestRemoveRoleFromUser(t *testing.T) {
	testCases := []struct {
		name             string
		userID           string
		roleID           string
		mockRemoveErr    error
		expectedError    string
		shouldCallRemove bool
	}{
		{
			name:             "success",
			userID:           "user-1",
			roleID:           "role-1",
			shouldCallRemove: true,
		},
		{
			name:          "empty userID",
			userID:        "",
			roleID:        "role-1",
			expectedError: "user_id is required",
		},
		{
			name:          "empty roleID",
			userID:        "user-1",
			roleID:        "",
			expectedError: "role_id is required",
		},
		{
			name:          "whitespace userID",
			userID:        "   ",
			roleID:        "role-1",
			expectedError: "user_id is required",
		},
		{
			name:          "whitespace roleID",
			userID:        "user-1",
			roleID:        "   ",
			expectedError: "role_id is required",
		},
		{
			name:             "service error",
			userID:           "user-1",
			roleID:           "role-1",
			mockRemoveErr:    errors.New("remove failed"),
			expectedError:    "remove failed",
			shouldCallRemove: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &tests.MockRolePermissionService{}
			mockService.On("RemoveUserRole", mock.Anything, mock.Anything, mock.Anything).Return(tt.mockRemoveErr).Maybe()

			uc := NewRolePermissionUseCase(mockService)
			err := uc.RemoveRoleFromUser(context.Background(), tt.userID, tt.roleID)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			}

			mockService.AssertExpectations(t)
		})
	}
}
