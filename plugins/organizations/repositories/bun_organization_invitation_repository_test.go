package repositories_test

import (
	"context"
	"fmt"
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

func TestBunOrganizationInvitationRepository_ListAllPendingByEmail(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
		t.Helper()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		plugintests.SeedOrganization(t, db, "org-2", "user-2", "Beta Inc", "beta-inc")
		plugintests.SeedOrganization(t, db, "org-3", "user-2", "Gamma Inc", "gamma-inc")
		plugintests.SeedOrganization(t, db, "org-4", "user-2", "Delta Inc", "delta-inc")
		plugintests.SeedOrganization(t, db, "org-5", "user-2", "Epsilon Inc", "epsilon-inc")

		repo := repositories.NewBunOrganizationInvitationRepository(db)
		ctx := context.Background()

		for i, invitation := range []*types.OrganizationInvitation{
			{ID: "inv-1", OrganizationID: "org-1", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)},
			{ID: "inv-2", OrganizationID: "org-2", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)},
			{ID: "inv-3", OrganizationID: "org-3", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)},
			{ID: "inv-4", OrganizationID: "org-4", Status: types.OrganizationInvitationStatusAccepted, ExpiresAt: time.Now().UTC().Add(time.Hour)},
			{ID: "inv-5", OrganizationID: "org-5", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(-time.Hour)},
		} {
			invitation.Email = "user@example.com"
			invitation.InviterID = "user-1"
			invitation.Role = "member"
			_, err := repo.Create(ctx, invitation)
			require.NoError(t, err, "seeding invitation %d", i)
		}

		return repo, ctx
	}

	tests := []struct {
		name          string
		email         string
		page          int
		limit         int
		expectedIDs   []string
		expectedTotal int
	}{
		{name: "first page returns the oldest pending invitations", email: "user@example.com", page: 1, limit: 10, expectedIDs: []string{"inv-1", "inv-2", "inv-3"}, expectedTotal: 3},
		{name: "page size splits the pending invitations", email: "user@example.com", page: 1, limit: 2, expectedIDs: []string{"inv-1", "inv-2"}, expectedTotal: 3},
		{name: "second page returns the remainder", email: "user@example.com", page: 2, limit: 2, expectedIDs: []string{"inv-3"}, expectedTotal: 3},
		{name: "unknown email has nothing pending", email: "missing@example.com", page: 1, limit: 10, expectedIDs: []string{}, expectedTotal: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := setup(t)

			pending, total, err := repo.ListAllPendingByEmail(ctx, tt.email, tt.page, tt.limit)
			require.NoError(t, err)
			require.Equal(t, tt.expectedTotal, total)

			ids := make([]string, 0, len(pending))
			for _, invitation := range pending {
				ids = append(ids, invitation.ID)
			}
			require.Equal(t, tt.expectedIDs, ids)
		})
	}
}

func TestBunOrganizationInvitationRepository_GetAllPendingByEmail(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
		t.Helper()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")

		repo := repositories.NewBunOrganizationInvitationRepository(db)
		ctx := context.Background()

		for i, invitation := range []*types.OrganizationInvitation{
			{ID: "inv-1", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)},
			{ID: "inv-2", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)},
			{ID: "inv-3", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(time.Hour)},
			{ID: "inv-4", Status: types.OrganizationInvitationStatusAccepted, ExpiresAt: time.Now().UTC().Add(time.Hour)},
			{ID: "inv-5", Status: types.OrganizationInvitationStatusPending, ExpiresAt: time.Now().UTC().Add(-time.Hour)},
		} {
			invitation.OrganizationID = "org-1"
			invitation.Email = "user@example.com"
			invitation.InviterID = "user-1"
			invitation.Role = "member"
			_, err := repo.Create(ctx, invitation)
			require.NoError(t, err, "seeding invitation %d", i)
		}

		return repo, ctx
	}

	tests := []struct {
		name        string
		email       string
		expectedIDs []string
	}{
		{name: "returns every pending invitation oldest first", email: "user@example.com", expectedIDs: []string{"inv-1", "inv-2", "inv-3"}},
		{name: "accepted and expired invitations are excluded", email: "user@example.com", expectedIDs: []string{"inv-1", "inv-2", "inv-3"}},
		{name: "unknown email has nothing pending", email: "missing@example.com", expectedIDs: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := setup(t)

			pending, err := repo.GetAllPendingByEmail(ctx, tt.email)
			require.NoError(t, err)

			ids := make([]string, 0, len(pending))
			for _, invitation := range pending {
				ids = append(ids, invitation.ID)
			}
			require.Equal(t, tt.expectedIDs, ids)
		})
	}
}

