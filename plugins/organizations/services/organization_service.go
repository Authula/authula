package services

import (
	"context"
	"database/sql"
	"sort"
	"sync"

	"github.com/uptrace/bun"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/constants"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
)

type organizationService struct {
	orgRepo              repositories.OrganizationRepository
	orgMemberRepo        repositories.OrganizationMemberRepository
	serviceUtils         *ServiceUtils
	accessControlService rootservices.AccessControlService
	organizationsLimit   *int
	txRunner             organizationTxRunner
}

type organizationTxRunner interface {
	RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(context.Context, bun.Tx) error) error
}

func NewOrganizationService(orgRepo repositories.OrganizationRepository, orgMemberRepo repositories.OrganizationMemberRepository, serviceUtils *ServiceUtils, accessControlService rootservices.AccessControlService, organizationsLimit *int, txRunner organizationTxRunner) *organizationService {
	return &organizationService{orgRepo: orgRepo, orgMemberRepo: orgMemberRepo, serviceUtils: serviceUtils, accessControlService: accessControlService, organizationsLimit: organizationsLimit, txRunner: txRunner}
}

func (s *organizationService) CreateOrganization(ctx context.Context, actor *models.Actor, request types.CreateOrganizationRequest) (*types.Organization, error) {
	if actor == nil || actor.ID == "" {
		return nil, internalerrors.ErrUnauthorized
	}

	if err := s.serviceUtils.authorizerOrDefault().Authorize(ctx, actor, ActionOrganizationsCreate, AuthorizerResource{}); err != nil {
		return nil, err
	}

	name := request.Name
	if name == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	role := request.Role
	if role == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}
	roleExists, err := s.accessControlService.RoleExists(ctx, role)
	if err != nil {
		return nil, err
	}
	if !roleExists {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	slug := ""
	if request.Slug != nil {
		slug = *request.Slug
	}
	if slug == "" {
		slug = s.serviceUtils.slugify(name)
	}
	if slug == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	organization := &types.Organization{
		ID:       util.GenerateUUID(),
		OwnerID:  actor.ID,
		Name:     name,
		Slug:     slug,
		Logo:     request.Logo,
		Metadata: request.Metadata,
	}
	if len(organization.Metadata) == 0 {
		organization.Metadata = make(map[string]any)
	}

	var created *types.Organization
	createFn := func(ctx context.Context, orgRepo repositories.OrganizationRepository, memberRepo repositories.OrganizationMemberRepository) error {
		if err := s.ensureOrganizationLimit(ctx, actor, orgRepo, memberRepo); err != nil {
			return err
		}

		createdOrganization, err := orgRepo.Create(ctx, organization)
		if err != nil {
			return err
		}

		member := &types.OrganizationMember{
			ID:             util.GenerateUUID(),
			OrganizationID: createdOrganization.ID,
			UserID:         actor.ID,
			Role:           role,
		}

		_, err = memberRepo.Create(ctx, member)
		if err != nil {
			return err
		}

		created = createdOrganization
		return nil
	}

	err = s.txRunner.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return createFn(ctx, s.orgRepo.WithTx(tx), s.orgMemberRepo.WithTx(tx))
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *organizationService) ensureOrganizationLimit(ctx context.Context, actor *models.Actor, orgRepo repositories.OrganizationRepository, memberRepo repositories.OrganizationMemberRepository) error {
	if actor == nil || actor.ID == "" {
		return internalerrors.ErrUnauthorized
	}

	if s.organizationsLimit == nil || *s.organizationsLimit <= 0 {
		return nil
	}

	var (
		ownedOrganizations    []types.Organization
		memberRecords         []types.OrganizationMember
		ownedOrganizationsErr error
		memberRecordsErr      error
	)

	wg := sync.WaitGroup{}

	wg.Go(func() {
		ownedOrganizations, ownedOrganizationsErr = orgRepo.GetAllByOwnerID(ctx, actor.ID)
	})

	wg.Go(func() {
		memberRecords, memberRecordsErr = memberRepo.GetAllByUserID(ctx, actor.ID)
	})

	wg.Wait()

	if ownedOrganizationsErr != nil {
		return ownedOrganizationsErr
	}
	if memberRecordsErr != nil {
		return memberRecordsErr
	}

	organizationIDs := make(map[string]struct{}, len(ownedOrganizations)+len(memberRecords))
	for _, organization := range ownedOrganizations {
		if organization.ID == "" {
			continue
		}
		organizationIDs[organization.ID] = struct{}{}
	}
	for _, member := range memberRecords {
		if member.OrganizationID == "" {
			continue
		}
		organizationIDs[member.OrganizationID] = struct{}{}
	}

	if len(organizationIDs) >= *s.organizationsLimit {
		return constants.ErrOrganizationsQuotaExceeded
	}

	return nil
}

