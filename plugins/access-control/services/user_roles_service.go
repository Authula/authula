package services

import (
	"context"
	"time"

	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
)

type UserRolesService struct {
	userRolesRepo repositories.UserRolesRepository
	rolesRepo     repositories.RolesRepository
}

func NewUserRolesService(userRolesRepo repositories.UserRolesRepository, rolesRepo repositories.RolesRepository) *UserRolesService {
	return &UserRolesService{userRolesRepo: userRolesRepo, rolesRepo: rolesRepo}
}

func (s *UserRolesService) GetUserRoles(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	if userID == "" {
		return nil, constants.ErrUnprocessableEntity
	}

	return s.userRolesRepo.GetUserRoles(ctx, userID)
}

func (s *UserRolesService) GetUserWithRolesByID(ctx context.Context, userID string) (*types.UserWithRoles, error) {
	if userID == "" {
		return nil, constants.ErrUnprocessableEntity
	}

	return s.userRolesRepo.GetUserWithRolesByID(ctx, userID)
}

func (s *UserRolesService) ReplaceUserRoles(ctx context.Context, userID string, roleIDs []string, assignedByUserID *string) error {
	if userID == "" {
		return constants.ErrBadRequest
	}

	normalized := make([]string, 0, len(roleIDs))
	seen := make(map[string]struct{}, len(roleIDs))
	for _, roleID := range roleIDs {
		if roleID == "" {
			continue
		}
		if _, ok := seen[roleID]; ok {
			continue
		}

		role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
		if err != nil {
			return err
		}
		if role == nil {
			return constants.ErrNotFound
		}

		seen[roleID] = struct{}{}
		normalized = append(normalized, roleID)
	}

	return s.userRolesRepo.ReplaceUserRoles(ctx, userID, normalized, assignedByUserID)
}

func (s *UserRolesService) AssignRoleToUser(ctx context.Context, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	if userID == "" {
		return constants.ErrBadRequest
	}

	roleID := req.RoleID
	if roleID == "" {
		return constants.ErrBadRequest
	}

	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now().UTC()) {
		return constants.ErrBadRequest
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return constants.ErrNotFound
	}

	return s.userRolesRepo.AssignUserRole(ctx, userID, roleID, assignedByUserID, req.ExpiresAt)
}

func (s *UserRolesService) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	if userID == "" || roleID == "" {
		return constants.ErrBadRequest
	}

	return s.userRolesRepo.RemoveUserRole(ctx, userID, roleID)
}
