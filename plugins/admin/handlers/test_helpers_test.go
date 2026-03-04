package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func mustJSON(t *testing.T, payload any) []byte {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	return body
}

func newAdminHandlerRequest(t *testing.T, method, path string, body []byte) (*http.Request, *httptest.ResponseRecorder, *models.RequestContext) {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}

	req := httptest.NewRequest(method, path, reader)
	w := httptest.NewRecorder()
	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Path:           path,
		Method:         method,
		Headers:        req.Header,
		ClientIP:       "127.0.0.1",
		Values:         make(map[string]any),
	}

	ctx := models.SetRequestContext(context.Background(), reqCtx)
	req = req.WithContext(ctx)
	reqCtx.Request = req

	return req, w, reqCtx
}

func decodeResponseJSON(t *testing.T, reqCtx *models.RequestContext) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal(reqCtx.ResponseBody, &payload); err != nil {
		t.Fatalf("failed to decode response json: %v body=%s", err, string(reqCtx.ResponseBody))
	}

	return payload
}

func assertErrorMessage(t *testing.T, reqCtx *models.RequestContext, status int, message string) {
	t.Helper()

	if !reqCtx.Handled {
		t.Fatal("expected request to be marked as handled")
	}
	if reqCtx.ResponseStatus != status {
		t.Fatalf("expected status %d, got %d", status, reqCtx.ResponseStatus)
	}

	payload := decodeResponseJSON(t, reqCtx)
	if payload["message"] != message {
		t.Fatalf("expected message %q, got %v", message, payload["message"])
	}
}

type mockUsersUseCase struct {
	mock.Mock
}

func (m *mockUsersUseCase) GetAll(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) {
	args := m.Called(ctx, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.UsersPage), args.Error(1)
}

func (m *mockUsersUseCase) Create(ctx context.Context, request types.CreateUserRequest) (*models.User, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockUsersUseCase) GetByID(ctx context.Context, userID string) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockUsersUseCase) Update(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error) {
	args := m.Called(ctx, userID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockUsersUseCase) Delete(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

type mockRolePermissionUseCase struct {
	mock.Mock
}

func (m *mockRolePermissionUseCase) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Permission), args.Error(1)
}

func (m *mockRolePermissionUseCase) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Permission), args.Error(1)
}

func (m *mockRolePermissionUseCase) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	args := m.Called(ctx, permissionID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Permission), args.Error(1)
}

func (m *mockRolePermissionUseCase) DeletePermission(ctx context.Context, permissionID string) error {
	args := m.Called(ctx, permissionID)
	return args.Error(0)
}

func (m *mockRolePermissionUseCase) CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Role), args.Error(1)
}

func (m *mockRolePermissionUseCase) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Role), args.Error(1)
}

func (m *mockRolePermissionUseCase) GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error) {
	args := m.Called(ctx, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.RoleDetails), args.Error(1)
}

func (m *mockRolePermissionUseCase) UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	args := m.Called(ctx, roleID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Role), args.Error(1)
}

func (m *mockRolePermissionUseCase) DeleteRole(ctx context.Context, roleID string) error {
	args := m.Called(ctx, roleID)
	return args.Error(0)
}

func (m *mockRolePermissionUseCase) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	args := m.Called(ctx, roleID, permissionID, grantedByUserID)
	return args.Error(0)
}

func (m *mockRolePermissionUseCase) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	args := m.Called(ctx, roleID, permissionID)
	return args.Error(0)
}

func (m *mockRolePermissionUseCase) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	args := m.Called(ctx, roleID, permissionIDs, grantedByUserID)
	return args.Error(0)
}

func (m *mockRolePermissionUseCase) AssignRoleToUser(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	args := m.Called(ctx, userID, req, assignedByUserID)
	return args.Error(0)
}

func (m *mockRolePermissionUseCase) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *mockRolePermissionUseCase) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	args := m.Called(ctx, userID, roleIDs, assignedByUserID)
	return args.Error(0)
}

type mockUserRolesUseCase struct {
	mock.Mock
}