func (s *organizationService) GetAllOrganizations(ctx context.Context, actor *models.Actor) ([]types.Organization, error) {
	if actor == nil || actor.ID == "" {
		return nil, internalerrors.ErrUnauthorized
	}

	if err := s.serviceUtils.authorizerOrDefault().Authorize(ctx, actor, ActionOrganizationsList, AuthorizerResource{}); err != nil {
		return nil, err
	}

	ownedOrganizations, err := s.orgRepo.GetAllByOwnerID(ctx, actor.ID)
	if err != nil {
		return nil, err
	}

	memberRecords, err := s.orgMemberRepo.GetAllByUserID(ctx, actor.ID)
	if err != nil {
		return nil, err
	}

	organizationMap := make(map[string]types.Organization, len(ownedOrganizations))
	for _, organization := range ownedOrganizations {
		organizationMap[organization.ID] = organization
	}

	for _, member := range memberRecords {
		if member.OrganizationID == "" {
			continue
		}
		if _, exists := organizationMap[member.OrganizationID]; exists {
			continue
		}

		organization, err := s.orgRepo.GetByID(ctx, member.OrganizationID)
		if err != nil {
			return nil, err
		}
		if organization == nil {
			continue
		}

		organizationMap[organization.ID] = *organization
	}

	organizationIDs := make([]string, 0, len(organizationMap))
	for organizationID := range organizationMap {
		organizationIDs = append(organizationIDs, organizationID)
	}
	sort.Strings(organizationIDs)

	organizations := make([]types.Organization, 0, len(organizationIDs))
	for _, organizationID := range organizationIDs {
		organizations = append(organizations, organizationMap[organizationID])
	}

	return organizations, nil
}

func (s *organizationService) GetOrganizationByID(ctx context.Context, actor *models.Actor, organizationID string) (*types.Organization, error) {
	organization, err := s.authorizeMember(ctx, actor, organizationID, ActionOrganizationsRead)
	if err != nil {
		return nil, err
	}

	return organization, nil
}

func (s *organizationService) UpdateOrganization(ctx context.Context, actor *models.Actor, organizationID string, request types.UpdateOrganizationRequest) (*types.Organization, error) {
	organization, err := s.authorizeMember(ctx, actor, organizationID, ActionOrganizationsUpdate)
	if err != nil {
		return nil, err
	}

	name := request.Name
	if name != nil && *name == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	slug := organization.Slug
	if request.Slug != nil {
		slug = *request.Slug
	}
	if slug == "" {
		slug = s.serviceUtils.slugify(*name)
	}
	if slug == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	if name != nil {
		organization.Name = *name
	}
	organization.Slug = slug
	if request.Logo != nil {
		organization.Logo = request.Logo
	}
	if request.Metadata != nil {
		organization.Metadata = request.Metadata
	}
	if len(organization.Metadata) == 0 {
		organization.Metadata = make(map[string]any)
	}

	updated, err := s.orgRepo.Update(ctx, organization)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *organizationService) DeleteOrganization(ctx context.Context, actor *models.Actor, organizationID string) error {
	_, err := s.serviceUtils.authorizeOwnerForAction(ctx, actor, ActionOrganizationsDelete, organizationID)
	if err != nil {
		return err
	}

	if err := s.orgRepo.Delete(ctx, organizationID); err != nil {
		return err
	}

	return nil
}

func (s *organizationService) authorizeMember(ctx context.Context, actor *models.Actor, organizationID string, action AuthorizerAction) (*types.Organization, error) {
	organization, _, err := s.serviceUtils.authorizeOrganizationAccessForAction(ctx, actor, action, organizationID)
	if err != nil {
		return nil, err
	}

	return organization, nil
}
