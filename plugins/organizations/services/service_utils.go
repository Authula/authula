package services

import (
	"context"
	"errors"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
)

type ServiceUtils struct {
	orgRepo           repositories.OrganizationRepository
	orgMemberRepo     repositories.OrganizationMemberRepository
	orgTeamRepo       repositories.OrganizationTeamRepository
	orgTeamMemberRepo repositories.OrganizationTeamMemberRepository
}

func NewServiceUtils(orgRepo repositories.OrganizationRepository, orgMemberRepo repositories.OrganizationMemberRepository, orgTeamRepo repositories.OrganizationTeamRepository, orgTeamMemberRepo repositories.OrganizationTeamMemberRepository) *ServiceUtils {
	return &ServiceUtils{
		orgRepo:           orgRepo,
		orgMemberRepo:     orgMemberRepo,
		orgTeamRepo:       orgTeamRepo,
		orgTeamMemberRepo: orgTeamMemberRepo,
	}
}

func (s *ServiceUtils) authorizeOwner(ctx context.Context, actor *models.Actor, organizationID string) (*types.Organization, error) {
	if actor == nil || actor.ID == "" || organizationID == "" {
		return nil, coreerrors.ErrUnauthorized
	}

	organization, err := s.orgRepo.GetByID(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if organization == nil {
		return nil, coreerrors.ErrNotFound
	}
	if err := verifyOrgClaim(actor, organizationID); err != nil {
		return nil, err
	}
	if organization.OwnerID != actor.ID {
		return nil, coreerrors.ErrForbidden
	}

	return organization, nil
}

func (s *ServiceUtils) AuthorizeOrganizationAccess(ctx context.Context, actor *models.Actor, organizationID string) (*types.Organization, *types.OrganizationMember, error) {
	if actor == nil || actor.ID == "" || organizationID == "" {
		return nil, nil, coreerrors.ErrUnauthorized
	}

	organization, err := s.orgRepo.GetByID(ctx, organizationID)
	if err != nil {
		return nil, nil, err
	}
	if organization == nil {
		return nil, nil, coreerrors.ErrNotFound
	}
	if err := verifyOrgClaim(actor, organizationID); err != nil {
		return nil, nil, err
	}

	if actor.Type == models.ActorMachine {
		return organization, nil, nil
	}

	member, err := s.orgMemberRepo.GetByOrganizationIDAndUserID(ctx, organizationID, actor.ID)
	if err != nil {
		return nil, nil, err
	}
	if member == nil {
		return nil, nil, coreerrors.ErrForbidden
	}

	return organization, member, nil
}

func (s *ServiceUtils) authorizeTeamAccess(ctx context.Context, actor *models.Actor, orgID string, teamID string) error {
	team, err := s.orgTeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil || team.OrganizationID != orgID {
		return coreerrors.ErrNotFound
	}

	if err := verifyOrgClaim(actor, orgID); err != nil {
		return err
	}

	if actor.Type == models.ActorMachine {
		return nil
	}

	member, err := s.orgMemberRepo.GetByOrganizationIDAndUserID(ctx, orgID, actor.ID)
	if err != nil {
		return err
	}
	if member == nil {
		return coreerrors.ErrForbidden
	}

	tm, err := s.orgTeamMemberRepo.GetByTeamIDAndMemberID(ctx, teamID, member.ID)
	if err != nil {
		return err
	}
	if tm == nil {
		return coreerrors.ErrForbidden
	}
	return nil
}

func verifyOrgClaim(actor *models.Actor, organizationID string) error {
	claimOrgID, hasClaim := actor.GetClaimString("organization_id")
	if !hasClaim || claimOrgID == "" {
		if actor.Type == models.ActorMachine {
			return coreerrors.ErrForbidden
		}
		return nil
	}
	if claimOrgID != organizationID {
		return coreerrors.ErrForbidden
	}
	return nil
}

func authorizeRoleWeight(ctx context.Context, accessControl rootservices.AccessControlService, actorMember *types.OrganizationMember, targetRole string) error {
	if actorMember == nil {
		return coreerrors.ErrForbidden
	}

	actorWeight, err := accessControl.GetRoleWeightByName(ctx, actorMember.Role)
	if err != nil {
		if errors.Is(err, coreerrors.ErrNotFound) {
			return coreerrors.ErrForbidden
		}
		return err
	}

	targetWeight, err := accessControl.GetRoleWeightByName(ctx, targetRole)
	if err != nil {
		if errors.Is(err, coreerrors.ErrNotFound) {
			return coreerrors.ErrBadRequest
		}
		return err
	}

	if targetWeight > actorWeight {
		return coreerrors.ErrForbidden
	}
	return nil
}
