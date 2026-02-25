package tests

import (
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

// Helper functions for creating test fixtures

// NewTestRole creates a test role with default values.
func NewTestRole(id string, name string) *types.Role {
	return &types.Role{
		ID:          id,
		Name:        name,
		Description: nil,
		IsSystem:    false,
	}
}

// NewTestSystemRole creates a test system role.
func NewTestSystemRole(id string, name string) *types.Role {
	role := NewTestRole(id, name)
	role.IsSystem = true
	return role
}

// NewTestPermission creates a test permission with default values.
func NewTestPermission(id string, key string) *types.Permission {
	return &types.Permission{
		ID:          id,
		Key:         key,
		Description: nil,
		IsSystem:    false,
	}
}

// NewTestSystemPermission creates a test system permission.
func NewTestSystemPermission(id string, key string) *types.Permission {
	perm := NewTestPermission(id, key)
	perm.IsSystem = true
	return perm
}

// NewTestUserPermissionInfo creates a test UserPermissionInfo.
func NewTestUserPermissionInfo(permissionID string, permissionKey string) types.UserPermissionInfo {
	return types.UserPermissionInfo{
		PermissionID:  permissionID,
		PermissionKey: permissionKey,
	}
}
