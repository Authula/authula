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

func newOrganizationTeamServiceWithHooks(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamRepo *orgtests.MockOrganizationTeamRepository, teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, executor *ServiceHookExecutor) *organizationTeamService {
	serviceUtils := &ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo, orgTeamRepo: teamRepo}
	return NewOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo, serviceUtils, &orgtests.MockTxRunner{}, executor)
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

	expectCreateBase := func(orgRepo *orgtests.MockOrganizationRepository, memberRepo *orgtests.MockOrganizationMemberRepository, teamRepo *orgtests.MockOrganizationTeamRepository, slug string) {
		expectTeamOrganization(orgRepo)
		expectTeamMemberAccess(memberRepo, 1)
		teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", slug).Return(nil, nil).Once()
	}
	expectCreatedTeam := func(teamRepo *orgtests.MockOrganizationTeamRepository, id string) {
		teamRepo.On("Create", mock.Anything, mock.Anything).
			Return(&orgtypes.OrganizationTeam{ID: id, OrganizationID: "org-1"}, nil).
			Once()
	}
	expectTeamMemberCreated := func(teamMemberRepo *orgtests.MockOrganizationTeamMemberRepository, memberID string) {
		teamMemberRepo.On("Create", mock.Anything, mock.MatchedBy(func(member *orgtypes.OrganizationTeamMember) bool {
			return member != nil && member.MemberID == memberID
		})).Return(&orgtypes.OrganizationTeamMember{ID: "team-member-1"}, nil).Once()
	}

	t.Run("constructor wires the hook executor and an explicit slug is used", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectCreateBase(orgRepo, memberRepo, teamRepo, "custom-slug")
		teamRepo.On("Create", mock.Anything, mock.MatchedBy(func(team *orgtypes.OrganizationTeam) bool {
			return team != nil && team.Slug == "custom-slug"
		})).Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Slug: "custom-slug"}, nil).Once()
		expectTeamMemberCreated(teamMemberRepo, "mem-1")

		var beforeSlug, afterSlug string
		executor := newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
			hooks.RegisterBeforeCreate(func(_ context.Context, _ *models.Actor, team *orgtypes.OrganizationTeam) error {
				beforeSlug = team.Slug
				return nil
			})
			hooks.RegisterAfterCreate(func(_ context.Context, _ *models.Actor, team *orgtypes.OrganizationTeam) error {
				afterSlug = team.Slug
				return nil
			})
		})

		svc := newOrganizationTeamServiceWithHooks(orgRepo, memberRepo, teamRepo, teamMemberRepo, executor)
		team, err := svc.CreateTeam(context.Background(), orgtests.Actor("user-1"), "org-1", orgtypes.CreateOrganizationTeamRequest{
			Name: "Acme Platform",
			Slug: new("custom-slug"),
		})

		require.NoError(t, err)
		require.Equal(t, "team-1", team.ID)
		require.Equal(t, "custom-slug", beforeSlug)
		require.Equal(t, "custom-slug", afterSlug)
		teamMemberRepo.AssertExpectations(t)
	})

	t.Run("before create hook error is propagated", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectCreateBase(orgRepo, memberRepo, teamRepo, "acme-platform")

		executor := newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
			hooks.RegisterBeforeCreate(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
				return errBeforeHook
			})
		})

		svc := newOrganizationTeamServiceWithHooks(orgRepo, memberRepo, teamRepo, teamMemberRepo, executor)
		team, err := svc.CreateTeam(context.Background(), orgtests.Actor("user-1"), "org-1", orgtypes.CreateOrganizationTeamRequest{Name: "Acme Platform"})

		require.ErrorIs(t, err, errBeforeHook)
		require.Nil(t, team)
		teamRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("machine actor resolves its member inside the transaction", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
		teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "acme-platform").Return(nil, nil).Once()
		expectCreatedTeam(teamRepo, "team-1")
		memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "machine-1").
			Return(&orgtypes.OrganizationMember{ID: "mem-machine", OrganizationID: "org-1", UserID: "machine-1"}, nil).Once()
		expectTeamMemberCreated(teamMemberRepo, "mem-machine")

		actor := &models.Actor{ID: "machine-1", Type: models.ActorMachine, Claims: map[string]any{"organization_id": "org-1"}}
		svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
		team, err := svc.CreateTeam(context.Background(), actor, "org-1", orgtypes.CreateOrganizationTeamRequest{Name: "Acme Platform"})

		require.NoError(t, err)
		require.Equal(t, "team-1", team.ID)
		memberRepo.AssertExpectations(t)
		teamMemberRepo.AssertExpectations(t)
	})

	t.Run("machine actor member lookup error is returned", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		orgRepo.On("GetByID", mock.Anything, "org-1").Return(&orgtypes.Organization{ID: "org-1", OwnerID: "user-1"}, nil).Once()
		teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "acme-platform").Return(nil, nil).Once()
		expectCreatedTeam(teamRepo, "team-1")
		memberRepo.On("GetByOrganizationIDAndUserID", mock.Anything, "org-1", "machine-1").Return(nil, errActorLookup).Once()

		actor := &models.Actor{ID: "machine-1", Type: models.ActorMachine, Claims: map[string]any{"organization_id": "org-1"}}
		svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
		team, err := svc.CreateTeam(context.Background(), actor, "org-1", orgtypes.CreateOrganizationTeamRequest{Name: "Acme Platform"})

		require.ErrorIs(t, err, errActorLookup)
		require.Nil(t, team)
		teamMemberRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("team member create error is returned", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectCreateBase(orgRepo, memberRepo, teamRepo, "acme-platform")
		expectCreatedTeam(teamRepo, "team-1")
		teamMemberRepo.On("Create", mock.Anything, mock.Anything).Return(nil, errTeamMemberCreate).Once()

		svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
		team, err := svc.CreateTeam(context.Background(), orgtests.Actor("user-1"), "org-1", orgtypes.CreateOrganizationTeamRequest{Name: "Acme Platform"})

		require.ErrorIs(t, err, errTeamMemberCreate)
		require.Nil(t, team)
	})

	t.Run("after create hook error is propagated in the transaction path", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectCreateBase(orgRepo, memberRepo, teamRepo, "acme-platform")
		expectCreatedTeam(teamRepo, "team-1")
		expectTeamMemberCreated(teamMemberRepo, "mem-1")

		executor := newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
			hooks.RegisterAfterCreate(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
				return errAfterHook
			})
		})

		svc := newOrganizationTeamServiceWithHooks(orgRepo, memberRepo, teamRepo, teamMemberRepo, executor)
		team, err := svc.CreateTeam(context.Background(), orgtests.Actor("user-1"), "org-1", orgtypes.CreateOrganizationTeamRequest{Name: "Acme Platform"})

		require.ErrorIs(t, err, errAfterHook)
		require.Nil(t, team)
	})

	t.Run("create error without a transaction runner is returned", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectCreateBase(orgRepo, memberRepo, teamRepo, "acme-platform")
		teamRepo.On("Create", mock.Anything, mock.Anything).Return(nil, errTeamCreate).Once()

		svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
		team, err := svc.CreateTeam(context.Background(), orgtests.Actor("user-1"), "org-1", orgtypes.CreateOrganizationTeamRequest{Name: "Acme Platform"})

		require.ErrorIs(t, err, errTeamCreate)
		require.Nil(t, team)
	})

	t.Run("after create hook error is propagated without a transaction runner", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectCreateBase(orgRepo, memberRepo, teamRepo, "acme-platform")
		expectCreatedTeam(teamRepo, "team-1")
		expectTeamMemberCreated(teamMemberRepo, "mem-1")

		executor := newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
			hooks.RegisterAfterCreate(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
				return errAfterHook
			})
		})

		serviceUtils := &ServiceUtils{orgRepo: orgRepo, orgMemberRepo: memberRepo, orgTeamRepo: teamRepo}
		svc := NewOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo, serviceUtils, nil, executor)
		team, err := svc.CreateTeam(context.Background(), orgtests.Actor("user-1"), "org-1", orgtypes.CreateOrganizationTeamRequest{Name: "Acme Platform"})

		require.ErrorIs(t, err, errAfterHook)
		require.Nil(t, team)
	})

	t.Run("success without a transaction runner", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectCreateBase(orgRepo, memberRepo, teamRepo, "acme-platform")
		expectCreatedTeam(teamRepo, "team-1")
		expectTeamMemberCreated(teamMemberRepo, "mem-1")

		svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
		team, err := svc.CreateTeam(context.Background(), orgtests.Actor("user-1"), "org-1", orgtypes.CreateOrganizationTeamRequest{Name: "Acme Platform"})

		require.NoError(t, err)
		require.Equal(t, "team-1", team.ID)
		teamMemberRepo.AssertExpectations(t)
	})
}

