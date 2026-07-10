package services

import (
	"context"
	"database/sql"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/uptrace/bun"

	emailconstants "github.com/Authula/authula/core/email/constants"
	emailtmpl "github.com/Authula/authula/core/email/template"
	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type organizationInvitationTxRunner interface {
	RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(context.Context, bun.Tx) error) error
}

type organizationInvitationService struct {
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
	emailTemplateManager *emailtmpl.Manager
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
	emailTemplateManager *emailtmpl.Manager,
) *organizationInvitationService {
	return &organizationInvitationService{
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
		emailTemplateManager: emailTemplateManager,
	}
}

func (s *organizationInvitationService) CreateOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationInvitationRequest) (*types.OrganizationInvitation, error) {
	reqCtx, _ := models.GetRequestContext(ctx)

	if actor == nil || actor.ID == "" || organizationID == "" {
		return nil, coreerrors.ErrUnauthorized
	}
	actorID := actor.ID

	organization, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actor, organizationID)
	if err != nil {
		return nil, err
	}

	role := request.Role
	if role == "" {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	validatedRoleAssignment, err := s.accessControlService.ValidateRoleAssignment(ctx, role, &actorID)
	if err != nil {
		if err.Error() == coreerrors.ErrForbidden.Error() {
			return nil, coreerrors.ErrForbidden
		}
		if err.Error() == coreerrors.ErrNotFound.Error() {
			return nil, coreerrors.ErrUnprocessableEntity
		}
		return nil, err
	}
	if !validatedRoleAssignment {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	var created *types.OrganizationInvitation
	err = s.txRunner.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		invitationRepo := s.orgInvitationRepo.WithTx(tx)
		memberRepo := s.orgMemberRepo.WithTx(tx)

		if err := ensureOrganizationMembersLimit(ctx, memberRepo, organizationID, s.pluginConfig.MembersLimit); err != nil {
			return err
		}

		if err := ensureOrganizationInvitationsLimit(ctx, invitationRepo, organizationID, request.Email, s.pluginConfig.InvitationsLimit); err != nil {
			return err
		}

		if existing, err := invitationRepo.GetByOrganizationIDAndEmail(ctx, organizationID, request.Email, types.OrganizationInvitationStatusPending); err != nil {
			return err
		} else if existing != nil {
			if err := s.expireOrganizationInvitationIfNeeded(ctx, existing); err != nil {
				return err
			}
			if existing.Status == types.OrganizationInvitationStatusPending {
				return coreerrors.ErrConflict
			}
		}

		expiresAt := time.Now().UTC().Add(s.pluginConfig.InvitationExpiresIn)
		if !expiresAt.After(time.Now().UTC()) {
			return coreerrors.ErrUnprocessableEntity
		}
		invitation := &types.OrganizationInvitation{
			ID:             util.GenerateUUID(),
			Email:          request.Email,
			InviterID:      actorID,
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

	acceptURL := s.buildOrganizationInvitationAcceptURL(created, request.RedirectURL)
	callbackHandled := false

	if s.pluginConfig.SendOrganizationInvitationEmail != nil {
		inviter, err := s.userService.GetByID(ctx, actorID)
		if err != nil {
			return nil, err
		}

		err = s.pluginConfig.SendOrganizationInvitationEmail(types.SendOrganizationInvitationEmailParams{
			Organization: organization,
			Invitation:   created,
			Inviter:      inviter,
			AcceptURL:    acceptURL,
		}, reqCtx)

		if err != nil {
			s.logger.Error("failed to send organization invitation email via plugin callback", "error", err.Error())
		} else {
			callbackHandled = true
		}
	}

	if !callbackHandled && s.mailerService != nil {
		go func() {
			detachedCtx := context.WithoutCancel(ctx)
			taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
			defer cancel()

			if err := s.sendOrganizationInvitationEmail(taskCtx, created, organization, acceptURL); err != nil {
				s.logger.Error("failed to send organization invitation email via built-in email service", "invitation_id", created.ID, "error", err)
			}
		}()
	}

	return created, nil
}

func (s *organizationInvitationService) sendOrganizationInvitationEmail(ctx context.Context, invitation *types.OrganizationInvitation, organization *types.Organization, acceptURL string) error {
	subject, textBody, htmlBody, err := s.emailTemplateManager.Render(emailconstants.OrganizationInvitationEmailTemplateName, types.OrganizationInvitationContext{
		CommonContext:    emailtmpl.NewCommonContext(s.globalConfig.AppName, s.globalConfig.BaseURL),
		InvitationEmail:  invitation.Email,
		OrganizationName: organization.Name,
		Role:             invitation.Role,
		AcceptLink:       acceptURL,
		Expiry:           s.pluginConfig.InvitationExpiresIn,
	})
	if err != nil {
		s.logger.Error("failed to render organization invitation template", "err", err.Error())
		return err
	}
	return s.mailerService.SendEmail(ctx, invitation.Email, subject, textBody, htmlBody)
}

func (s *organizationInvitationService) buildOrganizationInvitationAcceptURL(invitation *types.OrganizationInvitation, redirectURL string) string {
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

func (s *organizationInvitationService) publishOrganizationInvitationCreatedEvent(invitation *types.OrganizationInvitation, organization *types.Organization) {
	util.PublishEventAsync(s.eventBus, s.logger, models.Event{
		ID:        util.GenerateUUID(),
		Type:      orgconstants.EventOrganizationsInvitationCreated,
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"invitation_id":     invitation.ID,
			"organization_id":   invitation.OrganizationID,
			"organization_name": organization.Name,
			"invitee_email":     invitation.Email,
			"inviter_id":        invitation.InviterID,
			"role":              invitation.Role,
			"expires_at":        invitation.ExpiresAt,
		},
	})
}

func (s *organizationInvitationService) GetAllOrganizationInvitations(ctx context.Context, actor *models.Actor, organizationID string) ([]types.OrganizationInvitation, error) {
	if actor == nil || actor.ID == "" || organizationID == "" {
		return nil, coreerrors.ErrUnauthorized
	}

	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}

	invitations, err := s.orgInvitationRepo.GetAllByOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	return invitations, nil
}

func (s *organizationInvitationService) GetOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	if actor == nil || actor.ID == "" || organizationID == "" || invitationID == "" {
		return nil, coreerrors.ErrUnauthorized
	}

	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}

	invitation, err := s.orgInvitationRepo.GetByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if invitation == nil || invitation.OrganizationID != organizationID {
		return nil, coreerrors.ErrNotFound
	}

	return invitation, nil
}

