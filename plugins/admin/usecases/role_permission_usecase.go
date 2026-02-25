package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type RolePermissionServicer interface {
	CreatePermission(ctx context.Context, permission *types.Permission) error
	GetAllPermissions(ctx context.Context) ([]types.Permission, error)
	GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error)
	UpdatePermissionDescription(ctx context.Context, permissionID string, description *string) (bool, error)
	CountRoleAssignmentsByPermissionID(ctx context.Context, permissionID string) (int, error)
	DeletePermission(ctx context.Context, permissionID string) (bool, error)
	CreateRole(ctx context.Context, role *types.Role) error
	GetAllRoles(ctx context.Context) ([]types.Role, error)
	GetRoleByID(ctx context.Context, roleID string) (*types.Role, error)
	UpdateRole(ctx context.Context, roleID string, name *string, description *string) (bool, error)
	DeleteRole(ctx context.Context, roleID string) (bool, error)
	CountUserAssignmentsByRoleID(ctx context.Context, roleID string) (int, error)
	GetRolePermissions(ctx context.Context, roleID string) ([]types.UserPermissionInfo, error)
	ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error
	AddRolePermission(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error
	RemoveRolePermission(ctx context.Context, roleID string, permissionID string) error
	AssignUserRole(ctx context.Context, userID string, roleID string, assignedByUserID *string, expiresAt *time.Time) error
	RemoveUserRole(ctx context.Context, userID string, roleID string) error
	ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error
}

type rolePermissionUseCase struct {
	service RolePermissionServicer
}

func NewRolePermissionUseCase(service RolePermissionServicer) RolePermissionUseCase {
	return &rolePermissionUseCase{service: service}
}

func (u *rolePermissionUseCase) CreateRole(ctx context.Context, req types.CreateRoleRequest) (*types.Role, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("role name is required")
	}

	role := &types.Role{
		ID:          util.GenerateUUID(),
		Name:        name,
		Description: req.Description,
		IsSystem:    req.IsSystem,
	}

	if err := u.service.CreateRole(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

func (u *rolePermissionUseCase) GetAllRoles(ctx context.Context) ([]types.Role, error) {
	return u.service.GetAllRoles(ctx)
}

func (u *rolePermissionUseCase) GetRoleByID(ctx context.Context, roleID string) (*types.RoleDetails, error) {
	roleID = strings.TrimSpace(roleID)
	if roleID == "" {
		return nil, errors.New("role_id is required")
	}

	role, err := u.service.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("role not found")
	}

	permissions, err := u.service.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}

	return &types.RoleDetails{
		Role:        *role,
		Permissions: permissions,
	}, nil
}

func (u *rolePermissionUseCase) UpdateRole(ctx context.Context, roleID string, req types.UpdateRoleRequest) (*types.Role, error) {
	roleID = strings.TrimSpace(roleID)
	if roleID == "" {
		return nil, errors.New("role_id is required")
	}

	if req.Name == nil && req.Description == nil {
		return nil, errors.New("at least one field is required")
	}

	role, err := u.service.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("role not found")
	}
	if role.IsSystem {
		return nil, errors.New("cannot update system role")
	}

	var name *string
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return nil, errors.New("name is required")
		}
		name = &trimmed
	}

	var description *string
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		description = &trimmed
	}

	updated, err := u.service.UpdateRole(ctx, roleID, name, description)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, errors.New("role not found")
	}

	role, err = u.service.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("role not found")
	}

	return role, nil
}

func (u *rolePermissionUseCase) DeleteRole(ctx context.Context, roleID string) error {
	roleID = strings.TrimSpace(roleID)
	if roleID == "" {
		return errors.New("role_id is required")
	}

	role, err := u.service.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New("role not found")
	}
	if role.IsSystem {
		return errors.New("cannot delete system role")
	}

	assignmentsCount, err := u.service.CountUserAssignmentsByRoleID(ctx, roleID)
	if err != nil {
		return err
	}
	if assignmentsCount > 0 {
		return errors.New("role is assigned to one or more users")
	}

	deleted, err := u.service.DeleteRole(ctx, roleID)
	if err != nil {
		return err
	}
	if !deleted {
		return errors.New("role not found")
	}

	return nil
}

func (u *rolePermissionUseCase) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	key := strings.TrimSpace(req.Key)
	if key == "" {
		return nil, errors.New("permission key is required")
	}

	permission := &types.Permission{
		ID:          util.GenerateUUID(),
		Key:         key,
		Description: req.Description,
		IsSystem:    req.IsSystem,
	}

	if err := u.service.CreatePermission(ctx, permission); err != nil {
		return nil, err
	}

	return permission, nil
}

