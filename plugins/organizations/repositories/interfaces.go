package repositories

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/types"
)

type OrganizationHookExecutor interface {
	BeforeCreateOrganization(organization *types.Organization) error
	AfterCreateOrganization(organization types.Organization) error
	BeforeUpdateOrganization(organization *types.Organization) error
	AfterUpdateOrganization(organization types.Organization) error
	BeforeDeleteOrganization(organization *types.Organization) error
	AfterDeleteOrganization(organization types.Organization) error
}

type OrganizationInvitationHookExecutor interface {
	BeforeCreateOrganizationInvitation(invitation *types.OrganizationInvitation) error
	AfterCreateOrganizationInvitation(invitation types.OrganizationInvitation) error
	BeforeUpdateOrganizationInvitation(invitation *types.OrganizationInvitation) error
	AfterUpdateOrganizationInvitation(invitation types.OrganizationInvitation) error
}

type OrganizationMemberHookExecutor interface {
	BeforeCreateOrganizationMember(member *types.OrganizationMember) error
	AfterCreateOrganizationMember(member types.OrganizationMember) error
	BeforeUpdateOrganizationMember(member *types.OrganizationMember) error
	AfterUpdateOrganizationMember(member types.OrganizationMember) error
	BeforeDeleteOrganizationMember(member *types.OrganizationMember) error
	AfterDeleteOrganizationMember(member types.OrganizationMember) error
}

type OrganizationTeamHookExecutor interface {
	BeforeCreateOrganizationTeam(team *types.OrganizationTeam) error
	AfterCreateOrganizationTeam(team types.OrganizationTeam) error
	BeforeUpdateOrganizationTeam(team *types.OrganizationTeam) error
	AfterUpdateOrganizationTeam(team types.OrganizationTeam) error
	BeforeDeleteOrganizationTeam(team *types.OrganizationTeam) error
	AfterDeleteOrganizationTeam(team types.OrganizationTeam) error
}

type OrganizationTeamMemberHookExecutor interface {
	BeforeCreateOrganizationTeamMember(teamMember *types.OrganizationTeamMember) error
	AfterCreateOrganizationTeamMember(teamMember types.OrganizationTeamMember) error
	BeforeDeleteOrganizationTeamMember(teamMember *types.OrganizationTeamMember) error
	AfterDeleteOrganizationTeamMember(teamMember types.OrganizationTeamMember) error
}

type OrganizationRepository interface {
	Create(ctx context.Context, organization *types.Organization) (*types.Organization, error)
	GetByID(ctx context.Context, organizationID string) (*types.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*types.Organization, error)
	GetAllByOwnerID(ctx context.Context, ownerID string) ([]types.Organization, error)
	Update(ctx context.Context, organization *types.Organization) (*types.Organization, error)
	Delete(ctx context.Context, organizationID string) error
	WithTx(tx bun.IDB) OrganizationRepository
}

type OrganizationInvitationRepository interface {
	Create(ctx context.Context, invitation *types.OrganizationInvitation) (*types.OrganizationInvitation, error)
	GetByID(ctx context.Context, invitationID string) (*types.OrganizationInvitation, error)
	GetByOrganizationIDAndEmail(ctx context.Context, organizationID string, email string, status ...types.OrganizationInvitationStatus) (*types.OrganizationInvitation, error)
	GetAllByOrganizationID(ctx context.Context, organizationID string) ([]types.OrganizationInvitation, error)
	GetAllPendingByEmail(ctx context.Context, email string) ([]types.OrganizationInvitation, error)
	Update(ctx context.Context, invitation *types.OrganizationInvitation) (*types.OrganizationInvitation, error)
	CountByOrganizationIDAndEmail(ctx context.Context, organizationID string, email string) (int, error)
	WithTx(tx bun.IDB) OrganizationInvitationRepository
}

type OrganizationMemberRepository interface {
	Create(ctx context.Context, member *types.OrganizationMember) (*types.OrganizationMember, error)
	CountByOrganizationID(ctx context.Context, organizationID string) (int, error)
	GetAllByOrganizationID(ctx context.Context, organizationID string, page int, limit int) ([]types.OrganizationMember, error)
	GetByID(ctx context.Context, memberID string) (*types.OrganizationMember, error)
	GetByOrganizationIDAndUserID(ctx context.Context, organizationID string, userID string) (*types.OrganizationMember, error)
	GetAllByUserID(ctx context.Context, userID string) ([]types.OrganizationMember, error)
	Update(ctx context.Context, member *types.OrganizationMember) (*types.OrganizationMember, error)
	Delete(ctx context.Context, memberID string) error
	WithTx(tx bun.IDB) OrganizationMemberRepository
}

type OrganizationTeamRepository interface {
	Create(ctx context.Context, team *types.OrganizationTeam) (*types.OrganizationTeam, error)
	GetByID(ctx context.Context, teamID string) (*types.OrganizationTeam, error)
	GetByOrganizationIDAndSlug(ctx context.Context, organizationID, slug string) (*types.OrganizationTeam, error)
	GetAllByOrganizationID(ctx context.Context, organizationID string) ([]types.OrganizationTeam, error)
	Update(ctx context.Context, team *types.OrganizationTeam) (*types.OrganizationTeam, error)
	Delete(ctx context.Context, teamID string) error
	WithTx(tx bun.IDB) OrganizationTeamRepository
}

type OrganizationTeamMemberRepository interface {
	Create(ctx context.Context, teamMember *types.OrganizationTeamMember) (*types.OrganizationTeamMember, error)
	GetByID(ctx context.Context, teamMemberID string) (*types.OrganizationTeamMember, error)
	GetByTeamIDAndMemberID(ctx context.Context, teamID, memberID string) (*types.OrganizationTeamMember, error)
	GetAllByTeamID(ctx context.Context, teamID string, page int, limit int) ([]types.OrganizationTeamMember, error)
	DeleteByTeamIDAndMemberID(ctx context.Context, teamID, memberID string) error
	WithTx(tx bun.IDB) OrganizationTeamMemberRepository
}
