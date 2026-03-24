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
			teamMember: &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", UserID: "member-1"},
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
			teamMember: &types.OrganizationTeamMember{ID: "team-member-2", TeamID: "team-1", UserID: "member-2"},
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
			teamMember: &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", UserID: "member-1"},
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
			require.Equal(t, tt.teamMember.UserID, created.UserID)

			if tt.expectErr {
				duplicate := &types.OrganizationTeamMember{ID: tt.teamMember.ID, TeamID: tt.teamMember.TeamID, UserID: tt.teamMember.UserID}
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
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", UserID: "member-1"})
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
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", UserID: "member-1"})
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
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", UserID: "member-1"})
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
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", UserID: "member-1"})
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

func TestBunOrganizationTeamMemberRepository_GetAllByTeamID(t *testing.T) {
	t.Parallel()

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
			teamID:      "team-1",
			page:        1,
			limit:       2,
			expectCount: 2,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationMember(t, db, "member-2", "org-1", "user-2", "admin")
				plugintests.SeedUser(t, db, "user-3")
				plugintests.SeedOrganizationMember(t, db, "member-3", "org-1", "user-3", "member")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", UserID: "member-1"})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-2", TeamID: "team-1", UserID: "member-2"})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-3", TeamID: "team-1", UserID: "member-3"})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:        "second page",
			teamID:      "team-1",
			page:        2,
			limit:       2,
			expectCount: 1,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationMember(t, db, "member-2", "org-1", "user-2", "admin")
				plugintests.SeedUser(t, db, "user-3")
				plugintests.SeedOrganizationMember(t, db, "member-3", "org-1", "user-3", "member")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				repo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", UserID: "member-1"})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-2", TeamID: "team-1", UserID: "member-2"})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-3", TeamID: "team-1", UserID: "member-3"})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:        "empty",
			teamID:      "team-2",
			page:        1,
			limit:       10,
			expectCount: 0,
			setup: func(t *testing.T) (repositories.OrganizationTeamMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationMember(t, db, "member-2", "org-1", "user-2", "admin")
				plugintests.SeedUser(t, db, "user-3")
				plugintests.SeedOrganizationMember(t, db, "member-3", "org-1", "user-3", "member")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
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
				require.Equal(t, "team-1", teamMember.TeamID)
			}
			if tt.page == 1 && len(found) > 1 {
				require.Equal(t, "team-member-3", found[0].ID)
				require.Equal(t, "team-member-2", found[1].ID)
			}
			if tt.page == 2 && len(found) > 0 {
				require.Equal(t, "team-member-1", found[0].ID)
			}
		})
	}
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
				_, err := repo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", UserID: "member-1"})
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
			created, err := txRepo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", UserID: "member-1"})
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

func TestBunOrganizationTeamMemberRepository_Hooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "create hooks",
			run: func(t *testing.T) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")

				beforeCalled := false
				afterCalled := false
				hooks := &plugintests.MockOrganizationTeamMemberHooks{
					BeforeCreate: func(teamMember *types.OrganizationTeamMember) error {
						beforeCalled = true
						require.Equal(t, "team-member-1", teamMember.ID)
						return nil
					},
					AfterCreate: func(teamMember types.OrganizationTeamMember) error {
						afterCalled = true
						require.Equal(t, "team-member-1", teamMember.ID)
						return nil
					},
				}

				repo := repositories.NewBunOrganizationTeamMemberRepository(db, hooks)
				created, err := repo.Create(context.Background(), &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", UserID: "member-1"})
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
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "member-1", "org-1", "user-1", "member")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")

				seedRepo := repositories.NewBunOrganizationTeamMemberRepository(db)
				ctx := context.Background()
				_, err := seedRepo.Create(ctx, &types.OrganizationTeamMember{ID: "team-member-1", TeamID: "team-1", UserID: "member-1"})
				require.NoError(t, err)

				beforeCalled := false
				afterCalled := false
				hooks := &plugintests.MockOrganizationTeamMemberHooks{
					BeforeDelete: func(teamMember *types.OrganizationTeamMember) error {
						beforeCalled = true
						require.Equal(t, "team-member-1", teamMember.ID)
						return nil
					},
					AfterDelete: func(teamMember types.OrganizationTeamMember) error {
						afterCalled = true
						require.Equal(t, "team-member-1", teamMember.ID)
						return nil
					},
				}

				repo := repositories.NewBunOrganizationTeamMemberRepository(db, hooks)
				require.NoError(t, repo.DeleteByTeamIDAndMemberID(ctx, "team-1", "member-1"))
				found, err := repo.GetByTeamIDAndMemberID(ctx, "team-1", "member-1")
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
