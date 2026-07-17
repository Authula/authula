package usecases

import (
	"context"

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
	authorizer        rootservices.Authorizer
}

func NewUseCases(
	orgService orgservices.OrganizationService,
	invitationService orgservices.OrganizationInvitationService,
	memberService orgservices.OrganizationMemberService,
	teamService orgservices.OrganizationTeamService,
	teamMemberService orgservices.OrganizationTeamMemberService,
	authorizer rootservices.Authorizer,
) *UseCases {
	return &UseCases{
		orgService:        orgService,
		invitationService: invitationService,
		memberService:     memberService,
		teamService:       teamService,
		teamMemberService: teamMemberService,
		authorizer:        authorizer,
	}
}

// ------------- OrganizationService -------------

func (u *UseCases) CreateOrganization(ctx context.Context, actor *models.Actor, request types.CreateOrganizationRequest) (*types.Organization, error) {
	return u.orgService.CreateOrganization(ctx, actor, request)
}

func (u *UseCases) GetAllOrganizationsByOwner(ctx context.Context, actor *models.Actor) ([]types.Organization, error) {
	return u.orgService.GetAllOrganizationsByOwner(ctx, actor)
}

func (u *UseCases) GetOrganizationByID(ctx context.Context, actor *models.Actor, organizationID string) (*types.Organization, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsReadPermission); err != nil {
		return nil, err
	}
	return u.orgService.GetOrganizationByID(ctx, actor, organizationID)
}

func (u *UseCases) UpdateOrganization(ctx context.Context, actor *models.Actor, organizationID string, request types.UpdateOrganizationRequest) (*types.Organization, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsUpdatePermission); err != nil {
		return nil, err
	}
	return u.orgService.UpdateOrganization(ctx, actor, organizationID, request)
}

func (u *UseCases) DeleteOrganization(ctx context.Context, actor *models.Actor, organizationID string) error {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsDeletePermission); err != nil {
		return err
	}
	return u.orgService.DeleteOrganization(ctx, actor, organizationID)
}

func (u *UseCases) ExistsByID(ctx context.Context, organizationID string) (bool, error) {
	return u.orgService.ExistsByID(ctx, organizationID)
}

// ------------- OrganizationInvitationService -------------

func (u *UseCases) CreateOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationInvitationRequest) (*types.OrganizationInvitation, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsInvitationsCreatePermission); err != nil {
		return nil, err
	}
	return u.invitationService.CreateOrganizationInvitation(ctx, actor, organizationID, request)
}

func (u *UseCases) GetAllOrganizationInvitations(ctx context.Context, actor *models.Actor, organizationID string) ([]types.OrganizationInvitation, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsInvitationsListPermission); err != nil {
		return nil, err
	}
	return u.invitationService.GetAllOrganizationInvitations(ctx, actor, organizationID)
}

func (u *UseCases) GetOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsInvitationsReadPermission); err != nil {
		return nil, err
	}
	return u.invitationService.GetOrganizationInvitation(ctx, actor, organizationID, invitationID)
}

func (u *UseCases) RevokeOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsInvitationsRevokePermission); err != nil {
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
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsMembersAddPermission); err != nil {
		return nil, err
	}
	return u.memberService.AddMember(ctx, actor, organizationID, request)
}

func (u *UseCases) GetAllMembers(ctx context.Context, actor *models.Actor, organizationID string, page int, limit int) ([]types.OrganizationMember, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsMembersListPermission); err != nil {
		return nil, err
	}
	return u.memberService.GetAllMembers(ctx, actor, organizationID, page, limit)
}

func (u *UseCases) GetMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) (*types.OrganizationMember, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsMembersReadPermission); err != nil {
		return nil, err
	}
	return u.memberService.GetMember(ctx, actor, organizationID, memberID)
}

func (u *UseCases) UpdateMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string, request types.UpdateOrganizationMemberRequest) (*types.OrganizationMember, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsMembersUpdatePermission); err != nil {
		return nil, err
	}
	return u.memberService.UpdateMember(ctx, actor, organizationID, memberID, request)
}

func (u *UseCases) RemoveMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) error {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsMembersRemovePermission); err != nil {
		return err
	}
	return u.memberService.RemoveMember(ctx, actor, organizationID, memberID)
}

// ------------- OrganizationTeamService -------------

func (u *UseCases) CreateTeam(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsTeamsCreatePermission); err != nil {
		return nil, err
	}
	return u.teamService.CreateTeam(ctx, actor, organizationID, request)
}

func (u *UseCases) GetAllTeams(ctx context.Context, actor *models.Actor, organizationID string) ([]types.OrganizationTeam, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsTeamsListPermission); err != nil {
		return nil, err
	}
	return u.teamService.GetAllTeams(ctx, actor, organizationID)
}

func (u *UseCases) GetTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) (*types.OrganizationTeam, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsTeamsReadPermission); err != nil {
		return nil, err
	}
	return u.teamService.GetTeam(ctx, actor, organizationID, teamID)
}

func (u *UseCases) UpdateTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.UpdateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsTeamsUpdatePermission); err != nil {
		return nil, err
	}
	return u.teamService.UpdateTeam(ctx, actor, organizationID, teamID, request)
}

func (u *UseCases) DeleteTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) error {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsTeamsDeletePermission); err != nil {
		return err
	}
	return u.teamService.DeleteTeam(ctx, actor, organizationID, teamID)
}

// ------------- OrganizationTeamMemberService -------------

func (u *UseCases) AddTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.AddOrganizationTeamMemberRequest) (*types.OrganizationTeamMember, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsTeamMembersAddPermission); err != nil {
		return nil, err
	}
	return u.teamMemberService.AddTeamMember(ctx, actor, organizationID, teamID, request)
}

func (u *UseCases) GetAllTeamMembers(ctx context.Context, actor *models.Actor, organizationID string, teamID string, page int, limit int) ([]types.OrganizationTeamMember, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsTeamMembersListPermission); err != nil {
		return nil, err
	}
	return u.teamMemberService.GetAllTeamMembers(ctx, actor, organizationID, teamID, page, limit)
}

func (u *UseCases) GetTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) (*types.OrganizationTeamMember, error) {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return nil, err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsTeamMembersReadPermission); err != nil {
		return nil, err
	}
	return u.teamMemberService.GetTeamMember(ctx, actor, organizationID, teamID, memberID)
}

func (u *UseCases) RemoveTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) error {
	if err := u.authorizer.AuthorizeOrganizationAccess(ctx, actor, organizationID); err != nil {
		return err
	}
	if err := u.authorizer.AuthorizeScope(ctx, actor, orgconstants.OrganizationsTeamMembersRemovePermission); err != nil {
		return err
	}
	return u.teamMemberService.RemoveTeamMember(ctx, actor, organizationID, teamID, memberID)
}
