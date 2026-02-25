package admin_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	gobetterauth "github.com/GoBetterAuth/go-better-auth/v2"
	"github.com/GoBetterAuth/go-better-auth/v2/config"
	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	adminplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type adminFixture struct {
	*internaltests.BaseTestFixture
	Auth   *gobetterauth.Auth
	Plugin *adminplugin.AdminPlugin
	Router *gobetterauth.Router
}

func adminTestRouteMappings() []models.RouteMapping {
	return []models.RouteMapping{
		// User management
		{Path: "/admin/users", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},
		{Path: "/admin/users", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"users.write"}},
		{Path: "/admin/users/{user_id}", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},
		{Path: "/admin/users/{user_id}", Method: http.MethodPatch, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"users.write"}},
		{Path: "/admin/users/{user_id}", Method: http.MethodDelete, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"users.write"}},

		// Roles and permissions
		{Path: "/admin/roles", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},
		{Path: "/admin/roles", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"roles.write"}},
		{Path: "/admin/roles/{role_id}", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},
		{Path: "/admin/roles/{role_id}", Method: http.MethodPatch, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"roles.write"}},
		{Path: "/admin/roles/{role_id}", Method: http.MethodDelete, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"roles.write"}},
		{Path: "/admin/permissions", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},
		{Path: "/admin/permissions", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"roles.write"}},
		{Path: "/admin/impersonations", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"impersonation.manage"}},
		{Path: "/admin/impersonations/{impersonation_id}/end", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"impersonation.manage"}},
		{Path: "/admin/permissions/{permission_id}", Method: http.MethodPatch, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"roles.write"}},
		{Path: "/admin/permissions/{permission_id}", Method: http.MethodDelete, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"roles.write"}},
		{Path: "/admin/roles/{role_id}/permissions", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"roles.write"}},
		{Path: "/admin/roles/{role_id}/permissions", Method: http.MethodPut, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"roles.write"}},
		{Path: "/admin/roles/{role_id}/permissions/{permission_id}", Method: http.MethodDelete, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"roles.write"}},

		// User roles and permissions
		{Path: "/admin/users/{user_id}/roles", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},
		{Path: "/admin/users/{user_id}/roles", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"roles.write"}},
		{Path: "/admin/users/{user_id}/roles", Method: http.MethodPut, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"roles.write"}},
		{Path: "/admin/users/{user_id}/roles/{role_id}", Method: http.MethodDelete, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"roles.write"}},
		{Path: "/admin/users/{user_id}/permissions", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},

		// User state
		{Path: "/admin/users/{user_id}/state", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},
		{Path: "/admin/users/{user_id}/state", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"users.write"}},
		{Path: "/admin/users/{user_id}/state", Method: http.MethodDelete, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"users.write"}},
		{Path: "/admin/users/states/banned", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},
		{Path: "/admin/users/{user_id}/ban", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"users.write"}},
		{Path: "/admin/users/{user_id}/unban", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"users.write"}},
		{Path: "/admin/users/{user_id}/sessions", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},

		// Session state
		{Path: "/admin/sessions/{session_id}/state", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},
		{Path: "/admin/sessions/{session_id}/state", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"users.write"}},
		{Path: "/admin/sessions/{session_id}/state", Method: http.MethodDelete, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"users.write"}},
		{Path: "/admin/sessions/states/revoked", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},
		{Path: "/admin/sessions/{session_id}/revoke", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"users.write"}},

		// Impersonation
		{Path: "/admin/impersonations", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},
		{Path: "/admin/impersonations/{impersonation_id}", Method: http.MethodGet, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"admin.read"}},
		{Path: "/admin/impersonations", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"impersonation.manage"}},
		{Path: "/admin/impersonations/{impersonation_id}/end", Method: http.MethodPost, Plugins: []string{adminplugin.HookIDAdminRBAC.String()}, Permissions: []string{"impersonation.manage"}},
	}
}

func newAdminFixture(t *testing.T) *adminFixture {
	t.Helper()

	plugin := adminplugin.New(admintypes.AdminPluginConfig{Enabled: true})
	base := internaltests.NewBaseTestFixture(
		t,
		config.WithRouteMappings(adminTestRouteMappings()),
	)

	auth := gobetterauth.New(&gobetterauth.AuthConfig{
		Config:  base.Config,
		Plugins: []models.Plugin{plugin},
		DB:      base.DB,
	})
	_ = auth.Handler()

	return &adminFixture{
		BaseTestFixture: base,
		Auth:            auth,
		Plugin:          plugin,
		Router:          auth.Router(),
	}
}

