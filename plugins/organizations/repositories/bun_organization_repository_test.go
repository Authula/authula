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
			organization: &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc", Metadata: []byte(`{"tier":"core"}`)},
		},
		{
			name:         "keeps alternate metadata",
			organization: &types.Organization{ID: "org-2", OwnerID: "user-1", Name: "Platform", Slug: "platform", Metadata: []byte(`{"tier":"platform"}`)},
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
			require.Equal(t, string(tt.organization.Metadata), string(created.Metadata))
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

				_, err := repo.Create(ctx, &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc", Metadata: []byte(`{"tier":"core"}`)})
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

				_, err := repo.Create(ctx, &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc", Metadata: []byte(`{"tier":"core"}`)})
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

func TestBunOrganizationRepository_GetAllByOwnerID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		ownerID     string
		setup       func(*testing.T) (repositories.OrganizationRepository, context.Context)
		expectCount int
	}{
		{
			name:        "found",
			ownerID:     "user-1",
			expectCount: 2,
			setup: func(t *testing.T) (repositories.OrganizationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				repo := repositories.NewBunOrganizationRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc"})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.Organization{ID: "org-2", OwnerID: "user-1", Name: "Platform", Slug: "platform"})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:        "empty",
			ownerID:     "user-2",
			expectCount: 0,
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

			found, err := repo.GetAllByOwnerID(ctx, tt.ownerID)
			require.NoError(t, err)
			require.Len(t, found, tt.expectCount)
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

				created, err := repo.Create(ctx, &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc", Metadata: []byte(`{"tier":"core"}`)})
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

			created, err := txRepo.Create(ctx, &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc", Metadata: []byte(`{"tier":"core"}`)})
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

func TestBunOrganizationRepository_CreateHooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(*testing.T) (*repositories.BunOrganizationRepository, context.Context, *types.Organization)
		expectErr  bool
		expectBool bool
	}{
		{
			name: "hooks are called on create",
			setup: func(t *testing.T) (*repositories.BunOrganizationRepository, context.Context, *types.Organization) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				var beforeCalled bool
				var afterCalled bool
				hooks := &plugintests.MockOrganizationHooks{
					BeforeCreate: func(organization *types.Organization) error {
						beforeCalled = true
						require.Equal(t, "org-1", organization.ID)
						return nil
					},
					AfterCreate: func(organization types.Organization) error {
						afterCalled = true
						require.Equal(t, "org-1", organization.ID)
						return nil
					},
				}
				repo := repositories.NewBunOrganizationRepository(db, hooks).(*repositories.BunOrganizationRepository)
				org := &types.Organization{ID: "org-1", OwnerID: "user-1", Name: "Acme Inc", Slug: "acme-inc"}
				t.Cleanup(func() {
					require.True(t, beforeCalled)
					require.True(t, afterCalled)
				})
				return repo, context.Background(), org
			},
			expectBool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx, organization := tt.setup(t)
			created, err := repo.Create(ctx, organization)
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, organization.ID, created.ID)
			require.Equal(t, tt.expectBool, created.ID == organization.ID)
		})
	}
}
