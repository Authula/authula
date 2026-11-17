package types_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/core/pagination"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/types"
)

func TestOrganizationsPluginConfig_ApplyDefaultsPreservesConfiguredValues(t *testing.T) {
	t.Parallel()

	config := &types.OrganizationsPluginConfig{
		MembersLimit:        new(5),
		InvitationsLimit:    new(7),
		MaxPageLimit:        new(50),
		InvitationExpiresIn: 2 * time.Hour,
	}
	config.ApplyDefaults()

	require.Equal(t, 5, *config.MembersLimit)
	require.Equal(t, 7, *config.InvitationsLimit)
	require.Equal(t, 50, *config.MaxPageLimit)
	require.Equal(t, 2*time.Hour, config.InvitationExpiresIn)
}

func TestOrganizationsPluginConfig_ApplyDefaultsFillsEachGap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   types.OrganizationsPluginConfig
		expected func(*types.OrganizationsPluginConfig)
	}{
		{
			name:   "nil members limit",
			config: types.OrganizationsPluginConfig{MembersLimit: nil},
			expected: func(config *types.OrganizationsPluginConfig) {
				require.Equal(t, 100, *config.MembersLimit)
			},
		},
		{
			name:   "zero invitations limit",
			config: types.OrganizationsPluginConfig{InvitationsLimit: new(0)},
			expected: func(config *types.OrganizationsPluginConfig) {
				require.Equal(t, 100, *config.InvitationsLimit)
			},
		},
		{
			name:   "negative invitations limit",
			config: types.OrganizationsPluginConfig{InvitationsLimit: new(-1)},
			expected: func(config *types.OrganizationsPluginConfig) {
				require.Equal(t, 100, *config.InvitationsLimit)
			},
		},
		{
			name:   "zero max page limit",
			config: types.OrganizationsPluginConfig{MaxPageLimit: new(0)},
			expected: func(config *types.OrganizationsPluginConfig) {
				require.Equal(t, pagination.DefaultMaxLimit, *config.MaxPageLimit)
			},
		},
		{
			name:   "zero invitation expiry",
			config: types.OrganizationsPluginConfig{InvitationExpiresIn: 0},
			expected: func(config *types.OrganizationsPluginConfig) {
				require.Equal(t, 24*time.Hour, config.InvitationExpiresIn)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := tt.config
			config.ApplyDefaults()

			tt.expected(&config)
		})
	}
}

func assertHookRun(t *testing.T, hooks []types.OrganizationHook) {
	t.Helper()

	require.Len(t, hooks, 1)
	require.NoError(t, hooks[0](context.Background(), &models.Actor{ID: "user-1"}, &types.Organization{ID: "org-1"}))
}

func TestOrganizationServiceHooks_RegisterAndRead(t *testing.T) {
	t.Parallel()

	hooks := &types.OrganizationServiceHooks{}
	var called []string
	register := func(name string) types.OrganizationHook {
		return func(context.Context, *models.Actor, *types.Organization) error {
			called = append(called, name)
			return nil
		}
	}

	hooks.RegisterBeforeCreate(register("beforeCreate"))
	hooks.RegisterAfterCreate(register("afterCreate"))
	hooks.RegisterBeforeUpdate(register("beforeUpdate"))
	hooks.RegisterAfterUpdate(register("afterUpdate"))
	hooks.RegisterBeforeDelete(register("beforeDelete"))
	hooks.RegisterAfterDelete(register("afterDelete"))

	assertHookRun(t, hooks.BeforeCreateHooks())
	assertHookRun(t, hooks.AfterCreateHooks())
	assertHookRun(t, hooks.BeforeUpdateHooks())
	assertHookRun(t, hooks.AfterUpdateHooks())
	assertHookRun(t, hooks.BeforeDeleteHooks())
	assertHookRun(t, hooks.AfterDeleteHooks())

	require.Equal(t, []string{"beforeCreate", "afterCreate", "beforeUpdate", "afterUpdate", "beforeDelete", "afterDelete"}, called)
}

func assertMemberHookRun(t *testing.T, hooks []types.OrganizationMemberHook) {
	t.Helper()

	require.Len(t, hooks, 1)
	require.NoError(t, hooks[0](context.Background(), &models.Actor{ID: "user-1"}, &types.OrganizationMember{ID: "mem-1"}))
}

func TestOrganizationMemberServiceHooks_RegisterAndRead(t *testing.T) {
	t.Parallel()

	hooks := &types.OrganizationMemberServiceHooks{}
	var called []string
	register := func(name string) types.OrganizationMemberHook {
		return func(context.Context, *models.Actor, *types.OrganizationMember) error {
			called = append(called, name)
			return nil
		}
	}

	hooks.RegisterBeforeCreate(register("beforeCreate"))
	hooks.RegisterAfterCreate(register("afterCreate"))
	hooks.RegisterBeforeUpdate(register("beforeUpdate"))
	hooks.RegisterAfterUpdate(register("afterUpdate"))
	hooks.RegisterBeforeDelete(register("beforeDelete"))
	hooks.RegisterAfterDelete(register("afterDelete"))

	assertMemberHookRun(t, hooks.BeforeCreateHooks())
	assertMemberHookRun(t, hooks.AfterCreateHooks())
	assertMemberHookRun(t, hooks.BeforeUpdateHooks())
	assertMemberHookRun(t, hooks.AfterUpdateHooks())
	assertMemberHookRun(t, hooks.BeforeDeleteHooks())
	assertMemberHookRun(t, hooks.AfterDeleteHooks())

	require.Equal(t, []string{"beforeCreate", "afterCreate", "beforeUpdate", "afterUpdate", "beforeDelete", "afterDelete"}, called)
}

