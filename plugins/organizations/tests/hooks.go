package tests

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/types"
)

type MockOrganizationHooks struct {
	BeforeCreate func(ctx context.Context, actor *models.Actor, organization *types.Organization) error
	AfterCreate  func(ctx context.Context, actor *models.Actor, organization *types.Organization) error
	BeforeUpdate func(ctx context.Context, actor *models.Actor, organization *types.Organization) error
	AfterUpdate  func(ctx context.Context, actor *models.Actor, organization *types.Organization) error
	BeforeDelete func(ctx context.Context, actor *models.Actor, organization *types.Organization) error
	AfterDelete  func(ctx context.Context, actor *models.Actor, organization *types.Organization) error
}

type MockOrganizationMemberHooks struct {
	BeforeCreate func(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error
	AfterCreate  func(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error
	BeforeUpdate func(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error
	AfterUpdate  func(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error
	BeforeDelete func(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error
	AfterDelete  func(ctx context.Context, actor *models.Actor, member *types.OrganizationMember) error
}

type MockOrganizationInvitationHooks struct {
	BeforeCreate func(ctx context.Context, actor *models.Actor, invitation *types.OrganizationInvitation) error
	AfterCreate  func(ctx context.Context, actor *models.Actor, invitation *types.OrganizationInvitation) error
	BeforeUpdate func(ctx context.Context, actor *models.Actor, invitation *types.OrganizationInvitation) error
	AfterUpdate  func(ctx context.Context, actor *models.Actor, invitation *types.OrganizationInvitation) error
}

type MockOrganizationTeamHooks struct {
	BeforeCreate func(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error
	AfterCreate  func(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error
	BeforeUpdate func(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error
	AfterUpdate  func(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error
	BeforeDelete func(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error
	AfterDelete  func(ctx context.Context, actor *models.Actor, team *types.OrganizationTeam) error
}

type MockOrganizationTeamMemberHooks struct {
	BeforeCreate func(ctx context.Context, actor *models.Actor, member *types.OrganizationTeamMember) error
	AfterCreate  func(ctx context.Context, actor *models.Actor, member *types.OrganizationTeamMember) error
}
