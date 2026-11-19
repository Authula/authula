package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	orgtypes "github.com/Authula/authula/plugins/organizations/types"
)

func newTeamHooksExecutor(configure func(*orgtypes.OrganizationTeamServiceHooks)) *ServiceHookExecutor {
	hooks := &orgtypes.OrganizationTeamServiceHooks{}
	if configure != nil {
		configure(hooks)
	}
	return NewServiceHookExecutor(&orgtypes.OrganizationsServiceHooksConfig{Teams: hooks}, nil, nil)
}

type teamHooksFixture struct {
	orgRepo        *orgtests.MockOrganizationRepository
	memberRepo     *orgtests.MockOrganizationMemberRepository
	teamRepo       *orgtests.MockOrganizationTeamRepository
	teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository
	executor       *ServiceHookExecutor
	txRunner       organizationTeamTxRunner
	beforeSlug     string
	afterSlug      string
}

func newTeamHooksFixture() *teamHooksFixture {
	return &teamHooksFixture{
		orgRepo:        &orgtests.MockOrganizationRepository{},
		memberRepo:     &orgtests.MockOrganizationMemberRepository{},
		teamRepo:       &orgtests.MockOrganizationTeamRepository{},
		teamMemberRepo: &orgtests.MockOrganizationTeamMemberRepository{},
	}
}

func (f *teamHooksFixture) service() *organizationTeamService {
	serviceUtils := &ServiceUtils{orgRepo: f.orgRepo, orgMemberRepo: f.memberRepo, orgTeamRepo: f.teamRepo}
	return NewOrganizationTeamService(f.orgRepo, f.memberRepo, f.teamRepo, f.teamMemberRepo, serviceUtils, f.txRunner, f.executor)
}

func expectTeamOrganization(orgRepo *orgtests.MockOrganizationRepository) {
	orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
}

func expectTeamMemberAccess(memberRepo *orgtests.MockOrganizationMemberRepository, times int) {
	memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "user-1").
		Return(&orgtypes.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "owner"}, nil).
		Times(times)
}