func (s *organizationInvitationService) RevokeOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	if actor == nil || actor.ID == "" || organizationID == "" || invitationID == "" {
		return nil, coreerrors.ErrUnauthorized
	}

	if _, _, err := s.serviceUtils.authorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}

	invitation, err := s.orgInvitationRepo.GetByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if invitation == nil || invitation.OrganizationID != organizationID {
		return nil, coreerrors.ErrNotFound
	}
	if err := s.expireOrganizationInvitationIfNeeded(ctx, invitation); err != nil {
		return nil, err
	}
	if invitation.Status != types.OrganizationInvitationStatusPending {
		return nil, coreerrors.ErrConflict
	}

	invitation.Status = types.OrganizationInvitationStatusRevoked

	updated, err := s.orgInvitationRepo.Update(ctx, invitation)
	if err != nil {
		return nil, err
	}

	return updated, nil
}
func (s *organizationInvitationService) AcceptOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	if actor == nil || actor.ID == "" || organizationID == "" || invitationID == "" {
		return nil, coreerrors.ErrUnauthorized
	}

	actorID := actor.ID
	user, err := ensureEmailVerifiedForInvitationAcceptance(ctx, s.userService, actorID, s.pluginConfig.RequireEmailVerifiedOnInvitation)
	if err != nil {
		return nil, err
	}

	invitation, err := s.orgInvitationRepo.GetByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if invitation == nil || invitation.OrganizationID != organizationID {
		return nil, coreerrors.ErrNotFound
	}
	if err := s.expireOrganizationInvitationIfNeeded(ctx, invitation); err != nil {
		return nil, err
	}
	if invitation.Status != types.OrganizationInvitationStatusPending {
		return nil, coreerrors.ErrConflict
	}
	if !strings.EqualFold(invitation.Email, user.Email) {
		return nil, coreerrors.ErrForbidden
	}

	accepted, err := s.acceptOrganizationInvitations(ctx, actorID, []types.OrganizationInvitation{*invitation})
	if err != nil {
		return nil, err
	}
	if len(accepted) == 0 {
		return nil, coreerrors.ErrConflict
	}

	return &accepted[0], nil
}

