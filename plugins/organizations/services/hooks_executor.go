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

func (e *ServiceHookExecutor) BeforeCreateOrganization(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil || e.config.Organizations.BeforeCreate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Organizations.BeforeCreate(ctx, actor, organization)
}

func (e *ServiceHookExecutor) AfterCreateOrganization(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil || e.config.Organizations.AfterCreate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Organizations.AfterCreate(ctx, actor, organization)
}

func (e *ServiceHookExecutor) BeforeUpdateOrganization(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil || e.config.Organizations.BeforeUpdate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Organizations.BeforeUpdate(ctx, actor, organization)
}

func (e *ServiceHookExecutor) AfterUpdateOrganization(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil || e.config.Organizations.AfterUpdate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Organizations.AfterUpdate(ctx, actor, organization)
}

func (e *ServiceHookExecutor) BeforeDeleteOrganization(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil || e.config.Organizations.BeforeDelete == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Organizations.BeforeDelete(ctx, actor, organization)
}

func (e *ServiceHookExecutor) AfterDeleteOrganization(ctx context.Context, actor *models.Actor, organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil || e.config.Organizations.AfterDelete == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Organizations.AfterDelete(ctx, actor, organization)
}

func (e *ServiceHookExecutor) BeforeCreateOrganizationMember(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil || e.config.Members.BeforeCreate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Members.BeforeCreate(ctx, actor, member)
}

func (e *ServiceHookExecutor) AfterCreateOrganizationMember(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil || e.config.Members.AfterCreate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Members.AfterCreate(ctx, actor, member)
}

func (e *ServiceHookExecutor) BeforeUpdateOrganizationMember(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil || e.config.Members.BeforeUpdate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Members.BeforeUpdate(ctx, actor, member)
}

func (e *ServiceHookExecutor) AfterUpdateOrganizationMember(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil || e.config.Members.AfterUpdate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Members.AfterUpdate(ctx, actor, member)
}

func (e *ServiceHookExecutor) BeforeDeleteOrganizationMember(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil || e.config.Members.BeforeDelete == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Members.BeforeDelete(ctx, actor, member)
}

func (e *ServiceHookExecutor) AfterDeleteOrganizationMember(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil || e.config.Members.AfterDelete == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Members.AfterDelete(ctx, actor, member)
}

func (e *ServiceHookExecutor) BeforeCreateOrganizationInvitation(ctx context.Context, actor *models.Actor, invitation *types.OrganizationInvitation) error {
	if e == nil || e.config == nil || e.config.Invitations == nil || e.config.Invitations.BeforeCreate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Invitations.BeforeCreate(ctx, actor, invitation)
}

func (e *ServiceHookExecutor) AfterCreateOrganizationInvitation(ctx context.Context, actor *models.Actor, invitation *types.OrganizationInvitation) error {
	if e == nil || e.config == nil || e.config.Invitations == nil || e.config.Invitations.AfterCreate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Invitations.AfterCreate(ctx, actor, invitation)
}

func (e *ServiceHookExecutor) BeforeUpdateOrganizationInvitation(ctx context.Context, actor *models.Actor, invitation *types.OrganizationInvitation) error {
	if e == nil || e.config == nil || e.config.Invitations == nil || e.config.Invitations.BeforeUpdate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Invitations.BeforeUpdate(ctx, actor, invitation)
}

func (e *ServiceHookExecutor) AfterUpdateOrganizationInvitation(ctx context.Context, actor *models.Actor, invitation *types.OrganizationInvitation) error {
	if e == nil || e.config == nil || e.config.Invitations == nil || e.config.Invitations.AfterUpdate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Invitations.AfterUpdate(ctx, actor, invitation)
}

func (e *ServiceHookExecutor) BeforeCreateOrganizationTeam(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil || e.config.Teams.BeforeCreate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Teams.BeforeCreate(ctx, actor, team)
}

func (e *ServiceHookExecutor) AfterCreateOrganizationTeam(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil || e.config.Teams.AfterCreate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Teams.AfterCreate(ctx, actor, team)
}

func (e *ServiceHookExecutor) BeforeUpdateOrganizationTeam(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil || e.config.Teams.BeforeUpdate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Teams.BeforeUpdate(ctx, actor, team)
}

func (e *ServiceHookExecutor) AfterUpdateOrganizationTeam(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil || e.config.Teams.AfterUpdate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Teams.AfterUpdate(ctx, actor, team)
}

func (e *ServiceHookExecutor) BeforeDeleteOrganizationTeam(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil || e.config.Teams.BeforeDelete == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Teams.BeforeDelete(ctx, actor, team)
}

func (e *ServiceHookExecutor) AfterDeleteOrganizationTeam(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil || e.config.Teams.AfterDelete == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.Teams.AfterDelete(ctx, actor, team)
}

func (e *ServiceHookExecutor) BeforeCreateOrganizationTeamMember(ctx context.Context, actor *models.Actor, member *types.OrganizationTeamMember) error {
	if e == nil || e.config == nil || e.config.TeamMembers == nil || e.config.TeamMembers.BeforeCreate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.TeamMembers.BeforeCreate(ctx, actor, member)
}

func (e *ServiceHookExecutor) AfterCreateOrganizationTeamMember(ctx context.Context, actor *models.Actor, member *types.OrganizationTeamMember) error {
	if e == nil || e.config == nil || e.config.TeamMembers == nil || e.config.TeamMembers.AfterCreate == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.TeamMembers.AfterCreate(ctx, actor, member)
}

func (e *ServiceHookExecutor) BeforeDeleteOrganizationTeamMember(ctx context.Context, actor *models.Actor, member *types.OrganizationTeamMember) error {
	if e == nil || e.config == nil || e.config.TeamMembers == nil || e.config.TeamMembers.BeforeDelete == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.TeamMembers.BeforeDelete(ctx, actor, member)
}

func (e *ServiceHookExecutor) AfterDeleteOrganizationTeamMember(ctx context.Context, actor *models.Actor, member *types.OrganizationTeamMember) error {
	if e == nil || e.config == nil || e.config.TeamMembers == nil || e.config.TeamMembers.AfterDelete == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.TeamMembers.AfterDelete(ctx, actor, member)
}
