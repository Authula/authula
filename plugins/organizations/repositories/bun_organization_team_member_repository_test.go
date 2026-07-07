package repositories_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/repositories"
	plugintests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
)

func TestBunOrganizationTeamMemberRepository_Create(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	member1ID := uuid.New().String()
	member2ID := uuid.New().String()
	team1ID := uuid.New().String()
	teamMember1ID := uuid.New().String()
	teamMember2ID := uuid.New().String()

	tests := []struct {
		name       string
		teamMember *types.OrganizationTeamMember
		expectErr  bool
		setup      func(*testing.T) (repositories.OrganizationTeamMemberRepository, context.Context)
	}{
		{
			name:       "team member",
			teamMember: &types.OrganizationTeamMember{ID: teamMember1ID, TeamID: team1ID, MemberID: member1ID},
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationMember(t, db, member2ID, org1ID, user2ID, "admin")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
				return repositories.NewBunOrganizationTeamMemberRepository(db), context.Background()
			},
		},
		{
			name:       "another team member",
			teamMember: &types.OrganizationTeamMember{ID: teamMember2ID, TeamID: team1ID, MemberID: member2ID},
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationMember(t, db, member2ID, org1ID, user2ID, "admin")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
				return repositories.NewBunOrganizationTeamMemberRepository(db), context.Background()
			},
		},
		{
			name:       "duplicate id returns error",
			teamMember: &types.OrganizationTeamMember{ID: teamMember1ID, TeamID: team1ID, MemberID: member1ID},
			expectErr:  true,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationMember(t, db, member2ID, org1ID, user2ID, "admin")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
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

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	member1ID := uuid.New().String()
	member2ID := uuid.New().String()
	team1ID := uuid.New().String()
	teamMember1ID := uuid.New().String()

	tests := []struct {
		name         string
		teamMemberID string
		setup        func(*testing.T) (repositories.OrganizationTeamMemberRepository, context.Context)
		expectFound  bool
	}{
		{
			name:         "found",
			teamMemberID: teamMember1ID,
			expectFound:  true,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationMember(t, db, member2ID, org1ID, user2ID, "admin")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: teamMember1ID, TeamID: team1ID, MemberID: member1ID})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:         "not found",
			teamMemberID: "00000000-0000-0000-0000-000000000000",
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				return repositories.NewBunOrganizationTeamMemberRepository(plugintests.SetupRepoDB(t, user1ID, user2ID)), context.Background()
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
				require.Equal(t, teamMember1ID, found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationTeamMemberRepository_GetByTeamIDAndMemberID(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	member1ID := uuid.New().String()
	member2ID := uuid.New().String()
	team1ID := uuid.New().String()
	team2ID := uuid.New().String()
	teamMember1ID := uuid.New().String()

	tests := []struct {
		name        string
		teamID      string
		memberID    string
		setup       func(*testing.T) (repositories.OrganizationTeamMemberRepository, context.Context)
		expectFound bool
	}{
		{
			name:        "found",
			teamID:      team1ID,
			memberID:    member1ID,
			expectFound: true,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationMember(t, db, member2ID, org1ID, user2ID, "admin")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
				plugintests.SeedOrganizationTeam(t, db, team2ID, org1ID, "Core", "core")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: teamMember1ID, TeamID: team1ID, MemberID: member1ID})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:     "wrong member",
			teamID:   team1ID,
			memberID: member2ID,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationMember(t, db, member2ID, org1ID, user2ID, "admin")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
				plugintests.SeedOrganizationTeam(t, db, team2ID, org1ID, "Core", "core")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: teamMember1ID, TeamID: team1ID, MemberID: member1ID})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:     "wrong team",
			teamID:   team2ID,
			memberID: member1ID,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationMember(t, db, member2ID, org1ID, user2ID, "admin")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
				plugintests.SeedOrganizationTeam(t, db, team2ID, org1ID, "Core", "core")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: teamMember1ID, TeamID: team1ID, MemberID: member1ID})
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
				require.Equal(t, teamMember1ID, found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationTeamMemberRepository_GetAllByTeamID(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	user3ID := uuid.New().String()
	org1ID := uuid.New().String()
	member1ID := uuid.New().String()
	member2ID := uuid.New().String()
	member3ID := uuid.New().String()
	team1ID := uuid.New().String()
	team2ID := uuid.New().String()
	teamMember1ID := uuid.New().String()
	teamMember2ID := uuid.New().String()
	teamMember3ID := uuid.New().String()

	tests := []struct {
		name        string
		teamID      string
		page        int
		limit       int
		setup       func(*testing.T) (repositories.OrganizationTeamMemberRepository, context.Context)
		expectCount int
	}{
		{
			name:        "first page",
			teamID:      team1ID,
			page:        1,
			limit:       2,
			expectCount: 2,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationMember(t, db, member2ID, org1ID, user2ID, "admin")
				plugintests.SeedUser(t, db, user3ID)
				plugintests.SeedOrganizationMember(t, db, member3ID, org1ID, user3ID, "member")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: teamMember1ID, TeamID: team1ID, MemberID: member1ID})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationTeamMember{ID: teamMember2ID, TeamID: team1ID, MemberID: member2ID})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationTeamMember{ID: teamMember3ID, TeamID: team1ID, MemberID: member3ID})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:        "second page",
			teamID:      team1ID,
			page:        2,
			limit:       2,
			expectCount: 1,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationMember(t, db, member2ID, org1ID, user2ID, "admin")
				plugintests.SeedUser(t, db, user3ID)
				plugintests.SeedOrganizationMember(t, db, member3ID, org1ID, user3ID, "member")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: teamMember1ID, TeamID: team1ID, MemberID: member1ID})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationTeamMember{ID: teamMember2ID, TeamID: team1ID, MemberID: member2ID})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationTeamMember{ID: teamMember3ID, TeamID: team1ID, MemberID: member3ID})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:        "empty",
			teamID:      team2ID,
			page:        1,
			limit:       10,
			expectCount: 0,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationMember(t, db, member2ID, org1ID, user2ID, "admin")
				plugintests.SeedUser(t, db, user3ID)
				plugintests.SeedOrganizationMember(t, db, member3ID, org1ID, user3ID, "member")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
				return repositories.NewBunOrganizationTeamMemberRepository(db), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			found, err := repo.GetAllByTeamID(ctx, tt.teamID, tt.page, tt.limit)
			require.NoError(t, err)
			require.Len(t, found, tt.expectCount)
			for _, teamMember := range found {
				require.Equal(t, team1ID, teamMember.TeamID)
			}
			if tt.page == 1 && len(found) > 1 {
				require.Equal(t, teamMember3ID, found[0].ID)
				require.Equal(t, teamMember2ID, found[1].ID)
			}
			if tt.page == 2 && len(found) > 0 {
				require.Equal(t, teamMember1ID, found[0].ID)
			}
		})
	}
}

