package types

import (
	"context"
	"time"

	"github.com/Authula/authula/core/pagination"
	"github.com/Authula/authula/models"
)

type OrganizationsPluginConfig struct {
	Enabled                          bool          `json:"enabled" toml:"enabled"`
	OrganizationsLimit               *int          `json:"organizations_limit" toml:"organizations_limit"`
	MembersLimit                     *int          `json:"members_limit" toml:"members_limit"`
	InvitationsLimit                 *int          `json:"invitations_limit" toml:"invitations_limit"`
	MaxPageLimit                     *int          `json:"max_page_limit" toml:"max_page_limit"`
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
	if config.MaxPageLimit == nil || *config.MaxPageLimit <= 0 {
		config.MaxPageLimit = new(pagination.DefaultMaxLimit)
	}
	if config.InvitationExpiresIn == 0 {
		config.InvitationExpiresIn = 24 * time.Hour
	}
}

type OrganizationHook func(ctx context.Context, actor *models.Actor, organization *Organization) error

type OrganizationMemberHook func(ctx context.Context, actor *models.Actor, member *OrganizationMember) error

type OrganizationInvitationHook func(ctx context.Context, actor *models.Actor, invitation *OrganizationInvitation) error

type OrganizationTeamHook func(ctx context.Context, actor *models.Actor, team *OrganizationTeam) error

type OrganizationTeamMemberHook func(ctx context.Context, actor *models.Actor, member *OrganizationTeamMember) error

type OrganizationsServiceHooksConfig struct {
	Organizations *OrganizationServiceHooks
	Members       *OrganizationMemberServiceHooks
	Invitations   *OrganizationInvitationServiceHooks
	Teams         *OrganizationTeamServiceHooks
	TeamMembers   *OrganizationTeamMemberServiceHooks
}

type OrganizationServiceHooks struct {
	beforeCreate []OrganizationHook
	afterCreate  []OrganizationHook
	beforeUpdate []OrganizationHook
	afterUpdate  []OrganizationHook
	beforeDelete []OrganizationHook
	afterDelete  []OrganizationHook
}

func (h *OrganizationServiceHooks) RegisterBeforeCreate(fn OrganizationHook) {
	h.beforeCreate = append(h.beforeCreate, fn)
}

func (h *OrganizationServiceHooks) RegisterAfterCreate(fn OrganizationHook) {
	h.afterCreate = append(h.afterCreate, fn)
}

func (h *OrganizationServiceHooks) RegisterBeforeUpdate(fn OrganizationHook) {
	h.beforeUpdate = append(h.beforeUpdate, fn)
}

func (h *OrganizationServiceHooks) RegisterAfterUpdate(fn OrganizationHook) {
	h.afterUpdate = append(h.afterUpdate, fn)
}

func (h *OrganizationServiceHooks) RegisterBeforeDelete(fn OrganizationHook) {
	h.beforeDelete = append(h.beforeDelete, fn)
}

func (h *OrganizationServiceHooks) RegisterAfterDelete(fn OrganizationHook) {
	h.afterDelete = append(h.afterDelete, fn)
}

func (h *OrganizationServiceHooks) BeforeCreateHooks() []OrganizationHook {
	return h.beforeCreate
}

func (h *OrganizationServiceHooks) AfterCreateHooks() []OrganizationHook {
	return h.afterCreate
}

func (h *OrganizationServiceHooks) BeforeUpdateHooks() []OrganizationHook {
	return h.beforeUpdate
}

func (h *OrganizationServiceHooks) AfterUpdateHooks() []OrganizationHook {
	return h.afterUpdate
}

func (h *OrganizationServiceHooks) BeforeDeleteHooks() []OrganizationHook {
	return h.beforeDelete
}

func (h *OrganizationServiceHooks) AfterDeleteHooks() []OrganizationHook {
	return h.afterDelete
}

