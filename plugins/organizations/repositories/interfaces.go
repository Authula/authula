package repositories

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/types"
)

type OrganizationRepository interface {
	Create(ctx context.Context, organization *types.Organization) (*types.Organization, error)
	GetByID(ctx context.Context, organizationID string) (*types.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*types.Organization, error)
	ListAllAccessibleByUserID(ctx context.Context, userID string, page int, limit int) ([]types.Organization, int, error)
	GetAllAccessibleByUserID(ctx context.Context, userID string) ([]types.Organization, error)
	// GetAll returns every organization in the system. It applies no access
	// filtering, so callers are responsible for authorizing the request.
	GetAll(ctx context.Context) ([]types.Organization, error)
	CountAccessibleByUserID(ctx context.Context, userID string) (int, error)
	Update(ctx context.Context, organization *types.Organization) (*types.Organization, error)
	Delete(ctx context.Context, organizationID string) error
	WithTx(tx bun.IDB) OrganizationRepository
}

type OrganizationInvitationRepository interface {
	Create(ctx context.Context, invitation *types.OrganizationInvitation) (*types.OrganizationInvitation, error)
	GetByID(ctx context.Context, invitationID string) (*types.OrganizationInvitation, error)
	GetByIDWithOrg(ctx context.Context, invitationID string) (*types.GetOrganizationInvitationResponse, error)
	GetByOrganizationIDAndEmail(ctx context.Context, organizationID string, email string, status ...types.OrganizationInvitationStatus) (*types.OrganizationInvitation, error)
	ListAllByOrganizationIDWithOrg(ctx context.Context, organizationID string, page int, limit int) ([]types.GetOrganizationInvitationResponse, int, error)
	GetAllByOrganizationIDWithOrg(ctx context.Context, organizationID string) ([]types.GetOrganizationInvitationResponse, error)
	ListAllPendingByEmail(ctx context.Context, email string, page int, limit int) ([]types.OrganizationInvitation, int, error)
	GetAllPendingByEmail(ctx context.Context, email string) ([]types.OrganizationInvitation, error)
	Update(ctx context.Context, invitation *types.OrganizationInvitation) (*types.OrganizationInvitation, error)
	CountByOrganizationIDAndEmail(ctx context.Context, organizationID string, email string) (int, error)
	WithTx(tx bun.IDB) OrganizationInvitationRepository
}

type OrganizationMemberRepository interface {
	Create(ctx context.Context, member *types.OrganizationMember) (*types.OrganizationMember, error)
	CountByOrganizationID(ctx context.Context, organizationID string) (int, error)
	ListAllByOrganizationID(ctx context.Context, organizationID string, page int, limit int) ([]types.OrganizationMember, int, error)
	GetAllByOrganizationID(ctx context.Context, organizationID string) ([]types.OrganizationMember, error)
	ListAllByOrganizationIDWithUser(ctx context.Context, organizationID string, page int, limit int) ([]types.OrganizationMemberResponse, int, error)
	GetAllByOrganizationIDWithUser(ctx context.Context, organizationID string) ([]types.OrganizationMemberResponse, error)
	GetByID(ctx context.Context, memberID string) (*types.OrganizationMember, error)
	GetByIDWithUser(ctx context.Context, memberID string) (*types.OrganizationMemberResponse, error)
	GetByOrganizationIDAndUserID(ctx context.Context, organizationID string, userID string) (*types.OrganizationMember, error)
	GetByOrganizationIDAndUserIDWithUser(ctx context.Context, organizationID string, userID string) (*types.OrganizationMemberResponse, error)
	Update(ctx context.Context, member *types.OrganizationMember) (*types.OrganizationMember, error)
	Delete(ctx context.Context, memberID string) error
	WithTx(tx bun.IDB) OrganizationMemberRepository
}

type OrganizationTeamRepository interface {
	Create(ctx context.Context, team *types.OrganizationTeam) (*types.OrganizationTeam, error)
	GetByID(ctx context.Context, teamID string) (*types.OrganizationTeam, error)
	GetByOrganizationIDAndSlug(ctx context.Context, organizationID, slug string) (*types.OrganizationTeam, error)
	ListAllByOrganizationID(ctx context.Context, organizationID string, page int, limit int) ([]types.OrganizationTeam, int, error)
	GetAllByOrganizationID(ctx context.Context, organizationID string) ([]types.OrganizationTeam, error)
	Update(ctx context.Context, team *types.OrganizationTeam) (*types.OrganizationTeam, error)
	Delete(ctx context.Context, teamID string) error
	WithTx(tx bun.IDB) OrganizationTeamRepository
}

type OrganizationTeamMemberRepository interface {
	Create(ctx context.Context, teamMember *types.OrganizationTeamMember) (*types.OrganizationTeamMember, error)
	GetByID(ctx context.Context, teamMemberID string) (*types.OrganizationTeamMember, error)
	GetByTeamIDAndMemberID(ctx context.Context, teamID, memberID string) (*types.OrganizationTeamMember, error)
	ListAllByTeamID(ctx context.Context, teamID string, page int, limit int) ([]types.OrganizationTeamMember, int, error)
	GetAllByTeamID(ctx context.Context, teamID string) ([]types.OrganizationTeamMember, error)
	ListAllByTeamIDWithMemberAndUser(ctx context.Context, teamID string, page int, limit int) ([]types.OrganizationTeamMemberResponse, int, error)
	GetAllByTeamIDWithMemberAndUser(ctx context.Context, teamID string) ([]types.OrganizationTeamMemberResponse, error)
	GetByIDWithMemberAndUser(ctx context.Context, teamMemberID string) (*types.OrganizationTeamMemberResponse, error)
	DeleteByTeamIDAndMemberID(ctx context.Context, teamID, memberID string) error
	WithTx(tx bun.IDB) OrganizationTeamMemberRepository
}
