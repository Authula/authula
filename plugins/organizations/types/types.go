package types

import (
	"time"

	"github.com/Authula/authula/models"
)

type OrganizationsPluginConfig struct {
	Enabled                          bool          `json:"enabled" toml:"enabled"`
	OrganizationsLimit               *int          `json:"organizations_limit" toml:"organizations_limit"`
	MembersLimit                     *int          `json:"members_limit" toml:"members_limit"`
	InvitationsLimit                 *int          `json:"invitations_limit" toml:"invitations_limit"`
	InvitationExpiresIn              time.Duration `json:"invitation_expires_in" toml:"invitation_expires_in"`
	RequireEmailVerifiedOnInvitation bool          `json:"require_email_verified_on_invitation" toml:"require_email_verified_on_invitation"`

	SendOrganizationInvitationEmail func(organization *Organization, invitation *OrganizationInvitation, inviter *models.User) error `json:"-" toml:"-"`
	DatabaseHooks                   *OrganizationsDatabaseHooksConfig                                                                `json:"-" toml:"-"`
}

func (config *OrganizationsPluginConfig) ApplyDefaults() {
	if config.MembersLimit == nil {
		config.MembersLimit = new(100)
	}
	if config.InvitationsLimit == nil || *config.InvitationsLimit <= 0 {
		config.InvitationsLimit = new(100)
	}
	if config.InvitationExpiresIn == 0 {
		config.InvitationExpiresIn = 24 * time.Hour
	}
}

type OrganizationsDatabaseHooksConfig struct {
	Organizations *OrganizationDatabaseHooksConfig
	Members       *OrganizationMemberDatabaseHooksConfig
	Invitations   *OrganizationInvitationDatabaseHooksConfig
	Teams         *OrganizationTeamDatabaseHooksConfig
	TeamMembers   *OrganizationTeamMemberDatabaseHooksConfig
}

type OrganizationDatabaseHooksConfig struct {
	BeforeCreate func(organization *Organization) error
	AfterCreate  func(organization Organization) error
	BeforeUpdate func(organization *Organization) error
	AfterUpdate  func(organization Organization) error
	BeforeDelete func(organization *Organization) error
	AfterDelete  func(organization Organization) error
}

type OrganizationMemberDatabaseHooksConfig struct {
	BeforeCreate func(member *OrganizationMember) error
	AfterCreate  func(member OrganizationMember) error
	BeforeUpdate func(member *OrganizationMember) error
	AfterUpdate  func(member OrganizationMember) error
	BeforeDelete func(member *OrganizationMember) error
	AfterDelete  func(member OrganizationMember) error
}

type OrganizationInvitationDatabaseHooksConfig struct {
	BeforeCreate func(invitation *OrganizationInvitation) error
	AfterCreate  func(invitation OrganizationInvitation) error
	BeforeUpdate func(invitation *OrganizationInvitation) error
	AfterUpdate  func(invitation OrganizationInvitation) error
	BeforeDelete func(invitation *OrganizationInvitation) error
	AfterDelete  func(invitation OrganizationInvitation) error
}

type OrganizationTeamDatabaseHooksConfig struct {
	BeforeCreate func(team *OrganizationTeam) error
	AfterCreate  func(team OrganizationTeam) error
	BeforeUpdate func(team *OrganizationTeam) error
	AfterUpdate  func(team OrganizationTeam) error
	BeforeDelete func(team *OrganizationTeam) error
	AfterDelete  func(team OrganizationTeam) error
}

type OrganizationTeamMemberDatabaseHooksConfig struct {
	BeforeCreate func(member *OrganizationTeamMember) error
	AfterCreate  func(member OrganizationTeamMember) error
	BeforeDelete func(member *OrganizationTeamMember) error
	AfterDelete  func(member OrganizationTeamMember) error
}