type OrganizationMemberServiceHooks struct {
	beforeCreate []OrganizationMemberHook
	afterCreate  []OrganizationMemberHook
	beforeUpdate []OrganizationMemberHook
	afterUpdate  []OrganizationMemberHook
	beforeDelete []OrganizationMemberHook
	afterDelete  []OrganizationMemberHook
}

func (h *OrganizationMemberServiceHooks) RegisterBeforeCreate(fn OrganizationMemberHook) {
	h.beforeCreate = append(h.beforeCreate, fn)
}

func (h *OrganizationMemberServiceHooks) RegisterAfterCreate(fn OrganizationMemberHook) {
	h.afterCreate = append(h.afterCreate, fn)
}

func (h *OrganizationMemberServiceHooks) RegisterBeforeUpdate(fn OrganizationMemberHook) {
	h.beforeUpdate = append(h.beforeUpdate, fn)
}

func (h *OrganizationMemberServiceHooks) RegisterAfterUpdate(fn OrganizationMemberHook) {
	h.afterUpdate = append(h.afterUpdate, fn)
}

func (h *OrganizationMemberServiceHooks) RegisterBeforeDelete(fn OrganizationMemberHook) {
	h.beforeDelete = append(h.beforeDelete, fn)
}

func (h *OrganizationMemberServiceHooks) RegisterAfterDelete(fn OrganizationMemberHook) {
	h.afterDelete = append(h.afterDelete, fn)
}

func (h *OrganizationMemberServiceHooks) BeforeCreateHooks() []OrganizationMemberHook {
	return h.beforeCreate
}

func (h *OrganizationMemberServiceHooks) AfterCreateHooks() []OrganizationMemberHook {
	return h.afterCreate
}

func (h *OrganizationMemberServiceHooks) BeforeUpdateHooks() []OrganizationMemberHook {
	return h.beforeUpdate
}

func (h *OrganizationMemberServiceHooks) AfterUpdateHooks() []OrganizationMemberHook {
	return h.afterUpdate
}

func (h *OrganizationMemberServiceHooks) BeforeDeleteHooks() []OrganizationMemberHook {
	return h.beforeDelete
}

func (h *OrganizationMemberServiceHooks) AfterDeleteHooks() []OrganizationMemberHook {
	return h.afterDelete
}

type OrganizationInvitationServiceHooks struct {
	beforeCreate []OrganizationInvitationHook
	afterCreate  []OrganizationInvitationHook
	beforeUpdate []OrganizationInvitationHook
	afterUpdate  []OrganizationInvitationHook
}

func (h *OrganizationInvitationServiceHooks) RegisterBeforeCreate(fn OrganizationInvitationHook) {
	h.beforeCreate = append(h.beforeCreate, fn)
}

func (h *OrganizationInvitationServiceHooks) RegisterAfterCreate(fn OrganizationInvitationHook) {
	h.afterCreate = append(h.afterCreate, fn)
}

func (h *OrganizationInvitationServiceHooks) RegisterBeforeUpdate(fn OrganizationInvitationHook) {
	h.beforeUpdate = append(h.beforeUpdate, fn)
}

func (h *OrganizationInvitationServiceHooks) RegisterAfterUpdate(fn OrganizationInvitationHook) {
	h.afterUpdate = append(h.afterUpdate, fn)
}

func (h *OrganizationInvitationServiceHooks) BeforeCreateHooks() []OrganizationInvitationHook {
	return h.beforeCreate
}

func (h *OrganizationInvitationServiceHooks) AfterCreateHooks() []OrganizationInvitationHook {
	return h.afterCreate
}

func (h *OrganizationInvitationServiceHooks) BeforeUpdateHooks() []OrganizationInvitationHook {
	return h.beforeUpdate
}

func (h *OrganizationInvitationServiceHooks) AfterUpdateHooks() []OrganizationInvitationHook {
	return h.afterUpdate
}

type OrganizationTeamServiceHooks struct {
	beforeCreate []OrganizationTeamHook
	afterCreate  []OrganizationTeamHook
	beforeUpdate []OrganizationTeamHook
	afterUpdate  []OrganizationTeamHook
	beforeDelete []OrganizationTeamHook
	afterDelete  []OrganizationTeamHook
}