func TestOrganizationTeamService_GetTeamExtra(t *testing.T) {
	t.Parallel()

	errLookup := errors.New("team lookup failed")

	t.Run("team from another organization is not found", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectTeamOrganization(orgRepo)
		expectTeamMemberAccess(memberRepo, 1)
		teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-2"}, nil).Once()

		svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
		team, err := svc.GetTeam(context.Background(), orgtests.Actor("user-1"), "org-1", "team-1")

		require.ErrorIs(t, err, coreerrors.ErrNotFound)
		require.Nil(t, team)
	})

	t.Run("second team lookup error is returned", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectTeamOrganization(orgRepo)
		expectTeamMemberAccess(memberRepo, 2)
		teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Once()
		teamRepo.On("GetByID", mock.Anything, "team-1").Return(nil, errLookup).Once()

		svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
		team, err := svc.GetTeam(context.Background(), orgtests.Actor("user-1"), "org-1", "team-1")

		require.ErrorIs(t, err, errLookup)
		require.Nil(t, team)
	})

	t.Run("missing team on the second lookup is not found", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectTeamOrganization(orgRepo)
		expectTeamMemberAccess(memberRepo, 2)
		teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Once()
		teamRepo.On("GetByID", mock.Anything, "team-1").Return(nil, nil).Once()

		svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
		team, err := svc.GetTeam(context.Background(), orgtests.Actor("user-1"), "org-1", "team-1")

		require.ErrorIs(t, err, coreerrors.ErrNotFound)
		require.Nil(t, team)
	})
}

