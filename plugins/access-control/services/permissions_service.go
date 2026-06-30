package services

import (
	"context"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
	rootservices "github.com/Authula/authula/services"
)

type PermissionsService struct {
	permissionsRepo     repositories.PermissionsRepository
	rolePermissionsRepo repositories.RolePermissionsRepository
	authorizer          rootservices.Authorizer
}

func NewPermissionsService(permissionsRepo repositories.PermissionsRepository, rolePermissionsRepo repositories.RolePermissionsRepository, authorizer rootservices.Authorizer) *PermissionsService {
	return &PermissionsService{permissionsRepo: permissionsRepo, rolePermissionsRepo: rolePermissionsRepo, authorizer: authorizer}
}

func (s *PermissionsService) CreatePermission(ctx context.Context, actor *models.Actor, req types.CreatePermissionRequest) (*types.Permission, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, constants.PermissionsCreatePermission); err != nil {
		return nil, err
	}

	if req.Key == "" {
		return nil, internalerrors.ErrBadRequest
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
	if err := s.authorizer.AuthorizeScope(ctx, actor, constants.PermissionsListPermission); err != nil {
		return nil, err
	}

	return s.permissionsRepo.GetAllPermissions(ctx)
}

func (s *PermissionsService) GetPermissionByID(ctx context.Context, actor *models.Actor, permissionID string) (*types.Permission, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, constants.PermissionsReadPermission); err != nil {
		return nil, err
	}

	if permissionID == "" {
		return nil, internalerrors.ErrBadRequest
	}

	permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, internalerrors.ErrNotFound
	}

	return permission, nil
}

func (s *PermissionsService) GetPermissionByKey(ctx context.Context, actor *models.Actor, permissionKey string) (*types.Permission, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, constants.PermissionsReadPermission); err != nil {
		return nil, err
	}

	if permissionKey == "" {
		return nil, internalerrors.ErrBadRequest
	}

	permission, err := s.permissionsRepo.GetPermissionByKey(ctx, permissionKey)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, internalerrors.ErrNotFound
	}

	return permission, nil
}

func (s *PermissionsService) UpdatePermission(ctx context.Context, actor *models.Actor, permissionID string, req types.UpdatePermissionRequest) (*types.Permission, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, constants.PermissionsUpdatePermission); err != nil {
		return nil, err
	}

	if permissionID == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}
	if req.Description == nil {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	description := *req.Description
	if description == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, internalerrors.ErrNotFound
	}
	if permission.IsSystem {
		return nil, internalerrors.ErrBadRequest
	}

	updated, err := s.permissionsRepo.UpdatePermission(ctx, permissionID, &description)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, internalerrors.ErrNotFound
	}

	permission, err = s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, internalerrors.ErrNotFound
	}

	return permission, nil
}

func (s *PermissionsService) DeletePermission(ctx context.Context, actor *models.Actor, permissionID string) error {
	if err := s.authorizer.AuthorizeScope(ctx, actor, constants.PermissionsDeletePermission); err != nil {
		return err
	}

	if permissionID == "" {
		return internalerrors.ErrBadRequest
	}

	permission, err := s.permissionsRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return internalerrors.ErrNotFound
	}
	if permission.IsSystem {
		return internalerrors.ErrBadRequest
	}

	totalCountOfRolesByPermission, err := s.rolePermissionsRepo.CountRolesByPermission(ctx, permissionID)
	if err != nil {
		return err
	}
	if totalCountOfRolesByPermission > 0 {
		return internalerrors.ErrConflict
	}

	deleted, err := s.permissionsRepo.DeletePermission(ctx, permissionID)
	if err != nil {
		return err
	}
	if !deleted {
		return internalerrors.ErrNotFound
	}

	return nil
}
