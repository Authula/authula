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

func TestBunOrganizationMemberRepository_CreateGetUpdateDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(*testing.T, repositories.OrganizationMemberRepository, context.Context, *types.OrganizationMember)
	}{
		{
			name: "get by id returns created member",
			run: func(t *testing.T, orgMemberRepo repositories.OrganizationMemberRepository, ctx context.Context, created *types.OrganizationMember) {
				t.Helper()
				found, err := orgMemberRepo.GetByID(ctx, "mem-1")
				require.NoError(t, err)
				require.NotNil(t, found)
				require.Equal(t, created.Role, found.Role)
			},
		},
		{
			name: "list by organization returns created member",
			run: func(t *testing.T, orgMemberRepo repositories.OrganizationMemberRepository, ctx context.Context, created *types.OrganizationMember) {
				t.Helper()
				members, total, err := orgMemberRepo.ListAllByOrganizationID(ctx, "org-1", 1, 10)
				require.NoError(t, err)
				require.Len(t, members, 1)
				require.Equal(t, 1, total)
				require.Equal(t, created.ID, members[0].ID)
			},
		},
		{
			name: "update persists changed role",
			run: func(t *testing.T, orgMemberRepo repositories.OrganizationMemberRepository, ctx context.Context, created *types.OrganizationMember) {
				t.Helper()
				created.Role = "admin"
				updated, err := orgMemberRepo.Update(ctx, created)
				require.NoError(t, err)
				require.Equal(t, "admin", updated.Role)
			},
		},
		{
			name: "delete removes member",
			run: func(t *testing.T, orgMemberRepo repositories.OrganizationMemberRepository, ctx context.Context, created *types.OrganizationMember) {
				t.Helper()
				require.NoError(t, orgMemberRepo.Delete(ctx, created.ID))
				found, err := orgMemberRepo.GetByID(ctx, created.ID)
				require.NoError(t, err)
				require.Nil(t, found)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
			orgMemberRepo := repositories.NewBunOrganizationMemberRepository(db)
			ctx := context.Background()
			created, err := orgMemberRepo.Create(ctx, &types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-2", Role: "member"})
			require.NoError(t, err)
			require.NotNil(t, created)

			tt.run(t, orgMemberRepo, ctx, created)
		})
	}
}

func TestBunOrganizationMemberRepository_CountByOrganizationID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(*testing.T) (repositories.OrganizationMemberRepository, context.Context)
	}{
		{
			name: "counts members in organization",
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationMember{ID: "mem-2", OrganizationID: "org-1", UserID: "user-2", Role: "admin"})
				require.NoError(t, err)

				return repo, ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)
			count, err := repo.CountByOrganizationID(ctx, "org-1")
			require.NoError(t, err)
			require.Equal(t, 2, count)
		})
	}
}

func TestBunOrganizationMemberRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		member *types.OrganizationMember
	}{
		{name: "member", member: &types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"}},
		{name: "admin", member: &types.OrganizationMember{ID: "mem-2", OrganizationID: "org-1", UserID: "user-2", Role: "admin"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
			repo := repositories.NewBunOrganizationMemberRepository(db)
			created, err := repo.Create(context.Background(), tt.member)
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, tt.member.ID, created.ID)
			require.Equal(t, tt.member.Role, created.Role)
		})
	}
}

func seedMembers(t *testing.T, count int) (repositories.OrganizationMemberRepository, context.Context) {
	t.Helper()

	db := plugintests.SetupRepoDB(t)
	plugintests.SeedUsers(t, db, count)
	plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")

	repo := repositories.NewBunOrganizationMemberRepository(db)
	ctx := context.Background()
	for i := 1; i <= count; i++ {
		_, err := repo.Create(ctx, &types.OrganizationMember{
			ID:             fmt.Sprintf("mem-%d", i),
			OrganizationID: "org-1",
			UserID:         fmt.Sprintf("user-%d", i),
			Role:           "member",
		})
		require.NoError(t, err)
	}

	return repo, ctx
}

func memberIDs(members []types.OrganizationMember) []string {
	ids := make([]string, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.ID)
	}
	return ids
}

func memberResponseIDs(members []types.OrganizationMemberResponse) []string {
	ids := make([]string, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.ID)
	}
	return ids
}

func TestBunOrganizationMemberRepository_ListAllByOrganizationID(t *testing.T) {
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
		{name: "negative limit falls back to a bounded page", organizationID: "org-1", page: 1, limit: -1, expectCount: 5, expectTotal: 5},
		{name: "unknown organization is empty", organizationID: "org-2", page: 1, limit: 10, expectCount: 0, expectTotal: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := seedMembers(t, 5)

			members, total, err := repo.ListAllByOrganizationID(ctx, tt.organizationID, tt.page, tt.limit)
			require.NoError(t, err)
			require.Len(t, members, tt.expectCount)
			require.Equal(t, tt.expectTotal, total)
		})
	}
}

