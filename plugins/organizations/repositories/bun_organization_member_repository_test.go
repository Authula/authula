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

func TestBunOrganizationMemberRepository_CreateGetUpdateDelete(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	mem1ID := uuid.New().String()

	tests := []struct {
		name string
		run  func(*testing.T, repositories.OrganizationMemberRepository, context.Context, *types.OrganizationMember)
	}{
		{
			name: "get by id returns created member",
			run: func(t *testing.T, orgMemberRepo repositories.OrganizationMemberRepository, ctx context.Context, created *types.OrganizationMember) {
				t.Helper()
				found, err := orgMemberRepo.GetByID(ctx, mem1ID)
				require.NoError(t, err)
				require.NotNil(t, found)
				require.Equal(t, created.Role, found.Role)
			},
		},
		{
			name: "list by organization returns created member",
			run: func(t *testing.T, orgMemberRepo repositories.OrganizationMemberRepository, ctx context.Context, created *types.OrganizationMember) {
				t.Helper()
				members, err := orgMemberRepo.GetAllByOrganizationID(ctx, org1ID, 1, 10)
				require.NoError(t, err)
				require.Len(t, members, 1)
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

			db := plugintests.SetupRepoDB(t, user1ID, user2ID)
			plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
			orgMemberRepo := repositories.NewBunOrganizationMemberRepository(db)
			ctx := context.Background()
			created, err := orgMemberRepo.Create(ctx, &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user2ID, Role: "member"})
			require.NoError(t, err)
			require.NotNil(t, created)

			tt.run(t, orgMemberRepo, ctx, created)
		})
	}
}

func TestBunOrganizationMemberRepository_CountByOrganizationID(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	mem1ID := uuid.New().String()
	mem2ID := uuid.New().String()

	tests := []struct {
		name  string
		setup func(*testing.T) (repositories.OrganizationMemberRepository, context.Context)
	}{
		{
			name: "counts members in organization",
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user1ID, Role: "member"})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationMember{ID: mem2ID, OrganizationID: org1ID, UserID: user2ID, Role: "admin"})
				require.NoError(t, err)

				return repo, ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)
			count, err := repo.CountByOrganizationID(ctx, org1ID)
			require.NoError(t, err)
			require.Equal(t, 2, count)
		})
	}
}

func TestBunOrganizationMemberRepository_Create(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	mem1ID := uuid.New().String()
	mem2ID := uuid.New().String()

	tests := []struct {
		name   string
		member *types.OrganizationMember
	}{
		{name: "member", member: &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user1ID, Role: "member"}},
		{name: "admin", member: &types.OrganizationMember{ID: mem2ID, OrganizationID: org1ID, UserID: user2ID, Role: "admin"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t, user1ID, user2ID)
			plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
			repo := repositories.NewBunOrganizationMemberRepository(db)
			created, err := repo.Create(context.Background(), tt.member)
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, tt.member.ID, created.ID)
			require.Equal(t, tt.member.Role, created.Role)
		})
	}
}

func TestBunOrganizationMemberRepository_GetAllByOrganizationID(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	user3ID := uuid.New().String()
	org1ID := uuid.New().String()
	org2ID := uuid.New().String()
	mem1ID := uuid.New().String()
	mem2ID := uuid.New().String()
	mem3ID := uuid.New().String()

	tests := []struct {
		name           string
		organizationID string
		page           int
		limit          int
		setup          func(*testing.T) (repositories.OrganizationMemberRepository, context.Context)
		expectCount    int
	}{
		{
			name:           "first page",
			organizationID: org1ID,
			page:           1,
			limit:          1,
			expectCount:    1,
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedUser(t, db, user3ID)
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user1ID, Role: "member"})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationMember{ID: mem2ID, OrganizationID: org1ID, UserID: user2ID, Role: "admin"})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationMember{ID: mem3ID, OrganizationID: org1ID, UserID: user3ID, Role: "member"})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:           "second page",
			organizationID: org1ID,
			page:           2,
			limit:          1,
			expectCount:    1,
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedUser(t, db, user3ID)
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user1ID, Role: "member"})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationMember{ID: mem2ID, OrganizationID: org1ID, UserID: user2ID, Role: "admin"})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationMember{ID: mem3ID, OrganizationID: org1ID, UserID: user3ID, Role: "member"})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:           "empty result",
			organizationID: org2ID,
			page:           1,
			limit:          10,
			expectCount:    0,
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationMemberRepository(db)
				return repo, context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			members, err := repo.GetAllByOrganizationID(ctx, tt.organizationID, tt.page, tt.limit)
			require.NoError(t, err)
			require.Len(t, members, tt.expectCount)
			if tt.page == 1 && len(members) > 0 {
				require.Equal(t, mem3ID, members[0].ID)
			}
			if tt.page == 2 && len(members) > 0 {
				require.Equal(t, mem2ID, members[0].ID)
			}
		})
	}
}

func TestBunOrganizationMemberRepository_GetAllByUserID(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	user3ID := uuid.New().String()
	org1ID := uuid.New().String()
	org2ID := uuid.New().String()
	mem1ID := uuid.New().String()
	mem2ID := uuid.New().String()

	tests := []struct {
		name        string
		userID      string
		setup       func(*testing.T) (repositories.OrganizationMemberRepository, context.Context)
		expectCount int
	}{
		{
			name:        "found",
			userID:      user1ID,
			expectCount: 2,
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganization(t, db, org2ID, user2ID, "Platform", "platform")
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user1ID, Role: "member"})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationMember{ID: mem2ID, OrganizationID: org2ID, UserID: user1ID, Role: "admin"})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:        "empty",
			userID:      user3ID,
			expectCount: 0,
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				plugintests.SeedOrganization(t, db, org2ID, user2ID, "Platform", "platform")
				return repositories.NewBunOrganizationMemberRepository(db), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			members, err := repo.GetAllByUserID(ctx, tt.userID)
			require.NoError(t, err)
			require.Len(t, members, tt.expectCount)
		})
	}
}

