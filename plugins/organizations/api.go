package organizations

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/types"
	"github.com/Authula/authula/plugins/organizations/usecases"
)

type API struct {
	useCases *usecases.UseCases
}

func BuildAPI(plugin *OrganizationsPlugin) *API {
	return &API{
		useCases: plugin.useCases,
	}
}

// Organizations

func (a *API) ExistsByID(ctx context.Context, organizationID string) (bool, error) {
	return a.useCases.ExistsByID(ctx, organizationID)
}

func (a *API) CreateOrganization(ctx context.Context, actor *models.Actor, request types.CreateOrganizationRequest) (*types.Organization, error) {
	return a.useCases.CreateOrganization(ctx, actor, request)
}

func (a *API) GetAllOrganizationsByOwner(ctx context.Context, actor *models.Actor) ([]types.Organization, error) {
	return a.useCases.GetAllOrganizationsByOwner(ctx, actor)
}

func (a *API) GetOrganizationByID(ctx context.Context, actor *models.Actor, organizationID string) (*types.Organization, error) {
	return a.useCases.GetOrganizationByID(ctx, actor, organizationID)
}

func (a *API) UpdateOrganization(ctx context.Context, actor *models.Actor, organizationID string, request types.UpdateOrganizationRequest) (*types.Organization, error) {
	return a.useCases.UpdateOrganization(ctx, actor, organizationID, request)
}

func (a *API) DeleteOrganization(ctx context.Context, actor *models.Actor, organizationID string) error {
	return a.useCases.DeleteOrganization(ctx, actor, organizationID)
}

// Invitations

func (a *API) CreateInvitation(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationInvitationRequest, redirectURL string) (*types.OrganizationInvitation, error) {
	return a.useCases.CreateOrganizationInvitation(ctx, actor, organizationID, request, redirectURL)
}

func (a *API) GetAllInvitations(ctx context.Context, actor *models.Actor, organizationID string) ([]types.GetOrganizationInvitationResponse, error) {
	return a.useCases.GetAllOrganizationInvitations(ctx, actor, organizationID)
}

func (a *API) GetInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.GetOrganizationInvitationResponse, error) {
	return a.useCases.GetOrganizationInvitation(ctx, actor, organizationID, invitationID)
}

func (a *API) RevokeInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	return a.useCases.RevokeOrganizationInvitation(ctx, actor, organizationID, invitationID)
}

func (a *API) AcceptInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	return a.useCases.AcceptOrganizationInvitation(ctx, actor, organizationID, invitationID)
}

func (a *API) RejectInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	return a.useCases.RejectOrganizationInvitation(ctx, actor, organizationID, invitationID)
}

// Members

func (a *API) AddMember(ctx context.Context, actor *models.Actor, organizationID string, request types.AddOrganizationMemberRequest) (*types.OrganizationMember, error) {
	return a.useCases.AddMember(ctx, actor, organizationID, request)
}

func (a *API) GetAllMembers(ctx context.Context, actor *models.Actor, organizationID string, page int, limit int) ([]types.OrganizationMemberResponse, error) {
	return a.useCases.GetAllMembers(ctx, actor, organizationID, page, limit)
}

func (a *API) GetMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) (*types.OrganizationMemberResponse, error) {
	return a.useCases.GetMember(ctx, actor, organizationID, memberID)
}

func (a *API) GetMemberByUserID(ctx context.Context, actor *models.Actor, organizationID string, userID string) (*types.OrganizationMemberResponse, error) {
	return a.useCases.GetMemberByUserID(ctx, actor, organizationID, userID)
}

func (a *API) UpdateMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string, request types.UpdateOrganizationMemberRequest) (*types.OrganizationMember, error) {
	return a.useCases.UpdateMember(ctx, actor, organizationID, memberID, request)
}

func (a *API) RemoveMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) error {
	return a.useCases.RemoveMember(ctx, actor, organizationID, memberID)
}

// Teams

func (a *API) CreateTeam(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	return a.useCases.CreateTeam(ctx, actor, organizationID, request)
}

func (a *API) GetAllTeams(ctx context.Context, actor *models.Actor, organizationID string) ([]types.OrganizationTeam, error) {
	return a.useCases.GetAllTeams(ctx, actor, organizationID)
}

func (a *API) GetTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) (*types.OrganizationTeam, error) {
	return a.useCases.GetTeam(ctx, actor, organizationID, teamID)
}

func (a *API) UpdateTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.UpdateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	return a.useCases.UpdateTeam(ctx, actor, organizationID, teamID, request)
}

func (a *API) DeleteTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) error {
	return a.useCases.DeleteTeam(ctx, actor, organizationID, teamID)
}

// Team Members

func (a *API) AddTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.AddOrganizationTeamMemberRequest) (*types.OrganizationTeamMember, error) {
	return a.useCases.AddTeamMember(ctx, actor, organizationID, teamID, request)
}

func (a *API) GetAllTeamMembers(ctx context.Context, actor *models.Actor, organizationID string, teamID string, page int, limit int) ([]types.OrganizationTeamMemberResponse, error) {
	return a.useCases.GetAllTeamMembers(ctx, actor, organizationID, teamID, page, limit)
}

func (a *API) GetTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) (*types.OrganizationTeamMemberResponse, error) {
	return a.useCases.GetTeamMember(ctx, actor, organizationID, teamID, memberID)
}

func (a *API) RemoveTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) error {
	return a.useCases.RemoveTeamMember(ctx, actor, organizationID, teamID, memberID)
}
