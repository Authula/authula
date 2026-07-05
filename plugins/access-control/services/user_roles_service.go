package services

import (
	"context"
	"time"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/types"
	rootservices "github.com/Authula/authula/services"
)

type UserRolesService struct {
	userRolesRepo repositories.UserRolesRepository
	rolesRepo     repositories.RolesRepository
	authorizer    rootservices.Authorizer
}

func NewUserRolesService(userRolesRepo repositories.UserRolesRepository, rolesRepo repositories.RolesRepository, authorizer rootservices.Authorizer) *UserRolesService {
	return &UserRolesService{userRolesRepo: userRolesRepo, rolesRepo: rolesRepo, authorizer: authorizer}
}

func (s *UserRolesService) getUserRolesInternal(ctx context.Context, userID string) ([]types.UserRoleInfo, error) {
	if userID == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	return s.userRolesRepo.GetUserRoles(ctx, userID)
}

func (s *UserRolesService) GetUserRoles(ctx context.Context, actor *models.Actor, userID string) ([]types.UserRoleInfo, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, constants.UserRolesReadPermission); err != nil {
		return nil, err
	}

	return s.getUserRolesInternal(ctx, userID)
}

func (s *UserRolesService) ReplaceUserRoles(ctx context.Context, actor *models.Actor, userID string, roleIDs []string, assignedByUserID *string) error {
	if err := s.authorizer.AuthorizeScope(ctx, actor, constants.UserRolesAssignPermission); err != nil {
		return err
	}

	if userID == "" {
		return internalerrors.ErrBadRequest
	}

	normalized := make([]string, 0, len(roleIDs))
	targetRoles := make([]*types.Role, 0, len(roleIDs))
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
			return internalerrors.ErrNotFound
		}

		seen[roleID] = struct{}{}
		normalized = append(normalized, roleID)
		targetRoles = append(targetRoles, role)
	}

	if assignedByUserID != nil && *assignedByUserID != "" {
		highestWeight, err := s.highestActiveRoleWeight(ctx, *assignedByUserID)
		if err != nil {
			return err
		}

		for _, role := range targetRoles {
			if role.Weight > highestWeight {
				return internalerrors.ErrForbidden
			}
		}
	}

	return s.userRolesRepo.ReplaceUserRoles(ctx, userID, normalized, assignedByUserID)
}

func (s *UserRolesService) AssignRoleToUser(ctx context.Context, actor *models.Actor, userID string, req types.AssignUserRoleRequest, assignedByUserID *string) error {
	if err := s.authorizer.AuthorizeScope(ctx, actor, constants.UserRolesAssignPermission); err != nil {
		return err
	}

	if userID == "" {
		return internalerrors.ErrBadRequest
	}

	roleID := req.RoleID
	if roleID == "" {
		return internalerrors.ErrBadRequest
	}

	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now().UTC()) {
		return internalerrors.ErrBadRequest
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return internalerrors.ErrNotFound
	}

	if err := s.ensureRoleAssignable(ctx, role, assignedByUserID); err != nil {
		return err
	}

	return s.userRolesRepo.AssignUserRole(ctx, userID, roleID, assignedByUserID, req.ExpiresAt)
}

func (s *UserRolesService) assignRoleToUserInternal(ctx context.Context, userID string, roleID string, assignedByUserID *string) error {
	if userID == "" || roleID == "" {
		return internalerrors.ErrBadRequest
	}
	return s.userRolesRepo.AssignUserRole(ctx, userID, roleID, assignedByUserID, nil)
}

func (s *UserRolesService) RemoveRoleFromUser(ctx context.Context, actor *models.Actor, userID string, roleID string) error {
	if err := s.authorizer.AuthorizeScope(ctx, actor, constants.UserRolesRemovePermission); err != nil {
		return err
	}

	if userID == "" || roleID == "" {
		return internalerrors.ErrBadRequest
	}

	return s.userRolesRepo.RemoveUserRole(ctx, userID, roleID)
}

func (s *UserRolesService) ensureRoleAssignable(ctx context.Context, role *types.Role, assignedByUserID *string) error {
	if assignedByUserID == nil || *assignedByUserID == "" {
		return nil
	}

	highestWeight, err := s.highestActiveRoleWeight(ctx, *assignedByUserID)
	if err != nil {
		return err
	}

	if role.Weight > highestWeight {
		return internalerrors.ErrForbidden
	}

	return nil
}

func (s *UserRolesService) highestActiveRoleWeight(ctx context.Context, userID string) (int, error) {
	roles, err := s.userRolesRepo.GetUserRoles(ctx, userID)
	if err != nil {
		return 0, err
	}

	highestWeight, _ := determineHighestActiveRoleWeight(roles, time.Now().UTC())

	return highestWeight, nil
}

func determineHighestActiveRoleWeight(userRoles []types.UserRoleInfo, now time.Time) (int, int) {
	highestWeight := 0
	activeCount := 0

	for _, userRole := range userRoles {
		if userRole.ExpiresAt != nil && userRole.ExpiresAt.Before(now) {
			continue
		}
		activeCount++
		if userRole.RoleWeight > highestWeight {
			highestWeight = userRole.RoleWeight
		}
	}

	return highestWeight, activeCount
}