// GetAllPendingByEmail must not silently truncate. Its paginated sibling stops at
// the requested limit; this one is expected to return the whole set in one query,
// well past the 500-row batch cap that used to apply here.
func TestBunOrganizationInvitationRepository_GetAllPendingByEmailIsNotCapped(t *testing.T) {
	t.Parallel()

	const pendingCount = 600

	db := plugintests.SetupRepoDB(t)
	plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
	repo := repositories.NewBunOrganizationInvitationRepository(db)
	ctx := context.Background()

	for i := 1; i <= pendingCount; i++ {
		_, err := repo.Create(ctx, &types.OrganizationInvitation{
			ID:             fmt.Sprintf("inv-%04d", i),
			OrganizationID: "org-1",
			Email:          "user@example.com",
			InviterID:      "user-1",
			Role:           "member",
			Status:         types.OrganizationInvitationStatusPending,
			ExpiresAt:      time.Now().UTC().Add(time.Hour),
		})
		require.NoError(t, err, "seeding invitation %d", i)
	}

	pending, err := repo.GetAllPendingByEmail(ctx, "user@example.com")
	require.NoError(t, err)
	require.Len(t, pending, pendingCount)
	require.Equal(t, "inv-0001", pending[0].ID, "oldest invitation must come first")
	require.Equal(t, fmt.Sprintf("inv-%04d", pendingCount), pending[len(pending)-1].ID)
}

func TestBunOrganizationInvitationRepository_ListAllByOrganizationIDWithOrg(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (repositories.OrganizationInvitationRepository, context.Context) {
		t.Helper()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		plugintests.SeedOrganization(t, db, "org-2", "user-2", "Beta Inc", "beta-inc")

		repo := repositories.NewBunOrganizationInvitationRepository(db)
		ctx := context.Background()

		for i := 1; i <= 5; i++ {
			_, err := repo.Create(ctx, &types.OrganizationInvitation{
				ID:             fmt.Sprintf("inv-%d", i),
				Email:          fmt.Sprintf("user%d@example.com", i),
				InviterID:      "user-1",
				OrganizationID: "org-1",
				Role:           "member",
				Status:         types.OrganizationInvitationStatusPending,
				ExpiresAt:      time.Now().UTC().Add(time.Hour),
			})
			require.NoError(t, err)
		}

		return repo, ctx
	}

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
		{name: "zero limit falls back to a bounded page", organizationID: "org-1", page: 1, limit: 0, expectCount: 5, expectTotal: 5},
		{name: "organization without invitations is empty", organizationID: "org-2", page: 1, limit: 10, expectCount: 0, expectTotal: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := setup(t)

			invitations, total, err := repo.ListAllByOrganizationIDWithOrg(ctx, tt.organizationID, tt.page, tt.limit)
			require.NoError(t, err)
			require.Len(t, invitations, tt.expectCount)
			require.Equal(t, tt.expectTotal, total)
			for _, invitation := range invitations {
				require.Equal(t, tt.organizationID, invitation.Organization.ID, "joined organization must be populated")
			}
		})
	}
}

