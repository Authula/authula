package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/repositories"
	plugintests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
)

func TestBunOrganizationRepository_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seed    func(t *testing.T, db *bun.DB)
		closeDB bool
		run     func(*testing.T, repositories.OrganizationRepository, context.Context)
	}{
		{
			name: "create with a duplicate id fails",
			seed: func(t *testing.T, db *bun.DB) {
				t.Helper()
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
			},
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				created, err := repo.Create(ctx, &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme", Slug: "acme"})
				require.Error(t, err)
				require.Nil(t, created)
			},
		},
		{
			name: "update of a missing organization fails after the write",
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				updated, err := repo.Update(ctx, &types.Organization{ID: "ghost", OwnerID: "user-1", Name: "Acme", Slug: "acme"})
				require.Error(t, err)
				require.Nil(t, updated)
			},
		},
		{
			name:    "get by id",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByID(ctx, "org-1")
				require.Error(t, err)
			},
		},
		{
			name:    "get by slug",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetBySlug(ctx, "acme")
				require.Error(t, err)
			},
		},
		{
			name:    "list all accessible",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllAccessibleByUserID(ctx, "user-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name:    "get all accessible",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllAccessibleByUserID(ctx, "user-1")
				require.Error(t, err)
			},
		},
		{
			name:    "get all",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAll(ctx)
				require.Error(t, err)
			},
		},
		{
			name:    "count accessible",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				count, err := repo.CountAccessibleByUserID(ctx, "user-1")
				require.Error(t, err)
				require.Zero(t, count)
			},
		},
		{
			name:    "delete",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				require.Error(t, repo.Delete(ctx, "org-1"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			if tt.seed != nil {
				tt.seed(t, db)
			}
			if tt.closeDB {
				require.NoError(t, db.Close())
			}

			repo := repositories.NewBunOrganizationRepository(db)
			tt.run(t, repo, context.Background())
		})
	}
}

func TestBunOrganizationMemberRepository_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seed    func(t *testing.T, db *bun.DB)
		closeDB bool
		run     func(*testing.T, repositories.OrganizationMemberRepository, context.Context)
	}{
		{
			name: "create with a duplicate id fails",
			seed: func(t *testing.T, db *bun.DB) {
				t.Helper()
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "mem-1", "org-1", "user-2", "member")
			},
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				created, err := repo.Create(ctx, &types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "admin"})
				require.Error(t, err)
				require.Nil(t, created)
			},
		},
		{
			name: "update of a missing member fails after the write",
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				updated, err := repo.Update(ctx, &types.OrganizationMember{ID: "ghost", OrganizationID: "org-1", UserID: "user-2", Role: "admin"})
				require.Error(t, err)
				require.Nil(t, updated)
			},
		},
		{
			name:    "count by organization",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				count, err := repo.CountByOrganizationID(ctx, "org-1")
				require.Error(t, err)
				require.Zero(t, count)
			},
		},
		{
			name:    "list all by organization",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllByOrganizationID(ctx, "org-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name:    "get all by organization",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllByOrganizationID(ctx, "org-1")
				require.Error(t, err)
			},
		},
		{
			name:    "list all by organization with user",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllByOrganizationIDWithUser(ctx, "org-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name:    "get all by organization with user",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllByOrganizationIDWithUser(ctx, "org-1")
				require.Error(t, err)
			},
		},
		{
			name:    "get by organization and user",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByOrganizationIDAndUserID(ctx, "org-1", "user-2")
				require.Error(t, err)
			},
		},
		{
			name:    "delete",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				require.Error(t, repo.Delete(ctx, "mem-1"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			if tt.seed != nil {
				tt.seed(t, db)
			}
			if tt.closeDB {
				require.NoError(t, db.Close())
			}

			repo := repositories.NewBunOrganizationMemberRepository(db)
			tt.run(t, repo, context.Background())
		})
	}
}

func TestBunOrganizationTeamRepository_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seed    func(t *testing.T, db *bun.DB)
		closeDB bool
		run     func(*testing.T, repositories.OrganizationTeamRepository, context.Context)
	}{
		{
			name: "create with a duplicate id fails",
			seed: func(t *testing.T, db *bun.DB) {
				t.Helper()
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
			},
			run: func(t *testing.T, repo repositories.OrganizationTeamRepository, ctx context.Context) {
				t.Helper()
				created, err := repo.Create(ctx, &types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"})
				require.Error(t, err)
				require.Nil(t, created)
			},
		},
		{
			name: "update of a missing team fails after the write",
			run: func(t *testing.T, repo repositories.OrganizationTeamRepository, ctx context.Context) {
				t.Helper()
				updated, err := repo.Update(ctx, &types.OrganizationTeam{ID: "ghost", OrganizationID: "org-1", Name: "Platform", Slug: "platform"})
				require.Error(t, err)
				require.Nil(t, updated)
			},
		},
		{
			name:    "get by id",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationTeamRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByID(ctx, "team-1")
				require.Error(t, err)
			},
		},
		{
			name:    "get by organization and slug",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationTeamRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByOrganizationIDAndSlug(ctx, "org-1", "platform")
				require.Error(t, err)
			},
		},
		{
			name:    "list all by organization",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationTeamRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllByOrganizationID(ctx, "org-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name:    "get all by organization",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationTeamRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllByOrganizationID(ctx, "org-1")
				require.Error(t, err)
			},
		},
		{
			name:    "delete",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationTeamRepository, ctx context.Context) {
				t.Helper()
				require.Error(t, repo.Delete(ctx, "team-1"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			if tt.seed != nil {
				tt.seed(t, db)
			}
			if tt.closeDB {
				require.NoError(t, db.Close())
			}

			repo := repositories.NewBunOrganizationTeamRepository(db)
			tt.run(t, repo, context.Background())
		})
	}
}

