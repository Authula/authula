package services

import (
	"context"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/util"
)

type PermissionsService struct {
	permissionsRepo     repositories.PermissionsRepository
	rolePermissionsRepo repositories.RolePermissionsRepository
}

func NewPermissionsService(permissionsRepo repositories.PermissionsRepository, rolePermissionsRepo repositories.RolePermissionsRepository) *PermissionsService {
	return &PermissionsService{permissionsRepo: permissionsRepo, rolePermissionsRepo: rolePermissionsRepo}
}

func (s *PermissionsService) CreatePermission(ctx context.Context, actor *models.Actor, req types.CreatePermissionRequest) (*types.Permission, error) {
	if req.Key == "" {
		return nil, coreerrors.ErrBadRequest
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

func (s *PermissionsService) GetAllPermissions(ctx context.Context, actor *models.Actor) ([]types.Permission, error) {
	return s.permissionsRepo.GetAllPermissions(ctx)
}

func (s *PermissionsService) GetPermissionByID(ctx context.Context, actor *models.Actor, permissionID string) (*types.Permission, error) {
	if permissionID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, coreerrors.ErrNotFound
	}

	return permission, nil
}

func (s *PermissionsService) GetPermissionByKey(ctx context.Context, actor *models.Actor, permissionKey string) (*types.Permission, error) {
	if permissionKey == "" {
		return nil, coreerrors.ErrBadRequest
	}

	permission, err := s.permissionsRepo.GetPermissionByKey(ctx, permissionKey)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, coreerrors.ErrNotFound
	}

	return permission, nil
}

func (s *PermissionsService) UpdatePermission(ctx context.Context, actor *models.Actor, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	if permissionID == "" {
		return nil, coreerrors.ErrUnprocessableEntity
	}
	if req.Description == nil {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	description := *req.Description
	if description == "" {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, coreerrors.ErrNotFound
	}
	if permission.IsSystem {
		return nil, coreerrors.ErrBadRequest
	}

	updated, err := s.permissionsRepo.UpdatePermission(ctx, permissionID, &description)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, coreerrors.ErrNotFound
	}

	permission, err = s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, coreerrors.ErrNotFound
	}

	return permission, nil
}

func (s *PermissionsService) DeletePermission(ctx context.Context, actor *models.Actor, permissionID string) error {
	if permissionID == "" {
		return coreerrors.ErrBadRequest
	}

	permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return coreerrors.ErrNotFound
	}
	if permission.IsSystem {
		return coreerrors.ErrBadRequest
	}

	totalCountOfRolesByPermission, err := s.rolePermissionsRepo.CountRolesByPermission(ctx, permissionID)
	if err != nil {
		return err
	}
	if totalCountOfRolesByPermission > 0 {
		return coreerrors.ErrConflict
	}

	deleted, err := s.permissionsRepo.DeletePermission(ctx, permissionID)
	if err != nil {
		return err
	}
	if !deleted {
		return coreerrors.ErrNotFound
	}

	return nil
}