func TestBunOrganizationTeamMemberRepository_DeleteByTeamIDAndMemberID(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	member1ID := uuid.New().String()
	member2ID := uuid.New().String()
	team1ID := uuid.New().String()
	teamMember1ID := uuid.New().String()

	tests := []struct {
		name     string
		teamID   string
		memberID string
		setup    func(*testing.T) (repositories.OrganizationTeamMemberRepository, context.Context)
	}{
		{
			name:     "delete existing",
			teamID:   team1ID,
			memberID: member1ID,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationMember(t, db, member2ID, org1ID, user2ID, "admin")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: teamMember1ID, TeamID: team1ID, MemberID: member1ID})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:     "delete missing",
			teamID:   team1ID,
			memberID: "00000000-0000-0000-0000-000000000000",
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationMember(t, db, member2ID, org1ID, user2ID, "admin")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
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

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	member1ID := uuid.New().String()
	team1ID := uuid.New().String()
	teamMember1ID := uuid.New().String()

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
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
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
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")
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
			created, err := txRepo.Create(ctx, &types.OrganizationTeamMember{ID: teamMember1ID, TeamID: team1ID, MemberID: member1ID})
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, teamMember1ID, created.ID)

			if tt.commit {
				require.NoError(t, tx.Commit())
			} else {
				require.NoError(t, tx.Rollback())
			}

			found, err := repo.GetByID(ctx, teamMember1ID)
			require.NoError(t, err)
			if tt.commit {
				require.NotNil(t, found)
				require.Equal(t, teamMember1ID, found.ID)
			} else {
				require.Nil(t, found)
			}
		})
	}
}

func TestBunOrganizationTeamMemberRepository_Hooks(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	member1ID := uuid.New().String()
	team1ID := uuid.New().String()
	teamMember1ID := uuid.New().String()

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "create hooks",
			run: func(t *testing.T) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")

				beforeCalled := false
				afterCalled := false
				hooks := &plugintests.MockOrganizationTeamMemberHooks{
					BeforeCreate: func(teamMember *types.OrganizationTeamMember) error {
						beforeCalled = true
						require.Equal(t, teamMember1ID, teamMember.ID)
						return nil
					},
					AfterCreate: func(teamMember types.OrganizationTeamMember) error {
						afterCalled = true
						require.Equal(t, teamMember1ID, teamMember.ID)
						return nil
					},
				}

				repo := repositories.NewBunOrganizationTeamMemberRepository(db, hooks)
				created, err := repo.Create(context.Background(), &types.OrganizationTeamMember{ID: teamMember1ID, TeamID: team1ID, MemberID: member1ID})
				require.NoError(t, err)
				require.NotNil(t, created)
				require.True(t, beforeCalled)
				require.True(t, afterCalled)
			},
		},
		{
			name: "delete hooks",
			run: func(t *testing.T) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, member1ID, org1ID, user1ID, "member")
				plugintests.SeedOrganizationTeam(t, db, team1ID, org1ID, "Platform", "platform")

				seedRepo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := seedRepo.Create(ctx, &types.OrganizationTeamMember{ID: teamMember1ID, TeamID: team1ID, MemberID: member1ID})
				require.NoError(t, err)

				beforeCalled := false
				afterCalled := false
				hooks := &plugintests.MockOrganizationTeamMemberHooks{
					BeforeDelete: func(teamMember *types.OrganizationTeamMember) error {
						beforeCalled = true
						require.Equal(t, teamMember1ID, teamMember.ID)
						return nil
					},
					AfterDelete: func(teamMember types.OrganizationTeamMember) error {
						afterCalled = true
						require.Equal(t, teamMember1ID, teamMember.ID)
						return nil
					},
				}

				repo := repositories.NewBunOrganizationTeamMemberRepository(db, hooks)
				require.NoError(t, repo.DeleteByTeamIDAndMemberID(ctx, team1ID, member1ID))
				found, err := repo.GetByTeamIDAndMemberID(ctx, team1ID, member1ID)
				require.NoError(t, err)
				require.Nil(t, found)
				require.True(t, beforeCalled)
				require.True(t, afterCalled)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}
