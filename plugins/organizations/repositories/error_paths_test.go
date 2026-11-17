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

func closedRepoDB(t *testing.T) *bun.DB {
	t.Helper()

	db := plugintests.SetupRepoDB(t)
	require.NoError(t, db.Close())
	return db
}

func TestBunOrganizationRepository_ErrorPaths(t *testing.T) {
	t.Parallel()

	t.Run("create with a duplicate id fails", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		repo := repositories.NewBunOrganizationRepository(db)

		created, err := repo.Create(context.Background(), &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme", Slug: "acme"})
		require.Error(t, err)
		require.Nil(t, created)
	})

	t.Run("update of a missing organization fails after the write", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		repo := repositories.NewBunOrganizationRepository(db)

		updated, err := repo.Update(context.Background(), &types.Organization{ID: "ghost", OwnerID: "user-1", Name: "Acme", Slug: "acme"})
		require.Error(t, err)
		require.Nil(t, updated)
	})

	tests := []struct {
		name string
		run  func(*testing.T, repositories.OrganizationRepository, context.Context)
	}{
		{
			name: "get by id",
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByID(ctx, "org-1")
				require.Error(t, err)
			},
		},
		{
			name: "get by slug",
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetBySlug(ctx, "acme")
				require.Error(t, err)
			},
		},
		{
			name: "list all accessible",
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllAccessibleByUserID(ctx, "user-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name: "get all accessible",
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllAccessibleByUserID(ctx, "user-1")
				require.Error(t, err)
			},
		},
		{
			name: "get all",
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAll(ctx)
				require.Error(t, err)
			},
		},
		{
			name: "count accessible",
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				count, err := repo.CountAccessibleByUserID(ctx, "user-1")
				require.Error(t, err)
				require.Zero(t, count)
			},
		},
		{
			name: "delete",
			run: func(t *testing.T, repo repositories.OrganizationRepository, ctx context.Context) {
				t.Helper()
				require.Error(t, repo.Delete(ctx, "org-1"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := repositories.NewBunOrganizationRepository(closedRepoDB(t))
			tt.run(t, repo, context.Background())
		})
	}
}

func TestBunOrganizationMemberRepository_ErrorPaths(t *testing.T) {
	t.Parallel()

	t.Run("create with a duplicate id fails", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		plugintests.SeedOrganizationMember(t, db, "mem-1", "org-1", "user-2", "member")
		repo := repositories.NewBunOrganizationMemberRepository(db)

		created, err := repo.Create(context.Background(), &types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "admin"})
		require.Error(t, err)
		require.Nil(t, created)
	})

	t.Run("update of a missing member fails after the write", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		repo := repositories.NewBunOrganizationMemberRepository(db)

		updated, err := repo.Update(context.Background(), &types.OrganizationMember{ID: "ghost", OrganizationID: "org-1", UserID: "user-2", Role: "admin"})
		require.Error(t, err)
		require.Nil(t, updated)
	})

	tests := []struct {
		name string
		run  func(*testing.T, repositories.OrganizationMemberRepository, context.Context)
	}{
		{
			name: "count by organization",
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				count, err := repo.CountByOrganizationID(ctx, "org-1")
				require.Error(t, err)
				require.Zero(t, count)
			},
		},
		{
			name: "list all by organization",
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllByOrganizationID(ctx, "org-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name: "get all by organization",
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllByOrganizationID(ctx, "org-1")
				require.Error(t, err)
			},
		},
		{
			name: "list all by organization with user",
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllByOrganizationIDWithUser(ctx, "org-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name: "get all by organization with user",
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllByOrganizationIDWithUser(ctx, "org-1")
				require.Error(t, err)
			},
		},
		{
			name: "get by organization and user",
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByOrganizationIDAndUserID(ctx, "org-1", "user-2")
				require.Error(t, err)
			},
		},
		{
			name: "delete",
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				require.Error(t, repo.Delete(ctx, "mem-1"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := repositories.NewBunOrganizationMemberRepository(closedRepoDB(t))
			tt.run(t, repo, context.Background())
		})
	}
}

