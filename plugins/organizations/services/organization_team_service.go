package services

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/types"
)

type OrganizationTeamService struct {
	orgRepo           repositories.OrganizationRepository
	orgMemberRepo     repositories.OrganizationMemberRepository
	orgTeamRepo       repositories.OrganizationTeamRepository
	orgTeamMemberRepo repositories.OrganizationTeamMemberRepository
	serviceUtils      *ServiceUtils
	txRunner          organizationTeamTxRunner
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
) *OrganizationTeamService {
	return &OrganizationTeamService{orgRepo: orgRepo, orgTeamRepo: orgTeamRepo, orgMemberRepo: orgMemberRepo, orgTeamMemberRepo: orgTeamMemberRepo, serviceUtils: serviceUtils, txRunner: txRunner}
}

func (s *OrganizationTeamService) CreateTeam(ctx context.Context, actorUserID string, organizationID string, request types.CreateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	organization, actorMember, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID)
	if err != nil {
		return nil, err
	}

	name := request.Name
	if name == "" {
		return nil, internalerrors.ErrBadRequest
	}

	slug := ""
	if request.Slug != nil {
		slug = *request.Slug
	}
	if slug == "" {
		slug = s.serviceUtils.slugify(name)
	}
	if slug == "" {
		return nil, internalerrors.ErrBadRequest
	}

	if existing, err := s.orgTeamRepo.GetByOrganizationIDAndSlug(ctx, organizationID, slug); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, internalerrors.ErrConflict
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
		team.Metadata = []byte("{}")
	}

	var created *types.OrganizationTeam
	createFn := func(ctx context.Context, memberRepo repositories.OrganizationMemberRepository, teamRepo repositories.OrganizationTeamRepository, teamMemberRepo repositories.OrganizationTeamMemberRepository) error {
		createdTeam, err := teamRepo.Create(ctx, team)
		if err != nil {
			return err
		}

		member := actorMember
		if member == nil {
			member, err = memberRepo.GetByOrganizationIDAndUserID(ctx, organization.ID, actorUserID)
			if err != nil {
				return err
			}
		}
		if member != nil {
			teamMember := &types.OrganizationTeamMember{
				ID:     util.GenerateUUID(),
				TeamID: createdTeam.ID,
				UserID: member.ID,
			}

			_, err = teamMemberRepo.Create(ctx, teamMember)
			if err != nil {
				return err
			}
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
		return created, nil
	}

	if err := createFn(ctx, s.orgMemberRepo, s.orgTeamRepo, s.orgTeamMemberRepo); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *OrganizationTeamService) GetAllTeams(ctx context.Context, actorUserID string, organizationID string) ([]types.OrganizationTeam, error) {
	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID); err != nil {
		return nil, err
	}

	return s.orgTeamRepo.GetAllByOrganizationID(ctx, organizationID)
}

func (s *OrganizationTeamService) GetTeam(ctx context.Context, actorUserID string, organizationID string, teamID string) (*types.OrganizationTeam, error) {
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

	return team, nil
}

func (s *OrganizationTeamService) UpdateTeam(ctx context.Context, actorUserID string, organizationID string, teamID string, request types.UpdateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
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

	name := request.Name
	if name == "" {
		return nil, internalerrors.ErrBadRequest
	}

	slug := team.Slug
	if request.Slug != nil {
		slug = *request.Slug
	}
	if slug == "" {
		slug = s.serviceUtils.slugify(name)
	}
	if slug == "" {
		return nil, internalerrors.ErrBadRequest
	}

	if existing, err := s.orgTeamRepo.GetByOrganizationIDAndSlug(ctx, organizationID, slug); err != nil {
		return nil, err
	} else if existing != nil && existing.ID != teamID {
		return nil, internalerrors.ErrConflict
	}

	team.Name = name
	team.Slug = slug
	team.Description = request.Description
	team.Metadata = request.Metadata
	if len(team.Metadata) == 0 {
		team.Metadata = []byte("{}")
	}

	updated, err := s.orgTeamRepo.Update(ctx, team)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *OrganizationTeamService) DeleteTeam(ctx context.Context, actorUserID string, organizationID string, teamID string) error {
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

	if err := s.orgTeamRepo.Delete(ctx, teamID); err != nil {
		return err
	}

	return nil
}
