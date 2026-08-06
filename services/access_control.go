package services

import "context"

type PermissionDefinition struct {
	Key         string
	Description string
}

type AccessControlService interface {
	RoleExists(ctx context.Context, roleName string) (bool, error)
	GetRolePermissionsByName(ctx context.Context, roleName string) ([]string, error)
	GetRoleWeightByName(ctx context.Context, roleName string) (int, error)
	ValidatePermissionKeys(ctx context.Context, permissionKeys []string) error
	EnsurePermissions(ctx context.Context, permissions []PermissionDefinition) error
}
