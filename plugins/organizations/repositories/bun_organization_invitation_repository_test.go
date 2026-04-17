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

func TestBunOrganizationInvitationRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		invitation   *types.OrganizationInvitation
		expectStatus types.OrganizationInvitationStatus
	}{
		{
			name: "pending",
			invitation: &types.OrganizationInvitation{
				ID:             "inv-1",
				Email:          "user@example.com",
				InviterID:      "user-1",
				OrganizationID: "org-1",
				Role:           "member",
				Status:         types.OrganizationInvitationStatusPending,
				ExpiresAt:      time.Now().UTC().Add(time.Hour),
			},
			expectStatus: types.OrganizationInvitationStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
			repo := repositories.NewBunOrganizationInvitationRepository(db)

			created, err := repo.Create(context.Background(), tt.invitation)
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, tt.expectStatus, created.Status)
			require.Equal(t, tt.invitation.ID, created.ID)
		})
	}
}

func TestBunOrganizationInvitationRepository_GetByOrganizationIDAndEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		organizationID string
		email          string
		status         []types.OrganizationInvitationStatus
		setup          func(*testing.T) (repositories.OrganizationInvitationRepository, context.Context)
		expectFound    bool
		expectID       string
		expectStatus   types.OrganizationInvitationStatus
	}{
		{
			name:           "found latest invitation regardless of status",
			organizationID: "org-1",
			email:          "user@example.com",
			expectFound:    true,
			expectID:       "inv-3",
			expectStatus:   types.OrganizationInvitationStatusPending,
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationInvitationRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-1", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-2", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusAccepted, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-3", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:           "found pending invitation when status is filtered",
			organizationID: "org-1",
			email:          "user@example.com",
			status:         []types.OrganizationInvitationStatus{types.OrganizationInvitationStatusPending},
			expectFound:    true,
			expectID:       "inv-3",
			expectStatus:   types.OrganizationInvitationStatusPending,
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationInvitationRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-1", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-2", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusAccepted, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-3", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:           "found accepted invitation when status is filtered",
			organizationID: "org-1",
			email:          "user@example.com",
			status:         []types.OrganizationInvitationStatus{types.OrganizationInvitationStatusAccepted},
			expectFound:    true,
			expectID:       "inv-2",
			expectStatus:   types.OrganizationInvitationStatusAccepted,
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationInvitationRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-1", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-2", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusAccepted, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-3", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:           "found latest invitation across multiple statuses",
			organizationID: "org-1",
			email:          "user@example.com",
			status:         []types.OrganizationInvitationStatus{types.OrganizationInvitationStatusAccepted, types.OrganizationInvitationStatusPending},
			expectFound:    true,
			expectID:       "inv-3",
			expectStatus:   types.OrganizationInvitationStatusPending,
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationInvitationRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-1", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-2", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusAccepted, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-3", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:           "not found",
			organizationID: "org-1",
			email:          "missing@example.com",
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				return repositories.NewBunOrganizationInvitationRepository(db), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			found, err := repo.GetByOrganizationIDAndEmail(ctx, tt.organizationID, tt.email, tt.status...)
			require.NoError(t, err)
			if tt.expectFound {
				require.NotNil(t, found)
				require.Equal(t, tt.expectID, found.ID)
				require.Equal(t, tt.expectStatus, found.Status)
				return
			}
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationInvitationRepository_GetAllPendingByEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		email         string
		setup         func(*testing.T) (repositories.OrganizationInvitationRepository, context.Context)
		expectPending int
	}{
		{
			name:          "pending only",
			email:         "user@example.com",
			expectPending: 1,
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganization(t, db, "org-2", "user-2", "Beta Inc", "beta-inc")
				repo := repositories.NewBunOrganizationInvitationRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-1", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-2", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-2", Role: "member", Status: types.OrganizationInvitationStatusAccepted, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:          "missing",
			email:         "missing@example.com",
			expectPending: 0,
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganization(t, db, "org-2", "user-2", "Beta Inc", "beta-inc")
				return repositories.NewBunOrganizationInvitationRepository(db), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			pending, err := repo.GetAllPendingByEmail(ctx, tt.email)
			require.NoError(t, err)
			require.Len(t, pending, tt.expectPending)
		})
	}
}

