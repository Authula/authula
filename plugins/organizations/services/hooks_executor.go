package services

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/types"
)

type ServiceHookExecutor struct {
	config         *types.OrganizationsServiceHooksConfig
	pluginRegistry models.PluginRegistry
	registry       models.ServiceRegistry
}

func NewServiceHookExecutor(config *types.OrganizationsServiceHooksConfig, pluginRegistry models.PluginRegistry, registry models.ServiceRegistry) *ServiceHookExecutor {
	return &ServiceHookExecutor{config: config, pluginRegistry: pluginRegistry, registry: registry}
}

func (e *ServiceHookExecutor) withRegistries(ctx context.Context) context.Context {
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	return models.NewContextWithServiceRegistry(ctx, e.registry)
}

func runOrganizationHooks(ctx context.Context, hooks []types.OrganizationHook, actor *models.Actor, organization *types.Organization) error {
	for _, hook := range hooks {
		if err := hook(ctx, actor, organization); err != nil {
			return err
		}
	}
	return nil
}

func runOrganizationMemberHooks(ctx context.Context, hooks []types.OrganizationMemberHook, actor *models.Actor, member *types.OrganizationMember) error {
	for _, hook := range hooks {
		if err := hook(ctx, actor, member); err != nil {
			return err
		}
	}
	return nil
}

func runOrganizationInvitationHooks(ctx context.Context, hooks []types.OrganizationInvitationHook, actor *models.Actor, invitation *types.OrganizationInvitation) error {
	for _, hook := range hooks {
		if err := hook(ctx, actor, invitation); err != nil {
			return err
		}
	}
	return nil
}

func runOrganizationTeamHooks(ctx context.Context, hooks []types.OrganizationTeamHook, actor *models.Actor, team *types.OrganizationTeam) error {
	for _, hook := range hooks {
		if err := hook(ctx, actor, team); err != nil {
			return err
		}
	}
	return nil
}

func runOrganizationTeamMemberHooks(ctx context.Context, hooks []types.OrganizationTeamMemberHook, actor *models.Actor, member *types.OrganizationTeamMember) error {
	for _, hook := range hooks {
		if err := hook(ctx, actor, member); err != nil {
			return err
		}
	}
	return nil
}

func (e *ServiceHookExecutor) BeforeCreateOrganization(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil {
		return nil
	}
	return runOrganizationHooks(e.withRegistries(ctx), e.config.Organizations.BeforeCreateHooks(), actor, organization)
}

func (e *ServiceHookExecutor) AfterCreateOrganization(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil {
		return nil
	}
	return runOrganizationHooks(e.withRegistries(ctx), e.config.Organizations.AfterCreateHooks(), actor, organization)
}

func (e *ServiceHookExecutor) BeforeUpdateOrganization(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil {
		return nil
	}
	return runOrganizationHooks(e.withRegistries(ctx), e.config.Organizations.BeforeUpdateHooks(), actor, organization)
}

func (e *ServiceHookExecutor) AfterUpdateOrganization(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil {
		return nil
	}
	return runOrganizationHooks(e.withRegistries(ctx), e.config.Organizations.AfterUpdateHooks(), actor, organization)
}

func (e *ServiceHookExecutor) BeforeDeleteOrganization(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil {
		return nil
	}
	return runOrganizationHooks(e.withRegistries(ctx), e.config.Organizations.BeforeDeleteHooks(), actor, organization)
}

func (e *ServiceHookExecutor) AfterDeleteOrganization(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil {
		return nil
	}
	return runOrganizationHooks(e.withRegistries(ctx), e.config.Organizations.AfterDeleteHooks(), actor, organization)
}

func (e *ServiceHookExecutor) BeforeCreateOrganizationMember(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil {
		return nil
	}
	return runOrganizationMemberHooks(e.withRegistries(ctx), e.config.Members.BeforeCreateHooks(), actor, member)
}

func (e *ServiceHookExecutor) AfterCreateOrganizationMember(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil {
		return nil
	}
	return runOrganizationMemberHooks(e.withRegistries(ctx), e.config.Members.AfterCreateHooks(), actor, member)
}

func (e *ServiceHookExecutor) BeforeUpdateOrganizationMember(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil {
		return nil
	}
	return runOrganizationMemberHooks(e.withRegistries(ctx), e.config.Members.BeforeUpdateHooks(), actor, member)
}

func (e *ServiceHookExecutor) AfterUpdateOrganizationMember(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil {
		return nil
	}
	return runOrganizationMemberHooks(e.withRegistries(ctx), e.config.Members.AfterUpdateHooks(), actor, member)
}

func (e *ServiceHookExecutor) BeforeDeleteOrganizationMember(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil {
		return nil
	}
	return runOrganizationMemberHooks(e.withRegistries(ctx), e.config.Members.BeforeDeleteHooks(), actor, member)
}

