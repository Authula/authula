package services

import (
	"context"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/pagination"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/types"
	"github.com/Authula/authula/util"
)

type organizationTeamMemberService struct {
	orgRepo           repositories.OrganizationRepository
	orgMemberRepo     repositories.OrganizationMemberRepository
	orgTeamRepo       repositories.OrganizationTeamRepository
	orgTeamMemberRepo repositories.OrganizationTeamMemberRepository
	serviceUtils      *ServiceUtils
	hooks             *ServiceHookExecutor
}

func NewOrganizationTeamMemberService(
	orgRepo repositories.OrganizationRepository,
	orgMemberRepo repositories.OrganizationMemberRepository,
	teamRepo repositories.OrganizationTeamRepository,
	orgTeamMemberRepo repositories.OrganizationTeamMemberRepository,
	serviceUtils *ServiceUtils,
	hooks ...*ServiceHookExecutor,
) *organizationTeamMemberService {
	var hook *ServiceHookExecutor
	if len(hooks) > 0 {
		hook = hooks[0]
	}
	return &organizationTeamMemberService{orgRepo: orgRepo, orgMemberRepo: orgMemberRepo, orgTeamRepo: teamRepo, orgTeamMemberRepo: orgTeamMemberRepo, serviceUtils: serviceUtils, hooks: hook}
}

func (s *organizationTeamMemberService) AddTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.AddOrganizationTeamMemberRequest) (*types.OrganizationTeamMember, error) {
	orgMemberID := request.MemberID
	if orgMemberID == "" {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	if _, _, err := s.serviceUtils.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}

	if err := s.serviceUtils.authorizeTeamAccess(ctx, actor, organizationID, teamID); err != nil {
		return nil, err
	}

	team, err := s.orgTeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team == nil || team.OrganizationID != organizationID {
		return nil, coreerrors.ErrNotFound
	}

	orgMember, err := s.orgMemberRepo.GetByID(ctx, orgMemberID)
	if err != nil {
		return nil, err
	}
	if orgMember == nil || orgMember.OrganizationID != organizationID {
		return nil, coreerrors.ErrNotFound
	}

	if existing, err := s.orgTeamMemberRepo.GetByTeamIDAndMemberID(ctx, teamID, orgMemberID); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, coreerrors.ErrConflict
	}

	teamMember := &types.OrganizationTeamMember{
		ID:       util.GenerateUUID(),
		TeamID:   teamID,
		MemberID: orgMember.ID,
	}

	if s.hooks != nil {
		if err := s.hooks.BeforeCreateOrganizationTeamMember(ctx, actor, teamMember); err != nil {
			return nil, err
		}
	}

	created, err := s.orgTeamMemberRepo.Create(ctx, teamMember)
	if err != nil {
		return nil, err
	}

	if s.hooks != nil {
		if err := s.hooks.AfterCreateOrganizationTeamMember(ctx, actor, created); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *organizationTeamMemberService) GetAllTeamMembers(ctx context.Context, actor *models.Actor, organizationID string, teamID string, params pagination.Params) (*types.ListOrganizationTeamMembersResponse, error) {
	if _, _, err := s.serviceUtils.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}

	if err := s.serviceUtils.authorizeTeamAccess(ctx, actor, organizationID, teamID); err != nil {
		return nil, err
	}

	team, err := s.orgTeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team == nil || team.OrganizationID != organizationID {
		return nil, coreerrors.ErrNotFound
	}

	params = pagination.Clamp(params)

	teamMembers, total, err := s.orgTeamMemberRepo.GetAllByTeamIDWithMemberAndUser(ctx, teamID, params.Page, params.Limit)
	if err != nil {
		return nil, err
	}
	if teamMembers == nil {
		teamMembers = []types.OrganizationTeamMemberResponse{}
	}

	return &types.ListOrganizationTeamMembersResponse{
		Data:       teamMembers,
		Pagination: pagination.New(params.Page, params.Limit, total),
	}, nil
}

func (s *organizationTeamMemberService) GetTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) (*types.OrganizationTeamMemberResponse, error) {
	if _, _, err := s.serviceUtils.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}

	if err := s.serviceUtils.authorizeTeamAccess(ctx, actor, organizationID, teamID); err != nil {
		return nil, err
	}

	team, err := s.orgTeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team == nil || team.OrganizationID != organizationID {
		return nil, coreerrors.ErrNotFound
	}

	orgMember, err := s.orgMemberRepo.GetByIDWithUser(ctx, memberID)
	if err != nil {
		return nil, err
	}
	if orgMember == nil || orgMember.OrganizationID != organizationID {
		return nil, coreerrors.ErrNotFound
	}

	teamMember, err := s.orgTeamMemberRepo.GetByTeamIDAndMemberID(ctx, teamID, orgMember.ID)
	if err != nil {
		return nil, err
	}
	if teamMember == nil {
		return nil, coreerrors.ErrNotFound
	}

	return &types.OrganizationTeamMemberResponse{
		ID:        teamMember.ID,
		TeamID:    teamMember.TeamID,
		CreatedAt: teamMember.CreatedAt,
		Member:    *orgMember,
	}, nil
}

func (s *organizationTeamMemberService) RemoveTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) error {
	if _, _, err := s.serviceUtils.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return err
	}

	if err := s.serviceUtils.authorizeTeamAccess(ctx, actor, organizationID, teamID); err != nil {
		return err
	}

	team, err := s.orgTeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team == nil || team.OrganizationID != organizationID {
		return coreerrors.ErrNotFound
	}

	orgMember, err := s.orgMemberRepo.GetByID(ctx, memberID)
	if err != nil {
		return err
	}
	if orgMember == nil || orgMember.OrganizationID != organizationID {
		return coreerrors.ErrNotFound
	}

	teamMember, err := s.orgTeamMemberRepo.GetByTeamIDAndMemberID(ctx, teamID, orgMember.ID)
	if err != nil {
		return err
	}
	if teamMember == nil {
		return coreerrors.ErrNotFound
	}

	if s.hooks != nil {
		if err := s.hooks.BeforeDeleteOrganizationTeamMember(ctx, actor, teamMember); err != nil {
			return err
		}
	}

	if err := s.orgTeamMemberRepo.DeleteByTeamIDAndMemberID(ctx, teamID, orgMember.ID); err != nil {
		return err
	}

	if s.hooks != nil {
		if err := s.hooks.AfterDeleteOrganizationTeamMember(ctx, actor, teamMember); err != nil {
			return err
		}
	}

	return nil
}