func TestOrganizationTeamService_CreateTeamExtra(t *testing.T) {
	t.Parallel()

	errBeforeHook := errors.New("before create hook failed")
	errAfterHook := errors.New("after create hook failed")
	errActorLookup := errors.New("actor member lookup failed")
	errTeamMemberCreate := errors.New("team member create failed")
	errTeamCreate := errors.New("team create failed")

	expectCreateBase := func(f *teamHooksFixture, slug string) {
		expectTeamOrganization(f.orgRepo)
		expectTeamMemberAccess(f.memberRepo, 1)
		f.teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", slug).Return(nil, nil).Once()
	}
	expectCreatedTeam := func(f *teamHooksFixture, id string) {
		f.teamRepo.On("Create", mock.Anything, mock.Anything).
			Return(&orgtypes.OrganizationTeam{ID: id, OrganizationID: "org-1"}, nil).
			Once()
	}
	expectTeamMemberCreated := func(f *teamHooksFixture, memberID string) {
		f.teamMemberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *orgtypes.OrganizationTeamMember) bool {
			return member != nil && member.MemberID == memberID
		})).Return(&orgtypes.OrganizationTeamMember{ID: "team-member-1"}, nil).Once()
	}

	tests := []struct {
		name   string
		slug   *string
		actor  *models.Actor
		setup  func(*teamHooksFixture)
		assert func(*testing.T, *teamHooksFixture, *orgtypes.OrganizationTeam, error)
	}{
		{
			name: "constructor wires the hook executor and an explicit slug is used",
			slug: new("custom-slug"),
			setup: func(f *teamHooksFixture) {
				expectCreateBase(f, "custom-slug")
				f.teamRepo.On("Create", mock.Anything, mock.MatchedBy(func(team *orgtypes.OrganizationTeam) bool {
					return team != nil && team.Slug == "custom-slug"
				})).Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Slug: "custom-slug"}, nil).Once()
				expectTeamMemberCreated(f, "mem-1")
				f.txRunner = &orgtests.MockTxRunner{}
				f.executor = newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
					hooks.RegisterBeforeCreate(func(_ context.Context, _ *models.Actor, team *orgtypes.OrganizationTeam) error {
						f.beforeSlug = team.Slug
						return nil
					})
					hooks.RegisterAfterCreate(func(_ context.Context, _ *models.Actor, team *orgtypes.OrganizationTeam) error {
						f.afterSlug = team.Slug
						return nil
					})
				})
			},
			assert: func(t *testing.T, f *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.NoError(t, err)
				require.Equal(t, "team-1", team.ID)
				require.Equal(t, "custom-slug", f.beforeSlug)
				require.Equal(t, "custom-slug", f.afterSlug)
			},
		},
		{
			name: "before create hook error is propagated",
			setup: func(f *teamHooksFixture) {
				expectCreateBase(f, "acme-platform")
				f.txRunner = &orgtests.MockTxRunner{}
				f.executor = newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
					hooks.RegisterBeforeCreate(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
						return errBeforeHook
					})
				})
			},
			assert: func(t *testing.T, f *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.ErrorIs(t, err, errBeforeHook)
				require.Nil(t, team)
				f.teamRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			},
		},
		{
			name:  "machine actor resolves its member inside the transaction",
			actor: &models.Actor{ID: "machine-1", Type: models.ActorMachine, Claims: map[string]any{"organization_id": "org-1"}},
			setup: func(f *teamHooksFixture) {
				f.orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				f.teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "acme-platform").Return(nil, nil).Once()
				expectCreatedTeam(f, "team-1")
				f.memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "machine-1").
					Return(&orgtypes.OrganizationMember{ID: "mem-machine", OrganizationID: "org-1", UserID: "machine-1"}, nil).Once()
				expectTeamMemberCreated(f, "mem-machine")
			},
			assert: func(t *testing.T, _ *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.NoError(t, err)
				require.Equal(t, "team-1", team.ID)
			},
		},
		{
			name:  "machine actor member lookup error is returned",
			actor: &models.Actor{ID: "machine-1", Type: models.ActorMachine, Claims: map[string]any{"organization_id": "org-1"}},
			setup: func(f *teamHooksFixture) {
				f.orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
				f.teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "acme-platform").Return(nil, nil).Once()
				expectCreatedTeam(f, "team-1")
				f.memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "machine-1").Return(nil, errActorLookup).Once()
			},
			assert: func(t *testing.T, f *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.ErrorIs(t, err, errActorLookup)
				require.Nil(t, team)
				f.teamMemberRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			},
		},
		{
			name: "team member create error is returned",
			setup: func(f *teamHooksFixture) {
				expectCreateBase(f, "acme-platform")
				expectCreatedTeam(f, "team-1")
				f.teamMemberRepo.On("Create", mock.Anything, mock.Anything).Return(nil, errTeamMemberCreate).Once()
			},
			assert: func(t *testing.T, _ *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.ErrorIs(t, err, errTeamMemberCreate)
				require.Nil(t, team)
			},
		},
		{
			name: "after create hook error is propagated in the transaction path",
			setup: func(f *teamHooksFixture) {
				expectCreateBase(f, "acme-platform")
				expectCreatedTeam(f, "team-1")
				expectTeamMemberCreated(f, "mem-1")
				f.txRunner = &orgtests.MockTxRunner{}
				f.executor = newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
					hooks.RegisterAfterCreate(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
						return errAfterHook
					})
				})
			},
			assert: func(t *testing.T, _ *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.ErrorIs(t, err, errAfterHook)
				require.Nil(t, team)
			},
		},
		{
			name: "create error without a transaction runner is returned",
			setup: func(f *teamHooksFixture) {
				expectCreateBase(f, "acme-platform")
				f.teamRepo.On("Create", mock.Anything, mock.Anything).Return(nil, errTeamCreate).Once()
			},
			assert: func(t *testing.T, _ *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.ErrorIs(t, err, errTeamCreate)
				require.Nil(t, team)
			},
		},
		{
			name: "after create hook error is propagated without a transaction runner",
			setup: func(f *teamHooksFixture) {
				expectCreateBase(f, "acme-platform")
				expectCreatedTeam(f, "team-1")
				expectTeamMemberCreated(f, "mem-1")
				f.executor = newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
					hooks.RegisterAfterCreate(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
						return errAfterHook
					})
				})
			},
			assert: func(t *testing.T, _ *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.ErrorIs(t, err, errAfterHook)
				require.Nil(t, team)
			},
		},
		{
			name: "success without a transaction runner",
			setup: func(f *teamHooksFixture) {
				expectCreateBase(f, "acme-platform")
				expectCreatedTeam(f, "team-1")
				expectTeamMemberCreated(f, "mem-1")
			},
			assert: func(t *testing.T, _ *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.NoError(t, err)
				require.Equal(t, "team-1", team.ID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newTeamHooksFixture()
			tc.setup(f)

			actor := tc.actor
			if actor == nil {
				actor = orgtests.Actor("user-1")
			}

			team, err := f.service().CreateTeam(context.Background(), actor, "org-1", orgtypes.CreateOrganizationTeamRequest{
				Name: "Acme Platform",
				Slug: tc.slug,
			})

			tc.assert(t, f, team, err)
			f.orgRepo.AssertExpectations(t)
			f.memberRepo.AssertExpectations(t)
			f.teamRepo.AssertExpectations(t)
			f.teamMemberRepo.AssertExpectations(t)
		})
	}
}