func TestBunOrganizationMemberRepository_ListAllByOrganizationIDPagesPartitionCleanly(t *testing.T) {
	t.Parallel()

	repo, ctx := seedMembers(t, 5)

	seen := make([]string, 0, 5)
	for page := 1; page <= 3; page++ {
		members, total, err := repo.ListAllByOrganizationID(ctx, "org-1", page, 2)
		require.NoError(t, err)
		require.Equal(t, 5, total)
		seen = append(seen, memberIDs(members)...)
	}

	require.ElementsMatch(t, []string{"mem-1", "mem-2", "mem-3", "mem-4", "mem-5"}, seen)
}

func TestBunOrganizationMemberRepository_ListAllByOrganizationIDWithUser(t *testing.T) {
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
		{name: "limit beyond the maximum is capped", organizationID: "org-1", page: 1, limit: 100000, expectCount: 5, expectTotal: 5},
		{name: "unknown organization is empty", organizationID: "org-2", page: 1, limit: 10, expectCount: 0, expectTotal: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := seedMembers(t, 5)

			members, total, err := repo.ListAllByOrganizationIDWithUser(ctx, tt.organizationID, tt.page, tt.limit)
			require.NoError(t, err)
			require.Len(t, members, tt.expectCount)
			require.Equal(t, tt.expectTotal, total)
			for _, member := range members {
				require.NotEmpty(t, member.User.ID, "joined user must be populated")
			}
		})
	}
}

func TestBunOrganizationMemberRepository_ListAllByOrganizationIDWithUserPagesPartitionCleanly(t *testing.T) {
	t.Parallel()

	repo, ctx := seedMembers(t, 5)

	seen := make([]string, 0, 5)
	for page := 1; page <= 3; page++ {
		members, total, err := repo.ListAllByOrganizationIDWithUser(ctx, "org-1", page, 2)
		require.NoError(t, err)
		require.Equal(t, 5, total)
		seen = append(seen, memberResponseIDs(members)...)
	}

	require.ElementsMatch(t, []string{"mem-1", "mem-2", "mem-3", "mem-4", "mem-5"}, seen)
}

func TestBunOrganizationMemberRepository_GetByOrganizationIDAndUserID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		organizationID string
		userID         string
		setup          func(*testing.T) (repositories.OrganizationMemberRepository, context.Context)
		expectFound    bool
	}{
		{
			name:           "found",
			organizationID: "org-1",
			userID:         "user-1",
			expectFound:    true,
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:           "not found",
			organizationID: "org-1",
			userID:         "user-2",
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				return repositories.NewBunOrganizationMemberRepository(db), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			found, err := repo.GetByOrganizationIDAndUserID(ctx, tt.organizationID, tt.userID)
			require.NoError(t, err)
			if tt.expectFound {
				require.NotNil(t, found)
				require.Equal(t, "mem-1", found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationMemberRepository_GetByID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		memberID    string
		setup       func(*testing.T) (repositories.OrganizationMemberRepository, context.Context)
		expectFound bool
	}{
		{
			name:        "found",
			memberID:    "mem-1",
			expectFound: true,
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:     "not found",
			memberID: "missing",
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				return repositories.NewBunOrganizationMemberRepository(plugintests.SetupRepoDB(t)), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			found, err := repo.GetByID(ctx, tt.memberID)
			require.NoError(t, err)
			if tt.expectFound {
				require.NotNil(t, found)
				require.Equal(t, "mem-1", found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationMemberRepository_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(*testing.T) (repositories.OrganizationMemberRepository, context.Context, *types.OrganizationMember)
	}{
		{
			name: "change role",
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context, *types.OrganizationMember) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()

				created, err := repo.Create(ctx, &types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"})
				require.NoError(t, err)
				return repo, ctx, created
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx, member := tt.setup(t)
			member.Role = "admin"
			updated, err := repo.Update(ctx, member)
			require.NoError(t, err)
			require.Equal(t, "admin", updated.Role)
		})
	}
}

func TestBunOrganizationMemberRepository_Delete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		memberID string
		setup    func(*testing.T) (repositories.OrganizationMemberRepository, context.Context)
	}{
		{
			name:     "delete existing",
			memberID: "mem-1",
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"})
				require.NoError(t, err)
				return repo, ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			require.NoError(t, repo.Delete(ctx, tt.memberID))
			found, err := repo.GetByID(ctx, tt.memberID)
			require.NoError(t, err)
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationMemberRepository_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		commit bool
		setup  func(*testing.T) (repositories.OrganizationMemberRepository, context.Context, repositories.OrganizationMemberRepository, bun.Tx)
	}{
		{
			name:   "commits through tx",
			commit: true,
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context, repositories.OrganizationMemberRepository, bun.Tx) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()
				tx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)
				return repo, ctx, repo.WithTx(tx), tx
			},
		},
		{
			name:   "rolls back through tx",
			commit: false,
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context, repositories.OrganizationMemberRepository, bun.Tx) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationMemberRepository(db)
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
			require.IsType(t, &repositories.BunOrganizationMemberRepository{}, txRepo)

			created, err := txRepo.Create(ctx, &types.OrganizationMember{ID: "mem-1", OrganizationID: "org-1", UserID: "user-1", Role: "member"})
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, "mem-1", created.ID)

			if tt.commit {
				require.NoError(t, tx.Commit())
			} else {
				require.NoError(t, tx.Rollback())
			}

			found, err := repo.GetByID(ctx, "mem-1")
			require.NoError(t, err)
			if tt.commit {
				require.NotNil(t, found)
			} else {
				require.Nil(t, found)
			}
		})
	}
}

