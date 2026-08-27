package usecases

import (
	"context"
	"errors"
	"strings"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/pagination"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	orgservices "github.com/Authula/authula/plugins/organizations/services"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
)

type UseCases struct {
	orgService        orgservices.OrganizationService
	invitationService orgservices.OrganizationInvitationService
	memberService     orgservices.OrganizationMemberService
	teamService       orgservices.OrganizationTeamService
	teamMemberService orgservices.OrganizationTeamMemberService
	userService       rootservices.UserService
	globalConfig      *models.Config
	authorizer        rootservices.Authorizer
	serviceUtils      *orgservices.ServiceUtils
	accessControl     rootservices.AccessControlService
}

func NewUseCases(
	orgService orgservices.OrganizationService,
	invitationService orgservices.OrganizationInvitationService,
	memberService orgservices.OrganizationMemberService,
	teamService orgservices.OrganizationTeamService,
	teamMemberService orgservices.OrganizationTeamMemberService,
	userService rootservices.UserService,
	globalConfig *models.Config,
	authorizer rootservices.Authorizer,
	serviceUtils *orgservices.ServiceUtils,
	accessControl rootservices.AccessControlService,
) *UseCases {
	return &UseCases{
		orgService:        orgService,
		invitationService: invitationService,
		memberService:     memberService,
		teamService:       teamService,
		teamMemberService: teamMemberService,
		userService:       userService,
		globalConfig:      globalConfig,
		authorizer:        authorizer,
		serviceUtils:      serviceUtils,
		accessControl:     accessControl,
	}
}

func (u *UseCases) authorizeOrgAccess(ctx context.Context, actor *models.Actor, orgID, requiredScope string) error {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, orgID); err != nil {
		return err
	}
	_, member, err := u.serviceUtils.AuthorizeOrganizationAccess(ctx, actor, orgID)
	if err != nil {
		return err
	}
	if member == nil {
		if actor.Type != models.ActorMachine {
			return coreerrors.ErrForbidden
		}
		return u.authorizer.AuthorizeScope(ctx, actor, requiredScope)
	}

	perms, err := u.accessControl.GetRolePermissionsByName(ctx, member.Role)
	if err != nil {
		if errors.Is(err, coreerrors.ErrNotFound) {
			return coreerrors.ErrForbidden
		}
		return err
	}
	if !hasPermissionKey(perms, requiredScope) {
		return coreerrors.ErrInsufficientPermissions
	}
	return nil
}

func hasPermissionKey(permissions []string, required string) bool {
	for _, permission := range permissions {
		if permission == "*" || permission == required {
			return true
		}
		if strings.HasSuffix(permission, "*") && strings.HasPrefix(required, strings.TrimSuffix(permission, "*")) {
			return true
		}
	}
	return false
}

// ------------- OrganizationService -------------

func (u *UseCases) CreateOrganization(ctx context.Context, actor *models.Actor, request types.CreateOrganizationRequest) (*types.Organization, error) {
	return u.orgService.CreateOrganization(ctx, actor, request)
}

func (u *UseCases) ListAllOrganizations(ctx context.Context, actor *models.Actor, params pagination.Params) (*types.ListOrganizationsResponse, error) {
	return u.orgService.ListAllOrganizations(ctx, actor, params)
}

func (u *UseCases) GetOrganizationByID(ctx context.Context, actor *models.Actor, organizationID string) (*types.Organization, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsReadPermission); err != nil {
		return nil, err
	}
	return u.orgService.GetOrganizationByID(ctx, actor, organizationID)
}

func (u *UseCases) UpdateOrganization(ctx context.Context, actor *models.Actor, organizationID string, request types.UpdateOrganizationRequest) (*types.Organization, error) {
	return u.orgService.UpdateOrganization(ctx, actor, organizationID, request)
}

func (u *UseCases) DeleteOrganization(ctx context.Context, actor *models.Actor, organizationID string) error {
	return u.orgService.DeleteOrganization(ctx, actor, organizationID)
}

func (u *UseCases) ExistsByID(ctx context.Context, organizationID string) (bool, error) {
	return u.orgService.ExistsByID(ctx, organizationID)
}

// ------------- OrganizationInvitationService -------------

func (u *UseCases) CreateOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationInvitationRequest, redirectURL string) (*types.OrganizationInvitation, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsInvitationsCreatePermission); err != nil {
		return nil, err
	}
	return u.invitationService.CreateOrganizationInvitation(ctx, actor, organizationID, request, redirectURL)
}

