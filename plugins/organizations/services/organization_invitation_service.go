package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/uptrace/bun"

	internalerrors "github.com/Authula/authula/internal/errors"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgevents "github.com/Authula/authula/plugins/organizations/events"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
)

type organizationInvitationTxRunner interface {
	RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(context.Context, bun.Tx) error) error
}

type OrganizationInvitationService struct {
	txRunner             organizationInvitationTxRunner
	globalConfig         *models.Config
	pluginConfig         *types.OrganizationsPluginConfig
	logger               models.Logger
	eventBus             models.EventBus
	userService          rootservices.UserService
	mailerService        rootservices.MailerService
	accessControlService rootservices.AccessControlService
	organizationRepo     repositories.OrganizationRepository
	orgInvitationRepo    repositories.OrganizationInvitationRepository
	orgMemberRepo        repositories.OrganizationMemberRepository
	serviceUtils         *ServiceUtils
}

func NewOrganizationInvitationService(
	txRunner organizationInvitationTxRunner,
	globalConfig *models.Config,
	pluginConfig *types.OrganizationsPluginConfig,
	logger models.Logger,
	eventBus models.EventBus,
	userService rootservices.UserService,
	mailerService rootservices.MailerService,
	accessControlService rootservices.AccessControlService,
	organizationRepo repositories.OrganizationRepository,
	orgInvitationRepo repositories.OrganizationInvitationRepository,
	orgMemberRepo repositories.OrganizationMemberRepository,
	serviceUtils *ServiceUtils,
) *OrganizationInvitationService {
	return &OrganizationInvitationService{
		txRunner:             txRunner,
		globalConfig:         globalConfig,
		pluginConfig:         pluginConfig,
		logger:               logger,
		eventBus:             eventBus,
		userService:          userService,
		mailerService:        mailerService,
		accessControlService: accessControlService,
		organizationRepo:     organizationRepo,
		orgInvitationRepo:    orgInvitationRepo,
		orgMemberRepo:        orgMemberRepo,
		serviceUtils:         serviceUtils,
	}
}

func (s *OrganizationInvitationService) CreateOrganizationInvitation(ctx context.Context, actorUserID string, organizationID string, request types.CreateOrganizationInvitationRequest) (*types.OrganizationInvitation, error) {
	if actorUserID == "" || organizationID == "" {
		return nil, internalerrors.ErrUnauthorized
	}

	organization, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID)
	if err != nil {
		return nil, err
	}

	email := strings.ToLower(request.Email)
	if email == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	role := request.Role
	if role == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	validatedRoleAssignment, err := s.accessControlService.ValidateRoleAssignment(ctx, role, &actorUserID)
	if err != nil {
		if err.Error() == internalerrors.ErrForbidden.Error() {
			return nil, internalerrors.ErrForbidden
		}
		if err.Error() == internalerrors.ErrNotFound.Error() {
			return nil, internalerrors.ErrUnprocessableEntity
		}
		return nil, err
	}
	if !validatedRoleAssignment {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	var created *types.OrganizationInvitation
	err = s.txRunner.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		invitationRepo := s.orgInvitationRepo.WithTx(tx)
		memberRepo := s.orgMemberRepo.WithTx(tx)

		if err := s.serviceUtils.ensureOrganizationMembersLimit(ctx, memberRepo, organizationID, s.pluginConfig.MembersLimit); err != nil {
			return err
		}

		if err := s.serviceUtils.ensureOrganizationInvitationsLimit(ctx, invitationRepo, organizationID, email, s.pluginConfig.InvitationsLimit); err != nil {
			return err
		}

		if existing, err := invitationRepo.GetByOrganizationIDAndEmail(ctx, organizationID, email, types.OrganizationInvitationStatusPending); err != nil {
			return err
		} else if existing != nil {
			if err := s.expireOrganizationInvitationIfNeeded(ctx, existing); err != nil {
				return err
			}
			if existing.Status == types.OrganizationInvitationStatusPending {
				return internalerrors.ErrConflict
			}
		}

		expiresAt := time.Now().UTC().Add(s.pluginConfig.InvitationExpiresIn)
		if !expiresAt.After(time.Now().UTC()) {
			return internalerrors.ErrUnprocessableEntity
		}
		invitation := &types.OrganizationInvitation{
			ID:             util.GenerateUUID(),
			Email:          email,
			InviterID:      actorUserID,
			OrganizationID: organizationID,
			Role:           role,
			Status:         types.OrganizationInvitationStatusPending,
			ExpiresAt:      expiresAt,
		}

		createdInvitation, err := invitationRepo.Create(ctx, invitation)
		if err != nil {
			return err
		}

		created = createdInvitation
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.publishOrganizationInvitationCreatedEvent(created, organization)
	s.sendOrganizationInvitationEmailAsync(ctx, created, organization, request.RedirectURL)

	return created, nil
}

