package organizations

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/pagination"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
)

type apiFixture struct {
	api           *API
	orgSvc        *orgtests.MockOrganizationService
	invSvc        *orgtests.MockOrganizationInvitationService
	memberSvc     *orgtests.MockOrganizationMemberService
	teamSvc       *orgtests.MockOrganizationTeamService
	teamMemberSvc *orgtests.MockOrganizationTeamMemberService
	memberRepo    *orgtests.MockOrganizationMemberRepository
	accessControl *orgtests.AccessControlServiceStub
}

func newAPIFixture() *apiFixture {
	fixture := &apiFixture{
		orgSvc:        &orgtests.MockOrganizationService{},
		invSvc:        &orgtests.MockOrganizationInvitationService{},
		memberSvc:     &orgtests.MockOrganizationMemberService{},
		teamSvc:       &orgtests.MockOrganizationTeamService{},
		teamMemberSvc: &orgtests.MockOrganizationTeamMemberService{},
		memberRepo:    &orgtests.MockOrganizationMemberRepository{},
		accessControl: orgtests.NewAccessControlServiceStub(),
	}
	fixture.api = &API{
		organizationService:  fixture.orgSvc,
		invitationService:    fixture.invSvc,
		memberService:        fixture.memberSvc,
		teamService:          fixture.teamSvc,
		teamMemberService:    fixture.teamMemberSvc,
		memberRepo:           fixture.memberRepo,
		accessControlService: fixture.accessControl,
	}
	return fixture
}

func (f *apiFixture) assertExpectations(t *testing.T) {
	t.Helper()

	f.orgSvc.AssertExpectations(t)
	f.invSvc.AssertExpectations(t)
	f.memberSvc.AssertExpectations(t)
	f.teamSvc.AssertExpectations(t)
	f.teamMemberSvc.AssertExpectations(t)
	f.memberRepo.AssertExpectations(t)
}