func (u *UseCases) ListAllOrganizationInvitations(ctx context.Context, actor *models.Actor, organizationID string, params pagination.Params) (*types.ListOrganizationInvitationsResponse, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsInvitationsListPermission); err != nil {
		return nil, err
	}

	resp, err := u.invitationService.ListAllOrganizationInvitationsByOrgIDWithOrg(ctx, organizationID, params)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (u *UseCases) GetOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.GetOrganizationInvitationResponse, error) {
	resp, err := u.invitationService.GetOrganizationInvitationByIDWithOrg(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if resp.Invitation.OrganizationID != organizationID {
		return nil, coreerrors.ErrNotFound
	}

	user, err := u.userService.GetByID(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	if user != nil && strings.EqualFold(resp.Invitation.Email, user.Email) {
		return resp, nil
	}

	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsInvitationsReadPermission); err != nil {
		if errors.Is(err, coreerrors.ErrForbidden) || errors.Is(err, coreerrors.ErrInsufficientPermissions) {
			return nil, coreerrors.ErrNotFound
		}
		return nil, err
	}
	return resp, nil
}

func (u *UseCases) RevokeOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsInvitationsRevokePermission); err != nil {
		return nil, err
	}
	return u.invitationService.RevokeOrganizationInvitation(ctx, actor, organizationID, invitationID)
}

func (u *UseCases) AcceptOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	return u.invitationService.AcceptOrganizationInvitation(ctx, actor, organizationID, invitationID)
}

func (u *UseCases) RejectOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	return u.invitationService.RejectOrganizationInvitation(ctx, actor, organizationID, invitationID)
}

// ------------- OrganizationMemberService -------------

func (u *UseCases) AddMember(ctx context.Context, actor *models.Actor, organizationID string, request types.AddOrganizationMemberRequest) (*types.OrganizationMember, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsMembersAddPermission); err != nil {
		return nil, err
	}
	return u.memberService.AddMember(ctx, actor, organizationID, request)
}

func (u *UseCases) ListAllMembers(ctx context.Context, actor *models.Actor, organizationID string, params pagination.Params) (*types.ListOrganizationMembersResponse, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsMembersListPermission); err != nil {
		return nil, err
	}
	return u.memberService.ListAllMembers(ctx, actor, organizationID, params)
}

func (u *UseCases) GetMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) (*types.OrganizationMemberResponse, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsMembersReadPermission); err != nil {
		return nil, err
	}
	return u.memberService.GetMember(ctx, actor, organizationID, memberID)
}

func (u *UseCases) GetMemberByUserID(ctx context.Context, actor *models.Actor, organizationID string, userID string) (*types.OrganizationMemberResponse, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsMembersReadPermission); err != nil {
		return nil, err
	}
	return u.memberService.GetMemberByUserID(ctx, actor, organizationID, userID)
}

func (u *UseCases) UpdateMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string, request types.UpdateOrganizationMemberRequest) (*types.OrganizationMember, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsMembersUpdatePermission); err != nil {
		return nil, err
	}
	return u.memberService.UpdateMember(ctx, actor, organizationID, memberID, request)
}

func (u *UseCases) RemoveMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) error {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsMembersRemovePermission); err != nil {
		return err
	}
	return u.memberService.RemoveMember(ctx, actor, organizationID, memberID)
}

// ------------- OrganizationTeamService -------------

func (u *UseCases) CreateTeam(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsTeamsCreatePermission); err != nil {
		return nil, err
	}
	return u.teamService.CreateTeam(ctx, actor, organizationID, request)
}

func (u *UseCases) ListAllTeams(ctx context.Context, actor *models.Actor, organizationID string, params pagination.Params) (*types.ListOrganizationTeamsResponse, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsTeamsListPermission); err != nil {
		return nil, err
	}
	return u.teamService.ListAllTeams(ctx, actor, organizationID, params)
}

func (u *UseCases) GetTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) (*types.OrganizationTeam, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsTeamsReadPermission); err != nil {
		return nil, err
	}
	return u.teamService.GetTeam(ctx, actor, organizationID, teamID)
}

func (u *UseCases) UpdateTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.UpdateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsTeamsUpdatePermission); err != nil {
		return nil, err
	}
	return u.teamService.UpdateTeam(ctx, actor, organizationID, teamID, request)
}

func (u *UseCases) DeleteTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) error {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsTeamsDeletePermission); err != nil {
		return err
	}
	return u.teamService.DeleteTeam(ctx, actor, organizationID, teamID)
}

// ------------- OrganizationTeamMemberService -------------

func (u *UseCases) AddTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.AddOrganizationTeamMemberRequest) (*types.OrganizationTeamMember, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsTeamMembersAddPermission); err != nil {
		return nil, err
	}
	return u.teamMemberService.AddTeamMember(ctx, actor, organizationID, teamID, request)
}

func (u *UseCases) ListAllTeamMembers(ctx context.Context, actor *models.Actor, organizationID string, teamID string, params pagination.Params) (*types.ListOrganizationTeamMembersResponse, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsTeamMembersListPermission); err != nil {
		return nil, err
	}
	return u.teamMemberService.ListAllTeamMembers(ctx, actor, organizationID, teamID, params)
}

func (u *UseCases) GetTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) (*types.OrganizationTeamMemberResponse, error) {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsTeamMembersReadPermission); err != nil {
		return nil, err
	}
	return u.teamMemberService.GetTeamMember(ctx, actor, organizationID, teamID, memberID)
}

func (u *UseCases) RemoveTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) error {
	if err := u.authorizeOrgAccess(ctx, actor, organizationID, orgconstants.OrganizationsTeamMembersRemovePermission); err != nil {
		return err
	}
	return u.teamMemberService.RemoveTeamMember(ctx, actor, organizationID, teamID, memberID)
}
