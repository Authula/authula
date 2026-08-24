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

func TestBunOrganizationTeamMemberRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		teamMember *types.OrganizationTeamMember
		expectErr  bool
		setup      func(*testing.T) (repositories.OrganizationTeamMemberRepository, context.Context)
	}{
		{
			name:       "team member",
			teamMember: &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "member-1"},
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationMember(t, db, "member-2", "org-1", "user-2", "admin")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				return repositories.NewBunOrganizationTeamMemberRepository(db), context.Background()
			},
		},
		{
			name:       "another team member",
			teamMember: &types.OrganizationTeamMember{ID: "team-member-2", TeamID: "team-1", MemberID: "member-2"},
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationMember(t, db, "member-2", "org-1", "user-2", "admin")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				return repositories.NewBunOrganizationTeamMemberRepository(db), context.Background()
			},
		},
		{
			name:       "duplicate id returns error",
			teamMember: &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "member-1"},
			expectErr:  true,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationMember(t, db, "member-2", "org-1", "user-2", "admin")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				return repositories.NewBunOrganizationTeamMemberRepository(db), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			created, err := repo.Create(ctx, tt.teamMember)
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, tt.teamMember.ID, created.ID)
			require.Equal(t, tt.teamMember.TeamID, created.TeamID)
			require.Equal(t, tt.teamMember.MemberID, created.MemberID)

			if tt.expectErr {
				duplicate := &types.OrganizationTeamMember{ID: tt.teamMember.ID, TeamID: tt.teamMember.TeamID, MemberID: tt.teamMember.MemberID}
				duplicateCreated, duplicateErr := repo.Create(ctx, duplicate)
				require.Error(t, duplicateErr)
				require.Nil(t, duplicateCreated)

				found, err := repo.GetByID(ctx, tt.teamMember.ID)
				require.NoError(t, err)
				require.NotNil(t, found)
				require.Equal(t, tt.teamMember.ID, found.ID)
			}
		})
	}
}