func TestAPI_DelegatesToServices(t *testing.T) {
	t.Parallel()

	svcErr := errors.New("service failure")
	actor := orgtests.Actor("user-1")
	organization := &types.Organization{ID: "org-1", Name: "Acme"}
	organizationList := &types.ListOrganizationsResponse{Data: []types.Organization{*organization}}
	invitation := &types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1"}
	invitationList := &types.ListOrganizationInvitationsResponse{Data: []types.GetOrganizationInvitationResponse{}}
	member := &types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}
	memberResponse := &types.OrganizationMemberResponse{ID: "mem-1", OrganizationID: "org-1"}
	memberList := &types.ListOrganizationMembersResponse{Data: []types.OrganizationMemberResponse{}}
	team := &types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}
	teamList := &types.ListOrganizationTeamsResponse{Data: []types.OrganizationTeam{}}
	teamMember := &types.OrganizationTeamMember{ID: "tm-1", TeamID: "team-1", MemberID: "mem-1"}
	teamMemberResponse := &types.OrganizationTeamMemberResponse{ID: "tm-1", TeamID: "team-1"}
	teamMemberList := &types.ListOrganizationTeamMembersResponse{Data: []types.OrganizationTeamMemberResponse{}}

	tests := []struct {
		name    string
		call    func(*API) (any, error)
		arrange func(*apiFixture, any, error)
		want    any
	}{
		{
			name: "exists by id",
			call: func(api *API) (any, error) { return api.ExistsByID(context.Background(), "org-1") },
			arrange: func(f *apiFixture, result any, err error) {
				exists := false
				if value, ok := result.(bool); ok {
					exists = value
				}
				f.orgSvc.On("ExistsByID", mock.Anything, "org-1").Return(exists, err).Once()
			},
			want: true,
		},
		{
			name: "create organization",
			call: func(api *API) (any, error) {
				return api.CreateOrganization(context.Background(), actor, types.CreateOrganizationRequest{Name: "Acme", Role: "owner"})
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.orgSvc.On("CreateOrganization", mock.Anything, "user-1", types.CreateOrganizationRequest{Name: "Acme", Role: "owner"}).Return(result, err).Once()
			},
			want: organization,
		},
		{
			name: "list all organizations",
			call: func(api *API) (any, error) {
				return api.ListAllOrganizations(context.Background(), actor, pagination.Params{Page: 1, Limit: 10})
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.orgSvc.On("ListAllOrganizations", mock.Anything, "user-1", pagination.Params{Page: 1, Limit: 10}).Return(result, err).Once()
			},
			want: organizationList,
		},
		{
			name: "get all organizations",
			call: func(api *API) (any, error) { return api.GetAllOrganizations(context.Background(), actor) },
			arrange: func(f *apiFixture, result any, err error) {
				f.orgSvc.On("GetAllOrganizations", mock.Anything, actor).Return(result, err).Once()
			},
			want: []types.Organization{*organization},
		},
		{
			name: "get all organizations unscoped",
			call: func(api *API) (any, error) { return api.GetAllOrganizationsUnscoped(context.Background()) },
			arrange: func(f *apiFixture, result any, err error) {
				f.orgSvc.On("GetAllOrganizationsUnscoped", mock.Anything).Return(result, err).Once()
			},
			want: []types.Organization{*organization},
		},
		{
			name: "get organization by id",
			call: func(api *API) (any, error) { return api.GetOrganizationByID(context.Background(), actor, "org-1") },
			arrange: func(f *apiFixture, result any, err error) {
				f.orgSvc.On("GetOrganizationByID", mock.Anything, "user-1", "org-1").Return(result, err).Once()
			},
			want: organization,
		},
		{
			name: "update organization",
			call: func(api *API) (any, error) {
				return api.UpdateOrganization(context.Background(), actor, "org-1", types.UpdateOrganizationRequest{})
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.orgSvc.On("UpdateOrganization", mock.Anything, "user-1", "org-1", types.UpdateOrganizationRequest{}).Return(result, err).Once()
			},
			want: organization,
		},
		{
			name: "delete organization",
			call: func(api *API) (any, error) {
				return nil, api.DeleteOrganization(context.Background(), actor, "org-1")
			},
			arrange: func(f *apiFixture, _ any, err error) {
				f.orgSvc.On("DeleteOrganization", mock.Anything, "user-1", "org-1").Return(err).Once()
			},
		},
		{
			name: "create invitation",
			call: func(api *API) (any, error) {
				return api.CreateInvitation(context.Background(), actor, "org-1", types.CreateOrganizationInvitationRequest{Email: "bob@example.com", Role: "member"}, "https://app.example.com")
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.invSvc.On("CreateOrganizationInvitation", mock.Anything, "user-1", "org-1", types.CreateOrganizationInvitationRequest{Email: "bob@example.com", Role: "member"}, "https://app.example.com").Return(result, err).Once()
			},
			want: invitation,
		},
		{
			name: "list all invitations",
			call: func(api *API) (any, error) {
				return api.ListAllInvitations(context.Background(), actor, "org-1", pagination.Params{Page: 1, Limit: 10})
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.invSvc.On("ListAllOrganizationInvitationsByOrgIDWithOrg", mock.Anything, "org-1", pagination.Params{Page: 1, Limit: 10}).Return(result, err).Once()
			},
			want: invitationList,
		},
		{
			name: "get all invitations",
			call: func(api *API) (any, error) { return api.GetAllInvitations(context.Background(), actor, "org-1") },
			arrange: func(f *apiFixture, result any, err error) {
				f.invSvc.On("GetAllOrganizationInvitationsByOrgIDWithOrg", mock.Anything, "org-1").Return(result, err).Once()
			},
			want: []types.GetOrganizationInvitationResponse{},
		},
		{
			name: "get invitation",
			call: func(api *API) (any, error) { return api.GetInvitation(context.Background(), actor, "org-1", "inv-1") },
			arrange: func(f *apiFixture, result any, err error) {
				f.invSvc.On("GetOrganizationInvitationByIDWithOrg", mock.Anything, "inv-1").Return(result, err).Once()
			},
			want: &types.GetOrganizationInvitationResponse{Invitation: invitation},
		},
		{
			name: "revoke invitation",
			call: func(api *API) (any, error) {
				return api.RevokeInvitation(context.Background(), actor, "org-1", "inv-1")
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.invSvc.On("RevokeOrganizationInvitation", mock.Anything, "user-1", "org-1", "inv-1").Return(result, err).Once()
			},
			want: invitation,
		},
		{
			name: "accept invitation",
			call: func(api *API) (any, error) {
				return api.AcceptInvitation(context.Background(), actor, "org-1", "inv-1")
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.invSvc.On("AcceptOrganizationInvitation", mock.Anything, "user-1", "org-1", "inv-1").Return(result, err).Once()
			},
			want: invitation,
		},
		{
			name: "reject invitation",
			call: func(api *API) (any, error) {
				return api.RejectInvitation(context.Background(), actor, "org-1", "inv-1")
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.invSvc.On("RejectOrganizationInvitation", mock.Anything, "user-1", "org-1", "inv-1").Return(result, err).Once()
			},
			want: invitation,
		},
		{
			name: "add member",
			call: func(api *API) (any, error) {
				return api.AddMember(context.Background(), actor, "org-1", types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"})
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.memberSvc.On("AddMember", mock.Anything, "user-1", "org-1", types.AddOrganizationMemberRequest{UserID: "user-2", Role: "member"}).Return(result, err).Once()
			},
			want: member,
		},
		{
			name: "list all members",
			call: func(api *API) (any, error) {
				return api.ListAllMembers(context.Background(), actor, "org-1", pagination.Params{Page: 1, Limit: 10})
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.memberSvc.On("ListAllMembers", mock.Anything, "user-1", "org-1", pagination.Params{Page: 1, Limit: 10}).Return(result, err).Once()
			},
			want: memberList,
		},
		{
			name: "get all members",
			call: func(api *API) (any, error) { return api.GetAllMembers(context.Background(), actor, "org-1") },
			arrange: func(f *apiFixture, result any, err error) {
				f.memberSvc.On("GetAllMembers", mock.Anything, actor, "org-1").Return(result, err).Once()
			},
			want: []types.OrganizationMemberResponse{},
		},
		{
			name: "get member",
			call: func(api *API) (any, error) { return api.GetMember(context.Background(), actor, "org-1", "mem-1") },
			arrange: func(f *apiFixture, result any, err error) {
				f.memberSvc.On("GetMember", mock.Anything, "user-1", "org-1", "mem-1").Return(result, err).Once()
			},
			want: memberResponse,
		},
		{
			name: "get member by user id",
			call: func(api *API) (any, error) {
				return api.GetMemberByUserID(context.Background(), actor, "org-1", "user-2")
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.memberSvc.On("GetMemberByUserID", mock.Anything, "user-1", "org-1", "user-2").Return(result, err).Once()
			},
			want: memberResponse,
		},
		{
			name: "update member",
			call: func(api *API) (any, error) {
				return api.UpdateMember(context.Background(), actor, "org-1", "mem-1", types.UpdateOrganizationMemberRequest{Role: "admin"})
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.memberSvc.On("UpdateMember", mock.Anything, "user-1", "org-1", "mem-1", types.UpdateOrganizationMemberRequest{Role: "admin"}).Return(result, err).Once()
			},
			want: member,
		},
		{
			name: "remove member",
			call: func(api *API) (any, error) {
				return nil, api.RemoveMember(context.Background(), actor, "org-1", "mem-1")
			},
			arrange: func(f *apiFixture, _ any, err error) {
				f.memberSvc.On("RemoveMember", mock.Anything, "user-1", "org-1", "mem-1").Return(err).Once()
			},
		},
		{
			name: "create team",
			call: func(api *API) (any, error) {
				return api.CreateTeam(context.Background(), actor, "org-1", types.CreateOrganizationTeamRequest{Name: "Platform"})
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.teamSvc.On("CreateTeam", mock.Anything, "user-1", "org-1", types.CreateOrganizationTeamRequest{Name: "Platform"}).Return(result, err).Once()
			},
			want: team,
		},
		{
			name: "list all teams",
			call: func(api *API) (any, error) {
				return api.ListAllTeams(context.Background(), actor, "org-1", pagination.Params{Page: 1, Limit: 10})
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.teamSvc.On("ListAllTeams", mock.Anything, "user-1", "org-1", pagination.Params{Page: 1, Limit: 10}).Return(result, err).Once()
			},
			want: teamList,
		},
		{
			name: "get all teams",
			call: func(api *API) (any, error) { return api.GetAllTeams(context.Background(), actor, "org-1") },
			arrange: func(f *apiFixture, result any, err error) {
				f.teamSvc.On("GetAllTeams", mock.Anything, actor, "org-1").Return(result, err).Once()
			},
			want: []types.OrganizationTeam{},
		},
		{
			name: "get team",
			call: func(api *API) (any, error) { return api.GetTeam(context.Background(), actor, "org-1", "team-1") },
			arrange: func(f *apiFixture, result any, err error) {
				f.teamSvc.On("GetTeam", mock.Anything, "user-1", "org-1", "team-1").Return(result, err).Once()
			},
			want: team,
		},
		{
			name: "update team",
			call: func(api *API) (any, error) {
				return api.UpdateTeam(context.Background(), actor, "org-1", "team-1", types.UpdateOrganizationTeamRequest{Name: "Platform"})
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.teamSvc.On("UpdateTeam", mock.Anything, "user-1", "org-1", "team-1", types.UpdateOrganizationTeamRequest{Name: "Platform"}).Return(result, err).Once()
			},
			want: team,
		},
		{
			name: "delete team",
			call: func(api *API) (any, error) {
				return nil, api.DeleteTeam(context.Background(), actor, "org-1", "team-1")
			},
			arrange: func(f *apiFixture, _ any, err error) {
				f.teamSvc.On("DeleteTeam", mock.Anything, "user-1", "org-1", "team-1").Return(err).Once()
			},
		},
		{
			name: "add team member",
			call: func(api *API) (any, error) {
				return api.AddTeamMember(context.Background(), actor, "org-1", "team-1", types.AddOrganizationTeamMemberRequest{MemberID: "mem-1"})
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.teamMemberSvc.On("AddTeamMember", mock.Anything, "user-1", "org-1", "team-1", types.AddOrganizationTeamMemberRequest{MemberID: "mem-1"}).Return(result, err).Once()
			},
			want: teamMember,
		},
		{
			name: "list all team members",
			call: func(api *API) (any, error) {
				return api.ListAllTeamMembers(context.Background(), actor, "org-1", "team-1", pagination.Params{Page: 1, Limit: 10})
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.teamMemberSvc.On("ListAllTeamMembers", mock.Anything, "user-1", "org-1", "team-1", pagination.Params{Page: 1, Limit: 10}).Return(result, err).Once()
			},
			want: teamMemberList,
		},
		{
			name: "get all team members",
			call: func(api *API) (any, error) {
				return api.GetAllTeamMembers(context.Background(), actor, "org-1", "team-1")
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.teamMemberSvc.On("GetAllTeamMembers", mock.Anything, actor, "org-1", "team-1").Return(result, err).Once()
			},
			want: []types.OrganizationTeamMemberResponse{},
		},
		{
			name: "get team member",
			call: func(api *API) (any, error) {
				return api.GetTeamMember(context.Background(), actor, "org-1", "team-1", "mem-1")
			},
			arrange: func(f *apiFixture, result any, err error) {
				f.teamMemberSvc.On("GetTeamMember", mock.Anything, "user-1", "org-1", "team-1", "mem-1").Return(result, err).Once()
			},
			want: teamMemberResponse,
		},
		{
			name: "remove team member",
			call: func(api *API) (any, error) {
				return nil, api.RemoveTeamMember(context.Background(), actor, "org-1", "team-1", "mem-1")
			},
			arrange: func(f *apiFixture, _ any, err error) {
				f.teamMemberSvc.On("RemoveTeamMember", mock.Anything, "user-1", "org-1", "team-1", "mem-1").Return(err).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			t.Run("success delegates and returns the service result", func(t *testing.T) {
				t.Parallel()

				fixture := newAPIFixture()
				tt.arrange(fixture, tt.want, nil)
				got, err := tt.call(fixture.api)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
				fixture.assertExpectations(t)
			})

			t.Run("service error is propagated", func(t *testing.T) {
				t.Parallel()

				fixture := newAPIFixture()
				tt.arrange(fixture, nil, svcErr)
				got, err := tt.call(fixture.api)
				require.ErrorIs(t, err, svcErr)
				require.Empty(t, got)
				fixture.assertExpectations(t)
			})
		})
	}
}

