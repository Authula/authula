package repositories_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/repositories"
	plugintests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
)

func TestBunOrganizationRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		organization *types.Organization
	}{
		{
			name:         "keeps provided metadata",
			organization: &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc", Metadata: map[string]any{"tier": "core"}},
		},
		{
			name:         "keeps alternate metadata",
			organization: &types.Organization{ID: "org-2", OwnerID: "user-1", Name: "Platform", Slug: "platform", Metadata: map[string]any{"tier": "platform"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			repo := repositories.NewBunOrganizationRepository(db)

			created, err := repo.Create(context.Background(), tt.organization)
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, tt.organization.ID, created.ID)
			require.Equal(t, tt.organization.Metadata, created.Metadata)
		})
	}
}

func TestBunOrganizationRepository_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		organizationID string
		setup          func(*testing.T) (repositories.OrganizationRepository, context.Context)
		expectFound    bool
	}{
		{
			name:           "found",
			organizationID: "org-1",
			expectFound:    true,
			setup: func(t *testing.T) (repositories.OrganizationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				repo := repositories.NewBunOrganizationRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc", Metadata: map[string]any{"tier": "core"}})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:           "not found",
			organizationID: "missing",
			setup: func(t *testing.T) (repositories.OrganizationRepository, context.Context) {
				t.Helper()
				return repositories.NewBunOrganizationRepository(plugintests.SetupRepoDB(t)), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			found, err := repo.GetByID(ctx, tt.organizationID)
			require.NoError(t, err)
			if tt.expectFound {
				require.NotNil(t, found)
				require.Equal(t, "org-1", found.ID)
				require.Equal(t, "user-1", found.OwnerID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationRepository_GetBySlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		slug        string
		setup       func(*testing.T) (repositories.OrganizationRepository, context.Context)
		expectFound bool
	}{
		{
			name:        "found",
			slug:        "acme-inc",
			expectFound: true,
			setup: func(t *testing.T) (repositories.OrganizationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				repo := repositories.NewBunOrganizationRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc", Metadata: map[string]any{"tier": "core"}})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name: "not found",
			slug: "missing",
			setup: func(t *testing.T) (repositories.OrganizationRepository, context.Context) {
				t.Helper()
				return repositories.NewBunOrganizationRepository(plugintests.SetupRepoDB(t)), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			found, err := repo.GetBySlug(ctx, tt.slug)
			require.NoError(t, err)
			if tt.expectFound {
				require.NotNil(t, found)
				require.Equal(t, "org-1", found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func seedAccessibleOrganizations(t *testing.T) (repositories.OrganizationRepository, context.Context) {
	t.Helper()

	db := plugintests.SetupRepoDB(t)
	ctx := context.Background()

	plugintests.SeedOrganization(t, db, "org-a", "user-1", "Owned", "owned")
	plugintests.SeedOrganization(t, db, "org-b", "user-2", "Member Only", "member-only")
	plugintests.SeedOrganization(t, db, "org-c", "user-1", "Owner And Member", "owner-and-member")
	plugintests.SeedOrganization(t, db, "org-d", "user-2", "Unrelated", "unrelated")

	plugintests.SeedOrganizationMember(t, db, "mem-b", "org-b", "user-1", "member")
	plugintests.SeedOrganizationMember(t, db, "mem-c", "org-c", "user-1", "owner")
	plugintests.SeedOrganizationMember(t, db, "mem-d", "org-d", "user-2", "owner")

	return repositories.NewBunOrganizationRepository(db), ctx
}

func organizationIDs(organizations []types.Organization) []string {
	ids := make([]string, 0, len(organizations))
	for _, organization := range organizations {
		ids = append(ids, organization.ID)
	}
	return ids
}

func TestBunOrganizationRepository_GetAllAccessibleByUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userID      string
		page        int
		limit       int
		expectedIDs []string
		expectTotal int
	}{
		{
			name:        "returns owned and joined organizations without duplicates",
			userID:      "user-1",
			page:        1,
			limit:       10,
			expectedIDs: []string{"org-a", "org-b", "org-c"},
			expectTotal: 3,
		},
		{
			name:        "excludes organizations the user cannot access",
			userID:      "user-2",
			page:        1,
			limit:       10,
			expectedIDs: []string{"org-b", "org-d"},
			expectTotal: 2,
		},
		{
			name:        "page past the end is empty but keeps the true total",
			userID:      "user-1",
			page:        4,
			limit:       2,
			expectedIDs: []string{},
			expectTotal: 3,
		},
		{
			name:        "page zero does not produce a negative offset",
			userID:      "user-1",
			page:        0,
			limit:       10,
			expectedIDs: []string{"org-a", "org-b", "org-c"},
			expectTotal: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := seedAccessibleOrganizations(t)

			found, total, err := repo.GetAllAccessibleByUserID(ctx, tt.userID, tt.page, tt.limit)
			require.NoError(t, err)
			require.Equal(t, tt.expectTotal, total)
			require.ElementsMatch(t, tt.expectedIDs, organizationIDs(found))
		})
	}
}

func TestBunOrganizationRepository_GetAllAccessibleByUserIDPagesWithoutLoss(t *testing.T) {
	t.Parallel()

	repo, ctx := seedAccessibleOrganizations(t)

	seen := make([]string, 0, 3)
	for page := 1; page <= 3; page++ {
		found, total, err := repo.GetAllAccessibleByUserID(ctx, "user-1", page, 1)
		require.NoError(t, err)
		require.Equal(t, 3, total)
		require.Len(t, found, 1)
		seen = append(seen, found[0].ID)
	}

	require.ElementsMatch(t, []string{"org-a", "org-b", "org-c"}, seen)
}

func TestBunOrganizationRepository_CountAccessibleByUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		userID   string
		expected int
	}{
		{name: "counts owned and joined organizations once each", userID: "user-1", expected: 3},
		{name: "counts only what the user can access", userID: "user-2", expected: 2},
		{name: "unknown user has no organizations", userID: "user-unknown", expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := seedAccessibleOrganizations(t)

			count, err := repo.CountAccessibleByUserID(ctx, tt.userID)
			require.NoError(t, err)
			require.Equal(t, tt.expected, count)

			_, total, err := repo.GetAllAccessibleByUserID(ctx, tt.userID, 1, 10)
			require.NoError(t, err)
			require.Equal(t, count, total, "count must agree with the list total")
		})
	}
}

func TestBunOrganizationRepository_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(*testing.T) (repositories.OrganizationRepository, context.Context, *types.Organization)
	}{
		{
			name: "update name and logo",
			setup: func(t *testing.T) (repositories.OrganizationRepository, context.Context, *types.Organization) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				repo := repositories.NewBunOrganizationRepository(db)
				ctx := context.Background()

				created, err := repo.Create(ctx, &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc", Metadata: map[string]any{"tier": "core"}})
				require.NoError(t, err)
				return repo, ctx, created
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx, created := tt.setup(t)
			created.Name = "Acme Platform"
			logo := new(string)
			*logo = "http://example.com/logo.svg"
			created.Logo = logo
			updated, err := repo.Update(ctx, created)
			require.NoError(t, err)
			require.Equal(t, "Acme Platform", updated.Name)
			require.NotNil(t, updated.Logo)
			require.Equal(t, "http://example.com/logo.svg", *updated.Logo)
		})
	}
}

func TestBunOrganizationRepository_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		organizationID string
		setup          func(*testing.T) (repositories.OrganizationRepository, context.Context)
	}{
		{
			name:           "delete existing",
			organizationID: "org-1",
			setup: func(t *testing.T) (repositories.OrganizationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				repo := repositories.NewBunOrganizationRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc"})
				require.NoError(t, err)

				return repo, ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			require.NoError(t, repo.Delete(ctx, tt.organizationID))
			found, err := repo.GetByID(ctx, tt.organizationID)
			require.NoError(t, err)
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationRepository_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		commit bool
		setup  func(*testing.T) (repositories.OrganizationRepository, context.Context, repositories.OrganizationRepository, bun.Tx)
	}{
		{
			name:   "commits through tx",
			commit: true,
			setup: func(t *testing.T) (repositories.OrganizationRepository, context.Context, repositories.OrganizationRepository, bun.Tx) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				repo := repositories.NewBunOrganizationRepository(db)
				ctx := context.Background()
				tx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)
				return repo, ctx, repo.WithTx(tx), tx
			},
		},
		{
			name:   "rolls back through tx",
			commit: false,
			setup: func(t *testing.T) (repositories.OrganizationRepository, context.Context, repositories.OrganizationRepository, bun.Tx) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				repo := repositories.NewBunOrganizationRepository(db)
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
			require.IsType(t, &repositories.BunOrganizationRepository{}, txRepo)

			created, err := txRepo.Create(ctx, &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc", Metadata: map[string]any{"tier": "core"}})
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, "org-1", created.ID)

			if tt.commit {
				require.NoError(t, tx.Commit())
			} else {
				require.NoError(t, tx.Rollback())
			}

			found, err := repo.GetByID(ctx, "org-1")
			require.NoError(t, err)
			if tt.commit {
				require.NotNil(t, found)
			} else {
				require.Nil(t, found)
			}
		})
	}
}