func TestBunOrganizationMemberRepository_GetByOrganizationIDAndUserID(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	mem1ID := uuid.New().String()

	tests := []struct {
		name           string
		organizationID string
		userID         string
		setup          func(*testing.T) (repositories.OrganizationMemberRepository, context.Context)
		expectFound    bool
	}{
		{
			name:           "found",
			organizationID: org1ID,
			userID:         user1ID,
			expectFound:    true,
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user1ID, Role: "member"})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:           "not found",
			organizationID: org1ID,
			userID:         user2ID,
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
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
				require.Equal(t, mem1ID, found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationMemberRepository_GetByID(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	mem1ID := uuid.New().String()

	tests := []struct {
		name        string
		memberID    string
		setup       func(*testing.T) (repositories.OrganizationMemberRepository, context.Context)
		expectFound bool
	}{
		{
			name:        "found",
			memberID:    mem1ID,
			expectFound: true,
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user1ID, Role: "member"})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:     "not found",
			memberID: "00000000-0000-0000-0000-000000000000",
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				return repositories.NewBunOrganizationMemberRepository(plugintests.SetupRepoDB(t, user1ID, user2ID)), context.Background()
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
				require.Equal(t, mem1ID, found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationMemberRepository_Update(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	mem1ID := uuid.New().String()

	tests := []struct {
		name  string
		setup func(*testing.T) (repositories.OrganizationMemberRepository, context.Context, *types.OrganizationMember)
	}{
		{
			name: "change role",
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context, *types.OrganizationMember) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()

				created, err := repo.Create(ctx, &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user1ID, Role: "member"})
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

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	mem1ID := uuid.New().String()

	tests := []struct {
		name     string
		memberID string
		setup    func(*testing.T) (repositories.OrganizationMemberRepository, context.Context)
	}{
		{
			name:     "delete existing",
			memberID: mem1ID,
			setup: func(t *testing.T) (repositories.OrganizationMemberRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user1ID, Role: "member"})
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

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	mem1ID := uuid.New().String()

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
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
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
				db := plugintests.SetupRepoDB(t, user1ID, user2ID)
				plugintests.SeedOrganization(t, db, org1ID, user1ID, "Acme Inc", "acme-inc")
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

			created, err := txRepo.Create(ctx, &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user1ID, Role: "member"})
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, mem1ID, created.ID)

			if tt.commit {
				require.NoError(t, tx.Commit())
			} else {
				require.NoError(t, tx.Rollback())
			}

			found, err := repo.GetByID(ctx, mem1ID)
			require.NoError(t, err)
			if tt.commit {
				require.NotNil(t, found)
			} else {
				require.Nil(t, found)
			}
		})
	}
}

func TestBunOrganizationMemberRepository_Hooks(t *testing.T) {
	t.Parallel()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	org1ID := uuid.New().String()
	mem1ID := uuid.New().String()

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
				hooks := &plugintests.MockOrganizationMemberHooks{
					Before: func(member *types.OrganizationMember) error {
						beforeCalled = true
						require.Equal(t, mem1ID, member.ID)
						return nil
					},
					After: func(member types.OrganizationMember) error {
						afterCalled = true
						require.Equal(t, mem1ID, member.ID)
						return nil
					},
				}

				repo := repositories.NewBunOrganizationMemberRepository(db, hooks)
				created, err := repo.Create(context.Background(), &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user2ID, Role: "member"})
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

				seedRepo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()
				member, err := seedRepo.Create(ctx, &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user2ID, Role: "member"})
				require.NoError(t, err)

				beforeCalled := false
				afterCalled := false
				hooks := &plugintests.MockOrganizationMemberHooks{
					BeforeUpdate: func(member *types.OrganizationMember) error {
						beforeCalled = true
						require.Equal(t, "admin", member.Role)
						return nil
					},
					AfterUpdate: func(member types.OrganizationMember) error {
						afterCalled = true
						require.Equal(t, "admin", member.Role)
						return nil
					},
				}

				repo := repositories.NewBunOrganizationMemberRepository(db, hooks)
				member.Role = "admin"
				updated, err := repo.Update(ctx, member)
				require.NoError(t, err)
				require.Equal(t, "admin", updated.Role)
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

				seedRepo := repositories.NewBunOrganizationMemberRepository(db)
				ctx := context.Background()
				_, err := seedRepo.Create(ctx, &types.OrganizationMember{ID: mem1ID, OrganizationID: org1ID, UserID: user2ID, Role: "member"})
				require.NoError(t, err)

				beforeCalled := false
				afterCalled := false
				hooks := &plugintests.MockOrganizationMemberHooks{
					BeforeDelete: func(member *types.OrganizationMember) error {
						beforeCalled = true
						require.Equal(t, mem1ID, member.ID)
						return nil
					},
					AfterDelete: func(member types.OrganizationMember) error {
						afterCalled = true
						require.Equal(t, mem1ID, member.ID)
						return nil
					},
				}

				repo := repositories.NewBunOrganizationMemberRepository(db, hooks)
				require.NoError(t, repo.Delete(ctx, mem1ID))
				found, err := repo.GetByID(ctx, mem1ID)
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