func (e *ServiceHookExecutor) AfterDeleteOrganizationMember(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil {
		return nil
	}
	return runOrganizationMemberHooks(e.withRegistries(ctx), e.config.Members.AfterDeleteHooks(), actor, member)
}

func (e *ServiceHookExecutor) BeforeCreateOrganizationInvitation(ctx context.Context, actor *models.Actor, invitation *types.OrganizationInvitation) error {
	if e == nil || e.config == nil || e.config.Invitations == nil {
		return nil
	}
	return runOrganizationInvitationHooks(e.withRegistries(ctx), e.config.Invitations.BeforeCreateHooks(), actor, invitation)
}

func (e *ServiceHookExecutor) AfterCreateOrganizationInvitation(ctx context.Context, actor *models.Actor, invitation *types.OrganizationInvitation) error {
	if e == nil || e.config == nil || e.config.Invitations == nil {
		return nil
	}
	return runOrganizationInvitationHooks(e.withRegistries(ctx), e.config.Invitations.AfterCreateHooks(), actor, invitation)
}

func (e *ServiceHookExecutor) BeforeUpdateOrganizationInvitation(ctx context.Context, actor *models.Actor, invitation *types.OrganizationInvitation) error {
	if e == nil || e.config == nil || e.config.Invitations == nil {
		return nil
	}
	return runOrganizationInvitationHooks(e.withRegistries(ctx), e.config.Invitations.BeforeUpdateHooks(), actor, invitation)
}

func (e *ServiceHookExecutor) AfterUpdateOrganizationInvitation(ctx context.Context, actor *models.Actor, invitation *types.OrganizationInvitation) error {
	if e == nil || e.config == nil || e.config.Invitations == nil {
		return nil
	}
	return runOrganizationInvitationHooks(e.withRegistries(ctx), e.config.Invitations.AfterUpdateHooks(), actor, invitation)
}

func (e *ServiceHookExecutor) BeforeCreateOrganizationTeam(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil {
		return nil
	}
	return runOrganizationTeamHooks(e.withRegistries(ctx), e.config.Teams.BeforeCreateHooks(), actor, team)
}

func (e *ServiceHookExecutor) AfterCreateOrganizationTeam(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil {
		return nil
	}
	return runOrganizationTeamHooks(e.withRegistries(ctx), e.config.Teams.AfterCreateHooks(), actor, team)
}

func (e *ServiceHookExecutor) BeforeUpdateOrganizationTeam(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil {
		return nil
	}
	return runOrganizationTeamHooks(e.withRegistries(ctx), e.config.Teams.BeforeUpdateHooks(), actor, team)
}

func (e *ServiceHookExecutor) AfterUpdateOrganizationTeam(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil {
		return nil
	}
	return runOrganizationTeamHooks(e.withRegistries(ctx), e.config.Teams.AfterUpdateHooks(), actor, team)
}

func (e *ServiceHookExecutor) BeforeDeleteOrganizationTeam(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil {
		return nil
	}
	return runOrganizationTeamHooks(e.withRegistries(ctx), e.config.Teams.BeforeDeleteHooks(), actor, team)
}

func (e *ServiceHookExecutor) AfterDeleteOrganizationTeam(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil {
		return nil
	}
	return runOrganizationTeamHooks(e.withRegistries(ctx), e.config.Teams.AfterDeleteHooks(), actor, team)
}

func (e *ServiceHookExecutor) BeforeCreateOrganizationTeamMember(ctx context.Context, actor *models.Actor, member *types.OrganizationTeamMember) error {
	if e == nil || e.config == nil || e.config.TeamMembers == nil {
		return nil
	}
	return runOrganizationTeamMemberHooks(e.withRegistries(ctx), e.config.TeamMembers.BeforeCreateHooks(), actor, member)
}

func (e *ServiceHookExecutor) AfterCreateOrganizationTeamMember(ctx context.Context, actor *models.Actor, member *types.OrganizationTeamMember) error {
	if e == nil || e.config == nil || e.config.TeamMembers == nil {
		return nil
	}
	return runOrganizationTeamMemberHooks(e.withRegistries(ctx), e.config.TeamMembers.AfterCreateHooks(), actor, member)
}

func (e *ServiceHookExecutor) BeforeDeleteOrganizationTeamMember(ctx context.Context, actor *models.Actor, member *types.OrganizationTeamMember) error {
	if e == nil || e.config == nil || e.config.TeamMembers == nil {
		return nil
	}
	return runOrganizationTeamMemberHooks(e.withRegistries(ctx), e.config.TeamMembers.BeforeDeleteHooks(), actor, member)
}

func (e *ServiceHookExecutor) AfterDeleteOrganizationTeamMember(ctx context.Context, actor *models.Actor, member *types.OrganizationTeamMember) error {
	if e == nil || e.config == nil || e.config.TeamMembers == nil {
		return nil
	}
	return runOrganizationTeamMemberHooks(e.withRegistries(ctx), e.config.TeamMembers.AfterDeleteHooks(), actor, member)
}
