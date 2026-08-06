package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
)

func newTestOrganizationTeamService(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamRepo *orgtests.MockOrganizationTeamRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) *organizationTeamService {
	serviceUtils := &ServiceUtils{
		orgRepo:           orgRepo,
		orgMemberRepo:     memberRepo,
		orgTeamRepo:       teamRepo,
		orgTeamMemberRepo: teamMemberRepo,
	}

	return NewOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo, serviceUtils, nil)
}

func TestOrganizationTeamService_CreateTeam(t *testing.T) {
	t.Parallel()

	txRunner := &orgtests.MockTxRunner{}
	repoErr := errors.New("repository error")

	tests := []struct {
		name             string
		actorUserID      string
		organizationID   string
		request          types.CreateOrganizationTeamRequest
		setup            func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationTeamMemberRepository, *orgtests.MockOrganizationTeamHooks, *ServiceUtils)
		expectErr        error
		expectCalled     bool
		expectTeamMember bool
	}{
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.CreateOrganizationTeamRequest{Name: "Acme Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "acme-platform").Return(nil, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("Create", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					team := args.Get(1).(*types.OrganizationTeam)
					require.NotNil(t, team)
					require.Equal(t, "org-1", team.OrganizationID)
					require.Equal(t, "Acme Platform", team.Name)
					require.Equal(t, "acme-platform", team.Slug)
					require.Equal(t, map[string]any{}, team.Metadata)
				}).Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Acme Platform", Slug: "acme-platform", Metadata: map[string]any{}}, nil).Once()
				teamMemberRepo.On("Create", mock.Anything, mock.MatchedBy(func(teamMember *types.OrganizationTeamMember) bool {
					return teamMember != nil && teamMember.TeamID == "team-1" && teamMember.MemberID == "mem-1"
				})).Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectCalled:     true,
			expectTeamMember: true,
		},
		{
			name:           "org member can create",
			actorUserID:    "user-2",
			organizationID: "org-1",
			request:        types.CreateOrganizationTeamRequest{Name: "Acme Platform", Metadata: map[string]any{"tier": "core"}},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "acme-platform").Return(nil, nil).Once()
				teamRepo.On("Create", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					team := args.Get(1).(*types.OrganizationTeam)
					require.NotNil(t, team)
					require.Equal(t, "org-1", team.OrganizationID)
					require.Equal(t, "Acme Platform", team.Name)
					require.Equal(t, "acme-platform", team.Slug)
					require.Equal(t, map[string]any{"tier": "core"}, team.Metadata)
				}).Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Acme Platform", Slug: "acme-platform", Metadata: map[string]any{"tier": "core"}}, nil).Once()
				teamMemberRepo.On("Create", mock.Anything, mock.MatchedBy(func(teamMember *types.OrganizationTeamMember) bool {
					return teamMember != nil && teamMember.TeamID == "team-1" && teamMember.MemberID == "mem-1"
				})).Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectCalled:     true,
			expectTeamMember: true,
		},
	}

	tests = append(tests, []struct {
		name             string
		actorUserID      string
		organizationID   string
		request          types.CreateOrganizationTeamRequest
		setup            func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationTeamMemberRepository, *orgtests.MockOrganizationTeamHooks, *ServiceUtils)
		expectErr        error
		expectCalled     bool
		expectTeamMember bool
	}{
		{name: "unauthorized", actorUserID: "", organizationID: "org-1", request: types.CreateOrganizationTeamRequest{Name: "Platform"}, expectErr: coreerrors.ErrUnauthorized},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.CreateOrganizationTeamRequest{Name: "Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "organization lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.CreateOrganizationTeamRequest{Name: "Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return((*types.Organization)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.CreateOrganizationTeamRequest{Name: "Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "bad request empty name",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.CreateOrganizationTeamRequest{Name: ""},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
			},
			expectErr: coreerrors.ErrBadRequest,
		},
		{
			name:           "bad request empty slugify result",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.CreateOrganizationTeamRequest{Name: "!!!"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
			},
			expectErr: coreerrors.ErrBadRequest,
		},
		{
			name:           "slug lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.CreateOrganizationTeamRequest{Name: "Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "platform").Return((*types.OrganizationTeam)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "slug conflict",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.CreateOrganizationTeamRequest{Name: "Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "platform").Return(&types.OrganizationTeam{ID: "team-2"}, nil).Once()
			},
			expectErr: coreerrors.ErrConflict,
		},
		{
			name:           "create error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			request:        types.CreateOrganizationTeamRequest{Name: "Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, serviceUtils *ServiceUtils) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "platform").Return(nil, nil).Once()
				teamRepo.On("Create", mock.Anything, mock.MatchedBy(func(team *types.OrganizationTeam) bool {
					return team != nil && team.OrganizationID == "org-1" && team.Name == "Platform" && team.Slug == "platform"
				})).Return((*types.OrganizationTeam)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
	}...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			teamRepo := &orgtests.MockOrganizationTeamRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
			hooks := &orgtests.MockOrganizationTeamHooks{}
			serviceUtils := &ServiceUtils{
				orgRepo:           orgRepo,
				orgMemberRepo:     memberRepo,
				orgTeamRepo:       teamRepo,
				orgTeamMemberRepo: teamMemberRepo,
			}

			if tt.setup != nil {
				tt.setup(orgRepo, teamRepo, memberRepo, teamMemberRepo, hooks, serviceUtils)
			}

			svc := NewOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo, serviceUtils, txRunner)
			team, err := svc.CreateTeam(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.request)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, team)
			require.Equal(t, tt.expectCalled, orgRepo.AssertExpectations(t))
			require.Equal(t, tt.expectCalled, memberRepo.AssertExpectations(t))
			require.Equal(t, tt.expectCalled, teamRepo.AssertExpectations(t))
			if tt.expectTeamMember {
				require.True(t, teamMemberRepo.AssertExpectations(t))
			} else {
				teamMemberRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestOrganizationTeamService_GetAllTeams(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr      error
		expectLen      int
	}{
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetAllByOrganizationID", mock.Anything, "org-1").Return([]types.OrganizationTeam{{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}}, nil).Once()
			},
			expectLen: 1,
		},
		{
			name:           "org member can list",
			actorUserID:    "user-2",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				teamRepo.On("GetAllByOrganizationID", mock.Anything, "org-1").Return([]types.OrganizationTeam{{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}}, nil).Once()
			},
			expectLen: 1,
		},
	}

	tests = append(tests, []struct {
		name           string
		actorUserID    string
		organizationID string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository)
		expectErr      error
		expectLen      int
	}{
		{name: "unauthorized", actorUserID: "", organizationID: "org-1", expectErr: coreerrors.ErrUnauthorized},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "organization lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return((*types.Organization)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "repo error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Once()
				teamRepo.On("GetAllByOrganizationID", mock.Anything, "org-1").Return(([]types.OrganizationTeam)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
	}...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			teamRepo := &orgtests.MockOrganizationTeamRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			if tt.setup != nil {
				tt.setup(orgRepo, teamRepo, memberRepo)
			}

			svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, &orgtests.MockOrganizationTeamMemberRepository{})
			teams, err := svc.GetAllTeams(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, teams, tt.expectLen)
		})
	}
}