func TestBunOrganizationTeamMemberRepository_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		teamMemberID string
		setup        func(*testing.T) (repositories.OrganizationTeamMemberRepository, context.Context)
		expectFound  bool
	}{
		{
			name:         "found",
			teamMemberID: "team-member-1",
			expectFound:  true,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationMember(t, db, "member-2", "org-1", "user-2", "admin")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "member-1"})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:         "not found",
			teamMemberID: "missing",
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				return repositories.NewBunOrganizationTeamMemberRepository(plugintests.SetupRepoDB(t)), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			found, err := repo.GetByID(ctx, tt.teamMemberID)
			require.NoError(t, err)
			if tt.expectFound {
				require.NotNil(t, found)
				require.Equal(t, "team-member-1", found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationTeamMemberRepository_GetByTeamIDAndMemberID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		teamID      string
		memberID    string
		setup       func(*testing.T) (repositories.OrganizationTeamMemberRepository, context.Context)
		expectFound bool
	}{
		{
			name:        "found",
			teamID:      "team-1",
			memberID:    "member-1",
			expectFound: true,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationMember(t, db, "member-2", "org-1", "user-2", "admin")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				plugintests.SeedOrganizationTeam(t, db, "team-2", "org-1", "Core", "core")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "member-1"})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:     "wrong member",
			teamID:   "team-1",
			memberID: "member-2",
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationMember(t, db, "member-2", "org-1", "user-2", "admin")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				plugintests.SeedOrganizationTeam(t, db, "team-2", "org-1", "Core", "core")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "member-1"})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:     "wrong team",
			teamID:   "team-2",
			memberID: "member-1",
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationMember(t, db, "member-2", "org-1", "user-2", "admin")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				plugintests.SeedOrganizationTeam(t, db, "team-2", "org-1", "Core", "core")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "member-1"})
				require.NoError(t, err)
				return repo, ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			found, err := repo.GetByTeamIDAndMemberID(ctx, tt.teamID, tt.memberID)
			require.NoError(t, err)
			if tt.expectFound {
				require.NotNil(t, found)
				require.Equal(t, "team-member-1", found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func seedTeamMembers(t *testing.T, count int) (repositories.OrganizationTeamMemberRepository, context.Context) {
	t.Helper()

	db := plugintests.SetupRepoDB(t)
	plugintests.SeedUsers(t, db, count)
	plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
	plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")

	repo := repositories.NewBunOrganizationTeamMemberRepository(db)
	ctx := context.Background()
	for i := 1; i <= count; i++ {
		plugintests.SeedOrganizationMember(t, db, fmt.Sprintf("member-%d", i), "org-1", fmt.Sprintf("user-%d", i), "member")
		_, err := repo.Create(ctx, &types.OrganizationTeamMember{
			ID:       fmt.Sprintf("team-member-%d", i),
			TeamID:   "team-1",
			MemberID: fmt.Sprintf("member-%d", i),
		})
		require.NoError(t, err)
	}

	return repo, ctx
}

func teamMemberIDs(teamMembers []types.OrganizationTeamMember) []string {
	ids := make([]string, 0, len(teamMembers))
	for _, teamMember := range teamMembers {
		ids = append(ids, teamMember.ID)
	}
	return ids
}

func teamMemberResponseIDs(teamMembers []types.OrganizationTeamMemberResponse) []string {
	ids := make([]string, 0, len(teamMembers))
	for _, teamMember := range teamMembers {
		ids = append(ids, teamMember.ID)
	}
	return ids
}

func TestBunOrganizationTeamMemberRepository_GetAllByTeamID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		teamID      string
		page        int
		limit       int
		expectCount int
		expectTotal int
	}{
		{name: "first page", teamID: "team-1", page: 1, limit: 2, expectCount: 2, expectTotal: 5},
		{name: "last partial page", teamID: "team-1", page: 3, limit: 2, expectCount: 1, expectTotal: 5},
		{name: "page past the end keeps the true total", teamID: "team-1", page: 4, limit: 2, expectCount: 0, expectTotal: 5},
		{name: "page zero does not error", teamID: "team-1", page: 0, limit: 2, expectCount: 2, expectTotal: 5},
		{name: "negative page does not error", teamID: "team-1", page: -1, limit: 2, expectCount: 2, expectTotal: 5},
		{name: "zero limit falls back to a bounded page", teamID: "team-1", page: 1, limit: 0, expectCount: 5, expectTotal: 5},
		{name: "unknown team is empty", teamID: "team-2", page: 1, limit: 10, expectCount: 0, expectTotal: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := seedTeamMembers(t, 5)

			found, total, err := repo.GetAllByTeamID(ctx, tt.teamID, tt.page, tt.limit)
			require.NoError(t, err)
			require.Len(t, found, tt.expectCount)
			require.Equal(t, tt.expectTotal, total)
			for _, teamMember := range found {
				require.Equal(t, tt.teamID, teamMember.TeamID)
			}
		})
	}
}

func TestBunOrganizationTeamMemberRepository_GetAllByTeamIDPagesPartitionCleanly(t *testing.T) {
	t.Parallel()

	repo, ctx := seedTeamMembers(t, 5)

	seen := make([]string, 0, 5)
	for page := 1; page <= 3; page++ {
		found, total, err := repo.GetAllByTeamID(ctx, "team-1", page, 2)
		require.NoError(t, err)
		require.Equal(t, 5, total)
		seen = append(seen, teamMemberIDs(found)...)
	}

	require.ElementsMatch(t, []string{"team-member-1", "team-member-2", "team-member-3", "team-member-4", "team-member-5"}, seen)
}

