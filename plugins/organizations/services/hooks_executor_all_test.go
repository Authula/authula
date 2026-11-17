package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/types"
)

// hookCase describes one executor method: how to register a hook for it and how
// to invoke it.
type hookCase struct {
	name     string
	register func(config *types.OrganizationsServiceHooksConfig, hook func() error)
	invoke   func(*ServiceHookExecutor) error
}

func TestServiceHookExecutor_EveryHookRunsAndPropagatesErrors(t *testing.T) {
	t.Parallel()

	hookErr := errors.New("hook failure")
	actor := &models.Actor{ID: "user-1"}

	organization := &types.Organization{ID: "org-1"}
	member := &types.OrganizationMember{ID: "mem-1"}
	invitation := &types.OrganizationInvitation{ID: "inv-1"}
	team := &types.OrganizationTeam{ID: "team-1"}
	teamMember := &types.OrganizationTeamMember{ID: "tm-1"}

	orgHook := func(fn func() error) types.OrganizationHook {
		return func(context.Context, *models.Actor, *types.Organization) error { return fn() }
	}
	memberHook := func(fn func() error) types.OrganizationMemberHook {
		return func(context.Context, *models.Actor, *types.OrganizationMember) error { return fn() }
	}
	invitationHook := func(fn func() error) types.OrganizationInvitationHook {
		return func(context.Context, *models.Actor, *types.OrganizationInvitation) error { return fn() }
	}
	teamHook := func(fn func() error) types.OrganizationTeamHook {
		return func(context.Context, *models.Actor, *types.OrganizationTeam) error { return fn() }
	}
	teamMemberHook := func(fn func() error) types.OrganizationTeamMemberHook {
		return func(context.Context, *models.Actor, *types.OrganizationTeamMember) error { return fn() }
	}

	tests := []hookCase{
		{
			name: "organization before create",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Organizations = &types.OrganizationServiceHooks{}
				config.Organizations.RegisterBeforeCreate(orgHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.BeforeCreateOrganization(context.Background(), actor, organization)
			},
		},
		{
			name: "organization after create",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Organizations = &types.OrganizationServiceHooks{}
				config.Organizations.RegisterAfterCreate(orgHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.AfterCreateOrganization(context.Background(), actor, organization)
			},
		},
		{
			name: "organization before update",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Organizations = &types.OrganizationServiceHooks{}
				config.Organizations.RegisterBeforeUpdate(orgHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.BeforeUpdateOrganization(context.Background(), actor, organization)
			},
		},
		{
			name: "organization after update",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Organizations = &types.OrganizationServiceHooks{}
				config.Organizations.RegisterAfterUpdate(orgHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.AfterUpdateOrganization(context.Background(), actor, organization)
			},
		},
		{
			name: "organization before delete",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Organizations = &types.OrganizationServiceHooks{}
				config.Organizations.RegisterBeforeDelete(orgHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.BeforeDeleteOrganization(context.Background(), actor, organization)
			},
		},
		{
			name: "organization after delete",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Organizations = &types.OrganizationServiceHooks{}
				config.Organizations.RegisterAfterDelete(orgHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.AfterDeleteOrganization(context.Background(), actor, organization)
			},
		},
		{
			name: "member before create",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Members = &types.OrganizationMemberServiceHooks{}
				config.Members.RegisterBeforeCreate(memberHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.BeforeCreateOrganizationMember(context.Background(), actor, member)
			},
		},
		{
			name: "member after create",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Members = &types.OrganizationMemberServiceHooks{}
				config.Members.RegisterAfterCreate(memberHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.AfterCreateOrganizationMember(context.Background(), actor, member)
			},
		},
		{
			name: "member before update",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Members = &types.OrganizationMemberServiceHooks{}
				config.Members.RegisterBeforeUpdate(memberHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.BeforeUpdateOrganizationMember(context.Background(), actor, member)
			},
		},
		{
			name: "member after update",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Members = &types.OrganizationMemberServiceHooks{}
				config.Members.RegisterAfterUpdate(memberHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.AfterUpdateOrganizationMember(context.Background(), actor, member)
			},
		},
		{
			name: "member before delete",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Members = &types.OrganizationMemberServiceHooks{}
				config.Members.RegisterBeforeDelete(memberHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.BeforeDeleteOrganizationMember(context.Background(), actor, member)
			},
		},
		{
			name: "member after delete",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Members = &types.OrganizationMemberServiceHooks{}
				config.Members.RegisterAfterDelete(memberHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.AfterDeleteOrganizationMember(context.Background(), actor, member)
			},
		},
		{
			name: "invitation before create",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Invitations = &types.OrganizationInvitationServiceHooks{}
				config.Invitations.RegisterBeforeCreate(invitationHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.BeforeCreateOrganizationInvitation(context.Background(), actor, invitation)
			},
		},
		{
			name: "invitation after create",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Invitations = &types.OrganizationInvitationServiceHooks{}
				config.Invitations.RegisterAfterCreate(invitationHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.AfterCreateOrganizationInvitation(context.Background(), actor, invitation)
			},
		},
		{
			name: "invitation before update",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Invitations = &types.OrganizationInvitationServiceHooks{}
				config.Invitations.RegisterBeforeUpdate(invitationHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.BeforeUpdateOrganizationInvitation(context.Background(), actor, invitation)
			},
		},
		{
			name: "invitation after update",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Invitations = &types.OrganizationInvitationServiceHooks{}
				config.Invitations.RegisterAfterUpdate(invitationHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.AfterUpdateOrganizationInvitation(context.Background(), actor, invitation)
			},
		},
		{
			name: "team before create",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Teams = &types.OrganizationTeamServiceHooks{}
				config.Teams.RegisterBeforeCreate(teamHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.BeforeCreateOrganizationTeam(context.Background(), actor, team)
			},
		},
		{
			name: "team after create",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Teams = &types.OrganizationTeamServiceHooks{}
				config.Teams.RegisterAfterCreate(teamHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.AfterCreateOrganizationTeam(context.Background(), actor, team)
			},
		},
		{
			name: "team before update",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Teams = &types.OrganizationTeamServiceHooks{}
				config.Teams.RegisterBeforeUpdate(teamHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.BeforeUpdateOrganizationTeam(context.Background(), actor, team)
			},
		},
		{
			name: "team after update",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Teams = &types.OrganizationTeamServiceHooks{}
				config.Teams.RegisterAfterUpdate(teamHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.AfterUpdateOrganizationTeam(context.Background(), actor, team)
			},
		},
		{
			name: "team before delete",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Teams = &types.OrganizationTeamServiceHooks{}
				config.Teams.RegisterBeforeDelete(teamHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.BeforeDeleteOrganizationTeam(context.Background(), actor, team)
			},
		},
		{
			name: "team after delete",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.Teams = &types.OrganizationTeamServiceHooks{}
				config.Teams.RegisterAfterDelete(teamHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.AfterDeleteOrganizationTeam(context.Background(), actor, team)
			},
		},
		{
			name: "team member before create",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.TeamMembers = &types.OrganizationTeamMemberServiceHooks{}
				config.TeamMembers.RegisterBeforeCreate(teamMemberHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.BeforeCreateOrganizationTeamMember(context.Background(), actor, teamMember)
			},
		},
		{
			name: "team member after create",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.TeamMembers = &types.OrganizationTeamMemberServiceHooks{}
				config.TeamMembers.RegisterAfterCreate(teamMemberHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.AfterCreateOrganizationTeamMember(context.Background(), actor, teamMember)
			},
		},
		{
			name: "team member before delete",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.TeamMembers = &types.OrganizationTeamMemberServiceHooks{}
				config.TeamMembers.RegisterBeforeDelete(teamMemberHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.BeforeDeleteOrganizationTeamMember(context.Background(), actor, teamMember)
			},
		},
		{
			name: "team member after delete",
			register: func(config *types.OrganizationsServiceHooksConfig, hook func() error) {
				config.TeamMembers = &types.OrganizationTeamMemberServiceHooks{}
				config.TeamMembers.RegisterAfterDelete(teamMemberHook(hook))
			},
			invoke: func(e *ServiceHookExecutor) error {
				return e.AfterDeleteOrganizationTeamMember(context.Background(), actor, teamMember)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			t.Run("registered hook runs", func(t *testing.T) {
				t.Parallel()

				called := false
				config := &types.OrganizationsServiceHooksConfig{}
				tt.register(config, func() error {
					called = true
					return nil
				})

				require.NoError(t, tt.invoke(NewServiceHookExecutor(config, nil, nil)))
				require.True(t, called)
			})

			t.Run("hook error is propagated", func(t *testing.T) {
				t.Parallel()

				config := &types.OrganizationsServiceHooksConfig{}
				tt.register(config, func() error { return hookErr })

				require.ErrorIs(t, tt.invoke(NewServiceHookExecutor(config, nil, nil)), hookErr)
			})

			t.Run("absent hook group is a no-op", func(t *testing.T) {
				t.Parallel()

				require.NoError(t, tt.invoke(NewServiceHookExecutor(&types.OrganizationsServiceHooksConfig{}, nil, nil)))
			})
		})
	}
}