func TestBunOrganizationInvitationRepository_GetAllByOrganizationID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		organizationID string
		setup          func(*testing.T) (repositories.OrganizationInvitationRepository, context.Context)
		expectCount    int
		verify         func(*testing.T, []types.OrganizationInvitation)
	}{
		{
			name:           "returns invitations for organization",
			organizationID: "org-1",
			expectCount:    2,
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganization(t, db, "org-2", "user-2", "Beta Inc", "beta-inc")
				repo := repositories.NewBunOrganizationInvitationRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-1", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-2", Email: "other@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusRejected, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)

				return repo, ctx
			},
			verify: func(t *testing.T, invitations []types.OrganizationInvitation) {
				t.Helper()
				statusByID := map[string]types.OrganizationInvitationStatus{}
				for _, invitation := range invitations {
					statusByID[invitation.ID] = invitation.Status
				}

				require.Equal(t, types.OrganizationInvitationStatusPending, statusByID["inv-1"])
				require.Equal(t, types.OrganizationInvitationStatusRejected, statusByID["inv-2"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)
			invitations, err := repo.GetAllByOrganizationID(ctx, tt.organizationID)
			require.NoError(t, err)
			require.Len(t, invitations, tt.expectCount)
			if tt.verify != nil {
				tt.verify(t, invitations)
			}
		})
	}
}

func TestBunOrganizationInvitationRepository_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		invitationID string
		setup        func(*testing.T) (repositories.OrganizationInvitationRepository, context.Context)
		expectFound  bool
	}{
		{
			name:         "found",
			invitationID: "inv-1",
			expectFound:  true,
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationInvitationRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-1", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:         "not found",
			invitationID: "missing",
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				return repositories.NewBunOrganizationInvitationRepository(db), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			found, err := repo.GetByID(ctx, tt.invitationID)
			require.NoError(t, err)
			if tt.expectFound {
				require.NotNil(t, found)
				require.Equal(t, "inv-1", found.ID)
				return
			}
			require.Nil(t, found)
		})
	}
}

func TestBunOrganizationInvitationRepository_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(*testing.T) (repositories.OrganizationInvitationRepository, context.Context, *types.OrganizationInvitation)
	}{
		{
			name: "updates invitation status",
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context, *types.OrganizationInvitation) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationInvitationRepository(db)
				ctx := context.Background()

				created, err := repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-1", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				return repo, ctx, created
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx, created := tt.setup(t)
			created.Status = types.OrganizationInvitationStatusAccepted
			updated, err := repo.Update(ctx, created)
			require.NoError(t, err)
			require.NotNil(t, updated)
			require.Equal(t, types.OrganizationInvitationStatusAccepted, updated.Status)
		})
	}
}

func TestBunOrganizationInvitationRepository_CountByOrganizationIDAndEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		organizationID string
		email          string
		setup          func(*testing.T) (repositories.OrganizationInvitationRepository, context.Context)
		expectCount    int
	}{
		{
			name:           "counts all invitations for org/email pair",
			organizationID: "org-1",
			email:          "user@example.com",
			expectCount:    2,
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganization(t, db, "org-2", "user-2", "Beta Inc", "beta-inc")
				repo := repositories.NewBunOrganizationInvitationRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-1", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-2", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusAccepted, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-3", Email: "user@example.com", InviterID: "user-2", OrganizationID: "org-2", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:           "same email other org does not count",
			organizationID: "org-2",
			email:          "user@example.com",
			expectCount:    1,
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganization(t, db, "org-2", "user-2", "Beta Inc", "beta-inc")
				repo := repositories.NewBunOrganizationInvitationRepository(db)
				ctx := context.Background()

				_, err := repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-1", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-2", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusAccepted, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)
				_, err = repo.Create(ctx, &types.OrganizationInvitation{ID: "inv-3", Email: "user@example.com", InviterID: "user-2", OrganizationID: "org-2", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)

				return repo, ctx
			},
		},
		{
			name:           "missing returns zero",
			organizationID: "org-1",
			email:          "missing@example.com",
			expectCount:    0,
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganization(t, db, "org-2", "user-2", "Beta Inc", "beta-inc")
				return repositories.NewBunOrganizationInvitationRepository(db), context.Background()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := tt.setup(t)

			count, err := repo.CountByOrganizationIDAndEmail(ctx, tt.organizationID, tt.email)
			require.NoError(t, err)
			require.Equal(t, tt.expectCount, count)
		})
	}
}

