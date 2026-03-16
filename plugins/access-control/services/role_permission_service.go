package services

import (
	"context"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

type RolePermissionService struct {
	repo repositories.RolePermissionRepository
}

func NewRolePermissionService(repo repositories.RolePermissionRepository) *RolePermissionService {
	return &RolePermissionService{repo: repo}
}

func (s *RolePermissionService) CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, constants.ErrBadRequest
	}

	role := &types.Role{
		ID:          util.GenerateUUID(),
		Name:        name,
		Description: req.Description,
		IsSystem:    req.IsSystem,
	}

	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

func (s *RolePermissionService) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return s.repo.GetAllRoles(ctx)
}

func (s *RolePermissionService) GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error) {
	roleID = strings.TrimSpace(roleID)
	if roleID == "" {
		return nil, constants.ErrBadRequest
	}

	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, constants.ErrNotFound
	}

	permissions, err := s.repo.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}

	return &types.RoleDetails{Role: *role, Permissions: permissions}, nil
}

func (s *RolePermissionService) UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	roleID = strings.TrimSpace(roleID)
	if roleID == "" {
		return nil, constants.ErrBadRequest
	}

	if req.Name == nil && req.Description == nil {
		return nil, constants.ErrUnprocessableEntity
	}

	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, constants.ErrNotFound
	}
	if role.IsSystem {
		return nil, constants.ErrCannotUpdateSystemRole
	}

	var name *string
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return nil, constants.ErrBadRequest
		}
		name = &trimmed
	}

	var description *string
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		description = &trimmed
	}

	updated, err := s.repo.UpdateRole(ctx, roleID, name, description)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, constants.ErrNotFound
	}

	role, err = s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, constants.ErrNotFound
	}

	return role, nil
}

func (s *RolePermissionService) DeleteRole(ctx context.Context, roleID string) error {
	roleID = strings.TrimSpace(roleID)
	if roleID == "" {
		return constants.ErrBadRequest
	}

	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return constants.ErrNotFound
	}
	if role.IsSystem {
		return constants.ErrCannotUpdateSystemRole
	}

	assignmentsCount, err := s.repo.CountUserAssignmentsByRoleID(ctx, roleID)
	if err != nil {
		return err
	}
	if assignmentsCount > 0 {
		return constants.ErrConflict
	}

	deleted, err := s.repo.DeleteRole(ctx, roleID)
	if err != nil {
		return err
	}
	if !deleted {
		return constants.ErrNotFound
	}

	return nil
}

func (s *RolePermissionService) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	key := strings.TrimSpace(req.Key)
	if key == "" {
		return nil, constants.ErrBadRequest
	}

	permission := &types.Permission{
		ID:          util.GenerateUUID(),
		Key:         key,
		Description: req.Description,
		IsSystem:    req.IsSystem,
	}

	if err := s.repo.CreatePermission(ctx, permission); err != nil {
		return nil, err
	}

	return permission, nil
}

func (s *RolePermissionService) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return s.repo.GetAllPermissions(ctx)
}

func (s *RolePermissionService) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	permissionID = strings.TrimSpace(permissionID)
	if permissionID == "" {
		return nil, constants.ErrBadRequest
	}
	if req.Description == nil {
		return nil, constants.ErrBadRequest
	}

	description := strings.TrimSpace(*req.Description)
	if description == "" {
		return nil, constants.ErrBadRequest
	}

	permission, err := s.repo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, constants.ErrNotFound
	}
	if permission.IsSystem {
		return nil, constants.ErrBadRequest
	}

	updated, err := s.repo.UpdatePermission(ctx, permissionID, &description)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, constants.ErrNotFound
	}

	permission, err = s.repo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, constants.ErrNotFound
	}

	return permission, nil
}

