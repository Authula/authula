package tests

import (
	"github.com/Authula/authula/plugins/organizations/types"
)

type MockOrganizationHooks struct {
	BeforeCreate func(*types.Organization) error
	AfterCreate  func(types.Organization) error
	BeforeUpdate func(*types.Organization) error
	AfterUpdate  func(types.Organization) error
	BeforeDelete func(*types.Organization) error
	AfterDelete  func(types.Organization) error
}

func (h *MockOrganizationHooks) BeforeCreateOrganization(organization *types.Organization) error {
	if h.BeforeCreate == nil {
		return nil
	}
	return h.BeforeCreate(organization)
}

func (h *MockOrganizationHooks) AfterCreateOrganization(organization types.Organization) error {
	if h.AfterCreate == nil {
		return nil
	}
	return h.AfterCreate(organization)
}

func (h *MockOrganizationHooks) BeforeUpdateOrganization(organization *types.Organization) error {
	if h.BeforeUpdate == nil {
		return nil
	}
	return h.BeforeUpdate(organization)
}

func (h *MockOrganizationHooks) AfterUpdateOrganization(organization types.Organization) error {
	if h.AfterUpdate == nil {
		return nil
	}
	return h.AfterUpdate(organization)
}

func (h *MockOrganizationHooks) BeforeDeleteOrganization(organization *types.Organization) error {
	if h.BeforeDelete == nil {
		return nil
	}
	return h.BeforeDelete(organization)
}

func (h *MockOrganizationHooks) AfterDeleteOrganization(organization types.Organization) error {
	if h.AfterDelete == nil {
		return nil
	}
	return h.AfterDelete(organization)
}

type MockOrganizationMemberHooks struct {
	Before       func(*types.OrganizationMember) error
	After        func(types.OrganizationMember) error
	BeforeUpdate func(*types.OrganizationMember) error
	AfterUpdate  func(types.OrganizationMember) error
	BeforeDelete func(*types.OrganizationMember) error
	AfterDelete  func(types.OrganizationMember) error
}

func (h *MockOrganizationMemberHooks) BeforeCreateOrganizationMember(member *types.OrganizationMember) error {
	if h.Before == nil {
		return nil
	}
	return h.Before(member)
}

func (h *MockOrganizationMemberHooks) AfterCreateOrganizationMember(member types.OrganizationMember) error {
	if h.After == nil {
		return nil
	}
	return h.After(member)
}

func (h *MockOrganizationMemberHooks) BeforeUpdateOrganizationMember(member *types.OrganizationMember) error {
	if h.BeforeUpdate == nil {
		return nil
	}
	return h.BeforeUpdate(member)
}

func (h *MockOrganizationMemberHooks) AfterUpdateOrganizationMember(member types.OrganizationMember) error {
	if h.AfterUpdate == nil {
		return nil
	}
	return h.AfterUpdate(member)
}

func (h *MockOrganizationMemberHooks) BeforeDeleteOrganizationMember(member *types.OrganizationMember) error {
	if h.BeforeDelete == nil {
		return nil
	}
	return h.BeforeDelete(member)
}

func (h *MockOrganizationMemberHooks) AfterDeleteOrganizationMember(member types.OrganizationMember) error {
	if h.AfterDelete == nil {
		return nil
	}
	return h.AfterDelete(member)
}

type MockOrganizationInvitationHooks struct {
	Before       func(*types.OrganizationInvitation) error
	After        func(types.OrganizationInvitation) error
	BeforeUpdate func(*types.OrganizationInvitation) error
	AfterUpdate  func(types.OrganizationInvitation) error
}

func (h *MockOrganizationInvitationHooks) BeforeCreateOrganizationInvitation(invitation *types.OrganizationInvitation) error {
	if h.Before == nil {
		return nil
	}
	return h.Before(invitation)
}

func (h *MockOrganizationInvitationHooks) AfterCreateOrganizationInvitation(invitation types.OrganizationInvitation) error {
	if h.After == nil {
		return nil
	}
	return h.After(invitation)
}

func (h *MockOrganizationInvitationHooks) BeforeUpdateOrganizationInvitation(invitation *types.OrganizationInvitation) error {
	if h.BeforeUpdate == nil {
		return nil
	}
	return h.BeforeUpdate(invitation)
}

func (h *MockOrganizationInvitationHooks) AfterUpdateOrganizationInvitation(invitation types.OrganizationInvitation) error {
	if h.AfterUpdate == nil {
		return nil
	}
	return h.AfterUpdate(invitation)
}

type MockOrganizationTeamHooks struct {
	BeforeCreate func(*types.OrganizationTeam) error
	AfterCreate  func(types.OrganizationTeam) error
	BeforeUpdate func(*types.OrganizationTeam) error
	AfterUpdate  func(types.OrganizationTeam) error
	BeforeDelete func(*types.OrganizationTeam) error
	AfterDelete  func(types.OrganizationTeam) error
}

func (h *MockOrganizationTeamHooks) BeforeCreateOrganizationTeam(team *types.OrganizationTeam) error {
	if h.BeforeCreate == nil {
		return nil
	}
	return h.BeforeCreate(team)
}

func (h *MockOrganizationTeamHooks) AfterCreateOrganizationTeam(team types.OrganizationTeam) error {
	if h.AfterCreate == nil {
		return nil
	}
	return h.AfterCreate(team)
}

func (h *MockOrganizationTeamHooks) BeforeUpdateOrganizationTeam(team *types.OrganizationTeam) error {
	if h.BeforeUpdate == nil {
		return nil
	}
	return h.BeforeUpdate(team)
}

func (h *MockOrganizationTeamHooks) AfterUpdateOrganizationTeam(team types.OrganizationTeam) error {
	if h.AfterUpdate == nil {
		return nil
	}
	return h.AfterUpdate(team)
}

func (h *MockOrganizationTeamHooks) BeforeDeleteOrganizationTeam(team *types.OrganizationTeam) error {
	if h.BeforeDelete == nil {
		return nil
	}
	return h.BeforeDelete(team)
}

func (h *MockOrganizationTeamHooks) AfterDeleteOrganizationTeam(team types.OrganizationTeam) error {
	if h.AfterDelete == nil {
		return nil
	}
	return h.AfterDelete(team)
}

type MockOrganizationTeamMemberHooks struct {
	BeforeCreate func(*types.OrganizationTeamMember) error
	AfterCreate  func(types.OrganizationTeamMember) error
	BeforeDelete func(*types.OrganizationTeamMember) error
	AfterDelete  func(types.OrganizationTeamMember) error
}

func (h *MockOrganizationTeamMemberHooks) BeforeCreateOrganizationTeamMember(teamMember *types.OrganizationTeamMember) error {
	if h.BeforeCreate == nil {
		return nil
	}
	return h.BeforeCreate(teamMember)
}

func (h *MockOrganizationTeamMemberHooks) AfterCreateOrganizationTeamMember(teamMember types.OrganizationTeamMember) error {
	if h.AfterCreate == nil {
		return nil
	}
	return h.AfterCreate(teamMember)
}

func (h *MockOrganizationTeamMemberHooks) BeforeDeleteOrganizationTeamMember(teamMember *types.OrganizationTeamMember) error {
	if h.BeforeDelete == nil {
		return nil
	}
	return h.BeforeDelete(teamMember)
}

func (h *MockOrganizationTeamMemberHooks) AfterDeleteOrganizationTeamMember(teamMember types.OrganizationTeamMember) error {
	if h.AfterDelete == nil {
		return nil
	}
	return h.AfterDelete(teamMember)
}