func TestOrganizationTeamService_GetTeam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		teamID         string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationTeamMemberRepository)
		expectErr      error
		expectTeamID   string
	}{
		{
			name:           "owner can get team",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectTeamID: "team-1",
		},
		{
			name:           "org member can get team",
			actorUserID:    "user-2",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectTeamID: "team-1",
		},
		{
			name:           "non member forbidden",
			actorUserID:    "user-2",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			teamRepo := &orgtests.MockOrganizationTeamRepository{}
			teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
			if tt.setup != nil {
				tt.setup(orgRepo, teamRepo, memberRepo, teamMemberRepo)
			}

			svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
			team, err := svc.GetTeam(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.teamID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, team)
			require.Equal(t, tt.expectTeamID, team.ID)
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, teamRepo.AssertExpectations(t))
			require.True(t, memberRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationTeamService_UpdateTeam(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")
	updateErr := errors.New("update error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		teamID         string
		request        types.UpdateOrganizationTeamRequest
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationTeamHooks, *orgtests.MockOrganizationTeamMemberRepository)
		expectErr      error
	}{
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: "Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: "Platform Revamp"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "platform").Return(nil, nil).Once()
				teamRepo.On("Update", mock.Anything, mock.MatchedBy(func(team *types.OrganizationTeam) bool {
					return team != nil && team.ID == "team-1" && team.Name == "Platform Revamp" && team.Slug == "platform"
				})).Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform Revamp", Slug: "platform"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
		},
		{
			name:           "org member can update",
			actorUserID:    "user-2",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: "Platform Revamp"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "platform").Return(nil, nil).Once()
				teamRepo.On("Update", mock.Anything, mock.MatchedBy(func(team *types.OrganizationTeam) bool {
					return team != nil && team.ID == "team-1" && team.Name == "Platform Revamp" && team.Slug == "platform"
				})).Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform Revamp", Slug: "platform"}, nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
		},
	}

	tests = append(tests, []struct {
		name           string
		actorUserID    string
		organizationID string
		teamID         string
		request        types.UpdateOrganizationTeamRequest
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationTeamHooks, *orgtests.MockOrganizationTeamMemberRepository)
		expectErr      error
	}{
		{name: "unauthorized", actorUserID: "", organizationID: "org-1", teamID: "team-1", request: types.UpdateOrganizationTeamRequest{Name: "Platform"}, expectErr: coreerrors.ErrUnauthorized},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: "Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "organization lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: "Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return((*types.Organization)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: "Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "team lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: "Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return((*types.OrganizationTeam)(nil), repoErr).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "team not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: "Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(nil, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "team from another organization",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: "Platform"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-2", Name: "Platform", Slug: "platform"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "bad request empty name",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: ""},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrBadRequest,
		},
		{
			name:           "bad request empty slugify result",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: "!!!", Slug: func() *string { value := ""; return &value }()},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrBadRequest,
		},
		{
			name:           "slug lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: "Platform", Slug: func() *string { value := "platform"; return &value }()},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "platform").Return((*types.OrganizationTeam)(nil), repoErr).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "slug conflict",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: "Platform", Slug: func() *string { value := "platform"; return &value }()},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "platform").Return(&types.OrganizationTeam{ID: "team-2"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrConflict,
		},
		{
			name:           "update error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			request:        types.UpdateOrganizationTeamRequest{Name: "Platform Revamp"},
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "platform").Return(nil, nil).Once()
				teamRepo.On("Update", mock.Anything, mock.MatchedBy(func(team *types.OrganizationTeam) bool {
					return team != nil && team.ID == "team-1" && team.Name == "Platform Revamp" && team.Slug == "platform"
				})).Return((*types.OrganizationTeam)(nil), updateErr).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectErr: updateErr,
		},
	}...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			teamRepo := &orgtests.MockOrganizationTeamRepository{}
			teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
			hooks := &orgtests.MockOrganizationTeamHooks{}
			if tt.setup != nil {
				tt.setup(orgRepo, teamRepo, memberRepo, hooks, teamMemberRepo)
			}

			svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
			team, err := svc.UpdateTeam(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.teamID, tt.request)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, team)
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, teamRepo.AssertExpectations(t))
		})
	}
}

