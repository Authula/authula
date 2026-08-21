package services

import (
	"context"

	"github.com/Authula/authula/core/pagination"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/types"
)

type OrganizationService interface {
	CreateOrganization(ctx context.Context, actor *models.Actor, request types.CreateOrganizationRequest) (*types.Organization, error)
	GetAllOrganizations(ctx context.Context, actor *models.Actor, params pagination.Params) (*types.ListOrganizationsResponse, error)
	GetOrganizationByID(ctx context.Context, actor *models.Actor, organizationID string) (*types.Organization, error)
	UpdateOrganization(ctx context.Context, actor *models.Actor, organizationID string, request types.UpdateOrganizationRequest) (*types.Organization, error)
	DeleteOrganization(ctx context.Context, actor *models.Actor, organizationID string) error
	ExistsByID(ctx context.Context, organizationID string) (bool, error)
}

type OrganizationInvitationService interface {
	CreateOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationInvitationRequest, redirectURL string) (*types.OrganizationInvitation, error)
	GetOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error)
	GetOrganizationInvitationByID(ctx context.Context, invitationID string) (*types.OrganizationInvitation, error)
	GetOrganizationInvitationByIDWithOrg(ctx context.Context, invitationID string) (*types.GetOrganizationInvitationResponse, error)
	GetAllOrganizationInvitationsByOrgIDWithOrg(ctx context.Context, organizationID string, params pagination.Params) (*types.ListOrganizationInvitationsResponse, error)
	RevokeOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error)
	AcceptOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error)
	RejectOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error)
}

type OrganizationMemberService interface {
	AddMember(ctx context.Context, actor *models.Actor, organizationID string, request types.AddOrganizationMemberRequest) (*types.OrganizationMember, error)
	GetAllMembers(ctx context.Context, actor *models.Actor, organizationID string, params pagination.Params) (*types.ListOrganizationMembersResponse, error)
	GetMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) (*types.OrganizationMemberResponse, error)
	GetMemberByUserID(ctx context.Context, actor *models.Actor, organizationID string, userID string) (*types.OrganizationMemberResponse, error)
	UpdateMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string, request types.UpdateOrganizationMemberRequest) (*types.OrganizationMember, error)
	RemoveMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) error
}

type OrganizationTeamService interface {
	CreateTeam(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationTeamRequest) (*types.OrganizationTeam, error)
	GetAllTeams(ctx context.Context, actor *models.Actor, organizationID string, params pagination.Params) (*types.ListOrganizationTeamsResponse, error)
	GetTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) (*types.OrganizationTeam, error)
	UpdateTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.UpdateOrganizationTeamRequest) (*types.OrganizationTeam, error)
	DeleteTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) error
}

type OrganizationTeamMemberService interface {
	AddTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.AddOrganizationTeamMemberRequest) (*types.OrganizationTeamMember, error)
	GetAllTeamMembers(ctx context.Context, actor *models.Actor, organizationID string, teamID string, params pagination.Params) (*types.ListOrganizationTeamMembersResponse, error)
	GetTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) (*types.OrganizationTeamMemberResponse, error)
	RemoveTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) error
}
