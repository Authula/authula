package services

import (
	"context"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/types"
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
	if _, ok := actor.GetClaimString("organization_id"); ok {
		return organization, nil
	}
	if organization.OwnerID != actor.ID {
		return nil, coreerrors.ErrForbidden
	}

	return organization, nil
}

func (s *ServiceUtils) authorizeOrganizationAccess(ctx context.Context, actor *models.Actor, organizationID string) (*types.Organization, *types.OrganizationMember, error) {
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
	if _, ok := actor.GetClaimString("organization_id"); ok {
		return organization, nil, nil
	}
	if organization.OwnerID == actor.ID {
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

	if _, ok := actor.GetClaimString("organization_id"); ok {
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
