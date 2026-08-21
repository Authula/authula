package repositories_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/repositories"
	plugintests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
)

func TestBunOrganizationTeamRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		team *types.OrganizationTeam
	}{
		{
			name: "keeps provided metadata",
			team: &types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform", Metadata: map[string]any{"tier": "core"}},
		},
		{
			name: "keeps metadata and description",
			team: func() *types.OrganizationTeam {
				description := new(string)
				*description = "Core"
				return &types.OrganizationTeam{ID: "team-2", OrganizationID: "org-1", Name: "Core", Slug: "core", Description: description, Metadata: map[string]any{"tier": "core"}}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
			repo := repositories.NewBunOrganizationTeamRepository(db)
			created, err := repo.Create(context.Background(), tt.team)
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, tt.team.ID, created.ID)
			require.Equal(t, tt.team.Metadata, created.Metadata)
		})
	}
}

func TestBunOrganizationTeamRepository_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		teamID      string
		setup       func(*testing.T) (repositories.OrganizationTeamRepository, context.Context)
		expectFound bool
	}{
		{
			name:        "found",
			teamID:      "team-1",
			expectFound: true,
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationTeamRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:   "not found",
			teamID: "missing",
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context) {
				t.Helper()
				return repositories.NewBunOrganizationTeamRepository(plugintests.SetupRepoDB(t)), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			found, err := repo.GetByID(ctx, tt.teamID)
			require.NoError(t, err)
			if tt.expectFound {
				require.NotNil(t, found)
				require.Equal(t, "team-1", found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationTeamRepository_GetByOrganizationIDAndSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		organizationID string
		slug           string
		setup          func(*testing.T) (repositories.OrganizationTeamRepository, context.Context)
		expectFound    bool
	}{
		{
			name:           "found",
			organizationID: "org-1",
			slug:           "platform",
			expectFound:    true,
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationTeamRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:           "not found",
			organizationID: "org-1",
			slug:           "missing",
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context) {
				t.Helper()
				return repositories.NewBunOrganizationTeamRepository(plugintests.SetupRepoDB(t)), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			found, err := repo.GetByOrganizationIDAndSlug(ctx, tt.organizationID, tt.slug)
			require.NoError(t, err)
			if tt.expectFound {
				require.NotNil(t, found)
				require.Equal(t, "team-1", found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func seedTeams(t *testing.T, count int) (repositories.OrganizationTeamRepository, context.Context) {
	t.Helper()

	db := plugintests.SetupRepoDB(t)
	plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")

	repo := repositories.NewBunOrganizationTeamRepository(db)
	ctx := context.Background()
	for i := 1; i <= count; i++ {
		_, err := repo.Create(ctx, &types.OrganizationTeam{
			ID:             fmt.Sprintf("team-%d", i),
			OrganizationID: "org-1",
			Name:           fmt.Sprintf("Team %d", i),
			Slug:           fmt.Sprintf("team-%d", i),
		})
		require.NoError(t, err)
	}

	return repo, ctx
}

func teamIDs(teams []types.OrganizationTeam) []string {
	ids := make([]string, 0, len(teams))
	for _, team := range teams {
		ids = append(ids, team.ID)
	}
	return ids
}

func TestBunOrganizationTeamRepository_GetAllByOrganizationID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		organizationID string
		page           int
		limit          int
		expectCount    int
		expectTotal    int
	}{
		{name: "first page", organizationID: "org-1", page: 1, limit: 2, expectCount: 2, expectTotal: 5},
		{name: "last partial page", organizationID: "org-1", page: 3, limit: 2, expectCount: 1, expectTotal: 5},
		{name: "page past the end keeps the true total", organizationID: "org-1", page: 4, limit: 2, expectCount: 0, expectTotal: 5},
		{name: "page zero does not error", organizationID: "org-1", page: 0, limit: 2, expectCount: 2, expectTotal: 5},
		{name: "negative page does not error", organizationID: "org-1", page: -1, limit: 2, expectCount: 2, expectTotal: 5},
		{name: "zero limit falls back to a bounded page", organizationID: "org-1", page: 1, limit: 0, expectCount: 5, expectTotal: 5},
		{name: "unknown organization is empty", organizationID: "org-2", page: 1, limit: 10, expectCount: 0, expectTotal: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := seedTeams(t, 5)

			found, total, err := repo.GetAllByOrganizationID(ctx, tt.organizationID, tt.page, tt.limit)
			require.NoError(t, err)
			require.Len(t, found, tt.expectCount)
			require.Equal(t, tt.expectTotal, total)
		})
	}
}

func TestBunOrganizationTeamRepository_GetAllByOrganizationIDPagesPartitionCleanly(t *testing.T) {
	t.Parallel()

	repo, ctx := seedTeams(t, 5)

	seen := make([]string, 0, 5)
	for page := 1; page <= 3; page++ {
		found, total, err := repo.GetAllByOrganizationID(ctx, "org-1", page, 2)
		require.NoError(t, err)
		require.Equal(t, 5, total)
		seen = append(seen, teamIDs(found)...)
	}

	require.ElementsMatch(t, []string{"team-1", "team-2", "team-3", "team-4", "team-5"}, seen)
}

func TestBunOrganizationTeamRepository_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(*testing.T) (repositories.OrganizationTeamRepository, context.Context, *types.OrganizationTeam)
	}{
		{
			name: "change name and description",
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context, *types.OrganizationTeam) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationTeamRepository(db)
				ctx := context.Background()

				created, err := repo.Create(ctx, &types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"})
				require.NoError(t, err)
				return repo, ctx, created
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx, team := tt.setup(t)
			team.Name = "Platform Team"
			description := new(string)
			*description = "Core platform"
			team.Description = description
			updated, err := repo.Update(ctx, team)
			require.NoError(t, err)
			require.Equal(t, "Platform Team", updated.Name)
			require.NotNil(t, updated.Description)
			require.Equal(t, "Core platform", *updated.Description)
		})
	}
}

func TestBunOrganizationTeamRepository_Delete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		teamID string
		setup  func(*testing.T) (repositories.OrganizationTeamRepository, context.Context)
	}{
		{
			name:   "delete existing",
			teamID: "team-1",
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationTeamRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"})
				require.NoError(t, err)
				return repo, ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			require.NoError(t, repo.Delete(ctx, tt.teamID))
			found, err := repo.GetByID(ctx, tt.teamID)
			require.NoError(t, err)
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationTeamRepository_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		commit bool
		setup  func(*testing.T) (repositories.OrganizationTeamRepository, context.Context, repositories.OrganizationTeamRepository, bun.Tx)
	}{
		{
			name:   "commits through tx",
			commit: true,
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context, repositories.OrganizationTeamRepository, bun.Tx) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationTeamRepository(db)
				ctx := context.Background()
				tx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)
				return repo, ctx, repo.WithTx(tx), tx
			},
		},
		{
			name:   "rolls back through tx",
			commit: false,
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context, repositories.OrganizationTeamRepository, bun.Tx) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationTeamRepository(db)
				ctx := context.Background()
				tx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)
				return repo, ctx, repo.WithTx(tx), tx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx, txRepo, tx := tt.setup(t)
			require.NotNil(t, txRepo)
			require.IsType(t, &repositories.BunOrganizationTeamRepository{}, txRepo)

			created, err := txRepo.Create(ctx, &types.OrganizationTeam{ID: "team-1", OrganizationID: "org-1", Name: "Platform", Slug: "platform"})
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, "team-1", created.ID)

			if tt.commit {
				require.NoError(t, tx.Commit())
			} else {
				require.NoError(t, tx.Rollback())
			}

			found, err := repo.GetByID(ctx, "team-1")
			require.NoError(t, err)
			if tt.commit {
				require.NotNil(t, found)
			} else {
				require.Nil(t, found)
			}
		})
	}
}
