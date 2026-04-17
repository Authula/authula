package organizations

import (
	"context"

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

func (a *API) CreateOrganization(ctx context.Context, actorUserID string, request types.CreateOrganizationRequest) (*types.Organization, error) {
	return a.organizationService.CreateOrganization(ctx, actorUserID, request)
}

func (a *API) GetAllOrganizations(ctx context.Context, actorUserID string) ([]types.Organization, error) {
	return a.organizationService.GetAllOrganizations(ctx, actorUserID)
}

func (a *API) GetOrganizationByID(ctx context.Context, actorUserID string, organizationID string) (*types.Organization, error) {
	return a.organizationService.GetOrganizationByID(ctx, actorUserID, organizationID)
}

func (a *API) UpdateOrganization(ctx context.Context, actorUserID string, organizationID string, request types.UpdateOrganizationRequest) (*types.Organization, error) {
	return a.organizationService.UpdateOrganization(ctx, actorUserID, organizationID, request)
}

func (a *API) DeleteOrganization(ctx context.Context, actorUserID string, organizationID string) error {
	return a.organizationService.DeleteOrganization(ctx, actorUserID, organizationID)
}

// Invitations

func (a *API) CreateInvitation(ctx context.Context, actorUserID string, organizationID string, request types.CreateOrganizationInvitationRequest) (*types.OrganizationInvitation, error) {
	return a.invitationService.CreateOrganizationInvitation(ctx, actorUserID, organizationID, request)
}

func (a *API) GetInvitation(ctx context.Context, actorUserID string, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	return a.invitationService.GetOrganizationInvitation(ctx, actorUserID, organizationID, invitationID)
}

func (a *API) GetAllInvitations(ctx context.Context, actorUserID string, organizationID string) ([]types.OrganizationInvitation, error) {
	return a.invitationService.GetAllOrganizationInvitations(ctx, actorUserID, organizationID)
}

func (a *API) RevokeInvitation(ctx context.Context, actorUserID string, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	return a.invitationService.RevokeOrganizationInvitation(ctx, actorUserID, organizationID, invitationID)
}

func (a *API) AcceptInvitation(ctx context.Context, actorUserID string, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	return a.invitationService.AcceptOrganizationInvitation(ctx, actorUserID, organizationID, invitationID)
}

func (a *API) RejectInvitation(ctx context.Context, actorUserID string, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	return a.invitationService.RejectOrganizationInvitation(ctx, actorUserID, organizationID, invitationID)
}

// Members

func (a *API) AddMember(ctx context.Context, actorUserID string, organizationID string, request types.AddOrganizationMemberRequest) (*types.OrganizationMember, error) {
	return a.memberService.AddMember(ctx, actorUserID, organizationID, request)
}

func (a *API) GetAllMembers(ctx context.Context, actorUserID string, organizationID string, page int, limit int) ([]types.OrganizationMember, error) {
	return a.memberService.GetAllMembers(ctx, actorUserID, organizationID, page, limit)
}

func (a *API) GetMember(ctx context.Context, actorUserID string, organizationID string, memberID string) (*types.OrganizationMember, error) {
	return a.memberService.GetMember(ctx, actorUserID, organizationID, memberID)
}

func (a *API) UpdateMember(ctx context.Context, actorUserID string, organizationID string, memberID string, request types.UpdateOrganizationMemberRequest) (*types.OrganizationMember, error) {
	return a.memberService.UpdateMember(ctx, actorUserID, organizationID, memberID, request)
}

func (a *API) RemoveMember(ctx context.Context, actorUserID string, organizationID string, memberID string) error {
	return a.memberService.RemoveMember(ctx, actorUserID, organizationID, memberID)
}

// Teams

func (a *API) CreateTeam(ctx context.Context, actorUserID string, organizationID string, request types.CreateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	return a.teamService.CreateTeam(ctx, actorUserID, organizationID, request)
}

func (a *API) GetAllTeams(ctx context.Context, actorUserID string, organizationID string) ([]types.OrganizationTeam, error) {
	return a.teamService.GetAllTeams(ctx, actorUserID, organizationID)
}

func (a *API) GetTeam(ctx context.Context, actorUserID string, organizationID string, teamID string) (*types.OrganizationTeam, error) {
	return a.teamService.GetTeam(ctx, actorUserID, organizationID, teamID)
}

func (a *API) UpdateTeam(ctx context.Context, actorUserID string, organizationID string, teamID string, request types.UpdateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	return a.teamService.UpdateTeam(ctx, actorUserID, organizationID, teamID, request)
}

func (a *API) DeleteTeam(ctx context.Context, actorUserID string, organizationID string, teamID string) error {
	return a.teamService.DeleteTeam(ctx, actorUserID, organizationID, teamID)
}

// Team Members

func (a *API) AddTeamMember(ctx context.Context, actorUserID string, organizationID string, teamID string, request types.AddOrganizationTeamMemberRequest) (*types.OrganizationTeamMember, error) {
	return a.teamMemberService.AddTeamMember(ctx, actorUserID, organizationID, teamID, request)
}

func (a *API) GetAllTeamMembers(ctx context.Context, actorUserID string, organizationID string, teamID string, page int, limit int) ([]types.OrganizationTeamMember, error) {
	return a.teamMemberService.GetAllTeamMembers(ctx, actorUserID, organizationID, teamID, page, limit)
}

func (a *API) GetTeamMember(ctx context.Context, actorUserID string, organizationID string, teamID string, memberID string) (*types.OrganizationTeamMember, error) {
	return a.teamMemberService.GetTeamMember(ctx, actorUserID, organizationID, teamID, memberID)
}

func (a *API) RemoveTeamMember(ctx context.Context, actorUserID string, organizationID string, teamID string, memberID string) error {
	return a.teamMemberService.RemoveTeamMember(ctx, actorUserID, organizationID, teamID, memberID)
}