func (s *RolePermissionService) DeletePermission(ctx context.Context, permissionID string) error {
	permissionID = strings.TrimSpace(permissionID)
	if permissionID == "" {
		return constants.ErrBadRequest
	}

	permission, err := s.repo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return constants.ErrNotFound
	}
	if permission.IsSystem {
		return constants.ErrBadRequest
	}

	assignmentsCount, err := s.repo.CountRoleAssignmentsByPermissionID(ctx, permissionID)
	if err != nil {
		return err
	}
	if assignmentsCount > 0 {
		return constants.ErrConflict
	}

	deleted, err := s.repo.DeletePermission(ctx, permissionID)
	if err != nil {
		return err
	}
	if !deleted {
		return constants.ErrNotFound
	}

	return nil
}

func (s *RolePermissionService) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	roleID = strings.TrimSpace(roleID)
	permissionID = strings.TrimSpace(permissionID)

	if roleID == "" {
		return constants.ErrBadRequest
	}
	if permissionID == "" {
		return constants.ErrBadRequest
	}

	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return constants.ErrNotFound
	}
	if role.IsSystem {
		return constants.ErrBadRequest
	}

	permission, err := s.repo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return constants.ErrNotFound
	}
	if permission.IsSystem {
		return constants.ErrBadRequest
	}

	return s.repo.AddRolePermission(ctx, roleID, permissionID, grantedByUserID)
}

func (s *RolePermissionService) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	roleID = strings.TrimSpace(roleID)
	permissionID = strings.TrimSpace(permissionID)

	if roleID == "" {
		return constants.ErrBadRequest
	}
	if permissionID == "" {
		return constants.ErrBadRequest
	}

	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return constants.ErrNotFound
	}
	if role.IsSystem {
		return constants.ErrBadRequest
	}

	permission, err := s.repo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return constants.ErrNotFound
	}
	if permission.IsSystem {
		return constants.ErrBadRequest
	}

	return s.repo.RemoveRolePermission(ctx, roleID, permissionID)
}

func (s *RolePermissionService) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	if strings.TrimSpace(roleID) == "" {
		return constants.ErrBadRequest
	}

	normalized := make([]string, 0, len(permissionIDs))
	seen := make(map[string]struct{}, len(permissionIDs))
	for _, permissionID := range permissionIDs {
		permissionID = strings.TrimSpace(permissionID)
		if permissionID == "" {
			continue
		}
		if _, ok := seen[permissionID]; ok {
			continue
		}
		seen[permissionID] = struct{}{}
		normalized = append(normalized, permissionID)
	}

	return s.repo.ReplaceRolePermissions(ctx, roleID, normalized, grantedByUserID)
}

func (s *RolePermissionService) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	if strings.TrimSpace(userID) == "" {
		return constants.ErrBadRequest
	}

	normalized := make([]string, 0, len(roleIDs))
	seen := make(map[string]struct{}, len(roleIDs))
	for _, roleID := range roleIDs {
		roleID = strings.TrimSpace(roleID)
		if roleID == "" {
			continue
		}
		if _, ok := seen[roleID]; ok {
			continue
		}
		seen[roleID] = struct{}{}
		normalized = append(normalized, roleID)
	}

	return s.repo.ReplaceUserRoles(ctx, userID, normalized, assignedByUserID)
}

func (s *RolePermissionService) AssignRoleToUser(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return constants.ErrBadRequest
	}

	roleID := strings.TrimSpace(req.RoleID)
	if roleID == "" {
		return constants.ErrBadRequest
	}

	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now().UTC()) {
		return constants.ErrBadRequest
	}

	return s.repo.AssignUserRole(ctx, userID, roleID, assignedByUserID, req.ExpiresAt)
}

func (s *RolePermissionService) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	userID = strings.TrimSpace(userID)
	roleID = strings.TrimSpace(roleID)

	if userID == "" {
		return constants.ErrBadRequest
	}
	if roleID == "" {
		return constants.ErrBadRequest
	}

	return s.repo.RemoveUserRole(ctx, userID, roleID)
}
