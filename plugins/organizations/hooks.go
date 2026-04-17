package organizations

import "github.com/Authula/authula/plugins/organizations/types"

type OrganizationsHookExecutor struct {
	config *types.OrganizationsDatabaseHooksConfig
}

func NewOrganizationsHookExecutor(config *types.OrganizationsDatabaseHooksConfig) *OrganizationsHookExecutor {
	return &OrganizationsHookExecutor{config: config}
}

// Organization Hooks

func (e *OrganizationsHookExecutor) BeforeCreateOrganization(organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil || e.config.Organizations.BeforeCreate == nil {
		return nil
	}
	return e.config.Organizations.BeforeCreate(organization)
}

func (e *OrganizationsHookExecutor) AfterCreateOrganization(organization types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil || e.config.Organizations.AfterCreate == nil {
		return nil
	}
	return e.config.Organizations.AfterCreate(organization)
}

func (e *OrganizationsHookExecutor) BeforeUpdateOrganization(organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil || e.config.Organizations.BeforeUpdate == nil {
		return nil
	}
	return e.config.Organizations.BeforeUpdate(organization)
}

func (e *OrganizationsHookExecutor) AfterUpdateOrganization(organization types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil || e.config.Organizations.AfterUpdate == nil {
		return nil
	}
	return e.config.Organizations.AfterUpdate(organization)
}

func (e *OrganizationsHookExecutor) BeforeDeleteOrganization(organization *types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil || e.config.Organizations.BeforeDelete == nil {
		return nil
	}
	return e.config.Organizations.BeforeDelete(organization)
}

func (e *OrganizationsHookExecutor) AfterDeleteOrganization(organization types.Organization) error {
	if e == nil || e.config == nil || e.config.Organizations == nil || e.config.Organizations.AfterDelete == nil {
		return nil
	}
	return e.config.Organizations.AfterDelete(organization)
}

// Organization Member Hooks

func (e *OrganizationsHookExecutor) BeforeCreateOrganizationMember(member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil || e.config.Members.BeforeCreate == nil {
		return nil
	}
	return e.config.Members.BeforeCreate(member)
}

func (e *OrganizationsHookExecutor) AfterCreateOrganizationMember(member types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil || e.config.Members.AfterCreate == nil {
		return nil
	}
	return e.config.Members.AfterCreate(member)
}

func (e *OrganizationsHookExecutor) BeforeUpdateOrganizationMember(member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil || e.config.Members.BeforeUpdate == nil {
		return nil
	}
	return e.config.Members.BeforeUpdate(member)
}

func (e *OrganizationsHookExecutor) AfterUpdateOrganizationMember(member types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil || e.config.Members.AfterUpdate == nil {
		return nil
	}
	return e.config.Members.AfterUpdate(member)
}

func (e *OrganizationsHookExecutor) BeforeDeleteOrganizationMember(member *types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil || e.config.Members.BeforeDelete == nil {
		return nil
	}
	return e.config.Members.BeforeDelete(member)
}

func (e *OrganizationsHookExecutor) AfterDeleteOrganizationMember(member types.OrganizationMember) error {
	if e == nil || e.config == nil || e.config.Members == nil || e.config.Members.AfterDelete == nil {
		return nil
	}
	return e.config.Members.AfterDelete(member)
}

// Organization Invitation Hooks

func (e *OrganizationsHookExecutor) BeforeCreateOrganizationInvitation(invitation *types.OrganizationInvitation) error {
	if e == nil || e.config == nil || e.config.Invitations == nil || e.config.Invitations.BeforeCreate == nil {
		return nil
	}
	return e.config.Invitations.BeforeCreate(invitation)
}

func (e *OrganizationsHookExecutor) AfterCreateOrganizationInvitation(invitation types.OrganizationInvitation) error {
	if e == nil || e.config == nil || e.config.Invitations == nil || e.config.Invitations.AfterCreate == nil {
		return nil
	}
	return e.config.Invitations.AfterCreate(invitation)
}

func (e *OrganizationsHookExecutor) BeforeUpdateOrganizationInvitation(invitation *types.OrganizationInvitation) error {
	if e == nil || e.config == nil || e.config.Invitations == nil || e.config.Invitations.BeforeUpdate == nil {
		return nil
	}
	return e.config.Invitations.BeforeUpdate(invitation)
}

func (e *OrganizationsHookExecutor) AfterUpdateOrganizationInvitation(invitation types.OrganizationInvitation) error {
	if e == nil || e.config == nil || e.config.Invitations == nil || e.config.Invitations.AfterUpdate == nil {
		return nil
	}
	return e.config.Invitations.AfterUpdate(invitation)
}

// Organization Team Hooks

func (e *OrganizationsHookExecutor) BeforeCreateOrganizationTeam(team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil || e.config.Teams.BeforeCreate == nil {
		return nil
	}
	return e.config.Teams.BeforeCreate(team)
}

func (e *OrganizationsHookExecutor) AfterCreateOrganizationTeam(team types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil || e.config.Teams.AfterCreate == nil {
		return nil
	}
	return e.config.Teams.AfterCreate(team)
}

func (e *OrganizationsHookExecutor) BeforeUpdateOrganizationTeam(team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil || e.config.Teams.BeforeUpdate == nil {
		return nil
	}
	return e.config.Teams.BeforeUpdate(team)
}

func (e *OrganizationsHookExecutor) AfterUpdateOrganizationTeam(team types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil || e.config.Teams.AfterUpdate == nil {
		return nil
	}
	return e.config.Teams.AfterUpdate(team)
}

func (e *OrganizationsHookExecutor) BeforeDeleteOrganizationTeam(team *types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil || e.config.Teams.BeforeDelete == nil {
		return nil
	}
	return e.config.Teams.BeforeDelete(team)
}

func (e *OrganizationsHookExecutor) AfterDeleteOrganizationTeam(team types.OrganizationTeam) error {
	if e == nil || e.config == nil || e.config.Teams == nil || e.config.Teams.AfterDelete == nil {
		return nil
	}
	return e.config.Teams.AfterDelete(team)
}

// Organization Team Member Hooks

func (e *OrganizationsHookExecutor) BeforeCreateOrganizationTeamMember(member *types.OrganizationTeamMember) error {
	if e == nil || e.config == nil || e.config.TeamMembers == nil || e.config.TeamMembers.BeforeCreate == nil {
		return nil
	}
	return e.config.TeamMembers.BeforeCreate(member)
}

func (e *OrganizationsHookExecutor) AfterCreateOrganizationTeamMember(member types.OrganizationTeamMember) error {
	if e == nil || e.config == nil || e.config.TeamMembers == nil || e.config.TeamMembers.AfterCreate == nil {
		return nil
	}
	return e.config.TeamMembers.AfterCreate(member)
}

func (e *OrganizationsHookExecutor) BeforeDeleteOrganizationTeamMember(member *types.OrganizationTeamMember) error {
	if e == nil || e.config == nil || e.config.TeamMembers == nil || e.config.TeamMembers.BeforeDelete == nil {
		return nil
	}
	return e.config.TeamMembers.BeforeDelete(member)
}

func (e *OrganizationsHookExecutor) AfterDeleteOrganizationTeamMember(member types.OrganizationTeamMember) error {
	if e == nil || e.config == nil || e.config.TeamMembers == nil || e.config.TeamMembers.AfterDelete == nil {
		return nil
	}
	return e.config.TeamMembers.AfterDelete(member)
}
