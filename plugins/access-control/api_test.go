package accesscontrol

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func setupInitializedPlugin(t *testing.T) *AccessControlPlugin {
	t.Helper()

	db := accesscontroltests.SetupRepoDB(t)
	ctx := newTestPluginContext(nil)
	ctx.DB = db

	plugin := New(types.AccessControlPluginConfig{Enabled: true})
	require.NoError(t, plugin.Init(ctx))

	return plugin
}

func TestAPIWorkflows(t *testing.T) {
	t.Parallel()

	plugin := setupInitializedPlugin(t)
	api := plugin.Api
	ctx := context.Background()

	role, err := api.CreateRole(ctx, types.CreateRoleRequest{Name: "editor", Weight: new(20)})
	require.NoError(t, err)
	require.NotNil(t, role)
	require.NotEmpty(t, role.ID)

	_, err = api.CreateRole(ctx, types.CreateRoleRequest{Name: "editor"})
	require.Error(t, err)

	roles, err := api.GetAllRoles(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 1)

	byName, err := api.GetRoleByName(ctx, "editor")
	require.NoError(t, err)
	require.Equal(t, role.ID, byName.ID)

	details, err := api.GetRoleByID(ctx, role.ID)
	require.NoError(t, err)
	require.Equal(t, role.ID, details.Role.ID)
	require.Empty(t, details.Permissions)

	updatedRole, err := api.UpdateRole(ctx, role.ID, types.UpdateRoleRequest{Name: new("editor-updated")})
	require.NoError(t, err)
	require.Equal(t, "editor-updated", updatedRole.Name)

	_, err = api.GetRoleByName(ctx, "editor")
	require.True(t, errors.Is(err, coreerrors.ErrNotFound))

	_, err = api.GetRoleByID(ctx, "missing-role")
	require.True(t, errors.Is(err, coreerrors.ErrNotFound))

	permission, err := api.CreatePermission(ctx, types.CreatePermissionRequest{Key: "editor:read", Description: new("Read editor data")})
	require.NoError(t, err)
	require.NotNil(t, permission)

	permissions, err := api.GetAllPermissions(ctx)
	require.NoError(t, err)
	require.Len(t, permissions, 20)

	byID, err := api.GetPermissionByID(ctx, permission.ID)
	require.NoError(t, err)
	require.Equal(t, permission.ID, byID.ID)

	byKey, err := api.GetPermissionByKey(ctx, "editor:read")
	require.NoError(t, err)
	require.Equal(t, permission.ID, byKey.ID)

	rolePermissions, err := api.GetRolePermissions(ctx, role.ID)
	require.NoError(t, err)
	require.Empty(t, rolePermissions)

	require.NoError(t, api.AddPermissionToRole(ctx, role.ID, permission.ID, nil))

	rolePermissions, err = api.GetRolePermissions(ctx, role.ID)
	require.NoError(t, err)
	require.Len(t, rolePermissions, 1)
	require.Equal(t, "editor:read", rolePermissions[0].PermissionKey)

	require.NoError(t, api.ReplaceRolePermissions(ctx, role.ID, []string{permission.ID}, nil))

	rolePermissions, err = api.GetRolePermissions(ctx, role.ID)
	require.NoError(t, err)
	require.Len(t, rolePermissions, 1)

	updatedPermission, err := api.UpdatePermission(ctx, permission.ID, types.UpdatePermissionRequest{Description: new("Updated description")})
	require.NoError(t, err)
	require.NotNil(t, updatedPermission.Description)
	require.Equal(t, "Updated description", *updatedPermission.Description)

	userRoles, err := api.GetUserRoles(ctx, "u1")
	require.NoError(t, err)
	require.Empty(t, userRoles)

	require.NoError(t, api.AssignRoleToUser(ctx, "u1", types.AssignUserRoleRequest{RoleID: role.ID}, nil))

	userRoles, err = api.GetUserRoles(ctx, "u1")
	require.NoError(t, err)
	require.Len(t, userRoles, 1)
	require.Equal(t, role.ID, userRoles[0].RoleID)

	userPermissions, err := api.GetUserPermissions(ctx, "u1")
	require.NoError(t, err)
	require.Len(t, userPermissions, 1)
	require.Equal(t, "editor:read", userPermissions[0].PermissionKey)

	hasPermissions, err := api.HasPermissions(ctx, "u1", []string{"editor:read"})
	require.NoError(t, err)
	require.True(t, hasPermissions)

	hasPermissions, err = api.HasPermissions(ctx, "u1", []string{"editor:write"})
	require.NoError(t, err)
	require.False(t, hasPermissions)

	require.NoError(t, api.ReplaceUserRoles(ctx, "u1", []string{role.ID}, nil))
	require.NoError(t, api.RemoveRoleFromUser(ctx, "u1", role.ID))

	userRoles, err = api.GetUserRoles(ctx, "u1")
	require.NoError(t, err)
	require.Empty(t, userRoles)

	require.NoError(t, api.RemovePermissionFromRole(ctx, role.ID, permission.ID))
	require.NoError(t, api.DeletePermission(ctx, permission.ID))
	require.NoError(t, api.DeleteRole(ctx, role.ID))

	err = api.DeletePermission(ctx, "missing-permission")
	require.True(t, errors.Is(err, coreerrors.ErrNotFound))

	require.Error(t, api.DeleteRole(ctx, "missing-role"))
}