func (h *OrganizationTeamServiceHooks) RegisterBeforeCreate(fn OrganizationTeamHook) {
	h.beforeCreate = append(h.beforeCreate, fn)
}

func (h *OrganizationTeamServiceHooks) RegisterAfterCreate(fn OrganizationTeamHook) {
	h.afterCreate = append(h.afterCreate, fn)
}

func (h *OrganizationTeamServiceHooks) RegisterBeforeUpdate(fn OrganizationTeamHook) {
	h.beforeUpdate = append(h.beforeUpdate, fn)
}

func (h *OrganizationTeamServiceHooks) RegisterAfterUpdate(fn OrganizationTeamHook) {
	h.afterUpdate = append(h.afterUpdate, fn)
}

func (h *OrganizationTeamServiceHooks) RegisterBeforeDelete(fn OrganizationTeamHook) {
	h.beforeDelete = append(h.beforeDelete, fn)
}

func (h *OrganizationTeamServiceHooks) RegisterAfterDelete(fn OrganizationTeamHook) {
	h.afterDelete = append(h.afterDelete, fn)
}

func (h *OrganizationTeamServiceHooks) BeforeCreateHooks() []OrganizationTeamHook {
	return h.beforeCreate
}

func (h *OrganizationTeamServiceHooks) AfterCreateHooks() []OrganizationTeamHook {
	return h.afterCreate
}

func (h *OrganizationTeamServiceHooks) BeforeUpdateHooks() []OrganizationTeamHook {
	return h.beforeUpdate
}

func (h *OrganizationTeamServiceHooks) AfterUpdateHooks() []OrganizationTeamHook {
	return h.afterUpdate
}

func (h *OrganizationTeamServiceHooks) BeforeDeleteHooks() []OrganizationTeamHook {
	return h.beforeDelete
}

func (h *OrganizationTeamServiceHooks) AfterDeleteHooks() []OrganizationTeamHook {
	return h.afterDelete
}

type OrganizationTeamMemberServiceHooks struct {
	beforeCreate []OrganizationTeamMemberHook
	afterCreate  []OrganizationTeamMemberHook
	beforeDelete []OrganizationTeamMemberHook
	afterDelete  []OrganizationTeamMemberHook
}

func (h *OrganizationTeamMemberServiceHooks) RegisterBeforeCreate(fn OrganizationTeamMemberHook) {
	h.beforeCreate = append(h.beforeCreate, fn)
}

func (h *OrganizationTeamMemberServiceHooks) RegisterAfterCreate(fn OrganizationTeamMemberHook) {
	h.afterCreate = append(h.afterCreate, fn)
}

func (h *OrganizationTeamMemberServiceHooks) RegisterBeforeDelete(fn OrganizationTeamMemberHook) {
	h.beforeDelete = append(h.beforeDelete, fn)
}

func (h *OrganizationTeamMemberServiceHooks) RegisterAfterDelete(fn OrganizationTeamMemberHook) {
	h.afterDelete = append(h.afterDelete, fn)
}

func (h *OrganizationTeamMemberServiceHooks) BeforeCreateHooks() []OrganizationTeamMemberHook {
	return h.beforeCreate
}

func (h *OrganizationTeamMemberServiceHooks) AfterCreateHooks() []OrganizationTeamMemberHook {
	return h.afterCreate
}

func (h *OrganizationTeamMemberServiceHooks) BeforeDeleteHooks() []OrganizationTeamMemberHook {
	return h.beforeDelete
}

func (h *OrganizationTeamMemberServiceHooks) AfterDeleteHooks() []OrganizationTeamMemberHook {
	return h.afterDelete
}

type SendOrganizationInvitationEmailParams struct {
	Organization *Organization
	Invitation   *OrganizationInvitation
	Inviter      *models.User
	InviteURL    string
}
