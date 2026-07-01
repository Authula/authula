package services

import "context"

type PermissionDefinition struct {
	Key         string
	Description string
}

type AccessControlService interface {
	RoleExists(ctx context.Context, roleName string) (bool, error)
	ValidateRoleAssignment(ctx context.Context, roleName string, assignerUserID *string) (bool, error)
	ValidatePermissionKeys(ctx context.Context, permissionKeys []string) error
	EnsurePermissions(ctx context.Context, permissions []PermissionDefinition) error
}