func (s *organizationInvitationService) RejectOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	if actor == nil || actor.ID == "" || organizationID == "" || invitationID == "" {
		return nil, coreerrors.ErrUnauthorized
	}

	actorID := actor.ID
	user, err := s.userService.GetByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.Email == "" {
		return nil, coreerrors.ErrNotFound
	}

	invitation, err := s.orgInvitationRepo.GetByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if invitation == nil || invitation.OrganizationID != organizationID {
		return nil, coreerrors.ErrNotFound
	}
	if err := s.expireOrganizationInvitationIfNeeded(ctx, invitation); err != nil {
		return nil, err
	}
	if invitation.Status != types.OrganizationInvitationStatusPending {
		return nil, coreerrors.ErrConflict
	}
	if !strings.EqualFold(invitation.Email, user.Email) {
		return nil, coreerrors.ErrForbidden
	}

	invitation.Status = types.OrganizationInvitationStatusRejected

	updated, err := s.orgInvitationRepo.Update(ctx, invitation)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *organizationInvitationService) AcceptPendingOrganizationInvitationsForEmail(ctx context.Context, userID string, email string) ([]types.OrganizationInvitation, error) {
	email = strings.ToLower(email)
	if userID == "" || email == "" {
		return nil, coreerrors.ErrUnprocessableEntity
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return nil, coreerrors.ErrBadRequest
	}

	if s.pluginConfig.RequireEmailVerifiedOnInvitation {
		if _, err := ensureEmailVerifiedForInvitationAcceptance(ctx, s.userService, userID, true); err != nil {
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

func (s *organizationInvitationService) acceptOrganizationInvitations(ctx context.Context, userID string, invitations []types.OrganizationInvitation) ([]types.OrganizationInvitation, error) {
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
				return coreerrors.ErrUnprocessableEntity
			}

			existingMember, err := memberRepo.GetByOrganizationIDAndUserID(ctx, invitation.OrganizationID, userID)
			if err != nil {
				return err
			}
			if existingMember == nil {
				if err := ensureOrganizationMembersLimit(ctx, memberRepo, invitation.OrganizationID, s.pluginConfig.MembersLimit); err != nil {
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

func (s *organizationInvitationService) expireOrganizationInvitationIfNeeded(ctx context.Context, invitation *types.OrganizationInvitation) error {
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

func ensureOrganizationInvitationsLimit(ctx context.Context, invitationRepo repositories.OrganizationInvitationRepository, organizationID string, email string, invitationsLimit *int) error {
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

func ensureEmailVerifiedForInvitationAcceptance(ctx context.Context, userService rootservices.UserService, userID string, requireEmailVerified bool) (*models.User, error) {
	if userID == "" {
		return nil, coreerrors.ErrNotFound
	}

	user, err := userService.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.Email == "" {
		return nil, coreerrors.ErrNotFound
	}
	if requireEmailVerified && !user.EmailVerified {
		return nil, coreerrors.ErrForbidden
	}

	return user, nil
}
