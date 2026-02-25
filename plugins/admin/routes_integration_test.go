package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

// Automatically tests all admin plugin routes when RBAC isn't applied so no need to test this scenario for each individual route.
func TestAdminRouteIntegration_AllAdminRoutes_WithoutAuthentication_ReturnUnauthorized(t *testing.T) {
	f := newAdminFixture(t)
	f.ApplyRBACMappingsForAllPluginRoutes("admin.read")

	for _, route := range f.Plugin.Routes() {
		path := "/auth" + internaltests.ConcreteRoutePath(route.Path)
		w := f.JSONRequest(route.Method, path, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "route %s %s should return 401 for unauthenticated requests", route.Method, route.Path)
		envelope := internaltests.DecodeJSONResponse(t, w)
		assert.NotEmpty(t, internaltests.GetString(t, envelope, "message"))
	}
}

// User management

func TestAdminRouteIntegration_GetAllUsers_WithAdminReadPermission_ReturnsOK(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("user-1", "john.doe@example.com")
	f.GrantPermissionToUser("user-1", "admin.read")
	f.AuthenticateAs("user-1")
	w := f.JSONRequest(http.MethodGet, "/auth/admin/users", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	envelope := internaltests.DecodeJSONResponse(t, w)
	internaltests.AssertHasKey(t, envelope, "next_cursor")
	internaltests.AssertHasKey(t, envelope, "data")
}

func TestAdminRouteIntegration_GetAllUsers_Pagination(t *testing.T) {
	t.Parallel()

	f := newAdminFixture(t)

	f.SeedUser("user-1", "alice@example.com")
	f.SeedUser("user-2", "bob@example.com")
	f.SeedUser("user-3", "charlie@example.com")

	f.GrantPermissionToUser("user-1", "admin.read")
	f.AuthenticateAs("user-1")

	t.Run("should respect limit and return cursor", func(t *testing.T) {
		w := f.JSONRequest(http.MethodGet, "/auth/admin/users?limit=2", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		resp := internaltests.DecodeJSONResponse(t, w)

		data := resp["data"].([]any)
		assert.Equal(t, 2, len(data))
		assert.NotNil(t, resp["next_cursor"])
	})

	t.Run("should return 400 for invalid limit", func(t *testing.T) {
		w := f.JSONRequest(http.MethodGet, "/auth/admin/users?limit=abc", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAdminRouteIntegration_CreateUser_WithUsersWritePermission_ReturnsCreated(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("integration-user-writer", "integration-user-writer@example.com")
	f.GrantPermissionToUser("integration-user-writer", "users.write")
	f.AuthenticateAs("integration-user-writer")

	req := admintypes.CreateUserRequest{
		Name:  "Created By Admin",
		Email: "created-by-admin@example.com",
	}
	w := f.JSONRequest(http.MethodPost, "/auth/admin/users", req)

	assert.Equal(t, http.StatusCreated, w.Code)

	envelope := internaltests.DecodeJSONResponse(t, w)
	data := internaltests.GetMap(t, envelope, "user")
	assert.Equal(t, "created-by-admin@example.com", data["email"])

	// Verify persistence
	userID := data["id"].(string)
	user, err := f.Plugin.Api.GetUserByID(context.Background(), userID)
	assert.NoError(t, err)
	assert.Equal(t, "Created By Admin", user.Name)
}

func TestAdminRouteIntegration_GetUserByID_WithAdminReadPermission_ReturnsOK(t *testing.T) {
	f := newAdminFixture(t)
	targetID := f.SeedUser("target-user", "target@example.com")
	f.SeedUser("admin-user", "admin@example.com")
	f.GrantPermissionToUser("admin-user", "admin.read")
	f.AuthenticateAs("admin-user")

	w := f.JSONRequest(http.MethodGet, "/auth/admin/users/"+targetID, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := internaltests.DecodeJSONResponse(t, w)
	user := internaltests.GetMap(t, resp, "user")
	assert.Equal(t, targetID, user["id"])
	assert.Equal(t, "target@example.com", user["email"])
}

func TestAdminRouteIntegration_UpdateUser_WithUsersWritePermission_ReturnsOK(t *testing.T) {
	f := newAdminFixture(t)
	targetID := f.SeedUser("target-user", "target@example.com")
	f.SeedUser("admin-user", "admin@example.com")
	f.GrantPermissionToUser("admin-user", "users.write")
	f.AuthenticateAs("admin-user")

	newName := "Updated Name"
	req := admintypes.UpdateUserRequest{
		Name: &newName,
	}
	w := f.JSONRequest(http.MethodPatch, "/auth/admin/users/"+targetID, req)

	assert.Equal(t, http.StatusOK, w.Code)
	envelope := internaltests.DecodeJSONResponse(t, w)
	user := internaltests.GetMap(t, envelope, "user")
	assert.Equal(t, targetID, user["id"])
	assert.Equal(t, "Updated Name", user["name"])

	// Verify persistence
	updatedUser, err := f.Plugin.Api.GetUserByID(context.Background(), targetID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", updatedUser.Name)
}

func TestAdminRouteIntegration_DeleteUser_WithUsersWritePermission_ReturnsOK(t *testing.T) {
	f := newAdminFixture(t)
	targetID := f.SeedUser("target-user", "target@example.com")
	f.SeedUser("admin-user", "admin@example.com")
	f.GrantPermissionToUser("admin-user", "users.write")
	f.AuthenticateAs("admin-user")

	w := f.JSONRequest(http.MethodDelete, "/auth/admin/users/"+targetID, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	envelope := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "user deleted", internaltests.GetString(t, envelope, "message"))

	// Verify hard delete
	user, err := f.Plugin.Api.GetUserByID(context.Background(), targetID)
	assert.NoError(t, err)
	assert.Nil(t, user)
}

// Roles and permissions

func TestAdminRouteIntegration_RolesCollectionAndItem_CRUD_WithValidPayloads_ReturnExpectedStatuses(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("admin-user", "admin@example.com")
	f.GrantPermissionToUser("admin-user", "roles.write")
	f.GrantPermissionToUser("admin-user", "admin.read")
	f.AuthenticateAs("admin-user")

	// Create
	createReq := admintypes.CreateRoleRequest{Name: "test-role"}
	w := f.JSONRequest(http.MethodPost, "/auth/admin/roles", createReq)
	assert.Equal(t, http.StatusCreated, w.Code)

	createResp := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "role created", internaltests.GetString(t, createResp, "message"))
	role := internaltests.GetMap(t, createResp, "data")
	roleID := role["id"].(string)

	// List
	w = f.JSONRequest(http.MethodGet, "/auth/admin/roles", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	listResp := internaltests.DecodeJSONResponse(t, w)
	internaltests.AssertHasKey(t, listResp, "data")

	// Get
	w = f.JSONRequest(http.MethodGet, "/auth/admin/roles/"+roleID, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	getResp := internaltests.DecodeJSONResponse(t, w)
	roleData := internaltests.GetMap(t, getResp, "data")
	if roleData["id"] != nil {
		assert.Equal(t, roleID, roleData["id"])
	} else {
		role := internaltests.GetMap(t, roleData, "role")
		assert.Equal(t, roleID, role["id"])
	}

	// Update
	newName := "updated-role"
	updateReq := admintypes.UpdateRoleRequest{Name: &newName}
	w = f.JSONRequest(http.MethodPatch, "/auth/admin/roles/"+roleID, updateReq)
	assert.Equal(t, http.StatusOK, w.Code)
	updateResp := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "role updated", internaltests.GetString(t, updateResp, "message"))
	updatedRole := internaltests.GetMap(t, updateResp, "data")
	assert.Equal(t, "updated-role", updatedRole["name"])

	// Verify persistence
	roleDetails, err := f.Plugin.Api.GetRoleByID(context.Background(), roleID)
	assert.NoError(t, err)
	assert.Equal(t, "updated-role", roleDetails.Role.Name)

	// Delete
	w = f.JSONRequest(http.MethodDelete, "/auth/admin/roles/"+roleID, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	deleteResp := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "role deleted", internaltests.GetString(t, deleteResp, "message"))

	// Verify delete
	_, err = f.Plugin.Api.GetRoleByID(context.Background(), roleID)
	assert.Error(t, err)
}

func TestAdminRouteIntegration_GetAllPermissions_WithAuthenticationButNoPermission_ReturnsForbidden(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("integration-no-perm", "integration-no-perm@example.com")
	f.AuthenticateAs("integration-no-perm")
	w := f.JSONRequest(http.MethodGet, "/auth/admin/permissions", nil)

	assert.Equal(t, http.StatusForbidden, w.Code)
	envelope := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "Forbidden", internaltests.GetString(t, envelope, "message"))
}

func TestAdminRouteIntegration_GetAllPermissions_WithAdminReadPermission_ReturnsOK(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("integration-admin", "integration-admin@example.com")
	f.GrantPermissionToUser("integration-admin", "admin.read")
	f.ApplyRBACMappingsForAllPluginRoutes("admin.read")
	f.AuthenticateAs("integration-admin")
	w := f.JSONRequest(http.MethodGet, "/auth/admin/permissions", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	envelope := internaltests.DecodeJSONResponse(t, w)
	internaltests.AssertHasKey(t, envelope, "data")
}

func TestAdminRouteIntegration_CreatePermission_WithRolesWritePermission_ReturnsCreated(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("integration-perm-create-admin", "integration-perm-create-admin@example.com")
	f.GrantPermissionToUser("integration-perm-create-admin", "roles.write")
	f.AuthenticateAs("integration-perm-create-admin")

	body := admintypes.CreatePermissionRequest{
		Key:         "admin.permissions.create.route",
		Description: new(string),
	}
	*body.Description = "created via route"
	w := f.JSONRequest(http.MethodPost, "/auth/admin/permissions", body)

	assert.Equal(t, http.StatusCreated, w.Code)
	envelope := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "permission created", internaltests.GetString(t, envelope, "message"))
	data := internaltests.GetMap(t, envelope, "data")
	assert.Equal(t, "admin.permissions.create.route", data["key"])

	// Verify persistence
	allPerms, err := f.Plugin.Api.GetAllPermissions(context.Background())
	assert.NoError(t, err)
	found := false
	for _, p := range allPerms {
		if p.Key == "admin.permissions.create.route" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestAdminRouteIntegration_CreatePermission_WithInvalidPayload_ReturnsUnprocessableEntity(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("integration-perm-invalid-admin", "integration-perm-invalid-admin@example.com")
	f.GrantPermissionToUser("integration-perm-invalid-admin", "roles.write")
	f.AuthenticateAs("integration-perm-invalid-admin")

	w := f.Request(http.MethodPost, "/auth/admin/permissions", bytes.NewBufferString("{"))

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	envelope := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "invalid request body", internaltests.GetString(t, envelope, "message"))
}

func TestAdminRouteIntegration_UpdatePermission_WithRolesWritePermission_ReturnsOK(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("integration-perm-update-admin", "integration-perm-update-admin@example.com")
	f.GrantPermissionToUser("integration-perm-update-admin", "roles.write")
	f.AuthenticateAs("integration-perm-update-admin")

	description := "old description"
	permission, err := f.Plugin.Api.CreatePermission(context.Background(), admintypes.CreatePermissionRequest{
		Key:         "admin.permissions.update.route",
		Description: &description,
	})
	assert.NoError(t, err)

	newDesc := "updated description"
	req := admintypes.UpdatePermissionRequest{Description: &newDesc}
	w := f.JSONRequest(http.MethodPatch, "/auth/admin/permissions/"+permission.ID, req)

	assert.Equal(t, http.StatusOK, w.Code)
	envelope := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "permission updated", internaltests.GetString(t, envelope, "message"))
	data := internaltests.GetMap(t, envelope, "data")
	assert.Equal(t, permission.ID, data["id"])

	// Verify persistence
	updatedPermission, err := f.Plugin.Api.GetAllPermissions(context.Background())
	assert.NoError(t, err)

	var updated *admintypes.Permission
	for i := range updatedPermission {
		if updatedPermission[i].ID == permission.ID {
			updated = &updatedPermission[i]
			break
		}
	}
	assert.NotNil(t, updated)
	assert.Equal(t, "updated description", *updated.Description)
}

func TestAdminRouteIntegration_DeletePermission_WhenAssignedToRole_ReturnsConflict(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("integration-perm-delete-assigned-admin", "integration-perm-delete-assigned-admin@example.com")
	f.GrantPermissionToUser("integration-perm-delete-assigned-admin", "roles.write")
	f.AuthenticateAs("integration-perm-delete-assigned-admin")

	permission, err := f.Plugin.Api.CreatePermission(context.Background(), admintypes.CreatePermissionRequest{Key: "admin.permissions.delete.assigned"})
	assert.NoError(t, err)

	role, err := f.Plugin.Api.CreateRole(context.Background(), admintypes.CreateRoleRequest{Name: "permission-delete-assigned-role"})
	assert.NoError(t, err)

	err = f.Plugin.Api.ReplaceRolePermissions(context.Background(), role.ID, []string{permission.ID}, nil)
	assert.NoError(t, err)

	w := f.JSONRequest(http.MethodDelete, "/auth/admin/permissions/"+permission.ID, nil)

	assert.Equal(t, http.StatusConflict, w.Code)
	envelope := internaltests.DecodeJSONResponse(t, w)
	assert.NotEmpty(t, internaltests.GetString(t, envelope, "message"))
}

func TestAdminRouteIntegration_DeletePermission_WhenSystemPermission_ReturnsBadRequest(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("integration-perm-delete-system-admin", "integration-perm-delete-system-admin@example.com")
	f.GrantPermissionToUser("integration-perm-delete-system-admin", "roles.write")
	f.AuthenticateAs("integration-perm-delete-system-admin")

	permission, err := f.Plugin.Api.CreatePermission(context.Background(), admintypes.CreatePermissionRequest{
		Key:      "admin.permissions.delete.system",
		IsSystem: true,
	})
	assert.NoError(t, err)

	w := f.JSONRequest(http.MethodDelete, "/auth/admin/permissions/"+permission.ID, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	envelope := internaltests.DecodeJSONResponse(t, w)
	assert.NotEmpty(t, internaltests.GetString(t, envelope, "message"))
}

func TestAdminRouteIntegration_RolePermissions_AddReplaceRemove_WithRolesWritePermission_ReturnsOK(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("admin-user", "admin@example.com")
	f.GrantPermissionToUser("admin-user", "roles.write")
	f.AuthenticateAs("admin-user")

	role, _ := f.Plugin.Api.CreateRole(context.Background(), admintypes.CreateRoleRequest{Name: "test-role"})
	permission, _ := f.Plugin.Api.CreatePermission(context.Background(), admintypes.CreatePermissionRequest{Key: "test-perm"})

	// Add
	addReq := admintypes.AddRolePermissionRequest{PermissionID: permission.ID}
	w := f.JSONRequest(http.MethodPost, "/auth/admin/roles/"+role.ID+"/permissions", addReq)
	assert.Equal(t, http.StatusOK, w.Code)
	addResp := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "permission assigned to role", internaltests.GetString(t, addResp, "message"))

	// Replace
	replaceReq := admintypes.ReplaceRolePermissionsRequest{PermissionIDs: []string{permission.ID}}
	w = f.JSONRequest(http.MethodPut, "/auth/admin/roles/"+role.ID+"/permissions", replaceReq)
	assert.Equal(t, http.StatusOK, w.Code)
	replaceResp := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "role permissions replaced", internaltests.GetString(t, replaceResp, "message"))

	// Remove
	w = f.JSONRequest(http.MethodDelete, "/auth/admin/roles/"+role.ID+"/permissions/"+permission.ID, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	removeResp := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "permission removed from role", internaltests.GetString(t, removeResp, "message"))
}

// User roles and permissions

func TestAdminRouteIntegration_UserRolesAndEffectivePermissions_AssignListReplaceRemove_ReturnsOK(t *testing.T) {
	f := newAdminFixture(t)
	targetID := f.SeedUser("target-user", "target@example.com")
	f.SeedUser("admin-user", "admin@example.com")
	f.GrantPermissionToUser("admin-user", "roles.write")
	f.GrantPermissionToUser("admin-user", "admin.read")
	f.AuthenticateAs("admin-user")

	role, _ := f.Plugin.Api.CreateRole(context.Background(), admintypes.CreateRoleRequest{Name: "test-role"})

	// Assign
	assignReq := admintypes.AssignUserRoleRequest{RoleID: role.ID}
	w := f.JSONRequest(http.MethodPost, "/auth/admin/users/"+targetID+"/roles", assignReq)
	assert.Equal(t, http.StatusOK, w.Code)
	assignResp := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "role assigned", internaltests.GetString(t, assignResp, "message"))

	// List
	w = f.JSONRequest(http.MethodGet, "/auth/admin/users/"+targetID+"/roles", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	listResp := internaltests.DecodeJSONResponse(t, w)
	internaltests.AssertHasKey(t, listResp, "data")
	rolesData := listResp["data"].([]any)
	assert.Len(t, rolesData, 1)

	// Effective Permissions
	w = f.JSONRequest(http.MethodGet, "/auth/admin/users/"+targetID+"/permissions", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	permissionsResp := internaltests.DecodeJSONResponse(t, w)
	internaltests.AssertHasKey(t, permissionsResp, "data")

	// Replace
	replaceReq := admintypes.ReplaceUserRolesRequest{RoleIDs: []string{role.ID}}
	w = f.JSONRequest(http.MethodPut, "/auth/admin/users/"+targetID+"/roles", replaceReq)
	assert.Equal(t, http.StatusOK, w.Code)
	replaceResp := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "user roles replaced", internaltests.GetString(t, replaceResp, "message"))

	// Verify persistence via internal API
	userRoles, err := f.Plugin.Api.GetUserRoles(context.Background(), targetID)
	assert.NoError(t, err)
	assert.Len(t, userRoles, 1)
	assert.Equal(t, role.ID, userRoles[0].RoleID)

	// Remove
	w = f.JSONRequest(http.MethodDelete, "/auth/admin/users/"+targetID+"/roles/"+role.ID, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	removeResp := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "role removed", internaltests.GetString(t, removeResp, "message"))

	// Verify final state
	userRoles, err = f.Plugin.Api.GetUserRoles(context.Background(), targetID)
	assert.NoError(t, err)
	assert.Empty(t, userRoles)
}

// User state

func TestAdminRouteIntegration_UserState_UpsertGetBanListUnbanDelete_ReturnsOK(t *testing.T) {
	f := newAdminFixture(t)
	targetID := f.SeedUser("target-user", "target@example.com")
	f.SeedUser("admin-user", "admin@example.com")
	f.GrantPermissionToUser("admin-user", "users.write")
	f.GrantPermissionToUser("admin-user", "admin.read")
	f.AuthenticateAs("admin-user")

	// Upsert State
	upsertReq := admintypes.UpsertUserStateRequest{IsBanned: false}
	w := f.JSONRequest(http.MethodPost, "/auth/admin/users/"+targetID+"/state", upsertReq)
	assert.Equal(t, http.StatusOK, w.Code)
	upsertResp := internaltests.DecodeJSONResponse(t, w)
	_ = internaltests.GetMap(t, upsertResp, "data")

	// Get State
	w = f.JSONRequest(http.MethodGet, "/auth/admin/users/"+targetID+"/state", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	getResp := internaltests.DecodeJSONResponse(t, w)
	_ = internaltests.GetMap(t, getResp, "data")

	// Ban
	reason := "test"
	banReq := admintypes.BanUserRequest{Reason: &reason}
	w = f.JSONRequest(http.MethodPost, "/auth/admin/users/"+targetID+"/ban", banReq)
	assert.Equal(t, http.StatusOK, w.Code)
	banResp := internaltests.DecodeJSONResponse(t, w)
	_ = internaltests.GetMap(t, banResp, "data")

	// Verify ban status via internal API
	userState, err := f.Plugin.Api.GetUserState(context.Background(), targetID)
	assert.NoError(t, err)
	assert.True(t, userState.IsBanned)
	assert.Equal(t, "test", *userState.BannedReason)

	// List Banned
	w = f.JSONRequest(http.MethodGet, "/auth/admin/users/states/banned", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	bannedResp := internaltests.DecodeJSONResponse(t, w)
	internaltests.AssertHasKey(t, bannedResp, "data")

	// Unban
	w = f.JSONRequest(http.MethodPost, "/auth/admin/users/"+targetID+"/unban", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	unbanResp := internaltests.DecodeJSONResponse(t, w)
	_ = internaltests.GetMap(t, unbanResp, "data")

	// Verify unban status via internal API
	userState, err = f.Plugin.Api.GetUserState(context.Background(), targetID)
	assert.NoError(t, err)
	assert.False(t, userState.IsBanned)

	// Delete State
	w = f.JSONRequest(http.MethodDelete, "/auth/admin/users/"+targetID+"/state", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	deleteResp := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "user state deleted", internaltests.GetString(t, deleteResp, "message"))
}

func TestAdminRouteIntegration_GetUserSessions_WithAdminReadPermission_ReturnsOK(t *testing.T) {
	f := newAdminFixture(t)
	targetID := f.SeedUser("target-user", "target@example.com")
	f.SeedUser("admin-user", "admin@example.com")
	f.GrantPermissionToUser("admin-user", "admin.read")
	f.AuthenticateAs("admin-user")

	f.SeedSession("test-session", targetID)

	w := f.JSONRequest(http.MethodGet, "/auth/admin/users/"+targetID+"/sessions", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	envelope := internaltests.DecodeJSONResponse(t, w)
	internaltests.AssertHasKey(t, envelope, "data")
}

// Session state

func TestAdminRouteIntegration_SessionState_UpsertGetRevokeListDelete_ReturnsOK(t *testing.T) {
	f := newAdminFixture(t)
	targetID := f.SeedUser("target-user", "target@example.com")
	sessionID := f.SeedSession("target-session", targetID)
	f.SeedUser("admin-user", "admin@example.com")
	f.GrantPermissionToUser("admin-user", "users.write")
	f.GrantPermissionToUser("admin-user", "admin.read")
	f.AuthenticateAs("admin-user")

	// Upsert State
	upsertReq := admintypes.UpsertSessionStateRequest{Revoke: false}
	w := f.JSONRequest(http.MethodPost, "/auth/admin/sessions/"+sessionID+"/state", upsertReq)
	assert.Equal(t, http.StatusOK, w.Code)
	upsertResp := internaltests.DecodeJSONResponse(t, w)
	_ = internaltests.GetMap(t, upsertResp, "data")

	// Get State
	w = f.JSONRequest(http.MethodGet, "/auth/admin/sessions/"+sessionID+"/state", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	getResp := internaltests.DecodeJSONResponse(t, w)
	_ = internaltests.GetMap(t, getResp, "data")

	// Revoke
	reason := "security"
	revokeReq := admintypes.RevokeSessionRequest{Reason: &reason}
	w = f.JSONRequest(http.MethodPost, "/auth/admin/sessions/"+sessionID+"/revoke", revokeReq)
	assert.Equal(t, http.StatusOK, w.Code)
	revokeResp := internaltests.DecodeJSONResponse(t, w)
	_ = internaltests.GetMap(t, revokeResp, "data")

	// Verify revoke status via internal API
	sessionState, err := f.Plugin.Api.GetSessionState(context.Background(), sessionID)
	assert.NoError(t, err)
	assert.NotNil(t, sessionState.RevokedAt)
	assert.Equal(t, "security", *sessionState.RevokedReason)

	// List Revoked
	w = f.JSONRequest(http.MethodGet, "/auth/admin/sessions/states/revoked", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	revokedResp := internaltests.DecodeJSONResponse(t, w)
	internaltests.AssertHasKey(t, revokedResp, "data")

	// Delete State
	w = f.JSONRequest(http.MethodDelete, "/auth/admin/sessions/"+sessionID+"/state", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	deleteResp := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "session state deleted", internaltests.GetString(t, deleteResp, "message"))
}

// Impersonation

func TestAdminRouteIntegration_StartAndEndImpersonation_WithValidPayload_ReturnsCreatedAndOK(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("integration-imp-actor", "integration-imp-actor@example.com")
	f.SeedUser("integration-imp-target", "integration-imp-target@example.com")
	f.GrantPermissionToUser("integration-imp-actor", "impersonation.manage")
	f.AuthenticateAsWithSession("integration-imp-actor", "actor-session-1")

	startPayload := admintypes.StartImpersonationRequest{
		TargetUserID:     f.ResolveID("integration-imp-target"),
		Reason:           "support troubleshooting",
		ExpiresInSeconds: new(120),
	}
	startW := f.JSONRequest(http.MethodPost, "/auth/admin/impersonations", startPayload)

	if startW.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusCreated, startW.Code, startW.Body.String())
	}
	startEnvelope := internaltests.DecodeJSONResponse(t, startW)
	assert.Equal(t, "impersonation started", internaltests.GetString(t, startEnvelope, "message"))

	var startResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(startW.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("failed to decode start response: %v", err)
	}
	if startResp.Data.ID == "" {
		t.Fatalf("expected impersonation id in start response")
	}

	// Verify persistence
	imp, err := f.Plugin.Api.GetImpersonationByID(context.Background(), startResp.Data.ID)
	assert.NoError(t, err)
	assert.NotNil(t, imp)
	assert.Equal(t, f.ResolveID("integration-imp-target"), imp.TargetUserID)

	stopW := f.JSONRequest(http.MethodPost, "/auth/admin/impersonations/"+startResp.Data.ID+"/end", nil)

	if stopW.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, stopW.Code, stopW.Body.String())
	}
	stopEnvelope := internaltests.DecodeJSONResponse(t, stopW)
	assert.Equal(t, "impersonation ended", internaltests.GetString(t, stopEnvelope, "message"))

	// Verify ended state in DB
	impEnded, err := f.Plugin.Api.GetImpersonationByID(context.Background(), startResp.Data.ID)
	assert.NoError(t, err)
	assert.NotNil(t, impEnded.EndedAt)
}

func TestAdminRouteIntegration_StartImpersonation_WithoutReason_ReturnsBadRequest(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("integration-imp-missing-reason-actor", "integration-imp-missing-reason-actor@example.com")
	f.SeedUser("integration-imp-missing-reason-target", "integration-imp-missing-reason-target@example.com")
	f.GrantPermissionToUser("integration-imp-missing-reason-actor", "impersonation.manage")
	f.AuthenticateAs("integration-imp-missing-reason-actor")

	startPayload := admintypes.StartImpersonationRequest{
		TargetUserID: f.ResolveID("integration-imp-missing-reason-target"),
	}
	w := f.JSONRequest(http.MethodPost, "/auth/admin/impersonations", startPayload)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
	envelope := internaltests.DecodeJSONResponse(t, w)
	assert.NotEmpty(t, internaltests.GetString(t, envelope, "message"))
}

func TestAdminRouteIntegration_StartImpersonation_WithSelfTarget_ReturnsBadRequest(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("integration-imp-self-actor", "integration-imp-self-actor@example.com")
	f.GrantPermissionToUser("integration-imp-self-actor", "impersonation.manage")
	f.AuthenticateAs("integration-imp-self-actor")

	startPayload := admintypes.StartImpersonationRequest{
		TargetUserID: f.ResolveID("integration-imp-self-actor"),
		Reason:       "debugging",
	}
	w := f.JSONRequest(http.MethodPost, "/auth/admin/impersonations", startPayload)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
	envelope := internaltests.DecodeJSONResponse(t, w)
	assert.NotEmpty(t, internaltests.GetString(t, envelope, "message"))
}

func TestAdminRouteIntegration_StartImpersonation_WithExcessiveTTL_ReturnsBadRequest(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("integration-imp-ttl-actor", "integration-imp-ttl-actor@example.com")
	f.SeedUser("integration-imp-ttl-target", "integration-imp-ttl-target@example.com")
	f.GrantPermissionToUser("integration-imp-ttl-actor", "impersonation.manage")
	f.AuthenticateAs("integration-imp-ttl-actor")

	startPayload := admintypes.StartImpersonationRequest{
		TargetUserID:     f.ResolveID("integration-imp-ttl-target"),
		Reason:           "investigation",
		ExpiresInSeconds: new(100000),
	}
	w := f.JSONRequest(http.MethodPost, "/auth/admin/impersonations", startPayload)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
	envelope := internaltests.DecodeJSONResponse(t, w)
	assert.NotEmpty(t, internaltests.GetString(t, envelope, "message"))
}

func TestAdminRouteIntegration_EndImpersonation_ByDifferentActor_ReturnsBadRequest(t *testing.T) {
	f := newAdminFixture(t)
	f.SeedUser("integration-imp-stop-other-actor-a", "integration-imp-stop-other-actor-a@example.com")
	f.SeedUser("integration-imp-stop-other-actor-b", "integration-imp-stop-other-actor-b@example.com")
	f.SeedUser("integration-imp-stop-other-target", "integration-imp-stop-other-target@example.com")
	f.GrantPermissionToUser("integration-imp-stop-other-actor-a", "impersonation.manage")
	f.GrantPermissionToUser("integration-imp-stop-other-actor-b", "impersonation.manage")
	f.AuthenticateWithHeader("X-Test-User")

	startPayload := admintypes.StartImpersonationRequest{
		TargetUserID: f.ResolveID("integration-imp-stop-other-target"),
		Reason:       "support",
	}

	// Create request manually to set header
	body, _ := json.Marshal(startPayload)
	startReq := httptest.NewRequest(http.MethodPost, "/auth/admin/impersonations", bytes.NewReader(body))
	startReq.Header.Set("Content-Type", "application/json")
	startReq.Header.Set("X-Test-User", "integration-imp-stop-other-actor-a")
	startW := httptest.NewRecorder()
	f.Router.ServeHTTP(startW, startReq)

	if startW.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusCreated, startW.Code, startW.Body.String())
	}

	var startResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(startW.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("failed to decode start response: %v", err)
	}

	stopReq := httptest.NewRequest(http.MethodPost, "/auth/admin/impersonations/"+startResp.Data.ID+"/end", nil)
	stopReq.Header.Set("X-Test-User", "integration-imp-stop-other-actor-b")
	stopW := httptest.NewRecorder()
	f.Router.ServeHTTP(stopW, stopReq)

	if stopW.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, stopW.Code, stopW.Body.String())
	}
	stopEnvelope := internaltests.DecodeJSONResponse(t, stopW)
	assert.NotEmpty(t, internaltests.GetString(t, stopEnvelope, "message"))
}

func TestAdminRouteIntegration_Impersonations_ListAndGetByID_AfterStart_ReturnsOK(t *testing.T) {
	f := newAdminFixture(t)
	_ = f.SeedUser("actor", "actor@example.com")
	targetID := f.SeedUser("target", "target@example.com")
	f.GrantPermissionToUser("actor", "impersonation.manage")
	f.GrantPermissionToUser("actor", "admin.read")

	// Apply RBAC mappings to ensure the impersonation routes require permissions we granted
	f.ApplyRBACMappingsForAllPluginRoutes("admin.read")

	f.AuthenticateAsWithSession("actor", "sess-1")

	// Start one
	startPayload := admintypes.StartImpersonationRequest{
		TargetUserID: targetID,
		Reason:       "test",
	}
	w := f.JSONRequest(http.MethodPost, "/auth/admin/impersonations", startPayload)
	assert.Equal(t, http.StatusCreated, w.Code)
	startResp := internaltests.DecodeJSONResponse(t, w)
	assert.Equal(t, "impersonation started", internaltests.GetString(t, startResp, "message"))
	data := internaltests.GetMap(t, startResp, "data")
	impID := data["id"].(string)

	// List
	w = f.JSONRequest(http.MethodGet, "/auth/admin/impersonations", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	listResp := internaltests.DecodeJSONResponse(t, w)
	internaltests.AssertHasKey(t, listResp, "data")

	// GetByID
	w = f.JSONRequest(http.MethodGet, "/auth/admin/impersonations/"+impID, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	getResp := internaltests.DecodeJSONResponse(t, w)
	impersonation := internaltests.GetMap(t, getResp, "data")
	assert.Equal(t, impID, impersonation["id"])

	// Verify via direct API
	imp, err := f.Plugin.Api.GetImpersonationByID(context.Background(), impID)
	assert.NoError(t, err)
	assert.NotNil(t, imp)
	assert.Equal(t, targetID, imp.TargetUserID)
}
