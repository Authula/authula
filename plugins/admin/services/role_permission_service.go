package services

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type RolePermissionService struct {
	repo *repositories.RolePermissionRepository
}

func NewRolePermissionService(repo *repositories.RolePermissionRepository) *RolePermissionService {
	return &RolePermissionService{repo: repo}
}

func (s *RolePermissionService) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return s.repo.GetAllRoles(ctx)
}

func (s *RolePermissionService) GetRoleByID(ctx context.Context, roleID string) (*types.Role, error) {
	return s.repo.GetRoleByID(ctx, roleID)
}

func (s *RolePermissionService) CreateRole(ctx context.Context, role *types.Role) error {
	return s.repo.CreateRole(ctx, role)
}

func (s *RolePermissionService) UpdateRole(ctx context.Context, roleID string, name *string, description *string) (bool, error) {
	return s.repo.UpdateRole(ctx, roleID, name, description)
}

func (s *RolePermissionService) DeleteRole(ctx context.Context, roleID string) (bool, error) {
	return s.repo.DeleteRole(ctx, roleID)
}

func (s *RolePermissionService) CountUserAssignmentsByRoleID(ctx context.Context, roleID string) (int, error) {
	return s.repo.CountUserAssignmentsByRoleID(ctx, roleID)
}

func (s *RolePermissionService) CreatePermission(ctx context.Context, permission *types.Permission) error {
	return s.repo.CreatePermission(ctx, permission)
}

func (s *RolePermissionService) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return s.repo.GetAllPermissions(ctx)
}

func (s *RolePermissionService) GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error) {
	return s.repo.GetPermissionByID(ctx, permissionID)
}

func (s *RolePermissionService) UpdatePermissionDescription(ctx context.Context, permissionID string, description *string) (bool, error) {
	return s.repo.UpdatePermissionDescription(ctx, permissionID, description)
}

func (s *RolePermissionService) DeletePermission(ctx context.Context, permissionID string) (bool, error) {
	return s.repo.DeletePermission(ctx, permissionID)
}

func (s *RolePermissionService) CountRoleAssignmentsByPermissionID(ctx context.Context, permissionID string) (int, error) {
	return s.repo.CountRoleAssignmentsByPermissionID(ctx, permissionID)
}

func (s *RolePermissionService) GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error) {
	return s.repo.GetRolePermissions(ctx, roleID)
}

func (s *RolePermissionService) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	return s.repo.ReplaceRolePermissions(ctx, roleID, permissionIDs, grantedByUserID)
}

func (s *RolePermissionService) AddRolePermission(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	return s.repo.AddRolePermission(ctx, roleID, permissionID, grantedByUserID)
}

func (s *RolePermissionService) RemoveRolePermission(ctx context.Context, roleID string, permissionID string) error {
	return s.repo.RemoveRolePermission(ctx, roleID, permissionID)
}

func (s *RolePermissionService) AssignUserRole(ctx context.Context, userID string, roleID string, assignedByUserID *string, expiresAt *time.Time) error {
	return s.repo.AssignUserRole(ctx, userID, roleID, assignedByUserID, expiresAt)
}

func (s *RolePermissionService) RemoveUserRole(ctx context.Context, userID string, roleID string) error {
	return s.repo.RemoveUserRole(ctx, userID, roleID)
}

func (s *RolePermissionService) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	return s.repo.ReplaceUserRoles(ctx, userID, roleIDs, assignedByUserID)
}