func TestBunOrganizationTeamMemberRepository_GetAllByTeamIDWithMemberAndUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		teamID      string
		page        int
		limit       int
		expectCount int
		expectTotal int
	}{
		{name: "first page", teamID: "team-1", page: 1, limit: 2, expectCount: 2, expectTotal: 5},
		{name: "last partial page", teamID: "team-1", page: 3, limit: 2, expectCount: 1, expectTotal: 5},
		{name: "page past the end keeps the true total", teamID: "team-1", page: 4, limit: 2, expectCount: 0, expectTotal: 5},
		{name: "page zero does not error", teamID: "team-1", page: 0, limit: 2, expectCount: 2, expectTotal: 5},
		{name: "zero limit falls back to a bounded page", teamID: "team-1", page: 1, limit: 0, expectCount: 5, expectTotal: 5},
		{name: "limit beyond the maximum is capped", teamID: "team-1", page: 1, limit: 100000, expectCount: 5, expectTotal: 5},
		{name: "unknown team is empty", teamID: "team-2", page: 1, limit: 10, expectCount: 0, expectTotal: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := seedTeamMembers(t, 5)

			found, total, err := repo.GetAllByTeamIDWithMemberAndUser(ctx, tt.teamID, tt.page, tt.limit)
			require.NoError(t, err)
			require.Len(t, found, tt.expectCount)
			require.Equal(t, tt.expectTotal, total)
			for _, teamMember := range found {
				require.NotEmpty(t, teamMember.Member.User.ID, "joined user must be populated")
			}
		})
	}
}

func TestBunOrganizationTeamMemberRepository_GetAllByTeamIDWithMemberAndUserPagesPartitionCleanly(t *testing.T) {
	t.Parallel()

	repo, ctx := seedTeamMembers(t, 5)

	seen := make([]string, 0, 5)
	for page := 1; page <= 3; page++ {
		found, total, err := repo.GetAllByTeamIDWithMemberAndUser(ctx, "team-1", page, 2)
		require.NoError(t, err)
		require.Equal(t, 5, total)
		seen = append(seen, teamMemberResponseIDs(found)...)
	}

	require.ElementsMatch(t, []string{"team-member-1", "team-member-2", "team-member-3", "team-member-4", "team-member-5"}, seen)
}

func TestBunOrganizationTeamMemberRepository_DeleteByTeamIDAndMemberID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		teamID   string
		memberID string
		setup    func(*testing.T) (repositories.OrganizationTeamMemberRepository, context.Context)
	}{
		{
			name:     "delete existing",
			teamID:   "team-1",
			memberID: "member-1",
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationMember(t, db, "member-2", "org-1", "user-2", "admin")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "member-1"})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:     "delete missing",
			teamID:   "team-1",
			memberID: "missing",
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationMember(t, db, "member-2", "org-1", "user-2", "admin")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				return repositories.NewBunOrganizationTeamMemberRepository(db), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			require.NoError(t, repo.DeleteByTeamIDAndMemberID(ctx, tt.teamID, tt.memberID))
			found, err := repo.GetByTeamIDAndMemberID(ctx, tt.teamID, tt.memberID)
			require.NoError(t, err)
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationTeamMemberRepository_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		commit bool
		setup  func(*testing.T) (repositories.OrganizationTeamMemberRepository, context.Context, repositories.OrganizationTeamMemberRepository, bun.Tx)
	}{
		{
			name:   "commits through tx",
			commit: true,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context, repositories.OrganizationTeamMemberRepository, bun.Tx) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()

				tx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)
				txRepo := repo.WithTx(tx)
				require.NotNil(t, txRepo)
				require.IsType(t, &repositories.BunOrganizationTeamMemberRepository{}, txRepo)
				return repo, ctx, txRepo, tx
			},
		},
		{
			name:   "rolls back through tx",
			commit: false,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context, repositories.OrganizationTeamMemberRepository, bun.Tx) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()

				tx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)
				txRepo := repo.WithTx(tx)
				require.NotNil(t, txRepo)
				require.IsType(t, &repositories.BunOrganizationTeamMemberRepository{}, txRepo)
				return repo, ctx, txRepo, tx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx, txRepo, tx := tt.setup(t)
			created, err := txRepo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", MemberID: "member-1"})
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, "team-member-1", created.ID)

			if tt.commit {
				require.NoError(t, tx.Commit())
			} else {
				require.NoError(t, tx.Rollback())
			}

			found, err := repo.GetByID(ctx, "team-member-1")
			require.NoError(t, err)
			if tt.commit {
				require.NotNil(t, found)
				require.Equal(t, "team-member-1", found.ID)
			} else {
				require.Nil(t, found)
			}
		})
	}
}
