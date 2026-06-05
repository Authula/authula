package organizations

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/services"
	"github.com/Authula/authula/plugins/organizations/types"
)

type API struct {
	organizationService services.OrganizationService
	invitationService   services.OrganizationInvitationService
	memberService       services.OrganizationMemberService
	teamService         services.OrganizationTeamService
	teamMemberService   services.OrganizationTeamMemberService
}

func BuildAPI(plugin *OrganizationsPlugin) *API {
	return &API{
		organizationService: plugin.organizationService,
		invitationService:   plugin.invitationService,
		memberService:       plugin.memberService,
		teamService:         plugin.teamService,
		teamMemberService:   plugin.teamMemberService,
	}
}

// Organizations

func (a *API) CreateOrganization(ctx context.Context, actor *models.Actor, request types.CreateOrganizationRequest) (*types.Organization, error) {
	return a.organizationService.CreateOrganization(ctx, actor, request)
}

func (a *API) GetAllOrganizations(ctx context.Context, actor *models.Actor) ([]types.Organization, error) {
	return a.organizationService.GetAllOrganizations(ctx, actor)
}

func (a *API) GetOrganizationByID(ctx context.Context, actor *models.Actor, organizationID string) (*types.Organization, error) {
	return a.organizationService.GetOrganizationByID(ctx, actor, organizationID)
}

func (a *API) UpdateOrganization(ctx context.Context, actor *models.Actor, organizationID string, request types.UpdateOrganizationRequest) (*types.Organization, error) {
	return a.organizationService.UpdateOrganization(ctx, actor, organizationID, request)
}

func (a *API) DeleteOrganization(ctx context.Context, actor *models.Actor, organizationID string) error {
	return a.organizationService.DeleteOrganization(ctx, actor, organizationID)
}

// Invitations

func (a *API) CreateInvitation(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationInvitationRequest) (*types.OrganizationInvitation, error) {
	return a.invitationService.CreateOrganizationInvitation(ctx, actor, organizationID, request)
}

func (a *API) GetInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	return a.invitationService.GetOrganizationInvitation(ctx, actor, organizationID, invitationID)
}

func (a *API) GetAllInvitations(ctx context.Context, actor *models.Actor, organizationID string) ([]types.OrganizationInvitation, error) {
	return a.invitationService.GetAllOrganizationInvitations(ctx, actor, organizationID)
}

func (a *API) RevokeInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	return a.invitationService.RevokeOrganizationInvitation(ctx, actor, organizationID, invitationID)
}

func (a *API) AcceptInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	return a.invitationService.AcceptOrganizationInvitation(ctx, actor, organizationID, invitationID)
}

func (a *API) RejectInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	return a.invitationService.RejectOrganizationInvitation(ctx, actor, organizationID, invitationID)
}

// Members

func (a *API) AddMember(ctx context.Context, actor *models.Actor, organizationID string, request types.AddOrganizationMemberRequest) (*types.OrganizationMember, error) {
	return a.memberService.AddMember(ctx, actor, organizationID, request)
}

func (a *API) GetAllMembers(ctx context.Context, actor *models.Actor, organizationID string, page int, limit int) ([]types.OrganizationMember, error) {
	return a.memberService.GetAllMembers(ctx, actor, organizationID, page, limit)
}

func (a *API) GetMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) (*types.OrganizationMember, error) {
	return a.memberService.GetMember(ctx, actor, organizationID, memberID)
}

func (a *API) UpdateMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string, request types.UpdateOrganizationMemberRequest) (*types.OrganizationMember, error) {
	return a.memberService.UpdateMember(ctx, actor, organizationID, memberID, request)
}

func (a *API) RemoveMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) error {
	return a.memberService.RemoveMember(ctx, actor, organizationID, memberID)
}

// Teams

func (a *API) CreateTeam(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	return a.teamService.CreateTeam(ctx, actor, organizationID, request)
}

func (a *API) GetAllTeams(ctx context.Context, actor *models.Actor, organizationID string) ([]types.OrganizationTeam, error) {
	return a.teamService.GetAllTeams(ctx, actor, organizationID)
}

func (a *API) GetTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) (*types.OrganizationTeam, error) {
	return a.teamService.GetTeam(ctx, actor, organizationID, teamID)
}

func (a *API) UpdateTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.UpdateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	return a.teamService.UpdateTeam(ctx, actor, organizationID, teamID, request)
}

func (a *API) DeleteTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) error {
	return a.teamService.DeleteTeam(ctx, actor, organizationID, teamID)
}

// Team Members

func (a *API) AddTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.AddOrganizationTeamMemberRequest) (*types.OrganizationTeamMember, error) {
	return a.teamMemberService.AddTeamMember(ctx, actor, organizationID, teamID, request)
}

func (a *API) GetAllTeamMembers(ctx context.Context, actor *models.Actor, organizationID string, teamID string, page int, limit int) ([]types.OrganizationTeamMember, error) {
	return a.teamMemberService.GetAllTeamMembers(ctx, actor, organizationID, teamID, page, limit)
}

func (a *API) GetTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) (*types.OrganizationTeamMember, error) {
	return a.teamMemberService.GetTeamMember(ctx, actor, organizationID, teamID, memberID)
}

func (a *API) RemoveTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) error {
	return a.teamMemberService.RemoveTeamMember(ctx, actor, organizationID, teamID, memberID)
}
