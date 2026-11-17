package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/pagination"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgservices "github.com/Authula/authula/plugins/organizations/services"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
)

const (
	testOrgID  = "org-1"
	testUserID = "user-1"
)

type useCaseFixture struct {
	orgSvc        *orgtests.MockOrganizationService
	invSvc        *orgtests.MockOrganizationInvitationService
	memberSvc     *orgtests.MockOrganizationMemberService
	teamSvc       *orgtests.MockOrganizationTeamService
	teamMemberSvc *orgtests.MockOrganizationTeamMemberService
	orgRepo       *orgtests.MockOrganizationRepository
	memberRepo    *orgtests.MockOrganizationMemberRepository
	teamRepo      *orgtests.MockOrganizationTeamRepository
	userSvc       *internaltests.MockUserService
	accessControl *orgtests.AccessControlServiceStub
	uc            *UseCases
}

func newUseCaseFixture() *useCaseFixture {
	fixture := &useCaseFixture{
		orgSvc:        &orgtests.MockOrganizationService{},
		invSvc:        &orgtests.MockOrganizationInvitationService{},
		memberSvc:     &orgtests.MockOrganizationMemberService{},
		teamSvc:       &orgtests.MockOrganizationTeamService{},
		teamMemberSvc: &orgtests.MockOrganizationTeamMemberService{},
		orgRepo:       &orgtests.MockOrganizationRepository{},
		memberRepo:    &orgtests.MockOrganizationMemberRepository{},
		teamRepo:      &orgtests.MockOrganizationTeamRepository{},
		userSvc:       &internaltests.MockUserService{},
		accessControl: orgtests.NewAccessControlServiceStub(),
	}
	fixture.uc = &UseCases{
		orgService:        fixture.orgSvc,
		invitationService: fixture.invSvc,
		memberService:     fixture.memberSvc,
		teamService:       fixture.teamSvc,
		teamMemberService: fixture.teamMemberSvc,
		userService:       fixture.userSvc,
		globalConfig:      &models.Config{},
		authorizer:        rootservices.NewDefaultAuthorizer(),
		serviceUtils:      orgservices.NewServiceUtils(fixture.orgRepo, fixture.memberRepo, fixture.teamRepo, 0),
		accessControl:     fixture.accessControl,
	}
	return fixture
}

// grantAccess arranges the repository lookups authorizeOrgAccess performs and
// gives the actor's membership role the provided permissions.
func (f *useCaseFixture) grantAccess(role string, permissions []string) {
	f.orgRepo.On("GetByID", mock.Anything, testOrgID).Return(&types.Organization{ID: testOrgID, OwnerID: "owner-1"}, nil).Maybe()
	f.memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, testOrgID, testUserID).
		Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: testOrgID, UserID: testUserID, Role: role}, nil).Maybe()
	f.accessControl.RolePermissions[role] = permissions
}

func (f *useCaseFixture) assertExpectations(t *testing.T) {
	t.Helper()

	f.orgSvc.AssertExpectations(t)
	f.invSvc.AssertExpectations(t)
	f.memberSvc.AssertExpectations(t)
	f.teamSvc.AssertExpectations(t)
	f.teamMemberSvc.AssertExpectations(t)
	f.orgRepo.AssertExpectations(t)
	f.memberRepo.AssertExpectations(t)
	f.teamRepo.AssertExpectations(t)
	f.userSvc.AssertExpectations(t)
}