func (m *mockUserRolesUseCase) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.UserRoleInfo), args.Error(1)
}

func (m *mockUserRolesUseCase) GetUserEffectivePermissions(ctx context.Context, userID string) ([]types.UserPermissionInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.UserPermissionInfo), args.Error(1)
}

func (m *mockUserRolesUseCase) HasPermissions(ctx context.Context, userID string, requiredPermissions []string) (bool, error) {
	args := m.Called(ctx, userID, requiredPermissions)
	return args.Bool(0), args.Error(1)
}

func (m *mockUserRolesUseCase) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.UserWithRoles), args.Error(1)
}

func (m *mockUserRolesUseCase) GetUserWithPermissionsByID(ctx context.Context, userID string) (*types.UserWithPermissions, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.UserWithPermissions), args.Error(1)
}

func (m *mockUserRolesUseCase) GetUserAuthorizationProfile(ctx context.Context, userID string) (*types.UserAuthorizationProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.UserAuthorizationProfile), args.Error(1)
}

type mockImpersonationUseCase struct {
	mock.Mock
}

func (m *mockImpersonationUseCase) StartImpersonation(ctx context.Context, actorUserID string, actorSessionID *string, req types.StartImpersonationRequest) (*types.StartImpersonationResult, error) {
	args := m.Called(ctx, actorUserID, actorSessionID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.StartImpersonationResult), args.Error(1)
}

func (m *mockImpersonationUseCase) StopImpersonation(ctx context.Context, actorUserID string, request types.StopImpersonationRequest) error {
	args := m.Called(ctx, actorUserID, request)
	return args.Error(0)
}

func (m *mockImpersonationUseCase) GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Impersonation), args.Error(1)
}

func (m *mockImpersonationUseCase) GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error) {
	args := m.Called(ctx, impersonationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Impersonation), args.Error(1)
}

type mockStateUseCase struct {
	mock.Mock
}

func (m *mockStateUseCase) UpsertUserState(ctx context.Context, userID string, request types.UpsertUserStateRequest, actorUserID *string) (*types.AdminUserState, error) {
	args := m.Called(ctx, userID, request, actorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.AdminUserState), args.Error(1)
}

func (m *mockStateUseCase) BanUser(ctx context.Context, userID string, request types.BanUserRequest, actorUserID *string) (*types.AdminUserState, error) {
	args := m.Called(ctx, userID, request, actorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.AdminUserState), args.Error(1)
}

func (m *mockStateUseCase) UnbanUser(ctx context.Context, userID string) (*types.AdminUserState, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.AdminUserState), args.Error(1)
}

func (m *mockStateUseCase) GetUserState(ctx context.Context, userID string) (*types.AdminUserState, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.AdminUserState), args.Error(1)
}

func (m *mockStateUseCase) DeleteUserState(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockStateUseCase) GetBannedUserStates(ctx context.Context) ([]types.AdminUserState, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.AdminUserState), args.Error(1)
}

func (m *mockStateUseCase) UpsertSessionState(ctx context.Context, sessionID string, request types.UpsertSessionStateRequest, actorUserID *string) (*types.AdminSessionState, error) {
	args := m.Called(ctx, sessionID, request, actorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.AdminSessionState), args.Error(1)
}

func (m *mockStateUseCase) RevokeSession(ctx context.Context, sessionID string, reason *string, actorUserID *string) (*types.AdminSessionState, error) {
	args := m.Called(ctx, sessionID, reason, actorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.AdminSessionState), args.Error(1)
}

func (m *mockStateUseCase) GetUserAdminSessions(ctx context.Context, userID string) ([]types.AdminUserSession, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.AdminUserSession), args.Error(1)
}

func (m *mockStateUseCase) GetSessionState(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.AdminSessionState), args.Error(1)
}

func (m *mockStateUseCase) DeleteSessionState(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *mockStateUseCase) GetRevokedSessionStates(ctx context.Context) ([]types.AdminSessionState, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.AdminSessionState), args.Error(1)
}