func (s *OrganizationInvitationService) publishOrganizationInvitationCreatedEvent(invitation *types.OrganizationInvitation, organization *types.Organization) {
	payload, err := json.Marshal(orgevents.OrganizationInvitationCreatedEvent{
		ID:               util.GenerateUUID(),
		InvitationID:     invitation.ID,
		OrganizationID:   invitation.OrganizationID,
		OrganizationName: organization.Name,
		InviteeEmail:     invitation.Email,
		InviterID:        invitation.InviterID,
		Role:             invitation.Role,
		ExpiresAt:        invitation.ExpiresAt,
	})
	if err != nil {
		s.logger.Error("failed to marshal organization invitation created event", "error", err)
		return
	}

	util.PublishEventAsync(s.eventBus, s.logger, models.Event{
		ID:        util.GenerateUUID(),
		Type:      orgconstants.EventOrganizationsInvitationCreated,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	})
}

func (s *OrganizationInvitationService) sendOrganizationInvitationEmailAsync(ctx context.Context, invitation *types.OrganizationInvitation, organization *types.Organization, redirectURL string) {
	if s.mailerService == nil {
		s.logger.Warn("mailer service not available, skipping organization invitation email")
		return
	}

	go func() {
		detachedCtx := context.WithoutCancel(ctx)
		taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
		defer cancel()

		if err := s.sendOrganizationInvitationEmail(taskCtx, invitation, organization, redirectURL); err != nil {
			s.logger.Error("failed to send organization invitation email", "invitation_id", invitation.ID, "error", err)
		}
	}()
}

func (s *OrganizationInvitationService) sendOrganizationInvitationEmail(ctx context.Context, invitation *types.OrganizationInvitation, organization *types.Organization, redirectURL string) error {
	acceptURL := s.buildOrganizationInvitationAcceptURL(invitation, redirectURL)
	appName := "Authula"
	if s.globalConfig.AppName != "" {
		appName = s.globalConfig.AppName
	}
	subject := fmt.Sprintf("You're invited to join %s on %s", organization.Name, appName)
	textBody := fmt.Sprintf("You have been invited to join %s on %s as %s. Open this link to accept the invitation: %s", organization.Name, appName, invitation.Role, acceptURL)
	htmlBody := fmt.Sprintf(`
<div style="font-family: Arial, Helvetica, sans-serif; line-height: 1.5; color: #1f2937;">
  <p>Hello,</p>
  <p>You have been invited to join <strong>%s</strong> on <strong>%s</strong> as <strong>%s</strong>.</p>
  <p><a href="%s" style="display:inline-block;background:#111827;color:#ffffff;text-decoration:none;padding:10px 16px;border-radius:8px;">Accept invitation</a></p>
  <p>If the button does not work, copy this link:</p>
  <p><a href="%s">%s</a></p>
</div>`,
		html.EscapeString(organization.Name),
		html.EscapeString(appName),
		html.EscapeString(invitation.Role),
		html.EscapeString(acceptURL),
		html.EscapeString(acceptURL),
		html.EscapeString(acceptURL),
	)

	return s.mailerService.SendEmail(ctx, invitation.Email, subject, textBody, htmlBody)
}