func TestBunOrganizationMemberRepository_GetAllByOrganizationID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		seedCount      int
		organizationID string
		expectIDs      []string
	}{
		{name: "returns every member newest first", seedCount: 5, organizationID: "org-1", expectIDs: []string{"mem-5", "mem-4", "mem-3", "mem-2", "mem-1"}},
		{name: "unknown organization is empty", seedCount: 5, organizationID: "org-2", expectIDs: []string{}},
		{name: "organization with no members is empty", seedCount: 0, organizationID: "org-1", expectIDs: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := seedMembers(t, tt.seedCount)

			members, err := repo.GetAllByOrganizationID(ctx, tt.organizationID)
			require.NoError(t, err)
			require.NotNil(t, members, "an empty result must be an empty slice, not nil")
			require.Equal(t, tt.expectIDs, memberIDs(members))
		})
	}
}

// The unconstrained fetch must ignore pagination.DefaultLimit entirely.
func TestBunOrganizationMemberRepository_GetAllByOrganizationIDIgnoresTheDefaultLimit(t *testing.T) {
	t.Parallel()

	const memberCount = 25

	repo, ctx := seedMembers(t, memberCount)

	members, err := repo.GetAllByOrganizationID(ctx, "org-1")
	require.NoError(t, err)
	require.Len(t, members, memberCount)

	withUser, err := repo.GetAllByOrganizationIDWithUser(ctx, "org-1")
	require.NoError(t, err)
	require.Len(t, withUser, memberCount)
}

func TestBunOrganizationMemberRepository_GetAllByOrganizationIDWithUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		seedCount      int
		organizationID string
		expectIDs      []string
	}{
		{name: "returns every member newest first", seedCount: 5, organizationID: "org-1", expectIDs: []string{"mem-5", "mem-4", "mem-3", "mem-2", "mem-1"}},
		{name: "unknown organization is empty", seedCount: 5, organizationID: "org-2", expectIDs: []string{}},
		{name: "organization with no members is empty", seedCount: 0, organizationID: "org-1", expectIDs: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := seedMembers(t, tt.seedCount)

			members, err := repo.GetAllByOrganizationIDWithUser(ctx, tt.organizationID)
			require.NoError(t, err)
			require.NotNil(t, members, "an empty result must be an empty slice, not nil")
			require.Equal(t, tt.expectIDs, memberResponseIDs(members))
		})
	}
}

// The joined variant must hydrate the user, not just the membership row.
func TestBunOrganizationMemberRepository_GetAllByOrganizationIDWithUserHydratesTheUser(t *testing.T) {
	t.Parallel()

	repo, ctx := seedMembers(t, 3)

	members, err := repo.GetAllByOrganizationIDWithUser(ctx, "org-1")
	require.NoError(t, err)
	require.Len(t, members, 3)
	for _, member := range members {
		require.NotEmpty(t, member.User.ID)
		require.NotEmpty(t, member.User.Email)
		require.Equal(t, "org-1", member.OrganizationID)
	}
}

// Members of one organization must never leak into another organization's fetch.
func TestBunOrganizationMemberRepository_GetAllByOrganizationIDScopesToTheOrganization(t *testing.T) {
	t.Parallel()

	db := plugintests.SetupRepoDB(t)
	plugintests.SeedUsers(t, db, 4)
	plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
	plugintests.SeedOrganization(t, db, "org-2", "user-3", "Beta Inc", "beta-inc")

	repo := repositories.NewBunOrganizationMemberRepository(db)
	ctx := context.Background()
	plugintests.SeedOrganizationMember(t, db, "mem-a1", "org-1", "user-1", "owner")
	plugintests.SeedOrganizationMember(t, db, "mem-a2", "org-1", "user-2", "member")
	plugintests.SeedOrganizationMember(t, db, "mem-b1", "org-2", "user-3", "owner")

	first, err := repo.GetAllByOrganizationID(ctx, "org-1")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"mem-a1", "mem-a2"}, memberIDs(first))

	second, err := repo.GetAllByOrganizationID(ctx, "org-2")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"mem-b1"}, memberIDs(second))

	joined, err := repo.GetAllByOrganizationIDWithUser(ctx, "org-2")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"mem-b1"}, memberResponseIDs(joined))
}