func (u *rolePermissionUseCase) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return u.service.GetAllPermissions(ctx)
}

func (u *rolePermissionUseCase) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	permissionID = strings.TrimSpace(permissionID)
	if permissionID == "" {
		return nil, errors.New("permission_id is required")
	}
	if req.Description == nil {
		return nil, errors.New("description is required")
	}

	description := strings.TrimSpace(*req.Description)
	if description == "" {
		return nil, errors.New("description is required")
	}

	permission, err := u.service.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, errors.New("permission not found")
	}
	if permission.IsSystem {
		return nil, errors.New("cannot update system permission")
	}

	updated, err := u.service.UpdatePermissionDescription(ctx, permissionID, &description)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, errors.New("permission not found")
	}

	permission, err = u.service.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, errors.New("permission not found")
	}

	return permission, nil
}

func (u *rolePermissionUseCase) DeletePermission(ctx context.Context, permissionID string) error {
	permissionID = strings.TrimSpace(permissionID)
	if permissionID == "" {
		return errors.New("permission_id is required")
	}

	permission, err := u.service.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return errors.New("permission not found")
	}
	if permission.IsSystem {
		return errors.New("cannot delete system permission")
	}

	assignmentsCount, err := u.service.CountRoleAssignmentsByPermissionID(ctx, permissionID)
	if err != nil {
		return err
	}
	if assignmentsCount > 0 {
		return errors.New("permission is in use by one or more roles")
	}

	deleted, err := u.service.DeletePermission(ctx, permissionID)
	if err != nil {
		return err
	}
	if !deleted {
		return errors.New("permission not found")
	}

	return nil
}

func (u *rolePermissionUseCase) AddPermissionToRole(ctx context.Context, roleID string, permissionID string, grantedByUserID *string) error {
	roleID = strings.TrimSpace(roleID)
	permissionID = strings.TrimSpace(permissionID)

	if roleID == "" {
		return errors.New("role_id is required")
	}
	if permissionID == "" {
		return errors.New("permission_id is required")
	}

	role, err := u.service.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New("role not found")
	}
	if role.IsSystem {
		return errors.New("cannot modify system role")
	}

	permission, err := u.service.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return errors.New("permission not found")
	}
	if permission.IsSystem {
		return errors.New("cannot modify system permission")
	}

	return u.service.AddRolePermission(ctx, roleID, permissionID, grantedByUserID)
}

func (u *rolePermissionUseCase) RemovePermissionFromRole(ctx context.Context, roleID string, permissionID string) error {
	roleID = strings.TrimSpace(roleID)
	permissionID = strings.TrimSpace(permissionID)

	if roleID == "" {
		return errors.New("role_id is required")
	}
	if permissionID == "" {
		return errors.New("permission_id is required")
	}

	role, err := u.service.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New("role not found")
	}
	if role.IsSystem {
		return errors.New("cannot modify system role")
	}

	permission, err := u.service.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return errors.New("permission not found")
	}
	if permission.IsSystem {
		return errors.New("cannot modify system permission")
	}

	return u.service.RemoveRolePermission(ctx, roleID, permissionID)
}

func (u *rolePermissionUseCase) ReplaceRolePermissions(ctx context.Context, roleID string, permissionIDs []string, grantedByUserID *string) error {
	if strings.TrimSpace(roleID) == "" {
		return errors.New("role_id is required")
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

	return u.service.ReplaceRolePermissions(ctx, roleID, normalized, grantedByUserID)
}

func (u *rolePermissionUseCase) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	if strings.TrimSpace(userID) == "" {
		return errors.New("user_id is required")
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

	return u.service.ReplaceUserRoles(ctx, userID, normalized, assignedByUserID)
}

func (u *rolePermissionUseCase) AssignRoleToUser(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user_id is required")
	}

	roleID := strings.TrimSpace(req.RoleID)
	if roleID == "" {
		return errors.New("role_id is required")
	}

	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now().UTC()) {
		return errors.New("expires_at must be in the future")
	}

	return u.service.AssignUserRole(ctx, userID, roleID, assignedByUserID, req.ExpiresAt)
}

func (u *rolePermissionUseCase) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	userID = strings.TrimSpace(userID)
	roleID = strings.TrimSpace(roleID)

	if userID == "" {
		return errors.New("user_id is required")
	}
	if roleID == "" {
		return errors.New("role_id is required")
	}

	return u.service.RemoveUserRole(ctx, userID, roleID)
}
