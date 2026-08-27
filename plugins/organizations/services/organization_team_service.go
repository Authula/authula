package services

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/pagination"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/types"
	"github.com/Authula/authula/util"
)

type organizationTeamService struct {
	orgRepo           repositories.OrganizationRepository
	orgMemberRepo     repositories.OrganizationMemberRepository
	orgTeamRepo       repositories.OrganizationTeamRepository
	orgTeamMemberRepo repositories.OrganizationTeamMemberRepository
	serviceUtils      *ServiceUtils
	txRunner          organizationTeamTxRunner
	hooks             *ServiceHookExecutor
}

type organizationTeamTxRunner interface {
	RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(context.Context, bun.Tx) error) error
}

func NewOrganizationTeamService(
	orgRepo repositories.OrganizationRepository,
	orgMemberRepo repositories.OrganizationMemberRepository,
	orgTeamRepo repositories.OrganizationTeamRepository,
	orgTeamMemberRepo repositories.OrganizationTeamMemberRepository,
	serviceUtils *ServiceUtils,
	txRunner organizationTeamTxRunner,
	hooks ...*ServiceHookExecutor,
) *organizationTeamService {
	var hook *ServiceHookExecutor
	if len(hooks) > 0 {
		hook = hooks[0]
	}
	return &organizationTeamService{orgRepo: orgRepo, orgTeamRepo: orgTeamRepo, orgMemberRepo: orgMemberRepo, orgTeamMemberRepo: orgTeamMemberRepo, serviceUtils: serviceUtils, txRunner: txRunner, hooks: hook}
}

func (s *organizationTeamService) CreateTeam(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	organization, actorMember, err := s.serviceUtils.AuthorizeOrganizationAccess(ctx, actor, organizationID)
	if err != nil {
		return nil, err
	}

	actorID := actor.ID

	name := request.Name
	if name == "" {
		return nil, coreerrors.ErrBadRequest
	}

	slug := ""
	if request.Slug != nil {
		slug = *request.Slug
	}
	if slug == "" {
		slug = slugify(name)
	}
	if slug == "" {
		return nil, coreerrors.ErrBadRequest
	}

	if existing, err := s.orgTeamRepo.GetByOrganizationIDAndSlug(ctx, organizationID, slug); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, coreerrors.ErrConflict
	}

	team := &types.OrganizationTeam{
		ID:             util.GenerateUUID(),
		OrganizationID: organizationID,
		Name:           name,
		Slug:           slug,
		Description:    request.Description,
		Metadata:       request.Metadata,
	}
	if len(team.Metadata) == 0 {
		team.Metadata = make(map[string]any)
	}

	if s.hooks != nil {
		if err := s.hooks.BeforeCreateOrganizationTeam(ctx, actor, team); err != nil {
			return nil, err
		}
	}

	var created *types.OrganizationTeam
	createFn := func(ctx context.Context, memberRepo repositories.OrganizationMemberRepository, teamRepo repositories.OrganizationTeamRepository, teamMemberRepo repositories.OrganizationTeamMemberRepository) error {
		createdTeam, err := teamRepo.Create(ctx, team)
		if err != nil {
			return err
		}

		if actorMember == nil {
			actorMember, err = memberRepo.GetByOrganizationIDAndUserID(ctx, organization.ID, actorID)
			if err != nil {
				return err
			}
		}

		teamMember := &types.OrganizationTeamMember{
			ID:       util.GenerateUUID(),
			TeamID:   createdTeam.ID,
			MemberID: actorMember.ID,
		}

		_, err = teamMemberRepo.Create(ctx, teamMember)
		if err != nil {
			return err
		}

		created = createdTeam
		return nil
	}

	if s.txRunner != nil {
		err = s.txRunner.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			return createFn(ctx, s.orgMemberRepo.WithTx(tx), s.orgTeamRepo.WithTx(tx), s.orgTeamMemberRepo.WithTx(tx))
		})
		if err != nil {
			return nil, err
		}

		if s.hooks != nil {
			if err := s.hooks.AfterCreateOrganizationTeam(ctx, actor, created); err != nil {
				return nil, err
			}
		}

		return created, nil
	}

	if err := createFn(ctx, s.orgMemberRepo, s.orgTeamRepo, s.orgTeamMemberRepo); err != nil {
		return nil, err
	}

	if s.hooks != nil {
		if err := s.hooks.AfterCreateOrganizationTeam(ctx, actor, created); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *organizationTeamService) ListAllTeams(ctx context.Context, actor *models.Actor, organizationID string, params pagination.Params) (*types.ListOrganizationTeamsResponse, error) {
	if _, _, err := s.serviceUtils.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}

	params = s.serviceUtils.ClampPagination(params)

	teams, total, err := s.orgTeamRepo.ListAllByOrganizationID(ctx, organizationID, params.Page, params.Limit)
	if err != nil {
		return nil, err
	}
	if teams == nil {
		teams = []types.OrganizationTeam{}
	}

	return &types.ListOrganizationTeamsResponse{
		Data:       teams,
		Pagination: pagination.New(params.Page, params.Limit, total),
	}, nil
}

func (s *organizationTeamService) GetAllTeams(ctx context.Context, actor *models.Actor, organizationID string) ([]types.OrganizationTeam, error) {
	if _, _, err := s.serviceUtils.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}

	teams, err := s.orgTeamRepo.GetAllByOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if teams == nil {
		teams = []types.OrganizationTeam{}
	}

	return teams, nil
}

func (s *organizationTeamService) GetTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) (*types.OrganizationTeam, error) {
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

	return team, nil
}

func (s *organizationTeamService) UpdateTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.UpdateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
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

	name := request.Name
	if name == "" {
		return nil, coreerrors.ErrBadRequest
	}

	slug := team.Slug
	if request.Slug != nil {
		slug = *request.Slug
	}
	if slug == "" {
		slug = slugify(name)
	}
	if slug == "" {
		return nil, coreerrors.ErrBadRequest
	}

	if existing, err := s.orgTeamRepo.GetByOrganizationIDAndSlug(ctx, organizationID, slug); err != nil {
		return nil, err
	} else if existing != nil && existing.ID != teamID {
		return nil, coreerrors.ErrConflict
	}

	team.Name = name
	team.Slug = slug
	team.Description = request.Description
	team.Metadata = request.Metadata
	if len(team.Metadata) == 0 {
		team.Metadata = make(map[string]any)
	}

	if s.hooks != nil {
		if err := s.hooks.BeforeUpdateOrganizationTeam(ctx, actor, team); err != nil {
			return nil, err
		}
	}

	updated, err := s.orgTeamRepo.Update(ctx, team)
	if err != nil {
		return nil, err
	}

	if s.hooks != nil {
		if err := s.hooks.AfterUpdateOrganizationTeam(ctx, actor, updated); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

func (s *organizationTeamService) DeleteTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) error {
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

	if s.hooks != nil {
		if err := s.hooks.BeforeDeleteOrganizationTeam(ctx, actor, team); err != nil {
			return err
		}
	}

	if err := s.orgTeamRepo.Delete(ctx, teamID); err != nil {
		return err
	}

	if s.hooks != nil {
		if err := s.hooks.AfterDeleteOrganizationTeam(ctx, actor, team); err != nil {
			return err
		}
	}

	return nil
}
