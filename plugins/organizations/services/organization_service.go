package services

import (
	"context"
	"database/sql"
	"sort"
	"sync"

	"github.com/uptrace/bun"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
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

func (s *organizationService) CreateOrganization(ctx context.Context, actorUserID string, request types.CreateOrganizationRequest) (*types.Organization, error) {
	if actorUserID == "" {
		return nil, internalerrors.ErrUnauthorized
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
		OwnerID:  actorUserID,
		Name:     name,
		Slug:     slug,
		Logo:     request.Logo,
		Metadata: request.Metadata,
	}
	if len(organization.Metadata) == 0 {
		organization.Metadata = []byte("{}")
	}

	var created *types.Organization
	createFn := func(ctx context.Context, orgRepo repositories.OrganizationRepository, memberRepo repositories.OrganizationMemberRepository) error {
		if err := s.ensureOrganizationLimit(ctx, actorUserID, orgRepo, memberRepo); err != nil {
			return err
		}

		createdOrganization, err := orgRepo.Create(ctx, organization)
		if err != nil {
			return err
		}

		member := &types.OrganizationMember{
			ID:             util.GenerateUUID(),
			OrganizationID: createdOrganization.ID,
			UserID:         actorUserID,
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

func (s *organizationService) ensureOrganizationLimit(ctx context.Context, actorUserID string, orgRepo repositories.OrganizationRepository, memberRepo repositories.OrganizationMemberRepository) error {
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
		ownedOrganizations, ownedOrganizationsErr = orgRepo.GetAllByOwnerID(ctx, actorUserID)
	})

	wg.Go(func() {
		memberRecords, memberRecordsErr = memberRepo.GetAllByUserID(ctx, actorUserID)
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

func (s *organizationService) GetAllOrganizations(ctx context.Context, actorUserID string) ([]types.Organization, error) {
	if actorUserID == "" {
		return nil, internalerrors.ErrUnauthorized
	}

	ownedOrganizations, err := s.orgRepo.GetAllByOwnerID(ctx, actorUserID)
	if err != nil {
		return nil, err
	}

	memberRecords, err := s.orgMemberRepo.GetAllByUserID(ctx, actorUserID)
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

func (s *organizationService) GetOrganizationByID(ctx context.Context, actorUserID string, organizationID string) (*types.Organization, error) {
	organization, err := s.authorizeMember(ctx, actorUserID, organizationID)
	if err != nil {
		return nil, err
	}

	return organization, nil
}

func (s *organizationService) UpdateOrganization(ctx context.Context, actorUserID string, organizationID string, request types.UpdateOrganizationRequest) (*types.Organization, error) {
	organization, err := s.authorizeMember(ctx, actorUserID, organizationID)
	if err != nil {
		return nil, err
	}

	name := request.Name
	if name == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	slug := organization.Slug
	if request.Slug != nil {
		slug = *request.Slug
	}
	if slug == "" {
		slug = s.serviceUtils.slugify(name)
	}
	if slug == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	organization.Name = name
	organization.Slug = slug
	if request.Logo != nil {
		organization.Logo = request.Logo
	}
	if request.Metadata != nil {
		organization.Metadata = request.Metadata
	}
	if len(organization.Metadata) == 0 {
		organization.Metadata = []byte("{}")
	}

	updated, err := s.orgRepo.Update(ctx, organization)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *organizationService) DeleteOrganization(ctx context.Context, actorUserID string, organizationID string) error {
	_, err := s.serviceUtils.authorizeOwner(ctx, actorUserID, organizationID)
	if err != nil {
		return err
	}

	if err := s.orgRepo.Delete(ctx, organizationID); err != nil {
		return err
	}

	return nil
}

func (s *organizationService) authorizeMember(ctx context.Context, actorUserID string, organizationID string) (*types.Organization, error) {
	organization, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID)
	if err != nil {
		return nil, err
	}

	return organization, nil
}