func TestOrganizationTeamService_DeleteTeam(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository error")

	tests := []struct {
		name           string
		actorUserID    string
		organizationID string
		teamID         string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationTeamHooks, *orgtests.MockOrganizationTeamMemberRepository)
		expectErr      error
	}{
		{
			name:           "success",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				teamRepo.On("Delete", mock.Anything, "team-1").Return(nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
		},
		{
			name:           "org member can delete",
			actorUserID:    "user-2",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-2").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				teamRepo.On("Delete", mock.Anything, "team-1").Return(nil).Once()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
		},
	}

	tests = append(tests, []struct {
		name           string
		actorUserID    string
		organizationID string
		teamID         string
		setup          func(*orgtests.MockOrganizationRepository, *orgtests.MockOrganizationTeamRepository, *orgtests.MockOrganizationMemberRepository, *orgtests.MockOrganizationTeamHooks, *orgtests.MockOrganizationTeamMemberRepository)
		expectErr      error
	}{
		{name: "unauthorized", actorUserID: "", organizationID: "org-1", teamID: "team-1", expectErr: coreerrors.ErrUnauthorized},
		{
			name:           "organization not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "organization lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return((*types.Organization)(nil), repoErr).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "forbidden",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "owner-1"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(nil, nil).Once()
			},
			expectErr: coreerrors.ErrForbidden,
		},
		{
			name:           "team lookup error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return((*types.OrganizationTeam)(nil), repoErr).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectErr: repoErr,
		},
		{
			name:           "team not found",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(nil, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "team from another organization",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-2", Name: "Platform", Slug: "platform"}, nil).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectErr: coreerrors.ErrNotFound,
		},
		{
			name:           "delete error",
			actorUserID:    "user-1",
			organizationID: "org-1",
			teamID:         "team-1",
			setup: func(orgRepo *orgtests.MockOrganizationRepository, teamRepo *orgtests.MockOrganizationTeamRepository, memberRepo *orgtests.MockOrganizationMemberRepository, hooks *orgtests.MockOrganizationTeamHooks, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository) {
				orgRepo.On("GetByID", mock.Anything, "org-1").Return(&types.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				teamRepo.On("GetByID", mock.Anything, "team-1").Return(&types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"}, nil).Twice()
				teamRepo.On("Delete", mock.Anything, "team-1").Return(repoErr).Once()
				memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").Return(&types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).Twice()
				teamMemberRepo.On("GetByTeamIDAndMemberID", mock.Anything, "team-1", "mem-1").Return(&types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "mem-1"}, nil).Once()
			},
			expectErr: repoErr,
		},
	}...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgRepo := &orgtests.MockOrganizationRepository{}
			teamRepo := &orgtests.MockOrganizationTeamRepository{}
			memberRepo := &orgtests.MockOrganizationMemberRepository{}
			teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
			hooks := &orgtests.MockOrganizationTeamHooks{}
			if tt.setup != nil {
				tt.setup(orgRepo, teamRepo, memberRepo, hooks, teamMemberRepo)
			}

			svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
			err := svc.DeleteTeam(context.Background(), orgtests.Actor(tt.actorUserID), tt.organizationID, tt.teamID)
			if tt.expectErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			require.True(t, orgRepo.AssertExpectations(t))
			require.True(t, teamRepo.AssertExpectations(t))
		})
	}
}