func TestBunOrganizationInvitationRepository_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		commit bool
		setup  func(*testing.T) (repositories.OrganizationInvitationRepository, context.Context, repositories.OrganizationInvitationRepository, bun.Tx)
	}{
		{
			name:   "commits through tx",
			commit: true,
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context, repositories.OrganizationInvitationRepository, bun.Tx) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationInvitationRepository(db)
				ctx := context.Background()
				tx, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)
				return repo, ctx, repo.WithTx(tx), tx
			},
		},
		{
			name:   "rolls back through tx",
			commit: false,
			setup: func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context, repositories.OrganizationInvitationRepository, bun.Tx) {
				t.Helper()
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				repo := repositories.NewBunOrganizationInvitationRepository(db)
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
			require.IsType(t, &repositories.BunOrganizationInvitationRepository{}, txRepo)

			created, err := txRepo.Create(ctx, &types.OrganizationInvitation{ID: "inv-1", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
			require.NoError(t, err)
			require.NotNil(t, created)
			require.Equal(t, "inv-1", created.ID)

			if tt.commit {
				require.NoError(t, tx.Commit())
			} else {
				require.NoError(t, tx.Rollback())
			}

			found, err := repo.GetByID(ctx, "inv-1")
			require.NoError(t, err)
			if tt.commit {
				require.NotNil(t, found)
			} else {
				require.Nil(t, found)
			}
		})
	}
}

func TestBunOrganizationInvitationRepository_Hooks(t *testing.T) {
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

				beforeCalled := false
				afterCalled := false
				hooks := &plugintests.MockOrganizationInvitationHooks{
					Before: func(invitation *types.OrganizationInvitation) error {
						beforeCalled = true
						require.Equal(t, "inv-1", invitation.ID)
						return nil
					},
					After: func(invitation types.OrganizationInvitation) error {
						afterCalled = true
						require.Equal(t, "inv-1", invitation.ID)
						return nil
					},
				}

				repo := repositories.NewBunOrganizationInvitationRepository(db, hooks)
				created, err := repo.Create(context.Background(), &types.OrganizationInvitation{ID: "inv-1", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
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
				db := plugintests.SetupRepoDB(t)
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")

				seedRepo := repositories.NewBunOrganizationInvitationRepository(db)
				ctx := context.Background()
				invitation, err := seedRepo.Create(ctx, &types.OrganizationInvitation{ID: "inv-1", Email: "user@example.com", InviterID: "user-1", OrganizationID: "org-1", Role: "member", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)})
				require.NoError(t, err)

				beforeCalled := false
				afterCalled := false
				hooks := &plugintests.MockOrganizationInvitationHooks{
					BeforeUpdate: func(invitation *types.OrganizationInvitation) error {
						beforeCalled = true
						require.Equal(t, types.OrganizationInvitationStatusAccepted, invitation.Status)
						return nil
					},
					AfterUpdate: func(invitation types.OrganizationInvitation) error {
						afterCalled = true
						require.Equal(t, types.OrganizationInvitationStatusAccepted, invitation.Status)
						return nil
					},
				}

				repo := repositories.NewBunOrganizationInvitationRepository(db, hooks)
				invitation.Status = types.OrganizationInvitationStatusAccepted
				updated, err := repo.Update(ctx, invitation)
				require.NoError(t, err)
				require.Equal(t, types.OrganizationInvitationStatusAccepted, updated.Status)
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
