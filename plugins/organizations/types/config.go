package types

import (
	"context"
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

	ServiceHooks                    *OrganizationsServiceHooksConfig                                                        `json:"-" toml:"-"`
	SendOrganizationInvitationEmail func(params SendOrganizationInvitationEmailParams, reqCtx *models.RequestContext) error `json:"-" toml:"-"`
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

type OrganizationsServiceHooksConfig struct {
	Organizations *OrganizationServiceHooksConfig
	Members       *OrganizationMemberServiceHooksConfig
	Invitations   *OrganizationInvitationServiceHooksConfig
	Teams         *OrganizationTeamServiceHooksConfig
	TeamMembers   *OrganizationTeamMemberServiceHooksConfig
}

type OrganizationServiceHooksConfig struct {
	BeforeCreate func(ctx context.Context, actor *models.Actor, organization *Organization) error
	AfterCreate  func(ctx context.Context, actor *models.Actor, organization *Organization) error
	BeforeUpdate func(ctx context.Context, actor *models.Actor, organization *Organization) error
	AfterUpdate  func(ctx context.Context, actor *models.Actor, organization *Organization) error
	BeforeDelete func(ctx context.Context, actor *models.Actor, organization *Organization) error
	AfterDelete  func(ctx context.Context, actor *models.Actor, organization *Organization) error
}

type OrganizationMemberServiceHooksConfig struct {
	BeforeCreate func(ctx context.Context, actor *models.Actor, member *OrganizationMember) error
	AfterCreate  func(ctx context.Context, actor *models.Actor, member *OrganizationMember) error
	BeforeUpdate func(ctx context.Context, actor *models.Actor, member *OrganizationMember) error
	AfterUpdate  func(ctx context.Context, actor *models.Actor, member *OrganizationMember) error
	BeforeDelete func(ctx context.Context, actor *models.Actor, member *OrganizationMember) error
	AfterDelete  func(ctx context.Context, actor *models.Actor, member *OrganizationMember) error
}

type OrganizationInvitationServiceHooksConfig struct {
	BeforeCreate func(ctx context.Context, actor *models.Actor, invitation *OrganizationInvitation) error
	AfterCreate  func(ctx context.Context, actor *models.Actor, invitation *OrganizationInvitation) error
	BeforeUpdate func(ctx context.Context, actor *models.Actor, invitation *OrganizationInvitation) error
	AfterUpdate  func(ctx context.Context, actor *models.Actor, invitation *OrganizationInvitation) error
}

type OrganizationTeamServiceHooksConfig struct {
	BeforeCreate func(ctx context.Context, actor *models.Actor, team *OrganizationTeam) error
	AfterCreate  func(ctx context.Context, actor *models.Actor, team *OrganizationTeam) error
	BeforeUpdate func(ctx context.Context, actor *models.Actor, team *OrganizationTeam) error
	AfterUpdate  func(ctx context.Context, actor *models.Actor, team *OrganizationTeam) error
	BeforeDelete func(ctx context.Context, actor *models.Actor, team *OrganizationTeam) error
	AfterDelete  func(ctx context.Context, actor *models.Actor, team *OrganizationTeam) error
}

type OrganizationTeamMemberServiceHooksConfig struct {
	BeforeCreate func(ctx context.Context, actor *models.Actor, member *OrganizationTeamMember) error
	AfterCreate  func(ctx context.Context, actor *models.Actor, member *OrganizationTeamMember) error
	BeforeDelete func(ctx context.Context, actor *models.Actor, member *OrganizationTeamMember) error
	AfterDelete  func(ctx context.Context, actor *models.Actor, member *OrganizationTeamMember) error
}

type SendOrganizationInvitationEmailParams struct {
	Organization *Organization
	Invitation   *OrganizationInvitation
	Inviter      *models.User
	InviteURL    string
}
