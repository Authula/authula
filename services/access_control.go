package services

import "context"

type AccessControlService interface {
	RoleExists(ctx context.Context, roleName string) (bool, error)
	ValidateRoleAssignment(ctx context.Context, roleName string, assignerUserID *string) (bool, error)
}