func TestUseCases_DirectDelegation(t *testing.T) {
	t.Parallel()

	svcErr := errors.New("service failure")
	organization := &types.Organization{ID: testOrgID, Name: "Acme"}
	organizationList := &types.ListOrganizationsResponse{Data: []types.Organization{*organization}}
	invitation := &types.OrganizationInvitation{ID: "inv-1", OrganizationID: testOrgID}

	tests := []struct {
		name    string
		call    func(*UseCases) (any, error)
		arrange func(*useCaseFixture, any, error)
		want    any
	}{
		{
			name: "create organization",
			call: func(uc *UseCases) (any, error) {
				return uc.CreateOrganization(context.Background(), orgtests.Actor(testUserID), types.CreateOrganizationRequest{Name: "Acme", Role: "owner"})
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.orgSvc.On("CreateOrganization", mock.Anything, testUserID, types.CreateOrganizationRequest{Name: "Acme", Role: "owner"}).Return(result, err).Once()
			},
			want: organization,
		},
		{
			name: "list organizations",
			call: func(uc *UseCases) (any, error) {
				return uc.ListAllOrganizations(context.Background(), orgtests.Actor(testUserID), pagination.Params{Page: 1, Limit: 10})
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.orgSvc.On("ListAllOrganizations", mock.Anything, testUserID, pagination.Params{Page: 1, Limit: 10}).Return(result, err).Once()
			},
			want: organizationList,
		},
		{
			name: "update organization",
			call: func(uc *UseCases) (any, error) {
				return uc.UpdateOrganization(context.Background(), orgtests.Actor(testUserID), testOrgID, types.UpdateOrganizationRequest{})
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.orgSvc.On("UpdateOrganization", mock.Anything, testUserID, testOrgID, types.UpdateOrganizationRequest{}).Return(result, err).Once()
			},
			want: organization,
		},
		{
			name: "delete organization",
			call: func(uc *UseCases) (any, error) {
				return nil, uc.DeleteOrganization(context.Background(), orgtests.Actor(testUserID), testOrgID)
			},
			arrange: func(f *useCaseFixture, _ any, err error) {
				f.orgSvc.On("DeleteOrganization", mock.Anything, testUserID, testOrgID).Return(err).Once()
			},
		},
		{
			name: "exists by id",
			call: func(uc *UseCases) (any, error) {
				return uc.ExistsByID(context.Background(), testOrgID)
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				exists := false
				if value, ok := result.(bool); ok {
					exists = value
				}
				f.orgSvc.On("ExistsByID", mock.Anything, testOrgID).Return(exists, err).Once()
			},
			want: true,
		},
		{
			name: "accept invitation",
			call: func(uc *UseCases) (any, error) {
				return uc.AcceptOrganizationInvitation(context.Background(), orgtests.Actor(testUserID), testOrgID, "inv-1")
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.invSvc.On("AcceptOrganizationInvitation", mock.Anything, testUserID, testOrgID, "inv-1").Return(result, err).Once()
			},
			want: invitation,
		},
		{
			name: "reject invitation",
			call: func(uc *UseCases) (any, error) {
				return uc.RejectOrganizationInvitation(context.Background(), orgtests.Actor(testUserID), testOrgID, "inv-1")
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.invSvc.On("RejectOrganizationInvitation", mock.Anything, testUserID, testOrgID, "inv-1").Return(result, err).Once()
			},
			want: invitation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			t.Run("success delegates and returns the service result", func(t *testing.T) {
				t.Parallel()

				fixture := newUseCaseFixture()
				tt.arrange(fixture, tt.want, nil)
				got, err := tt.call(fixture.uc)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
				fixture.assertExpectations(t)
			})

			t.Run("service error is propagated", func(t *testing.T) {
				t.Parallel()

				fixture := newUseCaseFixture()
				tt.arrange(fixture, nil, svcErr)
				got, err := tt.call(fixture.uc)
				require.ErrorIs(t, err, svcErr)
				require.Empty(t, got)
				fixture.assertExpectations(t)
			})
		})
	}
}

