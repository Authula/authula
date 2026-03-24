package tests

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/Authula/authula/plugins/organizations/types"
)

type MockOrganizationService struct {
	mock.Mock
}

func (m *MockOrganizationService) CreateOrganization(ctx context.Context, actorUserID string, request types.CreateOrganizationRequest) (*types.Organization, error) {
	args := m.Called(ctx, actorUserID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
}

func (m *MockOrganizationService) GetAllOrganizations(ctx context.Context, actorUserID string) ([]types.Organization, error) {
	args := m.Called(ctx, actorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Organization), args.Error(1)
}

func (m *MockOrganizationService) GetOrganizationByID(ctx context.Context, actorUserID string, organizationID string) (*types.Organization, error) {
	args := m.Called(ctx, actorUserID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
}

func (m *MockOrganizationService) UpdateOrganization(ctx context.Context, actorUserID string, organizationID string, request types.UpdateOrganizationRequest) (*types.Organization, error) {
	args := m.Called(ctx, actorUserID, organizationID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
}

func (m *MockOrganizationService) DeleteOrganization(ctx context.Context, actorUserID string, organizationID string) error {
	args := m.Called(ctx, actorUserID, organizationID)
	return args.Error(0)
}

type MockOrganizationInvitationService struct {
	mock.Mock
}

func (m *MockOrganizationInvitationService) CreateOrganizationInvitation(ctx context.Context, actorUserID string, organizationID string, request types.CreateOrganizationInvitationRequest) (*types.OrganizationInvitation, error) {
	args := m.Called(ctx, actorUserID, organizationID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationService) GetOrganizationInvitation(ctx context.Context, actorUserID string, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	args := m.Called(ctx, actorUserID, organizationID, invitationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationService) GetAllOrganizationInvitations(ctx context.Context, actorUserID string, organizationID string) ([]types.OrganizationInvitation, error) {
	args := m.Called(ctx, actorUserID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationService) RevokeOrganizationInvitation(ctx context.Context, actorUserID string, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	args := m.Called(ctx, actorUserID, organizationID, invitationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationService) AcceptOrganizationInvitation(ctx context.Context, actorUserID string, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	args := m.Called(ctx, actorUserID, organizationID, invitationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
}

func (m *MockOrganizationInvitationService) RejectOrganizationInvitation(ctx context.Context, actorUserID string, organizationID string, invitationID string) (*types.OrganizationInvitation, error) {
	args := m.Called(ctx, actorUserID, organizationID, invitationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationInvitation), args.Error(1)
}

type MockOrganizationMemberService struct {
	mock.Mock
}

func (m *MockOrganizationMemberService) AddMember(ctx context.Context, actorUserID string, organizationID string, request types.AddOrganizationMemberRequest) (*types.OrganizationMember, error) {
	args := m.Called(ctx, actorUserID, organizationID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationMemberService) GetAllMembers(ctx context.Context, actorUserID string, organizationID string, page int, limit int) ([]types.OrganizationMember, error) {
	args := m.Called(ctx, actorUserID, organizationID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationMemberService) GetMember(ctx context.Context, actorUserID string, organizationID string, memberID string) (*types.OrganizationMember, error) {
	args := m.Called(ctx, actorUserID, organizationID, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationMemberService) UpdateMember(ctx context.Context, actorUserID string, organizationID string, memberID string, request types.UpdateOrganizationMemberRequest) (*types.OrganizationMember, error) {
	args := m.Called(ctx, actorUserID, organizationID, memberID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationMemberService) RemoveMember(ctx context.Context, actorUserID string, organizationID string, memberID string) error {
	args := m.Called(ctx, actorUserID, organizationID, memberID)
	return args.Error(0)
}

type MockOrganizationTeamService struct {
	mock.Mock
}

func (m *MockOrganizationTeamService) CreateTeam(ctx context.Context, actorUserID string, organizationID string, request types.CreateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	args := m.Called(ctx, actorUserID, organizationID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeam), args.Error(1)
}

func (m *MockOrganizationTeamService) GetAllTeams(ctx context.Context, actorUserID string, organizationID string) ([]types.OrganizationTeam, error) {
	args := m.Called(ctx, actorUserID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationTeam), args.Error(1)
}

func (m *MockOrganizationTeamService) GetTeam(ctx context.Context, actorUserID string, organizationID string, teamID string) (*types.OrganizationTeam, error) {
	args := m.Called(ctx, actorUserID, organizationID, teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeam), args.Error(1)
}

func (m *MockOrganizationTeamService) UpdateTeam(ctx context.Context, actorUserID string, organizationID string, teamID string, request types.UpdateOrganizationTeamRequest) (*types.OrganizationTeam, error) {
	args := m.Called(ctx, actorUserID, organizationID, teamID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeam), args.Error(1)
}

func (m *MockOrganizationTeamService) DeleteTeam(ctx context.Context, actorUserID string, organizationID string, teamID string) error {
	args := m.Called(ctx, actorUserID, organizationID, teamID)
	return args.Error(0)
}

type MockOrganizationTeamMemberService struct {
	mock.Mock
}

func (m *MockOrganizationTeamMemberService) AddTeamMember(ctx context.Context, actorUserID string, organizationID string, teamID string, request types.AddOrganizationTeamMemberRequest) (*types.OrganizationTeamMember, error) {
	args := m.Called(ctx, actorUserID, organizationID, teamID, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeamMember), args.Error(1)
}

func (m *MockOrganizationTeamMemberService) GetAllTeamMembers(ctx context.Context, actorUserID string, organizationID string, teamID string, page int, limit int) ([]types.OrganizationTeamMember, error) {
	args := m.Called(ctx, actorUserID, organizationID, teamID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.OrganizationTeamMember), args.Error(1)
}

func (m *MockOrganizationTeamMemberService) GetTeamMember(ctx context.Context, actorUserID string, organizationID string, teamID string, memberID string) (*types.OrganizationTeamMember, error) {
	args := m.Called(ctx, actorUserID, organizationID, teamID, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationTeamMember), args.Error(1)
}

func (m *MockOrganizationTeamMemberService) RemoveTeamMember(ctx context.Context, actorUserID string, organizationID string, teamID string, memberID string) error {
	args := m.Called(ctx, actorUserID, organizationID, teamID, memberID)
	return args.Error(0)
}