func (f *adminFixture) AuthenticateAs(userID string) {
	f.T.Helper()
	resolvedUserID := f.ResolveID(userID)
	f.Router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Order: 1,
		Handler: func(reqCtx *models.RequestContext) error {
			reqCtx.SetUserIDInContext(resolvedUserID)
			return nil
		},
	})
}

func (f *adminFixture) AuthenticateAsWithSession(userID, sessionID string) {
	f.T.Helper()
	resolvedUserID := f.ResolveID(userID)
	resolvedSessionID := f.SeedSession(sessionID, resolvedUserID)
	f.Router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Order: 1,
		Handler: func(reqCtx *models.RequestContext) error {
			reqCtx.SetUserIDInContext(resolvedUserID)
			reqCtx.Values[models.ContextSessionID.String()] = resolvedSessionID
			return nil
		},
	})
}

func (f *adminFixture) AuthenticateWithHeader(headerKey string) {
	f.T.Helper()
	f.Router.RegisterHook(models.Hook{
		Stage: models.HookBefore,
		Order: 1,
		Handler: func(reqCtx *models.RequestContext) error {
			userID := reqCtx.Request.Header.Get(headerKey)
			if userID != "" {
				reqCtx.SetUserIDInContext(f.ResolveID(userID))
			}
			return nil
		},
	})
}

func (f *adminFixture) GrantPermissionToUser(userID string, permissionKey string) {
	f.T.Helper()
	resolvedUserID := f.ResolveID(userID)
	var permissionID string

	allPermissions, err := f.Plugin.Api.GetAllPermissions(context.Background())
	if err != nil {
		f.T.Fatalf("failed to list permissions: %v", err)
	}
	for i := range allPermissions {
		if allPermissions[i].Key == permissionKey {
			permissionID = allPermissions[i].ID
			break
		}
	}

	if permissionID == "" {
		permission, createErr := f.Plugin.Api.CreatePermission(context.Background(), admintypes.CreatePermissionRequest{Key: permissionKey})
		if createErr != nil {
			f.T.Fatalf("failed to create permission %q: %v", permissionKey, createErr)
		}
		permissionID = permission.ID
	}

	roleName := fmt.Sprintf("%s-role-%s", userID, permissionKey)
	role, err := f.Plugin.Api.CreateRole(context.Background(), admintypes.CreateRoleRequest{Name: roleName})
	if err != nil {
		f.T.Fatalf("failed to create role %q: %v", roleName, err)
	}

	if err := f.Plugin.Api.ReplaceRolePermissions(context.Background(), role.ID, []string{permissionID}, nil); err != nil {
		f.T.Fatalf("failed to assign permission %q to role %q: %v", permissionKey, role.ID, err)
	}

	if err := f.Plugin.Api.AssignRoleToUser(context.Background(), resolvedUserID, admintypes.AssignUserRoleRequest{RoleID: role.ID}, nil); err != nil {
		f.T.Fatalf("failed to assign role %q to user %q: %v", role.ID, resolvedUserID, err)
	}
}

func (f *adminFixture) JSONRequest(method, path string, payload any) *httptest.ResponseRecorder {
	f.T.Helper()
	return internaltests.JSONRequest(f.T, f.Router, method, path, payload)
}

func (f *adminFixture) Request(method, path string, body *bytes.Buffer) *httptest.ResponseRecorder {
	f.T.Helper()
	return internaltests.Request(f.T, f.Router, method, path, body)
}

func (f *adminFixture) ApplyRBACMappingsForAllPluginRoutes(permissionKey string) {
	f.T.Helper()

	mappings := make([]models.RouteMapping, 0, len(f.Plugin.Routes()))
	for _, route := range f.Plugin.Routes() {
		mappings = append(mappings, models.RouteMapping{
			Path:        route.Path,
			Method:      route.Method,
			Plugins:     []string{adminplugin.HookIDAdminRBAC.String()},
			Permissions: []string{permissionKey},
		})
	}

	metadata, err := util.ConvertRouteMetadata(mappings)
	if err != nil {
		f.T.Fatalf("failed to convert route metadata for all plugin routes: %v", err)
	}
	f.Router.SetRouteMetadataFromConfig(metadata)
}
