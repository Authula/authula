package tests

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/types"
)

func actorID(actor *models.Actor) string {
	if actor == nil {
		return ""
	}

	return actor.ID
}

type MockOrganizationService struct {
	mock.Mock
}

func (m *MockOrganizationService) CreateOrganization(ctx context.Context, actor *models.Actor, request types.CreateOrganizationRequest) (*types.Organization, error) {
	args := m.Called(ctx, actorID(actor), request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
}

func (m *MockOrganizationService) GetAllOrganizationsByOwner(ctx context.Context, actor *models.Actor) ([]types.Organization, error) {
	args := m.Called(ctx, actorID(actor))
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Organization), args.Error(1)
}

func (m *MockOrganizationService) GetOrganizationByID(ctx context.Context, actor *models.Actor, organizationID string) (*types.Organization, error) {
	args := m.Called(ctx, actorID(actor), organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
}

func (m *MockOrganizationService) UpdateOrganization(ctx context.Context, actor *models.Actor, organizationID string, request types.UpdateOrganizationRequest) (*types.Organization, error) {
	args := m.Called(ctx, actorID(actor), organizationID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
}

func (m *MockOrganizationService) DeleteOrganization(ctx context.Context, actor *models.Actor, organizationID string) error {
	args := m.Called(ctx, actorID(actor), organizationID)
	return args.Error(0)
}

func (m *MockOrganizationService) ExistsByID(ctx context.Context, organizationID string) (bool, error) {
	args := m.Called(ctx, organizationID)
	return args.Bool(0), args.Error(1)
}

func (m *MockOrganizationService) GetByIDNoAuth(ctx context.Context, organizationID string) (*types.Organization, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
}

type MockOrganizationInvitationService struct {
	mock.Mock
}

func (m *MockOrganizationInvitationService) CreateOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationInvitationRequest, redirectURL string) (*types.OrganizationInvitation, error) {
	args := m.Called(ctx, actorID(actor), organizationID, request, redirectURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationService) GetOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	args := m.Called(ctx, actorID(actor), organizationID, invitationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationService) GetOrganizationInvitationByID(ctx context.Context, invitationID string) (*types.OrganizationInvitation, error) {
	args := m.Called(ctx, invitationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationService) GetAllOrganizationInvitations(ctx context.Context, actor *models.Actor, organizationID string) ([]types.OrganizationInvitation, error) {
	args := m.Called(ctx, actorID(actor), organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationService) RevokeOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	args := m.Called(ctx, actorID(actor), organizationID, invitationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationService) AcceptOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	args := m.Called(ctx, actorID(actor), organizationID, invitationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationService) RejectOrganizationInvitation(ctx context.Context, actor *models.Actor, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	args := m.Called(ctx, actorID(actor), organizationID, invitationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
}

type MockOrganizationMemberService struct {
	mock.Mock
}

func (m *MockOrganizationMemberService) AddMember(ctx context.Context, actor *models.Actor, organizationID string, request types.AddOrganizationMemberRequest) (*types.OrganizationMember, error) {
	args := m.Called(ctx, actorID(actor), organizationID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationMemberService) GetAllMembers(ctx context.Context, actor *models.Actor, organizationID string, page int, limit int) ([]types.OrganizationMemberResponse, error) {
	args := m.Called(ctx, actorID(actor), organizationID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationMemberResponse), args.Error(1)
}

func (m *MockOrganizationMemberService) GetMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) (*types.OrganizationMemberResponse, error) {
	args := m.Called(ctx, actorID(actor), organizationID, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationMemberResponse), args.Error(1)
}

func (m *MockOrganizationMemberService) UpdateMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string, request types.UpdateOrganizationMemberRequest) (*types.OrganizationMember, error) {
	args := m.Called(ctx, actorID(actor), organizationID, memberID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationMemberService) GetMemberByUserID(ctx context.Context, actor *models.Actor, organizationID string, userID string) (*types.OrganizationMemberResponse, error) {
	args := m.Called(ctx, actorID(actor), organizationID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationMemberResponse), args.Error(1)
}

func (m *MockOrganizationMemberService) RemoveMember(ctx context.Context, actor *models.Actor, organizationID string, memberID string) error {
	args := m.Called(ctx, actorID(actor), organizationID, memberID)
	return args.Error(0)
}

type MockOrganizationTeamService struct {
	mock.Mock
}

func (m *MockOrganizationTeamService) CreateTeam(ctx context.Context, actor *models.Actor, organizationID string, request types.CreateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	args := m.Called(ctx, actorID(actor), organizationID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeam), args.Error(1)
}

func (m *MockOrganizationTeamService) GetAllTeams(ctx context.Context, actor *models.Actor, organizationID string) ([]types.OrganizationTeam, error) {
	args := m.Called(ctx, actorID(actor), organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationTeam), args.Error(1)
}

func (m *MockOrganizationTeamService) GetTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) (*types.OrganizationTeam, error) {
	args := m.Called(ctx, actorID(actor), organizationID, teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeam), args.Error(1)
}

func (m *MockOrganizationTeamService) UpdateTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.UpdateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	args := m.Called(ctx, actorID(actor), organizationID, teamID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeam), args.Error(1)
}

func (m *MockOrganizationTeamService) DeleteTeam(ctx context.Context, actor *models.Actor, organizationID string, teamID string) error {
	args := m.Called(ctx, actorID(actor), organizationID, teamID)
	return args.Error(0)
}

type MockOrganizationTeamMemberService struct {
	mock.Mock
}

func (m *MockOrganizationTeamMemberService) AddTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, request types.AddOrganizationTeamMemberRequest) (*types.OrganizationTeamMember, error) {
	args := m.Called(ctx, actorID(actor), organizationID, teamID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeamMember), args.Error(1)
}

func (m *MockOrganizationTeamMemberService) GetAllTeamMembers(ctx context.Context, actor *models.Actor, organizationID string, teamID string, page int, limit int) ([]types.OrganizationTeamMemberResponse, error) {
	args := m.Called(ctx, actorID(actor), organizationID, teamID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationTeamMemberResponse), args.Error(1)
}

func (m *MockOrganizationTeamMemberService) GetTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) (*types.OrganizationTeamMemberResponse, error) {
	args := m.Called(ctx, actorID(actor), organizationID, teamID, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeamMemberResponse), args.Error(1)
}

func (m *MockOrganizationTeamMemberService) RemoveTeamMember(ctx context.Context, actor *models.Actor, organizationID string, teamID string, memberID string) error {
	args := m.Called(ctx, actorID(actor), organizationID, teamID, memberID)
	return args.Error(0)
}
