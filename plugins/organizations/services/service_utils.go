package services

import (
	"context"
	"strings"
	"unicode"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
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

func (s *ServiceUtils) authorizeOwner(ctx context.Context, actorUserID string, organizationID string) (*types.Organization, error) {
	if actorUserID == "" || organizationID == "" {
		return nil, internalerrors.ErrUnauthorized
	}

	organization, err := s.orgRepo.GetByID(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if organization == nil {
		return nil, internalerrors.ErrNotFound
	}
	if organization.OwnerID != actorUserID {
		return nil, internalerrors.ErrForbidden
	}

	return organization, nil
}

func (s *ServiceUtils) authorizeOrganizationAccess(ctx context.Context, actorUserID string, organizationID string) (*types.Organization, *types.OrganizationMember, error) {
	if actorUserID == "" || organizationID == "" {
		return nil, nil, internalerrors.ErrUnauthorized
	}

	organization, err := s.orgRepo.GetByID(ctx, organizationID)
	if err != nil {
		return nil, nil, err
	}
	if organization == nil {
		return nil, nil, internalerrors.ErrNotFound
	}
	if organization.OwnerID == actorUserID {
		return organization, nil, nil
	}

	member, err := s.orgMemberRepo.GetByOrganizationIDAndUserID(ctx, organizationID, actorUserID)
	if err != nil {
		return nil, nil, err
	}
	if member == nil {
		return nil, nil, internalerrors.ErrForbidden
	}

	return organization, member, nil
}

func (s *ServiceUtils) ensureOrganizationMembersLimit(ctx context.Context, memberRepo repositories.OrganizationMemberRepository, organizationID string, membersLimit *int) error {
	if membersLimit == nil || *membersLimit <= 0 {
		return nil
	}

	memberCount, err := memberRepo.CountByOrganizationID(ctx, organizationID)
	if err != nil {
		return err
	}
	if memberCount >= *membersLimit {
		return orgconstants.ErrMembersQuotaExceeded
	}

	return nil
}

func (s *ServiceUtils) ensureOrganizationInvitationsLimit(ctx context.Context, invitationRepo repositories.OrganizationInvitationRepository, organizationID string, email string, invitationsLimit *int) error {
	if invitationsLimit == nil || *invitationsLimit <= 0 {
		return nil
	}

	invitationCount, err := invitationRepo.CountByOrganizationIDAndEmail(ctx, organizationID, email)
	if err != nil {
		return err
	}
	if invitationCount >= *invitationsLimit {
		return orgconstants.ErrInvitationsQuotaExceeded
	}

	return nil
}

func (s *ServiceUtils) ensureEmailVerifiedForInvitationAcceptance(ctx context.Context, userService rootservices.UserService, userID string, requireEmailVerified bool) (*models.User, error) {
	if userID == "" {
		return nil, internalerrors.ErrNotFound
	}

	user, err := userService.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.Email == "" {
		return nil, internalerrors.ErrNotFound
	}
	if requireEmailVerified && !user.EmailVerified {
		return nil, internalerrors.ErrForbidden
	}

	return user, nil
}

func (s *ServiceUtils) slugify(input string) string {
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
