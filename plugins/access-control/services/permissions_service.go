package services

import (
	"context"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
)

type PermissionsService struct {
	permissionsRepo repositories.PermissionsRepository
	userAccessRepo  repositories.UserAccessRepository
}

func NewPermissionsService(permissionsRepo repositories.PermissionsRepository, userAccessRepo repositories.UserAccessRepository) *PermissionsService {
	return &PermissionsService{permissionsRepo: permissionsRepo, userAccessRepo: userAccessRepo}
}

func (s *PermissionsService) CreatePermission(ctx context.Context, req types.CreatePermissionRequest) (*types.Permission, error) {
	if req.Key == "" {
		return nil, constants.ErrBadRequest
	}

	var description *string
	if req.Description != nil {
		description = req.Description
	}

	permission := &types.Permission{
		ID:          util.GenerateUUID(),
		Key:         req.Key,
		Description: description,
		IsSystem:    req.IsSystem,
	}

	if err := s.permissionsRepo.CreatePermission(ctx, permission); err != nil {
		return nil, err
	}

	return permission, nil
}

func (s *PermissionsService) GetAllPermissions(ctx context.Context) ([]types.Permission, error) {
	return s.permissionsRepo.GetAllPermissions(ctx)
}

func (s *PermissionsService) GetPermissionByID(ctx context.Context, permissionID string) (*types.Permission, error) {
	if permissionID == "" {
		return nil, constants.ErrBadRequest
	}

	permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, constants.ErrNotFound
	}

	return permission, nil
}

func (s *PermissionsService) UpdatePermission(ctx context.Context, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	if permissionID == "" {
		return nil, constants.ErrUnprocessableEntity
	}
	if req.Description == nil {
		return nil, constants.ErrUnprocessableEntity
	}

	description := *req.Description
	if description == "" {
		return nil, constants.ErrUnprocessableEntity
	}

	permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, constants.ErrNotFound
	}
	if permission.IsSystem {
		return nil, constants.ErrBadRequest
	}

	updated, err := s.permissionsRepo.UpdatePermission(ctx, permissionID, &description)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, constants.ErrNotFound
	}

	permission, err = s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, constants.ErrNotFound
	}

	return permission, nil
}

func (s *PermissionsService) DeletePermission(ctx context.Context, permissionID string) error {
	if permissionID == "" {
		return constants.ErrBadRequest
	}

	permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return constants.ErrNotFound
	}
	if permission.IsSystem {
		return constants.ErrBadRequest
	}

	assignmentsCount, err := s.userAccessRepo.CountRoleAssignmentsByPermissionID(ctx, permissionID)
	if err != nil {
		return err
	}
	if assignmentsCount > 0 {
		return constants.ErrConflict
	}

	deleted, err := s.permissionsRepo.DeletePermission(ctx, permissionID)
	if err != nil {
		return err
	}
	if !deleted {
		return constants.ErrNotFound
	}

	return nil
}