func TestAPI_GetInvitation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	actor := orgtests.Actor("user-1")

	t.Run("invitation from another organization is not found", func(t *testing.T) {
		t.Parallel()

		fixture := newAPIFixture()
		fixture.invSvc.On("GetOrganizationInvitationByIDWithOrg", mock.Anything, "inv-1").
			Return(&types.GetOrganizationInvitationResponse{Invitation: &types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-2"}}, nil).Once()

		got, err := fixture.api.GetInvitation(ctx, actor, "org-1", "inv-1")
		require.ErrorIs(t, err, coreerrors.ErrNotFound)
		require.Nil(t, got)
		fixture.assertExpectations(t)
	})

	t.Run("invitation in the requested organization is returned", func(t *testing.T) {
		t.Parallel()

		response := &types.GetOrganizationInvitationResponse{Invitation: &types.OrganizationInvitation{ID: "inv-1", OrganizationID: "org-1"}}
		fixture := newAPIFixture()
		fixture.invSvc.On("GetOrganizationInvitationByIDWithOrg", mock.Anything, "inv-1").Return(response, nil).Once()

		got, err := fixture.api.GetInvitation(ctx, actor, "org-1", "inv-1")
		require.NoError(t, err)
		require.Equal(t, response, got)
		fixture.assertExpectations(t)
	})
}

func TestAPI_GetUserPermissionsInOrganizationRoleLookupError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("access control failure")
	fixture := newAPIFixture()
	fixture.orgSvc.On("ExistsByID", mock.Anything, "org-1").Return(true, nil).Once()
	fixture.memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").
		Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "ghost"}, nil).Once()
	fixture.accessControl.Err = repoErr

	perms, err := fixture.api.GetUserPermissionsInOrganization(context.Background(), "user-1", "org-1")
	require.ErrorIs(t, err, repoErr)
	require.Nil(t, perms)
	fixture.assertExpectations(t)
}