func TestOrganizationTeamService_UpdateTeamExtra(t *testing.T) {
	t.Parallel()

	errBeforeHook := errors.New("before update hook failed")
	errAfterHook := errors.New("after update hook failed")

	t.Run("team from another organization is rejected before the update", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectTeamOrganization(orgRepo)
		expectTeamMemberAccess(memberRepo, 1)
		teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-2"}, nil).Once()

		svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
		team, err := svc.UpdateTeam(context.Background(), orgtests.Actor("user-1"), "org-1", "team-1", orgtypes.UpdateOrganizationTeamRequest{Name: "Platform Revamp"})

		require.ErrorIs(t, err, coreerrors.ErrNotFound)
		require.Nil(t, team)
		teamRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("before update hook error is propagated", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectTeamOrganization(orgRepo)
		expectTeamMemberAccess(memberRepo, 2)
		teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Slug: "platform"}, nil).Twice()
		teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "platform").Return(nil, nil).Once()

		executor := newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
			hooks.RegisterBeforeUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
				return errBeforeHook
			})
		})

		svc := newOrganizationTeamServiceWithHooks(orgRepo, memberRepo, teamRepo, teamMemberRepo, executor)
		team, err := svc.UpdateTeam(context.Background(), orgtests.Actor("user-1"), "org-1", "team-1", orgtypes.UpdateOrganizationTeamRequest{Name: "Platform Revamp"})

		require.ErrorIs(t, err, errBeforeHook)
		require.Nil(t, team)
		teamRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("after update hook error is propagated", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectTeamOrganization(orgRepo)
		expectTeamMemberAccess(memberRepo, 2)
		teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Slug: "platform"}, nil).Twice()
		teamRepo.On("GetByOrganizationIDAndSlug", mock.Anything, "org-1", "platform").Return(nil, nil).Once()
		teamRepo.On("Update", mock.Anything, mock.Anything).Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Once()

		executor := newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
			hooks.RegisterAfterUpdate(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
				return errAfterHook
			})
		})

		svc := newOrganizationTeamServiceWithHooks(orgRepo, memberRepo, teamRepo, teamMemberRepo, executor)
		team, err := svc.UpdateTeam(context.Background(), orgtests.Actor("user-1"), "org-1", "team-1", orgtypes.UpdateOrganizationTeamRequest{Name: "Platform Revamp"})

		require.ErrorIs(t, err, errAfterHook)
		require.Nil(t, team)
	})
}

