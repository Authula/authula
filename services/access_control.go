package services

import "context"

type AccessControlService interface {
	RoleExists(ctx context.Context, roleName string) (bool, error)
}