func TestBunOrganizationInvitationRepository_ListAllByOrganizationIDWithOrgPagesPartitionCleanly(t *testing.T) {
	t.Parallel()

	db := plugintests.SetupRepoDB(t)
	plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
	repo := repositories.NewBunOrganizationInvitationRepository(db)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		_, err := repo.Create(ctx, &types.OrganizationInvitation{
			ID:             fmt.Sprintf("inv-%d", i),
			Email:          fmt.Sprintf("user%d@example.com", i),
			InviterID:      "user-1",
			OrganizationID: "org-1",
			Role:           "member",
			Status:         types.OrganizationInvitationStatusPending,
			ExpiresAt:      time.Now().UTC().Add(time.Hour),
		})
		require.NoError(t, err)
	}

	seen := make([]string, 0, 5)
	for page := 1; page <= 3; page++ {
		invitations, total, err := repo.ListAllByOrganizationIDWithOrg(ctx, "org-1", page, 2)
		require.NoError(t, err)
		require.Equal(t, 5, total)
		for _, invitation := range invitations {
			seen = append(seen, invitation.Invitation.ID)
		}
	}

	require.ElementsMatch(t, []string{"inv-1", "inv-2", "inv-3", "inv-4", "inv-5"}, seen)
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

func TestBunOrganizationInvitationRepository_GetAllByOrganizationIDWithOrg(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T, count int) (repositories.OrganizationInvitationRepository, context.Context) {
		t.Helper()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		plugintests.SeedOrganization(t, db, "org-2", "user-2", "Beta Inc", "beta-inc")

		repo := repositories.NewBunOrganizationInvitationRepository(db)
		ctx := context.Background()

		for i := 1; i <= count; i++ {
			_, err := repo.Create(ctx, &types.OrganizationInvitation{
				ID:             fmt.Sprintf("inv-%d", i),
				Email:          fmt.Sprintf("user%d@example.com", i),
				InviterID:      "user-1",
				OrganizationID: "org-1",
				Role:           "member",
				Status:         types.OrganizationInvitationStatusPending,
				ExpiresAt:      time.Now().UTC().Add(time.Hour),
			})
			require.NoError(t, err)
		}

		return repo, ctx
	}

	tests := []struct {
		name           string
		seedCount      int
		organizationID string
		expectCount    int
	}{
		{name: "returns every invitation for the organization", seedCount: 5, organizationID: "org-1", expectCount: 5},
		{name: "organization without invitations is empty", seedCount: 5, organizationID: "org-2", expectCount: 0},
		{name: "unknown organization is empty", seedCount: 5, organizationID: "org-99", expectCount: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, ctx := setup(t, tt.seedCount)

			invitations, err := repo.GetAllByOrganizationIDWithOrg(ctx, tt.organizationID)
			require.NoError(t, err)
			require.NotNil(t, invitations, "an empty result must be an empty slice, not nil")
			require.Len(t, invitations, tt.expectCount)
			for _, invitation := range invitations {
				require.NotNil(t, invitation.Invitation)
				require.Equal(t, tt.organizationID, invitation.Invitation.OrganizationID)
				require.Equal(t, tt.organizationID, invitation.Organization.ID, "the organization must be hydrated by the join")
				require.NotEmpty(t, invitation.Organization.Name)
			}
		})
	}
}

func TestBunOrganizationInvitationRepository_GetAllByOrganizationIDWithOrgIgnoresTheDefaultLimit(t *testing.T) {
	t.Parallel()

	const invitationCount = 25

	db := plugintests.SetupRepoDB(t)
	plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
	repo := repositories.NewBunOrganizationInvitationRepository(db)
	ctx := context.Background()

	for i := 1; i <= invitationCount; i++ {
		_, err := repo.Create(ctx, &types.OrganizationInvitation{
			ID:             fmt.Sprintf("inv-%02d", i),
			Email:          fmt.Sprintf("user%d@example.com", i),
			InviterID:      "user-1",
			OrganizationID: "org-1",
			Role:           "member",
			Status:         types.OrganizationInvitationStatusPending,
			ExpiresAt:      time.Now().UTC().Add(time.Hour),
		})
		require.NoError(t, err)
	}

	invitations, err := repo.GetAllByOrganizationIDWithOrg(ctx, "org-1")
	require.NoError(t, err)
	require.Len(t, invitations, invitationCount)
}