func TestBunOrganizationTeamMemberRepository_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seed    func(t *testing.T, db *bun.DB)
		closeDB bool
		run     func(*testing.T, repositories.OrganizationTeamMemberRepository, context.Context)
	}{
		{
			name: "create with a duplicate id fails",
			seed: func(t *testing.T, db *bun.DB) {
				t.Helper()
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "mem-1", "org-1", "user-2", "member")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				plugintests.SeedOrganizationTeamMember(t, db, "tm-1", "team-1", "mem-1")
			},
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				created, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: "tm-1", TeamID: "team-1", MemberID: "mem-1"})
				require.Error(t, err)
				require.Nil(t, created)
			},
		},
		{
			name:    "get by id",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByID(ctx, "tm-1")
				require.Error(t, err)
			},
		},
		{
			name:    "get by team and member",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByTeamIDAndMemberID(ctx, "team-1", "mem-1")
				require.Error(t, err)
			},
		},
		{
			name:    "list all by team",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllByTeamID(ctx, "team-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name:    "get all by team",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllByTeamID(ctx, "team-1")
				require.Error(t, err)
			},
		},
		{
			name:    "list all by team with member and user",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllByTeamIDWithMemberAndUser(ctx, "team-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name:    "get all by team with member and user",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllByTeamIDWithMemberAndUser(ctx, "team-1")
				require.Error(t, err)
			},
		},
		{
			name:    "delete by team and member",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				require.Error(t, repo.DeleteByTeamIDAndMemberID(ctx, "team-1", "mem-1"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			if tt.seed != nil {
				tt.seed(t, db)
			}
			if tt.closeDB {
				require.NoError(t, db.Close())
			}

			repo := repositories.NewBunOrganizationTeamMemberRepository(db)
			tt.run(t, repo, context.Background())
		})
	}
}

func TestBunOrganizationInvitationRepository_ErrorPaths(t *testing.T) {
	t.Parallel()

	expiresAt := time.Now().UTC().Add(time.Hour)
	seedInvitation := func(t *testing.T, db *bun.DB) {
		t.Helper()
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		plugintests.SeedOrganizationInvitation(t, db, "inv-1", "bob@example.com", "user-1", "org-1", "member", "pending", expiresAt)
	}

	tests := []struct {
		name    string
		seed    func(t *testing.T, db *bun.DB)
		closeDB bool
		run     func(*testing.T, repositories.OrganizationInvitationRepository, context.Context)
	}{
		{
			name: "create with a duplicate id fails",
			seed: seedInvitation,
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				created, err := repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-1", Email: "bob@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: expiresAt})
				require.Error(t, err)
				require.Nil(t, created)
			},
		},
		{
			name: "update of a missing invitation fails after the write",
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				updated, err := repo.Update(ctx, &types.OrganizationInvitation{ID: "ghost", Email: "bob@example.com", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: expiresAt})
				require.Error(t, err)
				require.Nil(t, updated)
			},
		},
		{
			name: "an empty status filter is ignored",
			seed: seedInvitation,
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				invitation, err := repo.GetByOrganizationIDAndEmail(ctx, "org-1", "bob@example.com", "")
				require.NoError(t, err)
				require.NotNil(t, invitation)
				require.Equal(t, "inv-1", invitation.ID)
			},
		},
		{
			name:    "get by id",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByID(ctx, "inv-1")
				require.Error(t, err)
			},
		},
		{
			name:    "get by organization and email",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByOrganizationIDAndEmail(ctx, "org-1", "bob@example.com")
				require.Error(t, err)
			},
		},
		{
			name:    "list all pending by email",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllPendingByEmail(ctx, "bob@example.com", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name:    "get all pending by email",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllPendingByEmail(ctx, "bob@example.com")
				require.Error(t, err)
			},
		},
		{
			name:    "count by organization and email",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				count, err := repo.CountByOrganizationIDAndEmail(ctx, "org-1", "bob@example.com")
				require.Error(t, err)
				require.Zero(t, count)
			},
		},
		{
			name:    "get by id with organization",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByIDWithOrg(ctx, "inv-1")
				require.Error(t, err)
			},
		},
		{
			name:    "list all by organization with organization",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllByOrganizationIDWithOrg(ctx, "org-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name:    "get all by organization with organization",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllByOrganizationIDWithOrg(ctx, "org-1")
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			if tt.seed != nil {
				tt.seed(t, db)
			}
			if tt.closeDB {
				require.NoError(t, db.Close())
			}

			repo := repositories.NewBunOrganizationInvitationRepository(db)
			tt.run(t, repo, context.Background())
		})
	}
}