func TestOrganizationTeamService_GetTeamExtra(t *testing.T) {
	t.Parallel()

	errLookup := errors.New("team lookup failed")

	tests := []struct {
		name   string
		setup  func(*teamHooksFixture)
		assert func(*testing.T, *teamHooksFixture, *orgtypes.OrganizationTeam, error)
	}{
		{
			name: "team from another organization is not found",
			setup: func(f *teamHooksFixture) {
				expectTeamOrganization(f.orgRepo)
				expectTeamMemberAccess(f.memberRepo, 1)
				f.teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-2"}, nil).Once()
			},
			assert: func(t *testing.T, _ *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.ErrorIs(t, err, coreerrors.ErrNotFound)
				require.Nil(t, team)
			},
		},
		{
			name: "second team lookup error is returned",
			setup: func(f *teamHooksFixture) {
				expectTeamOrganization(f.orgRepo)
				expectTeamMemberAccess(f.memberRepo, 2)
				f.teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Once()
				f.teamRepo.On("GetByID", mock.Anything, "team-1").Return(nil, errLookup).Once()
			},
			assert: func(t *testing.T, _ *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.ErrorIs(t, err, errLookup)
				require.Nil(t, team)
			},
		},
		{
			name: "missing team on the second lookup is not found",
			setup: func(f *teamHooksFixture) {
				expectTeamOrganization(f.orgRepo)
				expectTeamMemberAccess(f.memberRepo, 2)
				f.teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Once()
				f.teamRepo.On("GetByID", mock.Anything, "team-1").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, _ *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.ErrorIs(t, err, coreerrors.ErrNotFound)
				require.Nil(t, team)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newTeamHooksFixture()
			tc.setup(f)

			team, err := f.service().GetTeam(context.Background(), orgtests.Actor("user-1"), "org-1", "team-1")

			tc.assert(t, f, team, err)
			f.orgRepo.AssertExpectations(t)
			f.memberRepo.AssertExpectations(t)
			f.teamRepo.AssertExpectations(t)
			f.teamMemberRepo.AssertExpectations(t)
		})
	}
}