func assertInvitationHookRun(t *testing.T, hooks []types.OrganizationInvitationHook) {
	t.Helper()

	require.Len(t, hooks, 1)
	require.NoError(t, hooks[0](context.Background(), &models.Actor{ID: "user-1"}, &types.OrganizationInvitation{ID: "inv-1"}))
}

func TestOrganizationInvitationServiceHooks_RegisterAndRead(t *testing.T) {
	t.Parallel()

	hooks := &types.OrganizationInvitationServiceHooks{}
	var called []string
	register := func(name string) types.OrganizationInvitationHook {
		return func(context.Context, *models.Actor, *types.OrganizationInvitation) error {
			called = append(called, name)
			return nil
		}
	}

	hooks.RegisterBeforeCreate(register("beforeCreate"))
	hooks.RegisterAfterCreate(register("afterCreate"))
	hooks.RegisterBeforeUpdate(register("beforeUpdate"))
	hooks.RegisterAfterUpdate(register("afterUpdate"))

	assertInvitationHookRun(t, hooks.BeforeCreateHooks())
	assertInvitationHookRun(t, hooks.AfterCreateHooks())
	assertInvitationHookRun(t, hooks.BeforeUpdateHooks())
	assertInvitationHookRun(t, hooks.AfterUpdateHooks())

	require.Equal(t, []string{"beforeCreate", "afterCreate", "beforeUpdate", "afterUpdate"}, called)
}

func assertTeamHookRun(t *testing.T, hooks []types.OrganizationTeamHook) {
	t.Helper()

	require.Len(t, hooks, 1)
	require.NoError(t, hooks[0](context.Background(), &models.Actor{ID: "user-1"}, &types.OrganizationTeam{ID: "team-1"}))
}

func TestOrganizationTeamServiceHooks_RegisterAndRead(t *testing.T) {
	t.Parallel()

	hooks := &types.OrganizationTeamServiceHooks{}
	var called []string
	register := func(name string) types.OrganizationTeamHook {
		return func(context.Context, *models.Actor, *types.OrganizationTeam) error {
			called = append(called, name)
			return nil
		}
	}

	hooks.RegisterBeforeCreate(register("beforeCreate"))
	hooks.RegisterAfterCreate(register("afterCreate"))
	hooks.RegisterBeforeUpdate(register("beforeUpdate"))
	hooks.RegisterAfterUpdate(register("afterUpdate"))
	hooks.RegisterBeforeDelete(register("beforeDelete"))
	hooks.RegisterAfterDelete(register("afterDelete"))

	assertTeamHookRun(t, hooks.BeforeCreateHooks())
	assertTeamHookRun(t, hooks.AfterCreateHooks())
	assertTeamHookRun(t, hooks.BeforeUpdateHooks())
	assertTeamHookRun(t, hooks.AfterUpdateHooks())
	assertTeamHookRun(t, hooks.BeforeDeleteHooks())
	assertTeamHookRun(t, hooks.AfterDeleteHooks())

	require.Equal(t, []string{"beforeCreate", "afterCreate", "beforeUpdate", "afterUpdate", "beforeDelete", "afterDelete"}, called)
}

func assertTeamMemberHookRun(t *testing.T, hooks []types.OrganizationTeamMemberHook) {
	t.Helper()

	require.Len(t, hooks, 1)
	require.NoError(t, hooks[0](context.Background(), &models.Actor{ID: "user-1"}, &types.OrganizationTeamMember{ID: "tm-1"}))
}

func TestOrganizationTeamMemberServiceHooks_RegisterAndRead(t *testing.T) {
	t.Parallel()

	hooks := &types.OrganizationTeamMemberServiceHooks{}
	var called []string
	register := func(name string) types.OrganizationTeamMemberHook {
		return func(context.Context, *models.Actor, *types.OrganizationTeamMember) error {
			called = append(called, name)
			return nil
		}
	}

	hooks.RegisterBeforeCreate(register("beforeCreate"))
	hooks.RegisterAfterCreate(register("afterCreate"))
	hooks.RegisterBeforeDelete(register("beforeDelete"))
	hooks.RegisterAfterDelete(register("afterDelete"))

	assertTeamMemberHookRun(t, hooks.BeforeCreateHooks())
	assertTeamMemberHookRun(t, hooks.AfterCreateHooks())
	assertTeamMemberHookRun(t, hooks.BeforeDeleteHooks())
	assertTeamMemberHookRun(t, hooks.AfterDeleteHooks())

	require.Equal(t, []string{"beforeCreate", "afterCreate", "beforeDelete", "afterDelete"}, called)
}
