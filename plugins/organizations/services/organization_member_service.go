package services

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/constants"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type organizationMemberTxRunner interface {
	RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(context.Context, bun.Tx) error) error
}

type organizationMemberService struct {
	userService          rootservices.UserService
	accessControlService rootservices.AccessControlService
	orgRepo              repositories.OrganizationRepository
	orgMemberRepo        repositories.OrganizationMemberRepository
	serviceUtils         *ServiceUtils
	membersLimit         *int
	txRunner             organizationMemberTxRunner
	hooks                *ServiceHookExecutor
}

func NewOrganizationMemberService(userService rootservices.UserService, accessControlService rootservices.AccessControlService, orgRepo repositories.OrganizationRepository, orgMemberRepo repositories.OrganizationMemberRepository, membersLimit *int, txRunner organizationMemberTxRunner, serviceUtils *ServiceUtils, hooks ...*ServiceHookExecutor) *organizationMemberService {
	var hook *ServiceHookExecutor
	if len(hooks) > 0 {
		hook = hooks[0]
	}
	return &organizationMemberService{userService: userService, accessControlService: accessControlService, orgRepo: orgRepo, orgMemberRepo: orgMemberRepo, serviceUtils: serviceUtils, membersLimit: membersLimit, txRunner: txRunner, hooks: hook}
}

func (s *organizationMemberService) AddMember(ctx context.Context, actor *models.Actor, organizationID string, request types.AddOrganizationMemberRequest) (*types.OrganizationMember, error) {
	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}

	userID := request.UserID
	if userID == "" {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	role := request.Role
	if role == "" {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	user, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, coreerrors.ErrNotFound
	}

	if existing, err := s.orgMemberRepo.GetByOrganizationIDAndUserID(ctx, organizationID, userID); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, coreerrors.ErrConflict
	}

	actorID := actor.ID
	validatedRoleAssignment, err := s.accessControlService.ValidateRoleAssignment(ctx, role, &actorID)
	if err != nil {
		if err.Error() == coreerrors.ErrForbidden.Error() {
			return nil, coreerrors.ErrForbidden
		}
		if err.Error() == coreerrors.ErrNotFound.Error() {
			return nil, coreerrors.ErrBadRequest
		}
		return nil, err
	}
	if !validatedRoleAssignment {
		return nil, coreerrors.ErrBadRequest
	}

	member := &types.OrganizationMember{
		ID:             util.GenerateUUID(),
		OrganizationID: organizationID,
		UserID:         userID,
		Role:           role,
	}

	if s.hooks != nil {
		if err := s.hooks.BeforeCreateOrganizationMember(ctx, actor, member); err != nil {
			return nil, err
		}
	}

	var created *types.OrganizationMember
	err = s.txRunner.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		memberRepo := s.orgMemberRepo.WithTx(tx)
		if err := ensureOrganizationMembersLimit(ctx, memberRepo, organizationID, s.membersLimit); err != nil {
			return err
		}

		createdMember, err := memberRepo.Create(ctx, member)
		if err != nil {
			return err
		}

		created = createdMember
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.hooks != nil {
		if err := s.hooks.AfterCreateOrganizationMember(ctx, actor, created); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *organizationMemberService) GetAllMembers(ctx context.Context, actor *models.Actor, organizationID string, page int, limit int) ([]types.OrganizationMemberResponse, error) {
	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}

	return s.orgMemberRepo.GetAllByOrganizationIDWithUser(ctx, organizationID, page, limit)
}

func (s *organizationMemberService) GetMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) (*types.OrganizationMemberResponse, error) {
	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}

	if memberID == "" {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	member, err := s.orgMemberRepo.GetByIDWithUser(ctx, memberID)
	if err != nil {
		return nil, err
	}
	if member == nil || member.OrganizationID != organizationID {
		return nil, coreerrors.ErrNotFound
	}

	return member, nil
}

func (s *organizationMemberService) GetMemberByUserID(ctx context.Context, actor *models.Actor, organizationID string, userID string) (*types.OrganizationMemberResponse, error) {
	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}

	if userID == "" {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	member, err := s.orgMemberRepo.GetByOrganizationIDAndUserIDWithUser(ctx, organizationID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, coreerrors.ErrNotFound
	}

	return member, nil
}

func (s *organizationMemberService) UpdateMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string, request types.UpdateOrganizationMemberRequest) (*types.OrganizationMember, error) {
	_, actorMember, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actor, organizationID)
	if err != nil {
		return nil, err
	}

	member, err := s.orgMemberRepo.GetByID(ctx, memberID)
	if err != nil {
		return nil, err
	}
	if member == nil || member.OrganizationID != organizationID {
		return nil, coreerrors.ErrNotFound
	}

	role := request.Role
	if role == "" {
		return nil, coreerrors.ErrBadRequest
	}

	actorID := actor.ID
	validatedRoleAssignment, err := s.accessControlService.ValidateRoleAssignment(ctx, role, &actorID)
	if err != nil {
		if err.Error() == coreerrors.ErrForbidden.Error() {
			return nil, coreerrors.ErrForbidden
		}
		if err.Error() == coreerrors.ErrNotFound.Error() {
			return nil, coreerrors.ErrBadRequest
		}
		return nil, err
	}
	if !validatedRoleAssignment {
		return nil, coreerrors.ErrBadRequest
	}

	if actorMember != nil && actorMember.UserID == member.UserID {
		return nil, coreerrors.ErrForbidden
	}

	member.Role = role

	if s.hooks != nil {
		if err := s.hooks.BeforeUpdateOrganizationMember(ctx, actor, member); err != nil {
			return nil, err
		}
	}

	updated, err := s.orgMemberRepo.Update(ctx, member)
	if err != nil {
		return nil, err
	}

	if s.hooks != nil {
		if err := s.hooks.AfterUpdateOrganizationMember(ctx, actor, updated); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

func (s *organizationMemberService) RemoveMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) error {
	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return err
	}

	member, err := s.orgMemberRepo.GetByID(ctx, memberID)
	if err != nil {
		return err
	}
	if member == nil || member.OrganizationID != organizationID {
		return coreerrors.ErrNotFound
	}

	if s.hooks != nil {
		if err := s.hooks.BeforeDeleteOrganizationMember(ctx, actor, member); err != nil {
			return err
		}
	}

	if err := s.orgMemberRepo.Delete(ctx, member.ID); err != nil {
		return err
	}

	if s.hooks != nil {
		if err := s.hooks.AfterDeleteOrganizationMember(ctx, actor, member); err != nil {
			return err
		}
	}

	return nil
}

func ensureOrganizationMembersLimit(ctx context.Context, memberRepo repositories.OrganizationMemberRepository, organizationID string, membersLimit *int) error {
	if membersLimit == nil || *membersLimit <= 0 {
		return nil
	}

	memberCount, err := memberRepo.CountByOrganizationID(ctx, organizationID)
	if err != nil {
		return err
	}
	if memberCount >= *membersLimit {
		return constants.ErrMembersQuotaExceeded
	}

	return nil
}