func (s *OrganizationInvitationService) buildOrganizationInvitationAcceptURL(invitation *types.OrganizationInvitation, redirectURL string) string {
	baseURL := s.globalConfig.BaseURL
	basePath := s.globalConfig.BasePath
	acceptPath := fmt.Sprintf("/organizations/%s/invitations/%s/accept", url.PathEscape(invitation.OrganizationID), url.PathEscape(invitation.ID))

	fullURL := baseURL + basePath + acceptPath
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return fullURL
	}

	if redirectURL != "" {
		query := parsedURL.Query()
		query.Set("redirect_url", redirectURL)
		parsedURL.RawQuery = query.Encode()
	}

	return parsedURL.String()
}

func (s *OrganizationInvitationService) GetAllOrganizationInvitations(ctx context.Context, actorUserID string, organizationID string) ([]types.OrganizationInvitation, error) {
	if actorUserID == "" || organizationID == "" {
		return nil, internalerrors.ErrUnauthorized
	}

	_, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID)
	if err != nil {
		return nil, err
	}

	invitations, err := s.orgInvitationRepo.GetAllByOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return invitations, nil
}

func (s *OrganizationInvitationService) GetOrganizationInvitation(ctx context.Context, actorUserID string, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	if actorUserID == "" || organizationID == "" || invitationID == "" {
		return nil, internalerrors.ErrUnauthorized
	}

	_, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID)
	if err != nil {
		return nil, err
	}

	invitation, err := s.orgInvitationRepo.GetByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if invitation == nil || invitation.OrganizationID != organizationID {
		return nil, internalerrors.ErrNotFound
	}

	return invitation, nil
}

func (s *OrganizationInvitationService) RevokeOrganizationInvitation(ctx context.Context, actorUserID string, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	if actorUserID == "" || organizationID == "" || invitationID == "" {
		return nil, internalerrors.ErrUnauthorized
	}

	_, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actorUserID, organizationID)
	if err != nil {
		return nil, err
	}

	invitation, err := s.orgInvitationRepo.GetByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if invitation == nil || invitation.OrganizationID != organizationID {
		return nil, internalerrors.ErrNotFound
	}
	if err := s.expireOrganizationInvitationIfNeeded(ctx, invitation); err != nil {
		return nil, err
	}
	if invitation.Status != types.OrganizationInvitationStatusPending {
		return nil, internalerrors.ErrConflict
	}

	invitation.Status = types.OrganizationInvitationStatusRevoked

	updated, err := s.orgInvitationRepo.Update(ctx, invitation)
	if err != nil {
		return nil, err
	}

	return updated, nil
}
func (s *OrganizationInvitationService) AcceptOrganizationInvitation(ctx context.Context, actorUserID string, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	if actorUserID == "" || organizationID == "" || invitationID == "" {
		return nil, internalerrors.ErrUnauthorized
	}

	user, err := s.serviceUtils.ensureEmailVerifiedForInvitationAcceptance(ctx, s.userService, actorUserID, s.pluginConfig.RequireEmailVerifiedOnInvitation)
	if err != nil {
		return nil, err
	}

	invitation, err := s.orgInvitationRepo.GetByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if invitation == nil || invitation.OrganizationID != organizationID {
		return nil, internalerrors.ErrNotFound
	}
	if err := s.expireOrganizationInvitationIfNeeded(ctx, invitation); err != nil {
		return nil, err
	}
	if invitation.Status != types.OrganizationInvitationStatusPending {
		return nil, internalerrors.ErrConflict
	}
	if !strings.EqualFold(invitation.Email, user.Email) {
		return nil, internalerrors.ErrForbidden
	}

	accepted, err := s.acceptOrganizationInvitations(ctx, actorUserID, []types.OrganizationInvitation{*invitation})
	if err != nil {
		return nil, err
	}
	if len(accepted) == 0 {
		return nil, internalerrors.ErrConflict
	}

	return &accepted[0], nil
}

