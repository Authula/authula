package services

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
)

type organizationMemberTxRunner interface {
	RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(context.Context, bun.Tx) error) error
}

type OrganizationMemberService struct {
	userService          rootservices.UserService
	accessControlService rootservices.AccessControlService
	orgRepo              repositories.OrganizationRepository
	orgMemberRepo        repositories.OrganizationMemberRepository
	serviceUtils         *ServiceUtils
	membersLimit         *int
	txRunner             organizationMemberTxRunner
}

func NewOrganizationMemberService(userService rootservices.UserService, accessControlService rootservices.AccessControlService, orgRepo repositories.OrganizationRepository, orgMemberRepo repositories.OrganizationMemberRepository, membersLimit *int, txRunner organizationMemberTxRunner, serviceUtils *ServiceUtils) *OrganizationMemberService {
	return &OrganizationMemberService{userService: userService, accessControlService: accessControlService, orgRepo: orgRepo, orgMemberRepo: orgMemberRepo, serviceUtils: serviceUtils, membersLimit: membersLimit, txRunner: txRunner}
}

func (s *OrganizationMemberService) AddMember(ctx context.Context, actorUserID string, organizationID string, request types.AddOrganizationMemberRequest) (*types.OrganizationMember, error) {
	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID); err != nil {
		return nil, err
	}

	userID := request.UserID
	if userID == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	role := request.Role
	if role == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	user, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, internalerrors.ErrNotFound
	}

	if existing, err := s.orgMemberRepo.GetByOrganizationIDAndUserID(ctx, organizationID, userID); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, internalerrors.ErrConflict
	}

	validatedRoleAssignment, err := s.accessControlService.ValidateRoleAssignment(ctx, role, &actorUserID)
	if err != nil {
		if err.Error() == internalerrors.ErrForbidden.Error() {
			return nil, internalerrors.ErrForbidden
		}
		if err.Error() == internalerrors.ErrNotFound.Error() {
			return nil, internalerrors.ErrBadRequest
		}
		return nil, err
	}
	if !validatedRoleAssignment {
		return nil, internalerrors.ErrBadRequest
	}

	var created *types.OrganizationMember
	err = s.txRunner.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		memberRepo := s.orgMemberRepo.WithTx(tx)
		if err := s.serviceUtils.ensureOrganizationMembersLimit(ctx, memberRepo, organizationID, s.membersLimit); err != nil {
			return err
		}

		member := &types.OrganizationMember{
			ID:             util.GenerateUUID(),
			OrganizationID: organizationID,
			UserID:         userID,
			Role:           role,
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

	return created, nil
}

func (s *OrganizationMemberService) GetAllMembers(ctx context.Context, actorUserID string, organizationID string, page int, limit int) ([]types.OrganizationMember, error) {
	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID); err != nil {
		return nil, err
	}

	return s.orgMemberRepo.GetAllByOrganizationID(ctx, organizationID, page, limit)
}

func (s *OrganizationMemberService) GetMember(ctx context.Context, actorUserID string, organizationID string, memberID string) (*types.OrganizationMember, error) {
	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID); err != nil {
		return nil, err
	}

	if memberID == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	member, err := s.orgMemberRepo.GetByID(ctx, memberID)
	if err != nil {
		return nil, err
	}
	if member == nil || member.OrganizationID != organizationID {
		return nil, internalerrors.ErrNotFound
	}

	return member, nil
}

func (s *OrganizationMemberService) UpdateMember(ctx context.Context, actorUserID string, organizationID string, memberID string, request types.UpdateOrganizationMemberRequest) (*types.OrganizationMember, error) {
	_, actorMember, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID)
	if err != nil {
		return nil, err
	}

	member, err := s.orgMemberRepo.GetByID(ctx, memberID)
	if err != nil {
		return nil, err
	}
	if member == nil || member.OrganizationID != organizationID {
		return nil, internalerrors.ErrNotFound
	}

	role := request.Role
	if role == "" {
		return nil, internalerrors.ErrBadRequest
	}

	validatedRoleAssignment, err := s.accessControlService.ValidateRoleAssignment(ctx, role, &actorUserID)
	if err != nil {
		if err.Error() == internalerrors.ErrForbidden.Error() {
			return nil, internalerrors.ErrForbidden
		}
		if err.Error() == internalerrors.ErrNotFound.Error() {
			return nil, internalerrors.ErrBadRequest
		}
		return nil, err
	}
	if !validatedRoleAssignment {
		return nil, internalerrors.ErrBadRequest
	}

	if actorMember != nil && actorMember.UserID == member.UserID {
		return nil, internalerrors.ErrForbidden
	}

	member.Role = role

	updated, err := s.orgMemberRepo.Update(ctx, member)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *OrganizationMemberService) RemoveMember(ctx context.Context, actorUserID string, organizationID string, memberID string) error {
	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID); err != nil {
		return err
	}

	member, err := s.orgMemberRepo.GetByID(ctx, memberID)
	if err != nil {
		return err
	}
	if member == nil || member.OrganizationID != organizationID {
		return internalerrors.ErrNotFound
	}

	if err := s.orgMemberRepo.Delete(ctx, member.ID); err != nil {
		return err
	}

	return nil
}