func TestOrganizationTeamService_UpdateTeamExtra(t *testing.T) {
	t.Parallel()

	errBeforeHook := errors.New("before update hook failed")
	errAfterHook := errors.New("after update hook failed")

	tests := []struct {
		name   string
		setup  func(*teamHooksFixture)
		assert func(*testing.T, *teamHooksFixture, *orgtypes.OrganizationTeam, error)
	}{
		{
			name: "team from another organization is rejected before the update",
			setup: func(f *teamHooksFixture) {
				expectTeamOrganization(f.orgRepo)
				expectTeamMemberAccess(f.memberRepo, 1)
				f.teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-2"}, nil).Once()
			},
			assert: func(t *testing.T, f *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.ErrorIs(t, err, coreerrors.ErrNotFound)
				require.Nil(t, team)
				f.teamRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			},
		},
		{
			name: "before update hook error is propagated",
			setup: func(f *teamHooksFixture) {
				expectTeamOrganization(f.orgRepo)
				expectTeamMemberAccess(f.memberRepo, 2)
				f.teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Slug: "platform"}, nil).Twice()
				f.teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "platform").Return(nil, nil).Once()
				f.txRunner = &orgtests.MockTxRunner{}
				f.executor = newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
					hooks.RegisterBeforeUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
						return errBeforeHook
					})
				})
			},
			assert: func(t *testing.T, f *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.ErrorIs(t, err, errBeforeHook)
				require.Nil(t, team)
				f.teamRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			},
		},
		{
			name: "after update hook error is propagated",
			setup: func(f *teamHooksFixture) {
				expectTeamOrganization(f.orgRepo)
				expectTeamMemberAccess(f.memberRepo, 2)
				f.teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Slug: "platform"}, nil).Twice()
				f.teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "platform").Return(nil, nil).Once()
				f.teamRepo.On("Update", mock.Anything, mock.Anything).Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Once()
				f.txRunner = &orgtests.MockTxRunner{}
				f.executor = newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
					hooks.RegisterAfterUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
						return errAfterHook
					})
				})
			},
			assert: func(t *testing.T, _ *teamHooksFixture, team *orgtypes.OrganizationTeam, err error) {
				require.ErrorIs(t, err, errAfterHook)
				require.Nil(t, team)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newTeamHooksFixture()
			tc.setup(f)

			team, err := f.service().UpdateTeam(context.Background(), orgtests.Actor("user-1"), "org-1", "team-1", orgtypes.UpdateOrganizationTeamRequest{Name: "Platform Revamp"})

			tc.assert(t, f, team, err)
			f.orgRepo.AssertExpectations(t)
			f.memberRepo.AssertExpectations(t)
			f.teamRepo.AssertExpectations(t)
			f.teamMemberRepo.AssertExpectations(t)
		})
	}
}

func TestOrganizationTeamService_DeleteTeamExtra(t *testing.T) {
	t.Parallel()

	errBeforeHook := errors.New("before delete hook failed")
	errAfterHook := errors.New("after delete hook failed")

	tests := []struct {
		name   string
		setup  func(*teamHooksFixture)
		assert func(*testing.T, *teamHooksFixture, error)
	}{
		{
			name: "team from another organization is rejected before the delete",
			setup: func(f *teamHooksFixture) {
				expectTeamOrganization(f.orgRepo)
				expectTeamMemberAccess(f.memberRepo, 1)
				f.teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-2"}, nil).Once()
			},
			assert: func(t *testing.T, f *teamHooksFixture, err error) {
				require.ErrorIs(t, err, coreerrors.ErrNotFound)
				f.teamRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
			},
		},
		{
			name: "before delete hook error is propagated",
			setup: func(f *teamHooksFixture) {
				expectTeamOrganization(f.orgRepo)
				expectTeamMemberAccess(f.memberRepo, 2)
				f.teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				f.executor = newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
					hooks.RegisterBeforeDelete(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
						return errBeforeHook
					})
				})
			},
			assert: func(t *testing.T, f *teamHooksFixture, err error) {
				require.ErrorIs(t, err, errBeforeHook)
				f.teamRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
			},
		},
		{
			name: "after delete hook error is propagated",
			setup: func(f *teamHooksFixture) {
				expectTeamOrganization(f.orgRepo)
				expectTeamMemberAccess(f.memberRepo, 2)
				f.teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
				f.teamRepo.On("Delete", mock.Anything, "team-1").Return(nil).Once()
				f.executor = newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
					hooks.RegisterAfterDelete(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
						return errAfterHook
					})
				})
			},
			assert: func(t *testing.T, _ *teamHooksFixture, err error) {
				require.ErrorIs(t, err, errAfterHook)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newTeamHooksFixture()
			tc.setup(f)

			err := f.service().DeleteTeam(context.Background(), orgtests.Actor("user-1"), "org-1", "team-1")

			tc.assert(t, f, err)
			f.orgRepo.AssertExpectations(t)
			f.memberRepo.AssertExpectations(t)
			f.teamRepo.AssertExpectations(t)
			f.teamMemberRepo.AssertExpectations(t)
		})
	}
}
