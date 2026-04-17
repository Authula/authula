package services

import (
	"context"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/types"
)

type organizationTeamMemberService struct {
	orgRepo           repositories.OrganizationRepository
	orgMemberRepo     repositories.OrganizationMemberRepository
	orgTeamRepo       repositories.OrganizationTeamRepository
	orgTeamMemberRepo repositories.OrganizationTeamMemberRepository
	serviceUtils      *ServiceUtils
}

func NewOrganizationTeamMemberService(
	orgRepo repositories.OrganizationRepository,
	orgMemberRepo repositories.OrganizationMemberRepository,
	teamRepo repositories.OrganizationTeamRepository,
	orgTeamMemberRepo repositories.OrganizationTeamMemberRepository,
	serviceUtils *ServiceUtils,
) *organizationTeamMemberService {
	return &organizationTeamMemberService{orgRepo: orgRepo, orgMemberRepo: orgMemberRepo, orgTeamRepo: teamRepo, orgTeamMemberRepo: orgTeamMemberRepo, serviceUtils: serviceUtils}
}

func (s *organizationTeamMemberService) AddTeamMember(ctx context.Context, actorUserID string, organizationID string, teamID string, request types.AddOrganizationTeamMemberRequest) (*types.OrganizationTeamMember, error) {
	orgMemberID := request.MemberID
	if orgMemberID == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID); err != nil {
		return nil, err
	}

	team, err := s.orgTeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team == nil || team.OrganizationID != organizationID {
		return nil, internalerrors.ErrNotFound
	}

	orgMember, err := s.orgMemberRepo.GetByID(ctx, orgMemberID)
	if err != nil {
		return nil, err
	}
	if orgMember == nil || orgMember.OrganizationID != organizationID {
		return nil, internalerrors.ErrNotFound
	}

	if existing, err := s.orgTeamMemberRepo.GetByTeamIDAndMemberID(ctx, teamID, orgMemberID); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, internalerrors.ErrConflict
	}

	teamMember := &types.OrganizationTeamMember{
		ID:       util.GenerateUUID(),
		TeamID:   teamID,
		MemberID: orgMember.ID,
	}

	created, err := s.orgTeamMemberRepo.Create(ctx, teamMember)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *organizationTeamMemberService) GetAllTeamMembers(ctx context.Context, actorUserID string, organizationID string, teamID string, page int, limit int) ([]types.OrganizationTeamMember, error) {
	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID); err != nil {
		return nil, err
	}

	team, err := s.orgTeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team == nil || team.OrganizationID != organizationID {
		return nil, internalerrors.ErrNotFound
	}

	return s.orgTeamMemberRepo.GetAllByTeamID(ctx, teamID, page, limit)
}

func (s *organizationTeamMemberService) GetTeamMember(ctx context.Context, actorUserID string, organizationID string, teamID string, memberID string) (*types.OrganizationTeamMember, error) {
	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID); err != nil {
		return nil, err
	}

	team, err := s.orgTeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team == nil || team.OrganizationID != organizationID {
		return nil, internalerrors.ErrNotFound
	}

	orgMember, err := s.orgMemberRepo.GetByID(ctx, memberID)
	if err != nil {
		return nil, err
	}
	if orgMember == nil || orgMember.OrganizationID != organizationID {
		return nil, internalerrors.ErrNotFound
	}

	teamMember, err := s.orgTeamMemberRepo.GetByTeamIDAndMemberID(ctx, teamID, orgMember.ID)
	if err != nil {
		return nil, err
	}
	if teamMember == nil {
		return nil, internalerrors.ErrNotFound
	}

	return teamMember, nil
}

func (s *organizationTeamMemberService) RemoveTeamMember(ctx context.Context, actorUserID string, organizationID string, teamID string, memberID string) error {
	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID); err != nil {
		return err
	}

	team, err := s.orgTeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil || team.OrganizationID != organizationID {
		return internalerrors.ErrNotFound
	}

	orgMember, err := s.orgMemberRepo.GetByID(ctx, memberID)
	if err != nil {
		return err
	}
	if orgMember == nil || orgMember.OrganizationID != organizationID {
		return internalerrors.ErrNotFound
	}

	teamMember, err := s.orgTeamMemberRepo.GetByTeamIDAndMemberID(ctx, teamID, orgMember.ID)
	if err != nil {
		return err
	}
	if teamMember == nil {
		return internalerrors.ErrNotFound
	}

	if err := s.orgTeamMemberRepo.DeleteByTeamIDAndMemberID(ctx, teamID, orgMember.ID); err != nil {
		return err
	}

	return nil
}
