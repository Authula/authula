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

func TestBunOrganizationTeamRepository_Create(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	team1ID := uuid.New().String()
	team2ID := uuid.New().String()

	tests := []struct {
		name string
		team *types.OrganizationTeam
	}{
		{
			name: "keeps provided metadata",
			team: &types.OrganizationTeam{ID: team1ID, OrganizationID: org1ID, Name: "Platform", Slug: "platform", Metadata: map[string]any{"tier": "core"}},
		},
		{
			name: "keeps metadata and description",
			team: func() *types.OrganizationTeam {
				description := new(string)
				*description = "Core"
				return &types.OrganizationTeam{ID: team2ID, OrganizationID: org1ID, Name: "Core", Slug: "core", Description: description, Metadata: map[string]any{"tier": "core"}}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t, user1ID, user2ID)
			plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
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

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	team1ID := uuid.New().String()

	tests := []struct {
		name        string
		teamID      string
		setup       func(*testing.T) (repositories.OrganizationTeamRepository, context.Context)
		expectFound bool
	}{
		{
			name:        "found",
			teamID:      team1ID,
			expectFound: true,
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationTeamRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationTeam{ID: team1ID, OrganizationID: org1ID, Name: "Platform", Slug: "platform"})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:   "not found",
			teamID: "00000000-0000-0000-0000-000000000000",
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context) {
				t.Helper()
				return repositories.NewBunOrganizationTeamRepository(plugintests.SetupRepoDB(t, user1ID, user2ID)), context.Background()
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
				require.Equal(t, team1ID, found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationTeamRepository_GetByOrganizationIDAndSlug(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	team1ID := uuid.New().String()

	tests := []struct {
		name           string
		organizationID string
		slug           string
		setup          func(*testing.T) (repositories.OrganizationTeamRepository, context.Context)
		expectFound    bool
	}{
		{
			name:           "found",
			organizationID: org1ID,
			slug:           "platform",
			expectFound:    true,
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationTeamRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationTeam{ID: team1ID, OrganizationID: org1ID, Name: "Platform", Slug: "platform"})
				require.NoError(t, err)
				return repo, ctx
			},
		},
		{
			name:           "not found",
			organizationID: org1ID,
			slug:           "missing",
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context) {
				t.Helper()
				return repositories.NewBunOrganizationTeamRepository(plugintests.SetupRepoDB(t, user1ID, user2ID)), context.Background()
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
				require.Equal(t, team1ID, found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationTeamRepository_GetAllByOrganizationID(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	org2ID := uuid.New().String()
	team1ID := uuid.New().String()
	team2ID := uuid.New().String()

	tests := []struct {
		name           string
		organizationID string
		setup          func(*testing.T) (repositories.OrganizationTeamRepository, context.Context)
		expectCount    int
	}{
		{
			name:           "found",
			organizationID: org1ID,
			expectCount:    2,
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationTeamRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationTeam{ID: team1ID, OrganizationID: org1ID, Name: "Platform", Slug: "platform"})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationTeam{ID: team2ID, OrganizationID: org1ID, Name: "Core", Slug: "core"})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:           "empty",
			organizationID: org2ID,
			expectCount:    0,
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context) {
				t.Helper()
				return repositories.NewBunOrganizationTeamRepository(plugintests.SetupRepoDB(t, user1ID, user2ID)), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			found, err := repo.GetAllByOrganizationID(ctx, tt.organizationID)
			require.NoError(t, err)
			require.Len(t, found, tt.expectCount)
		})
	}
}

func TestBunOrganizationTeamRepository_Update(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	team1ID := uuid.New().String()

	tests := []struct {
		name  string
		setup func(*testing.T) (repositories.OrganizationTeamRepository, context.Context, *types.OrganizationTeam)
	}{
		{
			name: "change name and description",
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context, *types.OrganizationTeam) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationTeamRepository(db)
				ctx := context.Background()

				created, err := repo.Create(ctx, &types.OrganizationTeam{ID: team1ID, OrganizationID: org1ID, Name: "Platform", Slug: "platform"})
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

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	team1ID := uuid.New().String()

	tests := []struct {
		name   string
		teamID string
		setup  func(*testing.T) (repositories.OrganizationTeamRepository, context.Context)
	}{
		{
			name:   "delete existing",
			teamID: team1ID,
			setup: func(t *testing.T) (repositories.OrganizationTeamRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationTeamRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationTeam{ID: team1ID, OrganizationID: org1ID, Name: "Platform", Slug: "platform"})
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

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	team1ID := uuid.New().String()

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
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
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
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
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

			created, err := txRepo.Create(ctx, &types.OrganizationTeam{ID: team1ID, OrganizationID: org1ID, Name: "Platform", Slug: "platform"})
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, team1ID, created.ID)

			if tt.commit {
				require.NoError(t, tx.Commit())
			} else {
				require.NoError(t, tx.Rollback())
			}

			found, err := repo.GetByID(ctx, team1ID)
			require.NoError(t, err)
			if tt.commit {
				require.NotNil(t, found)
			} else {
				require.Nil(t, found)
			}
		})
	}
}

func TestBunOrganizationTeamRepository_Hooks(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	team1ID := uuid.New().String()

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

				beforeCalled := false
				afterCalled := false
				hooks := &plugintests.MockOrganizationTeamHooks{
					BeforeCreate: func(team *types.OrganizationTeam) error {
						beforeCalled = true
						require.Equal(t, team1ID, team.ID)
						require.Equal(t, "Platform", team.Name)
						return nil
					},
					AfterCreate: func(team types.OrganizationTeam) error {
						afterCalled = true
						require.Equal(t, team1ID, team.ID)
						return nil
					},
				}

				repo := repositories.NewBunOrganizationTeamRepository(db, hooks)
				created, err := repo.Create(context.Background(), &types.OrganizationTeam{ID: team1ID, OrganizationID: org1ID, Name: "Platform", Slug: "platform"})
				require.NoError(t, err)
				require.NotNil(t, created)
				require.True(t, beforeCalled)
				require.True(t, afterCalled)
			},
		},
		{
			name: "update hooks",
			run: func(t *testing.T) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")

				seedRepo := repositories.NewBunOrganizationTeamRepository(db)
				ctx := context.Background()
				team, err := seedRepo.Create(ctx, &types.OrganizationTeam{ID: team1ID, OrganizationID: org1ID, Name: "Platform", Slug: "platform"})
				require.NoError(t, err)

				beforeCalled := false
				afterCalled := false
				hooks := &plugintests.MockOrganizationTeamHooks{
					BeforeUpdate: func(team *types.OrganizationTeam) error {
						beforeCalled = true
						require.Equal(t, "Platform Revamp", team.Name)
						return nil
					},
					AfterUpdate: func(team types.OrganizationTeam) error {
						afterCalled = true
						require.Equal(t, "Platform Revamp", team.Name)
						return nil
					},
				}

				repo := repositories.NewBunOrganizationTeamRepository(db, hooks)
				team.Name = "Platform Revamp"
				description := new(string)
				*description = "Core platform"
				team.Description = description
				updated, err := repo.Update(ctx, team)
				require.NoError(t, err)
				require.Equal(t, "Platform Revamp", updated.Name)
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

				seedRepo := repositories.NewBunOrganizationTeamRepository(db)
				ctx := context.Background()
				_, err := seedRepo.Create(ctx, &types.OrganizationTeam{ID: team1ID, OrganizationID: org1ID, Name: "Platform", Slug: "platform"})
				require.NoError(t, err)

				beforeCalled := false
				afterCalled := false
				hooks := &plugintests.MockOrganizationTeamHooks{
					BeforeDelete: func(team *types.OrganizationTeam) error {
						beforeCalled = true
						require.Equal(t, team1ID, team.ID)
						return nil
					},
					AfterDelete: func(team types.OrganizationTeam) error {
						afterCalled = true
						require.Equal(t, team1ID, team.ID)
						return nil
					},
				}

				repo := repositories.NewBunOrganizationTeamRepository(db, hooks)
				require.NoError(t, repo.Delete(ctx, team1ID))
				found, err := repo.GetByID(ctx, team1ID)
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