func TestUseCases_AuthorizedDelegation(t *testing.T) {
	t.Parallel()

	svcErr := errors.New("service failure")
	organization := &types.Organization{ID: testOrgID, Name: "Acme"}
	invitation := &types.OrganizationInvitation{ID: "inv-1", OrganizationID: testOrgID}
	invitationList := &types.ListOrganizationInvitationsResponse{Data: []types.GetOrganizationInvitationResponse{}}
	member := &types.OrganizationMember{ID: "mem-2", OrganizationID: testOrgID, UserID: "user-2", Role: "member"}
	memberResponse := &types.OrganizationMemberResponse{ID: "mem-2", OrganizationID: testOrgID}
	memberList := &types.ListOrganizationMembersResponse{Data: []types.OrganizationMemberResponse{}}
	team := &types.OrganizationTeam{ID: "team-1", OrganizationID: testOrgID}
	teamList := &types.ListOrganizationTeamsResponse{Data: []types.OrganizationTeam{}}
	teamMember := &types.OrganizationTeamMember{ID: "tm-1", TeamID: "team-1", MemberID: "mem-2"}
	teamMemberResponse := &types.OrganizationTeamMemberResponse{ID: "tm-1", TeamID: "team-1"}
	teamMemberList := &types.ListOrganizationTeamMembersResponse{Data: []types.OrganizationTeamMemberResponse{}}

	tests := []struct {
		name    string
		call    func(*UseCases) (any, error)
		arrange func(*useCaseFixture, any, error)
		want    any
	}{
		{
			name: "get organization by id",
			call: func(uc *UseCases) (any, error) {
				return uc.GetOrganizationByID(context.Background(), orgtests.Actor(testUserID), testOrgID)
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.orgSvc.On("GetOrganizationByID", mock.Anything, testUserID, testOrgID).Return(result, err).Once()
			},
			want: organization,
		},
		{
			name: "create invitation",
			call: func(uc *UseCases) (any, error) {
				return uc.CreateOrganizationInvitation(context.Background(), orgtests.Actor(testUserID), testOrgID, types.CreateOrganizationInvitationRequest{Email: "bob@example.com", Role: "member"}, "https://app.example.com")
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.invSvc.On("CreateOrganizationInvitation", mock.Anything, testUserID, testOrgID, types.CreateOrganizationInvitationRequest{Email: "bob@example.com", Role: "member"}, "https://app.example.com").Return(result, err).Once()
			},
			want: invitation,
		},
		{
			name: "list invitations",
			call: func(uc *UseCases) (any, error) {
				return uc.ListAllOrganizationInvitations(context.Background(), orgtests.Actor(testUserID), testOrgID, pagination.Params{Page: 1, Limit: 10})
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.invSvc.On("ListAllOrganizationInvitationsByOrgIDWithOrg", mock.Anything, testOrgID, pagination.Params{Page: 1, Limit: 10}).Return(result, err).Once()
			},
			want: invitationList,
		},
		{
			name: "revoke invitation",
			call: func(uc *UseCases) (any, error) {
				return uc.RevokeOrganizationInvitation(context.Background(), orgtests.Actor(testUserID), testOrgID, "inv-1")
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.invSvc.On("RevokeOrganizationInvitation", mock.Anything, testUserID, testOrgID, "inv-1").Return(result, err).Once()
			},
			want: invitation,
		},
		{
			name: "add member",
			call: func(uc *UseCases) (any, error) {
				return uc.AddMember(context.Background(), orgtests.Actor(testUserID), testOrgID, types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"})
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.memberSvc.On("AddMember", mock.Anything, testUserID, testOrgID, types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"}).Return(result, err).Once()
			},
			want: member,
		},
		{
			name: "list members",
			call: func(uc *UseCases) (any, error) {
				return uc.ListAllMembers(context.Background(), orgtests.Actor(testUserID), testOrgID, pagination.Params{Page: 1, Limit: 10})
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.memberSvc.On("ListAllMembers", mock.Anything, testUserID, testOrgID, pagination.Params{Page: 1, Limit: 10}).Return(result, err).Once()
			},
			want: memberList,
		},
		{
			name: "get member",
			call: func(uc *UseCases) (any, error) {
				return uc.GetMember(context.Background(), orgtests.Actor(testUserID), testOrgID, "mem-2")
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.memberSvc.On("GetMember", mock.Anything, testUserID, testOrgID, "mem-2").Return(result, err).Once()
			},
			want: memberResponse,
		},
		{
			name: "get member by user id",
			call: func(uc *UseCases) (any, error) {
				return uc.GetMemberByUserID(context.Background(), orgtests.Actor(testUserID), testOrgID, "user-2")
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.memberSvc.On("GetMemberByUserID", mock.Anything, testUserID, testOrgID, "user-2").Return(result, err).Once()
			},
			want: memberResponse,
		},
		{
			name: "update member",
			call: func(uc *UseCases) (any, error) {
				return uc.UpdateMember(context.Background(), orgtests.Actor(testUserID), testOrgID, "mem-2", types.UpdateOrganizationMemberRequest{Role: "admin"})
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.memberSvc.On("UpdateMember", mock.Anything, testUserID, testOrgID, "mem-2", types.UpdateOrganizationMemberRequest{Role: "admin"}).Return(result, err).Once()
			},
			want: member,
		},
		{
			name: "remove member",
			call: func(uc *UseCases) (any, error) {
				return nil, uc.RemoveMember(context.Background(), orgtests.Actor(testUserID), testOrgID, "mem-2")
			},
			arrange: func(f *useCaseFixture, _ any, err error) {
				f.memberSvc.On("RemoveMember", mock.Anything, testUserID, testOrgID, "mem-2").Return(err).Once()
			},
		},
		{
			name: "create team",
			call: func(uc *UseCases) (any, error) {
				return uc.CreateTeam(context.Background(), orgtests.Actor(testUserID), testOrgID, types.CreateOrganizationTeamRequest{Name: "Platform"})
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.teamSvc.On("CreateTeam", mock.Anything, testUserID, testOrgID, types.CreateOrganizationTeamRequest{Name: "Platform"}).Return(result, err).Once()
			},
			want: team,
		},
		{
			name: "list teams",
			call: func(uc *UseCases) (any, error) {
				return uc.ListAllTeams(context.Background(), orgtests.Actor(testUserID), testOrgID, pagination.Params{Page: 1, Limit: 10})
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.teamSvc.On("ListAllTeams", mock.Anything, testUserID, testOrgID, pagination.Params{Page: 1, Limit: 10}).Return(result, err).Once()
			},
			want: teamList,
		},
		{
			name: "get team",
			call: func(uc *UseCases) (any, error) {
				return uc.GetTeam(context.Background(), orgtests.Actor(testUserID), testOrgID, "team-1")
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.teamSvc.On("GetTeam", mock.Anything, testUserID, testOrgID, "team-1").Return(result, err).Once()
			},
			want: team,
		},
		{
			name: "update team",
			call: func(uc *UseCases) (any, error) {
				return uc.UpdateTeam(context.Background(), orgtests.Actor(testUserID), testOrgID, "team-1", types.UpdateOrganizationTeamRequest{Name: "Platform"})
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.teamSvc.On("UpdateTeam", mock.Anything, testUserID, testOrgID, "team-1", types.UpdateOrganizationTeamRequest{Name: "Platform"}).Return(result, err).Once()
			},
			want: team,
		},
		{
			name: "delete team",
			call: func(uc *UseCases) (any, error) {
				return nil, uc.DeleteTeam(context.Background(), orgtests.Actor(testUserID), testOrgID, "team-1")
			},
			arrange: func(f *useCaseFixture, _ any, err error) {
				f.teamSvc.On("DeleteTeam", mock.Anything, testUserID, testOrgID, "team-1").Return(err).Once()
			},
		},
		{
			name: "add team member",
			call: func(uc *UseCases) (any, error) {
				return uc.AddTeamMember(context.Background(), orgtests.Actor(testUserID), testOrgID, "team-1", types.AddOrganizationTeamMemberRequest{MemberID: "mem-2"})
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.teamMemberSvc.On("AddTeamMember", mock.Anything, testUserID, testOrgID, "team-1", types.AddOrganizationTeamMemberRequest{MemberID: "mem-2"}).Return(result, err).Once()
			},
			want: teamMember,
		},
		{
			name: "list team members",
			call: func(uc *UseCases) (any, error) {
				return uc.ListAllTeamMembers(context.Background(), orgtests.Actor(testUserID), testOrgID, "team-1", pagination.Params{Page: 1, Limit: 10})
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.teamMemberSvc.On("ListAllTeamMembers", mock.Anything, testUserID, testOrgID, "team-1", pagination.Params{Page: 1, Limit: 10}).Return(result, err).Once()
			},
			want: teamMemberList,
		},
		{
			name: "get team member",
			call: func(uc *UseCases) (any, error) {
				return uc.GetTeamMember(context.Background(), orgtests.Actor(testUserID), testOrgID, "team-1", "mem-2")
			},
			arrange: func(f *useCaseFixture, result any, err error) {
				f.teamMemberSvc.On("GetTeamMember", mock.Anything, testUserID, testOrgID, "team-1", "mem-2").Return(result, err).Once()
			},
			want: teamMemberResponse,
		},
		{
			name: "remove team member",
			call: func(uc *UseCases) (any, error) {
				return nil, uc.RemoveTeamMember(context.Background(), orgtests.Actor(testUserID), testOrgID, "team-1", "mem-2")
			},
			arrange: func(f *useCaseFixture, _ any, err error) {
				f.teamMemberSvc.On("RemoveTeamMember", mock.Anything, testUserID, testOrgID, "team-1", "mem-2").Return(err).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			t.Run("success delegates and returns the service result", func(t *testing.T) {
				t.Parallel()

				fixture := newUseCaseFixture()
				fixture.grantAccess("admin", []string{"*"})
				tt.arrange(fixture, tt.want, nil)
				got, err := tt.call(fixture.uc)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
				fixture.assertExpectations(t)
			})

			t.Run("service error is propagated", func(t *testing.T) {
				t.Parallel()

				fixture := newUseCaseFixture()
				fixture.grantAccess("admin", []string{"*"})
				tt.arrange(fixture, nil, svcErr)
				got, err := tt.call(fixture.uc)
				require.ErrorIs(t, err, svcErr)
				require.Nil(t, got)
				fixture.assertExpectations(t)
			})

			t.Run("missing permission short-circuits the service call", func(t *testing.T) {
				t.Parallel()

				fixture := newUseCaseFixture()
				fixture.grantAccess("viewer", []string{"users:read"})
				got, err := tt.call(fixture.uc)
				require.ErrorIs(t, err, coreerrors.ErrInsufficientPermissions)
				require.Nil(t, got)
				fixture.assertExpectations(t)
			})
		})
	}
}

func TestNewUseCases_WiresDependencies(t *testing.T) {
	t.Parallel()

	fixture := newUseCaseFixture()
	globalConfig := &models.Config{AppName: "Authula"}
	authorizer := rootservices.NewDefaultAuthorizer()
	serviceUtils := orgservices.NewServiceUtils(fixture.orgRepo, fixture.memberRepo, fixture.teamRepo, 25)

	uc := NewUseCases(
		fixture.orgSvc,
		fixture.invSvc,
		fixture.memberSvc,
		fixture.teamSvc,
		fixture.teamMemberSvc,
		fixture.userSvc,
		globalConfig,
		authorizer,
		serviceUtils,
		fixture.accessControl,
	)

	require.NotNil(t, uc)
	require.Equal(t, fixture.orgSvc, uc.orgService)
	require.Equal(t, fixture.invSvc, uc.invitationService)
	require.Equal(t, fixture.memberSvc, uc.memberService)
	require.Equal(t, fixture.teamSvc, uc.teamService)
	require.Equal(t, fixture.teamMemberSvc, uc.teamMemberService)
	require.Equal(t, fixture.userSvc, uc.userService)
	require.Equal(t, globalConfig, uc.globalConfig)
	require.Equal(t, authorizer, uc.authorizer)
	require.Equal(t, serviceUtils, uc.serviceUtils)
	require.Equal(t, fixture.accessControl, uc.accessControl)
}