func TestOrganizationTeamService_DeleteTeamExtra(t *testing.T) {
	t.Parallel()

	errBeforeHook := errors.New("before delete hook failed")
	errAfterHook := errors.New("after delete hook failed")

	t.Run("team from another organization is rejected before the delete", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectTeamOrganization(orgRepo)
		expectTeamMemberAccess(memberRepo, 1)
		teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-2"}, nil).Once()

		svc := newTestOrganizationTeamService(orgRepo, memberRepo, teamRepo, teamMemberRepo)
		err := svc.DeleteTeam(context.Background(), orgtests.Actor("user-1"), "org-1", "team-1")

		require.ErrorIs(t, err, coreerrors.ErrNotFound)
		teamRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("before delete hook error is propagated", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectTeamOrganization(orgRepo)
		expectTeamMemberAccess(memberRepo, 2)
		teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()

		executor := newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
			hooks.RegisterBeforeDelete(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
				return errBeforeHook
			})
		})

		svc := newOrganizationTeamServiceWithHooks(orgRepo, memberRepo, teamRepo, teamMemberRepo, executor)
		err := svc.DeleteTeam(context.Background(), orgtests.Actor("user-1"), "org-1", "team-1")

		require.ErrorIs(t, err, errBeforeHook)
		teamRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("after delete hook error is propagated", func(t *testing.T) {
		t.Parallel()

		orgRepo := &orgtests.MockOrganizationRepository{}
		memberRepo := &orgtests.MockOrganizationMemberRepository{}
		teamRepo := &orgtests.MockOrganizationTeamRepository{}
		teamMemberRepo := &orgtests.MockOrganizationTeamMemberRepository{}
		expectTeamOrganization(orgRepo)
		expectTeamMemberAccess(memberRepo, 2)
		teamRepo.On("GetByID", mock.Anything, "team-1").Return(&orgtypes.OrganizationTeam{ID: "team-1", OrganizationID: "org-1"}, nil).Twice()
		teamRepo.On("Delete", mock.Anything, "team-1").Return(nil).Once()

		executor := newTeamHooksExecutor(func(hooks *orgtypes.OrganizationTeamServiceHooks) {
			hooks.RegisterAfterDelete(func(context.Context, *models.Actor, *orgtypes.OrganizationTeam) error {
				return errAfterHook
			})
		})

		svc := newOrganizationTeamServiceWithHooks(orgRepo, memberRepo, teamRepo, teamMemberRepo, executor)
		err := svc.DeleteTeam(context.Background(), orgtests.Actor("user-1"), "org-1", "team-1")

		require.ErrorIs(t, err, errAfterHook)
		teamRepo.AssertExpectations(t)
	})
}
