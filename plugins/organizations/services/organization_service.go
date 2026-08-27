package services

import (
	"context"
	"database/sql"
	"strings"
	"unicode"

	"github.com/uptrace/bun"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/pagination"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/constants"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type organizationService struct {
	orgRepo              repositories.OrganizationRepository
	orgMemberRepo        repositories.OrganizationMemberRepository
	serviceUtils         *ServiceUtils
	accessControlService rootservices.AccessControlService
	organizationsLimit   *int
	txRunner             organizationTxRunner
	hooks                *ServiceHookExecutor
}

type organizationTxRunner interface {
	RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(context.Context, bun.Tx) error) error
}

func NewOrganizationService(
	orgRepo repositories.OrganizationRepository,
	orgMemberRepo repositories.OrganizationMemberRepository,
	serviceUtils *ServiceUtils,
	accessControlService rootservices.AccessControlService,
	organizationsLimit *int,
	txRunner organizationTxRunner,
	hooks ...*ServiceHookExecutor,
) *organizationService {
	var hook *ServiceHookExecutor
	if len(hooks) > 0 {
		hook = hooks[0]
	}
	return &organizationService{
		orgRepo:              orgRepo,
		orgMemberRepo:        orgMemberRepo,
		serviceUtils:         serviceUtils,
		accessControlService: accessControlService,
		organizationsLimit:   organizationsLimit,
		txRunner:             txRunner,
		hooks:                hook,
	}
}

func (s *organizationService) CreateOrganization(ctx context.Context, actor *models.Actor, request types.CreateOrganizationRequest) (*types.Organization, error) {
	if actor == nil || actor.ID == "" {
		return nil, coreerrors.ErrUnauthorized
	}

	name := request.Name
	if name == "" {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	role := request.Role
	if role == "" {
		return nil, coreerrors.ErrUnprocessableEntity
	}
	roleExists, err := s.accessControlService.RoleExists(ctx, role)
	if err != nil {
		return nil, err
	}
	if !roleExists {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	rolePermissions, err := s.accessControlService.GetRolePermissionsByName(ctx, role)
	if err != nil {
		return nil, err
	}
	if !constants.CoversAllOrganizationPermissions(rolePermissions) {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	slug := ""
	if request.Slug != nil {
		slug = *request.Slug
	}
	if slug == "" {
		slug = slugify(name)
	}
	if slug == "" {
		return nil, coreerrors.ErrUnprocessableEntity
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

	if s.hooks != nil {
		if err := s.hooks.BeforeCreateOrganization(ctx, actor, organization); err != nil {
			return nil, err
		}
	}

	var created *types.Organization
	createFn := func(ctx context.Context, orgRepo repositories.OrganizationRepository, memberRepo repositories.OrganizationMemberRepository) error {
		if err := s.ensureOrganizationLimit(ctx, actor, orgRepo); err != nil {
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

	if s.hooks != nil {
		if err := s.hooks.AfterCreateOrganization(ctx, actor, created); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *organizationService) ensureOrganizationLimit(ctx context.Context, actor *models.Actor, orgRepo repositories.OrganizationRepository) error {
	if actor == nil || actor.ID == "" {
		return coreerrors.ErrUnauthorized
	}

	if s.organizationsLimit == nil || *s.organizationsLimit <= 0 {
		return nil
	}

	organizationCount, err := orgRepo.CountAccessibleByUserID(ctx, actor.ID)
	if err != nil {
		return err
	}

	if organizationCount >= *s.organizationsLimit {
		return constants.ErrOrganizationsQuotaExceeded
	}

	return nil
}

func (s *organizationService) ListAllOrganizations(ctx context.Context, actor *models.Actor, params pagination.Params) (*types.ListOrganizationsResponse, error) {
	if actor == nil || actor.ID == "" {
		return nil, coreerrors.ErrUnauthorized
	}

	params = s.serviceUtils.ClampPagination(params)

	organizations, total, err := s.orgRepo.ListAllAccessibleByUserID(ctx, actor.ID, params.Page, params.Limit)
	if err != nil {
		return nil, err
	}
	if organizations == nil {
		organizations = []types.Organization{}
	}

	return &types.ListOrganizationsResponse{
		Data:       organizations,
		Pagination: pagination.New(params.Page, params.Limit, total),
	}, nil
}

func (s *organizationService) GetAllOrganizations(ctx context.Context, actor *models.Actor) ([]types.Organization, error) {
	if actor == nil || actor.ID == "" {
		return nil, coreerrors.ErrUnauthorized
	}

	organizations, err := s.orgRepo.GetAllAccessibleByUserID(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	if organizations == nil {
		organizations = []types.Organization{}
	}

	return organizations, nil
}

// GetAllOrganizationsUnscoped bypasses actor scoping entirely. See the interface
// documentation before adding a caller.
func (s *organizationService) GetAllOrganizationsUnscoped(ctx context.Context) ([]types.Organization, error) {
	organizations, err := s.orgRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	if organizations == nil {
		organizations = []types.Organization{}
	}

	return organizations, nil
}

func (s *organizationService) GetOrganizationByID(ctx context.Context, actor *models.Actor, organizationID string) (*types.Organization, error) {
	organization, _, err := s.serviceUtils.AuthorizeOrganizationAccess(ctx, actor, organizationID)
	if err != nil {
		return nil, err
	}

	return organization, nil
}

func (s *organizationService) UpdateOrganization(ctx context.Context, actor *models.Actor, organizationID string, request types.UpdateOrganizationRequest) (*types.Organization, error) {
	organization, err := s.serviceUtils.authorizeOwner(ctx, actor, organizationID)
	if err != nil {
		return nil, err
	}

	name := request.Name
	if name != nil && *name == "" {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	slug := organization.Slug
	if request.Slug != nil {
		slug = *request.Slug
	}
	if slug == "" {
		slug = slugify(*name)
	}
	if slug == "" {
		return nil, coreerrors.ErrUnprocessableEntity
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

	if s.hooks != nil {
		if err := s.hooks.BeforeUpdateOrganization(ctx, actor, organization); err != nil {
			return nil, err
		}
	}

	updated, err := s.orgRepo.Update(ctx, organization)
	if err != nil {
		return nil, err
	}

	if s.hooks != nil {
		if err := s.hooks.AfterUpdateOrganization(ctx, actor, updated); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

func (s *organizationService) ExistsByID(ctx context.Context, organizationID string) (bool, error) {
	org, err := s.orgRepo.GetByID(ctx, organizationID)
	if err != nil {
		return false, err
	}
	return org != nil, nil
}

func (s *organizationService) DeleteOrganization(ctx context.Context, actor *models.Actor, organizationID string) error {
	organization, err := s.serviceUtils.authorizeOwner(ctx, actor, organizationID)
	if err != nil {
		return err
	}

	if s.hooks != nil {
		if err := s.hooks.BeforeDeleteOrganization(ctx, actor, organization); err != nil {
			return err
		}
	}

	if err := s.orgRepo.Delete(ctx, organizationID); err != nil {
		return err
	}

	if s.hooks != nil {
		if err := s.hooks.AfterDeleteOrganization(ctx, actor, organization); err != nil {
			return err
		}
	}

	return nil
}

func slugify(input string) string {
	input = strings.ToLower(input)
	var builder strings.Builder
	lastDash := false

	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}

	return strings.Trim(builder.String(), "-")
}