func (s *OrganizationInvitationService) RejectOrganizationInvitation(ctx context.Context, actorUserID string, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	if actorUserID == "" || organizationID == "" || invitationID == "" {
		return nil, internalerrors.ErrUnauthorized
	}

	user, err := s.userService.GetByID(ctx, actorUserID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.Email == "" {
		return nil, internalerrors.ErrNotFound
	}

	invitation, err := s.orgInvitationRepo.GetByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if invitation == nil || invitation.OrganizationID != organizationID {
		return nil, internalerrors.ErrNotFound
	}
	if err := s.expireOrganizationInvitationIfNeeded(ctx, invitation); err != nil {
		return nil, err
	}
	if invitation.Status != types.OrganizationInvitationStatusPending {
		return nil, internalerrors.ErrConflict
	}
	if !strings.EqualFold(invitation.Email, user.Email) {
		return nil, internalerrors.ErrForbidden
	}

	invitation.Status = types.OrganizationInvitationStatusRejected

	updated, err := s.orgInvitationRepo.Update(ctx, invitation)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *OrganizationInvitationService) AcceptPendingOrganizationInvitationsForEmail(ctx context.Context, userID string, email string) ([]types.OrganizationInvitation, error) {
	email = strings.ToLower(email)
	if userID == "" || email == "" {
		return nil, internalerrors.ErrUnprocessableEntity
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return nil, internalerrors.ErrBadRequest
	}

	if s.pluginConfig.RequireEmailVerifiedOnInvitation {
		if _, err := s.serviceUtils.ensureEmailVerifiedForInvitationAcceptance(ctx, s.userService, userID, true); err != nil {
			return nil, err
		}
	}

	pendingInvitations, err := s.orgInvitationRepo.GetAllPendingByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if len(pendingInvitations) == 0 {
		return []types.OrganizationInvitation{}, nil
	}

	return s.acceptOrganizationInvitations(ctx, userID, pendingInvitations)
}

func (s *OrganizationInvitationService) acceptOrganizationInvitations(ctx context.Context, userID string, invitations []types.OrganizationInvitation) ([]types.OrganizationInvitation, error) {
	accepted := make([]types.OrganizationInvitation, 0, len(invitations))
	err := s.txRunner.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		invitationRepo := s.orgInvitationRepo.WithTx(tx)
		memberRepo := s.orgMemberRepo.WithTx(tx)

		for _, pendingInvitation := range invitations {
			invitation := pendingInvitation
			roleExists, err := s.accessControlService.RoleExists(ctx, invitation.Role)
			if err != nil {
				return err
			}
			if !roleExists {
				return internalerrors.ErrUnprocessableEntity
			}

			existingMember, err := memberRepo.GetByOrganizationIDAndUserID(ctx, invitation.OrganizationID, userID)
			if err != nil {
				return err
			}
			if existingMember == nil {
				if err := s.serviceUtils.ensureOrganizationMembersLimit(ctx, memberRepo, invitation.OrganizationID, s.pluginConfig.MembersLimit); err != nil {
					return err
				}

				member := &types.OrganizationMember{
					ID:             util.GenerateUUID(),
					OrganizationID: invitation.OrganizationID,
					UserID:         userID,
					Role:           invitation.Role,
				}

				_, err = memberRepo.Create(ctx, member)
				if err != nil {
					return err
				}
			}

			invitation.Status = types.OrganizationInvitationStatusAccepted
			updatedInvitation, err := invitationRepo.Update(ctx, &invitation)
			if err != nil {
				return err
			}

			accepted = append(accepted, *updatedInvitation)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return accepted, nil
}

func (s *OrganizationInvitationService) expireOrganizationInvitationIfNeeded(ctx context.Context, invitation *types.OrganizationInvitation) error {
	if invitation.Status != types.OrganizationInvitationStatusPending {
		return nil
	}
	if invitation.ExpiresAt.After(time.Now().UTC()) {
		return nil
	}

	invitation.Status = types.OrganizationInvitationStatusExpired

	updated, err := s.orgInvitationRepo.Update(ctx, invitation)
	if err != nil {
		return err
	}
	*invitation = *updated

	return nil
}