func TestBunOrganizationTeamRepository_ErrorPaths(t *testing.T) {
	t.Parallel()

	t.Run("create with a duplicate id fails", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
		repo := repositories.NewBunOrganizationTeamRepository(db)

		created, err := repo.Create(context.Background(), &types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"})
		require.Error(t, err)
		require.Nil(t, created)
	})

	t.Run("update of a missing team fails after the write", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		repo := repositories.NewBunOrganizationTeamRepository(db)

		updated, err := repo.Update(context.Background(), &types.OrganizationTeam{ID: "ghost", OrganizationID: "org-1", Name: "Platform", Slug: "platform"})
		require.Error(t, err)
		require.Nil(t, updated)
	})

	tests := []struct {
		name string
		run  func(*testing.T, repositories.OrganizationTeamRepository, context.Context)
	}{
		{
			name: "get by id",
			run: func(t *testing.T, repo repositories.OrganizationTeamRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByID(ctx, "team-1")
				require.Error(t, err)
			},
		},
		{
			name: "get by organization and slug",
			run: func(t *testing.T, repo repositories.OrganizationTeamRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByOrganizationIDAndSlug(ctx, "org-1", "platform")
				require.Error(t, err)
			},
		},
		{
			name: "list all by organization",
			run: func(t *testing.T, repo repositories.OrganizationTeamRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllByOrganizationID(ctx, "org-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name: "get all by organization",
			run: func(t *testing.T, repo repositories.OrganizationTeamRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllByOrganizationID(ctx, "org-1")
				require.Error(t, err)
			},
		},
		{
			name: "delete",
			run: func(t *testing.T, repo repositories.OrganizationTeamRepository, ctx context.Context) {
				t.Helper()
				require.Error(t, repo.Delete(ctx, "team-1"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := repositories.NewBunOrganizationTeamRepository(closedRepoDB(t))
			tt.run(t, repo, context.Background())
		})
	}
}

func TestBunOrganizationTeamMemberRepository_ErrorPaths(t *testing.T) {
	t.Parallel()

	t.Run("create with a duplicate id fails", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		plugintests.SeedOrganizationMember(t, db, "mem-1", "org-1", "user-2", "member")
		plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
		plugintests.SeedOrganizationTeamMember(t, db, "tm-1", "team-1", "mem-1")
		repo := repositories.NewBunOrganizationTeamMemberRepository(db)

		created, err := repo.Create(context.Background(), &types.OrganizationTeamMember{ID: "tm-1", TeamID: "team-1", MemberID: "mem-1"})
		require.Error(t, err)
		require.Nil(t, created)
	})

	tests := []struct {
		name string
		run  func(*testing.T, repositories.OrganizationTeamMemberRepository, context.Context)
	}{
		{
			name: "get by id",
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByID(ctx, "tm-1")
				require.Error(t, err)
			},
		},
		{
			name: "get by team and member",
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByTeamIDAndMemberID(ctx, "team-1", "mem-1")
				require.Error(t, err)
			},
		},
		{
			name: "list all by team",
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllByTeamID(ctx, "team-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name: "get all by team",
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllByTeamID(ctx, "team-1")
				require.Error(t, err)
			},
		},
		{
			name: "list all by team with member and user",
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllByTeamIDWithMemberAndUser(ctx, "team-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name: "get all by team with member and user",
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllByTeamIDWithMemberAndUser(ctx, "team-1")
				require.Error(t, err)
			},
		},
		{
			name: "delete by team and member",
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				require.Error(t, repo.DeleteByTeamIDAndMemberID(ctx, "team-1", "mem-1"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := repositories.NewBunOrganizationTeamMemberRepository(closedRepoDB(t))
			tt.run(t, repo, context.Background())
		})
	}
}

func TestBunOrganizationInvitationRepository_ErrorPaths(t *testing.T) {
	t.Parallel()

	expiresAt := time.Now().UTC().Add(time.Hour)

	t.Run("create with a duplicate id fails", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		plugintests.SeedOrganizationInvitation(t, db, "inv-1", "bob@example.com", "user-1", "org-1", "member", "pending", expiresAt)
		repo := repositories.NewBunOrganizationInvitationRepository(db)

		created, err := repo.Create(context.Background(), &types.OrganizationInvitation{ID: "inv-1", Email: "bob@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: expiresAt})
		require.Error(t, err)
		require.Nil(t, created)
	})

	t.Run("update of a missing invitation fails after the write", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		repo := repositories.NewBunOrganizationInvitationRepository(db)

		updated, err := repo.Update(context.Background(), &types.OrganizationInvitation{ID: "ghost", Email: "bob@example.com", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: expiresAt})
		require.Error(t, err)
		require.Nil(t, updated)
	})

	t.Run("an empty status filter is ignored", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		plugintests.SeedOrganizationInvitation(t, db, "inv-1", "bob@example.com", "user-1", "org-1", "member", "pending", expiresAt)
		repo := repositories.NewBunOrganizationInvitationRepository(db)

		invitation, err := repo.GetByOrganizationIDAndEmail(context.Background(), "org-1", "bob@example.com", "")
		require.NoError(t, err)
		require.NotNil(t, invitation)
		require.Equal(t, "inv-1", invitation.ID)
	})

	tests := []struct {
		name string
		run  func(*testing.T, repositories.OrganizationInvitationRepository, context.Context)
	}{
		{
			name: "get by id",
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByID(ctx, "inv-1")
				require.Error(t, err)
			},
		},
		{
			name: "get by organization and email",
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByOrganizationIDAndEmail(ctx, "org-1", "bob@example.com")
				require.Error(t, err)
			},
		},
		{
			name: "list all pending by email",
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllPendingByEmail(ctx, "bob@example.com", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name: "get all pending by email",
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetAllPendingByEmail(ctx, "bob@example.com")
				require.Error(t, err)
			},
		},
		{
			name: "count by organization and email",
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				count, err := repo.CountByOrganizationIDAndEmail(ctx, "org-1", "bob@example.com")
				require.Error(t, err)
				require.Zero(t, count)
			},
		},
		{
			name: "get by id with organization",
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				_, err := repo.GetByIDWithOrg(ctx, "inv-1")
				require.Error(t, err)
			},
		},
		{
			name: "list all by organization with organization",
			run: func(t *testing.T, repo repositories.OrganizationInvitationRepository, ctx context.Context) {
				t.Helper()
				_, total, err := repo.ListAllByOrganizationIDWithOrg(ctx, "org-1", 1, 10)
				require.Error(t, err)
				require.Zero(t, total)
			},
		},
		{
			name: "get all by organization with organization",
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

			repo := repositories.NewBunOrganizationInvitationRepository(closedRepoDB(t))
			tt.run(t, repo, context.Background())
		})
	}
}
