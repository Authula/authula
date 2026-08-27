package services

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

func newTestOrganizationTeamMemberService(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamRepo *orgtests.MockOrganizationTeamRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) *organizationTeamMemberService {
	serviceUtils := &ServiceUtils{
		orgRepo:       orgRepo,
		orgMemberRepo: memberRepo,
		orgTeamRepo:   teamRepo,
	}

	return NewOrganizationTeamMemberService(orgRepo, memberRepo, teamRepo, teamMemberRepo, serviceUtils)
}

func TestOrganizationTeamService_ListAllTeamMembers(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		teamID         string
		params         pagination.Params
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationTeamMemberRepository)
		expectErr      error
		expectLen      int
		expectCalled   bool
	}{
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				teamMemberRepo.On("ListAllByTeamIDWithMemberAndUser", mock.Anything, "team-1", 1, 10).Return([]types.OrganizationTeamMemberResponse{{ID: "tm-1", TeamID: "team-1"}}, 1, nil).Once()
			},
			expectLen:    1,
			expectCalled: true,
		},
		{
			name:           "org member can list",
			actorUserID:    "user-2",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "org-member-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				teamMemberRepo.On("ListAllByTeamIDWithMemberAndUser", mock.Anything, "team-1", 1, 10).Return([]types.OrganizationTeamMemberResponse{{ID: "tm-1", TeamID: "team-1"}}, 1, nil).Once()
			},
			expectLen:    1,
			expectCalled: true,
		},
		{
			name:           "unauthorized",
			actorUserID:    "",
			organizationID: "org-1",
			teamID:         "team-1",
			expectErr:      coreerrors.ErrUnauthorized,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrForbidden,
			expectCalled: true,
		},
		{
			name:           "team lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return((*types.OrganizationTeam)(nil), repoErr).Once()
			},
			expectErr:    repoErr,
			expectCalled: true,
		},
		{
			name:           "team not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "team from another organization",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-2"}, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "repo error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				teamMemberRepo.On("ListAllByTeamIDWithMemberAndUser", mock.Anything, "team-1", 1, 10).Return(([]types.OrganizationTeamMemberResponse)(nil), 0, repoErr).Once()
			},
			expectErr:    repoErr,
			expectCalled: true,
		},
		{
			name:           "a negative page and an oversized limit are clamped before reaching the repository",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			params:         pagination.Params{Page: -4, Limit: 5000},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				teamMemberRepo.On("ListAllByTeamIDWithMemberAndUser", mock.Anything, "team-1", 1, pagination.DefaultMaxLimit).Return(([]types.OrganizationTeamMemberResponse)(nil), 0, nil).Once()
			},
			expectLen:    0,
			expectCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			teamRepo := &orgtests.MockOrganizationTeamRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
			if tt.setup != nil {
				tt.setup(orgRepo, teamRepo, memberRepo, teamMemberRepo)
			}

			params := tt.params
			if params == (pagination.Params{}) {
				params = pagination.Params{Page: 1, Limit: 10}
			}

			svc := newTestOrganizationTeamMemberService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
			resp, err := svc.ListAllTeamMembers(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.teamID, params)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				require.True(t, orgRepo.AssertExpectations(t))
				require.True(t, memberRepo.AssertExpectations(t))
				require.True(t, teamRepo.AssertExpectations(t))
				require.True(t, teamMemberRepo.AssertExpectations(t))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, resp.Data)
			require.Len(t, resp.Data, tt.expectLen)
			require.Equal(t, svc.serviceUtils.ClampPagination(params).Page, resp.Pagination.Page)
			require.Equal(t, svc.serviceUtils.ClampPagination(params).Limit, resp.Pagination.Limit)
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
			require.True(t, teamRepo.AssertExpectations(t))
			require.True(t, teamMemberRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationTeamService_AddTeamMember(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")
	createErr := errors.New("create error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		teamID         string
		request        types.AddOrganizationTeamMemberRequest
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationTeamMemberRepository, *orgtests.MockOrganizationTeamMemberHooks, *ServiceUtils)
		expectErr      error
		expectCalled   bool
	}{
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				memberRepo.On("GetByID", mock.Anything, "member-1").Return(&types.OrganizationMember{ID: "member-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return(nil, nil).Once()
				teamMemberRepo.On("Create", mock.Anything, mock.MatchedBy(func(teamMember *types.OrganizationTeamMember) bool {
					return teamMember != nil && teamMember.TeamID == "team-1" && teamMember.MemberID == "member-1"
				})).Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "member-1"}, nil).Once()
			},
			expectCalled: true,
		},
		{
			name:           "org member can add",
			actorUserID:    "user-2",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "org-member-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				memberRepo.On("GetByID", mock.Anything, "member-1").Return(&types.OrganizationMember{ID: "member-1", OrganizationID: "org-1", UserID: "user-3", Role: "member"}, nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return(nil, nil).Once()
				teamMemberRepo.On("Create", mock.Anything, mock.MatchedBy(func(teamMember *types.OrganizationTeamMember) bool {
					return teamMember != nil && teamMember.TeamID == "team-1" && teamMember.MemberID == "member-1"
				})).Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "member-1"}, nil).Once()
			},
			expectCalled: true,
		},
		{
			name:           "unauthorized",
			actorUserID:    "",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			expectErr:      coreerrors.ErrUnauthorized,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "organization lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return((*types.Organization)(nil), repoErr).Once()
			},
			expectErr:    repoErr,
			expectCalled: true,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrForbidden,
			expectCalled: true,
		},
		{
			name:           "team lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return((*types.OrganizationTeam)(nil), repoErr).Once()
			},
			expectErr:    repoErr,
			expectCalled: true,
		},
		{
			name:           "team not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "team from another organization",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-2", Name: "Platform", Slug: "platform"}, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "member from another organization",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				memberRepo.On("GetByID", mock.Anything, "member-1").Return(&types.OrganizationMember{ID: "member-1", OrganizationID: "org-2", UserID: "user-2", Role: "member"}, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "bad request empty member id",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: ""},
			expectErr:      coreerrors.ErrUnprocessableEntity,
			expectCalled:   true,
		},
		{
			name:           "member lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				memberRepo.On("GetByID", mock.Anything, "member-1").Return((*types.OrganizationMember)(nil), repoErr).Once()
			},
			expectErr:    repoErr,
			expectCalled: true,
		},
		{
			name:           "member not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				memberRepo.On("GetByID", mock.Anything, "member-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "team member conflict",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				memberRepo.On("GetByID", mock.Anything, "member-1").Return(&types.OrganizationMember{ID: "member-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return(&types.OrganizationTeamMember{ID: "team-member-1"}, nil).Once()
			},
			expectErr:    coreerrors.ErrConflict,
			expectCalled: true,
		},
		{
			name:           "team member lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				memberRepo.On("GetByID", mock.Anything, "member-1").Return(&types.OrganizationMember{ID: "member-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return((*types.OrganizationTeamMember)(nil), repoErr).Once()
			},
			expectErr:    repoErr,
			expectCalled: true,
		},
		{
			name:           "create error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.AddOrganizationTeamMemberRequest{MemberID: "member-1"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamMemberHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				memberRepo.On("GetByID", mock.Anything, "member-1").Return(&types.OrganizationMember{ID: "member-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return(nil, nil).Once()
				teamMemberRepo.On("Create", mock.Anything, mock.MatchedBy(func(teamMember *types.OrganizationTeamMember) bool {
					return teamMember != nil && teamMember.TeamID == "team-1" && teamMember.MemberID == "member-1"
				})).Return((*types.OrganizationTeamMember)(nil), createErr).Once()
			},
			expectErr:    createErr,
			expectCalled: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			teamRepo := &orgtests.MockOrganizationTeamRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
			hooks := &orgtests.MockOrganizationTeamMemberHooks{}
			serviceUtils := &ServiceUtils{
				orgRepo:       orgRepo,
				orgMemberRepo: memberRepo,
				orgTeamRepo:   teamRepo,
			}

			if tt.setup != nil {
				tt.setup(orgRepo, teamRepo, memberRepo, teamMemberRepo, hooks, serviceUtils)
			}

			svc := NewOrganizationTeamMemberService(orgRepo, memberRepo, teamRepo, teamMemberRepo, serviceUtils)
			teamMember, err := svc.AddTeamMember(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.teamID, tt.request)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				require.True(t, orgRepo.AssertExpectations(t))
				require.True(t, teamRepo.AssertExpectations(t))
				require.True(t, memberRepo.AssertExpectations(t))
				require.True(t, teamMemberRepo.AssertExpectations(t))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, teamMember)
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, teamRepo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
			require.True(t, teamMemberRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationTeamService_GetTeamMember(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		teamID         string
		memberID       string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationTeamMemberRepository)
		expectErr      error
		expectID       string
		expectCalled   bool
	}{
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				memberRepo.On("GetByIDWithUser", mock.Anything, "member-1").Return(&types.OrganizationMemberResponse{ID: "member-1", OrganizationID: "org-1", Role: "member"}, nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return(&types.OrganizationTeamMember{ID: "tm-1", TeamID: "team-1", MemberID: "member-1"}, nil).Once()
			},
			expectID:     "tm-1",
			expectCalled: true,
		},
		{
			name:           "org member can get",
			actorUserID:    "user-2",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "org-member-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				memberRepo.On("GetByIDWithUser", mock.Anything, "member-1").Return(&types.OrganizationMemberResponse{ID: "member-1", OrganizationID: "org-1", Role: "member"}, nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return(&types.OrganizationTeamMember{ID: "tm-1", TeamID: "team-1", MemberID: "member-1"}, nil).Once()
			},
			expectID:     "tm-1",
			expectCalled: true,
		},
		{
			name:           "unauthorized",
			actorUserID:    "",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			expectErr:      coreerrors.ErrUnauthorized,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrForbidden,
			expectCalled: true,
		},
		{
			name:           "team lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return((*types.OrganizationTeam)(nil), repoErr).Once()
			},
			expectErr:    repoErr,
			expectCalled: true,
		},
		{
			name:           "team not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "team from another organization",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-2"}, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "repo error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				memberRepo.On("GetByIDWithUser", mock.Anything, "member-1").Return(&types.OrganizationMemberResponse{ID: "member-1", OrganizationID: "org-1", Role: "member"}, nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return((*types.OrganizationTeamMember)(nil), repoErr).Once()
			},
			expectErr:    repoErr,
			expectCalled: true,
		},
		{
			name:           "not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				memberRepo.On("GetByIDWithUser", mock.Anything, "member-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "whitespace member id",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				memberRepo.On("GetByIDWithUser", mock.Anything, "").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			teamRepo := &orgtests.MockOrganizationTeamRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
			if tt.setup != nil {
				tt.setup(orgRepo, teamRepo, memberRepo, teamMemberRepo)
			}

			svc := newTestOrganizationTeamMemberService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
			teamMember, err := svc.GetTeamMember(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.teamID, tt.memberID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				require.True(t, orgRepo.AssertExpectations(t))
				require.True(t, memberRepo.AssertExpectations(t))
				require.True(t, teamRepo.AssertExpectations(t))
				require.True(t, teamMemberRepo.AssertExpectations(t))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, teamMember)
			require.Equal(t, tt.expectID, teamMember.ID)
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
			require.True(t, teamRepo.AssertExpectations(t))
			require.True(t, teamMemberRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationTeamService_RemoveTeamMember(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")
	deleteErr := errors.New("delete error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		teamID         string
		memberID       string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationTeamMemberRepository)
		expectErr      error
		expectCalled   bool
	}{
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				memberRepo.On("GetByID", mock.Anything, "member-1").Return(&types.OrganizationMember{ID: "member-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return(&types.OrganizationTeamMember{ID: "tm-1", TeamID: "team-1", MemberID: "member-1"}, nil).Once()
				teamMemberRepo.On("DeleteByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return(nil).Once()
			},
			expectCalled: true,
		},
		{
			name:           "org member can remove",
			actorUserID:    "user-2",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "org-member-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				memberRepo.On("GetByID", mock.Anything, "member-1").Return(&types.OrganizationMember{ID: "member-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return(&types.OrganizationTeamMember{ID: "tm-1", TeamID: "team-1", MemberID: "member-1"}, nil).Once()
				teamMemberRepo.On("DeleteByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return(nil).Once()
			},
			expectCalled: true,
		},
		{
			name:           "unauthorized",
			actorUserID:    "",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			expectErr:      coreerrors.ErrUnauthorized,
		},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrForbidden,
			expectCalled: true,
		},
		{
			name:           "team lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return((*types.OrganizationTeam)(nil), repoErr).Once()
			},
			expectErr:    repoErr,
			expectCalled: true,
		},
		{
			name:           "team not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "team from another organization",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-2"}, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "team member lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				memberRepo.On("GetByID", mock.Anything, "member-1").Return(&types.OrganizationMember{ID: "member-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return((*types.OrganizationTeamMember)(nil), repoErr).Once()
			},
			expectErr:    repoErr,
			expectCalled: true,
		},
		{
			name:           "member not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				memberRepo.On("GetByID", mock.Anything, "member-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "delete error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			memberID:       "member-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				memberRepo.On("GetByID", mock.Anything, "member-1").Return(&types.OrganizationMember{ID: "member-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return(&types.OrganizationTeamMember{ID: "tm-1", TeamID: "team-1", MemberID: "member-1"}, nil).Once()
				teamMemberRepo.On("DeleteByTeamIDAndMemberID", mock.Anything, "team-1", "member-1").Return(deleteErr).Once()
			},
			expectErr:    deleteErr,
			expectCalled: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			teamRepo := &orgtests.MockOrganizationTeamRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
			if tt.setup != nil {
				tt.setup(orgRepo, teamRepo, memberRepo, teamMemberRepo)
			}

			svc := newTestOrganizationTeamMemberService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
			err := svc.RemoveTeamMember(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.teamID, tt.memberID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				require.True(t, orgRepo.AssertExpectations(t))
				require.True(t, memberRepo.AssertExpectations(t))
				require.True(t, teamRepo.AssertExpectations(t))
				require.True(t, teamMemberRepo.AssertExpectations(t))
				return
			}
			require.NoError(t, err)
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
			require.True(t, teamRepo.AssertExpectations(t))
			require.True(t, teamMemberRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationTeamMemberService_GetAllTeamMembers(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		teamID         string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationTeamMemberRepository)
		expectErr      error
		expectIDs      []string
		expectCalled   bool
	}{
		{
			name:           "success returns every team member",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				teamMemberRepo.On("GetAllByTeamIDWithMemberAndUser", mock.Anything, "team-1").Return([]types.OrganizationTeamMemberResponse{
					{ID: "tm-1", TeamID: "team-1"},
					{ID: "tm-2", TeamID: "team-1"},
				}, nil).Once()
			},
			expectIDs:    []string{"tm-1", "tm-2"},
			expectCalled: true,
		},
		{
			name:           "unauthorized",
			actorUserID:    "",
			organizationID: "org-1",
			teamID:         "team-1",
			expectErr:      coreerrors.ErrUnauthorized,
		},
		{
			name:           "forbidden when the actor is not an organization member",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrForbidden,
			expectCalled: true,
		},
		{
			name:           "team not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(nil, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "team belonging to another organization is not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-2"}, nil).Once()
			},
			expectErr:    coreerrors.ErrNotFound,
			expectCalled: true,
		},
		{
			name:           "repository error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				teamMemberRepo.On("GetAllByTeamIDWithMemberAndUser", mock.Anything, "team-1").Return(([]types.OrganizationTeamMemberResponse)(nil), repoErr).Once()
			},
			expectErr:    repoErr,
			expectCalled: true,
		},
		{
			name:           "nil result is normalised to an empty slice",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "owner-member-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				teamMemberRepo.On("GetAllByTeamIDWithMemberAndUser", mock.Anything, "team-1").Return(([]types.OrganizationTeamMemberResponse)(nil), nil).Once()
			},
			expectIDs:    []string{},
			expectCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			teamRepo := &orgtests.MockOrganizationTeamRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
			if tt.setup != nil {
				tt.setup(orgRepo, teamRepo, memberRepo, teamMemberRepo)
			}

			svc := newTestOrganizationTeamMemberService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
			teamMembers, err := svc.GetAllTeamMembers(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.teamID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				require.Nil(t, teamMembers)
			} else {
				require.NoError(t, err)
				require.NotNil(t, teamMembers)
				ids := make([]string, 0, len(teamMembers))
				for _, teamMember := range teamMembers {
					ids = append(ids, teamMember.ID)
				}
				require.Equal(t, tt.expectIDs, ids)
			}

			if tt.expectCalled {
				require.True(t, orgRepo.AssertExpectations(t))
				require.True(t, teamRepo.AssertExpectations(t))
				require.True(t, memberRepo.AssertExpectations(t))
				require.True(t, teamMemberRepo.AssertExpectations(t))
			}
		})
	}
}
