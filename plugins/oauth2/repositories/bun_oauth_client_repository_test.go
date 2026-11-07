package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/oauth2/types"
)

func assertOAuthClientEqual(t *testing.T, want *types.OAuthClient, got *types.OAuthClient) {
	t.Helper()

	require.NotNil(t, got)
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.ClientID, got.ClientID)
	assert.Equal(t, want.ClientSecretHash, got.ClientSecretHash)
	assert.Equal(t, want.ClientSecretSalt, got.ClientSecretSalt)
	assert.Equal(t, want.ClientSecretPrefix, got.ClientSecretPrefix)
	assert.Equal(t, want.Name, got.Name)
	assert.Equal(t, want.RedirectURIs, got.RedirectURIs)
	assert.Equal(t, want.AllowedGrants, got.AllowedGrants)
	assert.Equal(t, want.AllowedScopes, got.AllowedScopes)
	assert.Equal(t, want.OwnerType, got.OwnerType)
	assert.Equal(t, want.OwnerID, got.OwnerID)
	assert.Equal(t, want.CreatorID, got.CreatorID)
	assert.Equal(t, want.OrganizationID, got.OrganizationID)
	assert.Equal(t, want.IsActive, got.IsActive)
	assert.Equal(t, want.DeletedAt, got.DeletedAt)
	assert.True(t, want.CreatedAt.Equal(got.CreatedAt))
	assert.True(t, want.UpdatedAt.Equal(got.UpdatedAt))
}

func seedOAuthClients(t *testing.T, ctx context.Context, repo OAuthClientRepository, clients []*types.OAuthClient, deletedIDs []string) {
	t.Helper()

	for _, client := range clients {
		_, err := repo.Create(ctx, client)
		require.NoError(t, err)
	}
	for _, id := range deletedIDs {
		deleted, err := repo.SoftDelete(ctx, id, testNow())
		require.NoError(t, err)
		require.True(t, deleted)
	}
}

func TestBunOAuthClientRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing []*types.OAuthClient
		client   *types.OAuthClient
		wantErr  bool
	}{
		{
			name:   "creates and returns the client",
			client: createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
		},
		{
			name: "rejects a duplicate client record id",
			existing: []*types.OAuthClient{
				createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			},
			client:  createTestClient("id-1", "client-2", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			wantErr: true,
		},
		{
			name: "rejects a duplicate public client id",
			existing: []*types.OAuthClient{
				createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			},
			client:  createTestClient("id-2", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunOAuthClientRepository(newTestOAuth2DB(t))

			for _, client := range tt.existing {
				_, err := repo.Create(ctx, client)
				require.NoError(t, err)
			}

			created, err := repo.Create(ctx, tt.client)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assertOAuthClientEqual(t, tt.client, created)

			stored, err := repo.GetByID(ctx, tt.client.ID)
			require.NoError(t, err)
			assertOAuthClientEqual(t, tt.client, stored)
		})
	}
}

func TestBunOAuthClientRepository_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		clients    []*types.OAuthClient
		deletedIDs []string
		id         string
		wantFound  bool
	}{
		{
			name: "returns an active client",
			clients: []*types.OAuthClient{
				createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			},
			id:        "id-1",
			wantFound: true,
		},
		{
			name: "returns nil for a deleted client",
			clients: []*types.OAuthClient{
				createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			},
			deletedIDs: []string{"id-1"},
			id:         "id-1",
			wantFound:  false,
		},
		{
			name:      "returns nil for a missing client",
			id:        "missing",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunOAuthClientRepository(newTestOAuth2DB(t))
			seedOAuthClients(t, ctx, repo, tt.clients, tt.deletedIDs)

			client, err := repo.GetByID(ctx, tt.id)
			require.NoError(t, err)
			if !tt.wantFound {
				assert.Nil(t, client)
				return
			}

			require.NotNil(t, client)
			assert.Equal(t, tt.id, client.ID)
		})
	}
}

func TestBunOAuthClientRepository_GetByIDIncludingDeleted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		clients    []*types.OAuthClient
		deletedIDs []string
		id         string
		wantFound  bool
		wantDelete bool
	}{
		{
			name: "returns an active client",
			clients: []*types.OAuthClient{
				createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			},
			id:        "id-1",
			wantFound: true,
		},
		{
			name: "returns a deleted client",
			clients: []*types.OAuthClient{
				createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			},
			deletedIDs: []string{"id-1"},
			id:         "id-1",
			wantFound:  true,
			wantDelete: true,
		},
		{
			name:      "returns nil for a missing client",
			id:        "missing",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunOAuthClientRepository(newTestOAuth2DB(t))
			seedOAuthClients(t, ctx, repo, tt.clients, tt.deletedIDs)

			client, err := repo.GetByIDIncludingDeleted(ctx, tt.id)
			require.NoError(t, err)
			if !tt.wantFound {
				assert.Nil(t, client)
				return
			}

			require.NotNil(t, client)
			assert.Equal(t, tt.id, client.ID)
			if tt.wantDelete {
				assert.NotNil(t, client.DeletedAt)
				assert.False(t, client.IsActive)
			} else {
				assert.Nil(t, client.DeletedAt)
				assert.True(t, client.IsActive)
			}
		})
	}
}

func TestBunOAuthClientRepository_GetByClientID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		clients    []*types.OAuthClient
		deletedIDs []string
		clientID   string
		wantID     string
	}{
		{
			name: "returns an active client",
			clients: []*types.OAuthClient{
				createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			},
			clientID: "client-1",
			wantID:   "id-1",
		},
		{
			name: "returns nil for a deleted client",
			clients: []*types.OAuthClient{
				createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			},
			deletedIDs: []string{"id-1"},
			clientID:   "client-1",
		},
		{
			name:     "returns nil for a missing client",
			clientID: "missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunOAuthClientRepository(newTestOAuth2DB(t))
			seedOAuthClients(t, ctx, repo, tt.clients, tt.deletedIDs)

			client, err := repo.GetByClientID(ctx, tt.clientID)
			require.NoError(t, err)
			if tt.wantID == "" {
				assert.Nil(t, client)
				return
			}

			require.NotNil(t, client)
			assert.Equal(t, tt.wantID, client.ID)
		})
	}
}

func TestBunOAuthClientRepository_GetByClientIDIncludingDeleted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		clients    []*types.OAuthClient
		deletedIDs []string
		clientID   string
		wantID     string
		wantDelete bool
	}{
		{
			name: "returns an active client",
			clients: []*types.OAuthClient{
				createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			},
			clientID: "client-1",
			wantID:   "id-1",
		},
		{
			name: "returns a deleted client",
			clients: []*types.OAuthClient{
				createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			},
			deletedIDs: []string{"id-1"},
			clientID:   "client-1",
			wantID:     "id-1",
			wantDelete: true,
		},
		{
			name:     "returns nil for a missing client",
			clientID: "missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunOAuthClientRepository(newTestOAuth2DB(t))
			seedOAuthClients(t, ctx, repo, tt.clients, tt.deletedIDs)

			client, err := repo.GetByClientIDIncludingDeleted(ctx, tt.clientID)
			require.NoError(t, err)
			if tt.wantID == "" {
				assert.Nil(t, client)
				return
			}

			require.NotNil(t, client)
			assert.Equal(t, tt.wantID, client.ID)
			if tt.wantDelete {
				assert.NotNil(t, client.DeletedAt)
			} else {
				assert.Nil(t, client.DeletedAt)
			}
		})
	}
}

func TestBunOAuthClientRepository_List(t *testing.T) {
	t.Parallel()

	base := testNow()
	orgID := "org-1"
	newDefaultClients := func() []*types.OAuthClient {
		clients := []*types.OAuthClient{
			createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", base.Add(-3*time.Minute)),
			createTestClient("id-2", "client-2", types.OAuthClientOwnerTypeUser, "user-1", base.Add(-2*time.Minute)),
			createTestClient("id-3", "client-3", types.OAuthClientOwnerTypeOrganization, "org-1", base.Add(-time.Minute)),
		}
		clients[2].OrganizationID = &orgID
		return clients
	}

	tests := []struct {
		name       string
		deletedIDs []string
		filter     OAuthClientListFilter
		wantIDs    []string
	}{
		{
			name: "filters by owner type and owner id",
			filter: OAuthClientListFilter{
				OwnerType: types.OAuthClientOwnerTypeUser,
				OwnerID:   "user-1",
			},
			wantIDs: []string{"id-2", "id-1"},
		},
		{
			name:    "filters by organization id",
			filter:  OAuthClientListFilter{OrganizationID: &orgID},
			wantIDs: []string{"id-3"},
		},
		{
			name:       "excludes deleted clients by default",
			deletedIDs: []string{"id-1"},
			filter: OAuthClientListFilter{
				OwnerType: types.OAuthClientOwnerTypeUser,
				OwnerID:   "user-1",
			},
			wantIDs: []string{"id-2"},
		},
		{
			name:       "includes deleted clients when requested",
			deletedIDs: []string{"id-1"},
			filter: OAuthClientListFilter{
				OwnerType:      types.OAuthClientOwnerTypeUser,
				OwnerID:        "user-1",
				IncludeDeleted: true,
			},
			wantIDs: []string{"id-2", "id-1"},
		},
		{
			name: "applies a limit",
			filter: OAuthClientListFilter{
				OwnerType: types.OAuthClientOwnerTypeUser,
				OwnerID:   "user-1",
				Limit:     1,
			},
			wantIDs: []string{"id-2"},
		},
		{
			name: "paginates with a cursor",
			filter: OAuthClientListFilter{
				OwnerType: types.OAuthClientOwnerTypeUser,
				OwnerID:   "user-1",
				Limit:     1,
				Cursor:    &OAuthClientCursor{CreatedAt: base.Add(-2 * time.Minute), ID: "id-2"},
			},
			wantIDs: []string{"id-1"},
		},
		{
			name:    "returns an empty list when nothing matches",
			filter:  OAuthClientListFilter{OwnerID: "user-2"},
			wantIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunOAuthClientRepository(newTestOAuth2DB(t))
			seedOAuthClients(t, ctx, repo, newDefaultClients(), tt.deletedIDs)

			clients, err := repo.List(ctx, tt.filter)
			require.NoError(t, err)
			require.Len(t, clients, len(tt.wantIDs))
			for i, wantID := range tt.wantIDs {
				assert.Equal(t, wantID, clients[i].ID)
			}
		})
	}
}

func TestBunOAuthClientRepository_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		createClient   bool
		nilExpected    bool
		expectedOffset time.Duration
		newName        string
		wantErr        bool
	}{
		{
			name:         "updates without an expected timestamp",
			createClient: true,
			nilExpected:  true,
			newName:      "renamed without version",
		},
		{
			name:           "updates with a matching timestamp",
			createClient:   true,
			expectedOffset: 0,
			newName:        "renamed with version",
		},
		{
			name:           "rejects a stale timestamp",
			createClient:   true,
			expectedOffset: -time.Second,
			newName:        "stale rename",
			wantErr:        true,
		},
		{
			name:         "rejects a missing client",
			createClient: false,
			nilExpected:  true,
			newName:      "missing rename",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunOAuthClientRepository(newTestOAuth2DB(t))
			client := createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow())

			if tt.createClient {
				_, err := repo.Create(ctx, client)
				require.NoError(t, err)

				stored, err := repo.GetByID(ctx, client.ID)
				require.NoError(t, err)
				require.NotNil(t, stored)
				client = stored
			}

			var expectedUpdatedAt *time.Time
			if !tt.nilExpected {
				expected := client.UpdatedAt.Add(tt.expectedOffset)
				expectedUpdatedAt = &expected
			}
			client.Name = tt.newName
			client.UpdatedAt = client.UpdatedAt.Add(time.Second)

			updated, err := repo.Update(ctx, client, expectedUpdatedAt)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrOAuthClientConflict)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, updated)
			assert.Equal(t, tt.newName, updated.Name)

			stored, err := repo.GetByID(ctx, client.ID)
			require.NoError(t, err)
			require.NotNil(t, stored)
			assert.Equal(t, tt.newName, stored.Name)
		})
	}
}

func TestBunOAuthClientRepository_SoftDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		clients     []*types.OAuthClient
		preDeleteID string
		id          string
		wantDeleted bool
		wantExists  bool
	}{
		{
			name: "soft deletes an active client",
			clients: []*types.OAuthClient{
				createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			},
			id:          "id-1",
			wantDeleted: true,
			wantExists:  true,
		},
		{
			name: "returns false for an already deleted client",
			clients: []*types.OAuthClient{
				createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()),
			},
			preDeleteID: "id-1",
			id:          "id-1",
			wantDeleted: false,
			wantExists:  true,
		},
		{
			name:        "returns false for a missing client",
			id:          "missing",
			wantDeleted: false,
			wantExists:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := NewBunOAuthClientRepository(newTestOAuth2DB(t))
			seedOAuthClients(t, ctx, repo, tt.clients, nil)

			if tt.preDeleteID != "" {
				deleted, err := repo.SoftDelete(ctx, tt.preDeleteID, testNow())
				require.NoError(t, err)
				require.True(t, deleted)
			}

			deleted, err := repo.SoftDelete(ctx, tt.id, testNow())
			require.NoError(t, err)
			assert.Equal(t, tt.wantDeleted, deleted)

			client, err := repo.GetByIDIncludingDeleted(ctx, tt.id)
			require.NoError(t, err)
			if !tt.wantExists {
				assert.Nil(t, client)
				return
			}

			require.NotNil(t, client)
			assert.NotNil(t, client.DeletedAt)
			assert.False(t, client.IsActive)
		})
	}
}

func TestBunOAuthClientRepository_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rollback  bool
		wantStore bool
	}{
		{name: "commit persists the client", wantStore: true},
		{name: "rollback discards the client", rollback: true, wantStore: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestOAuth2DB(t)
			repo := NewBunOAuthClientRepository(db)
			client := createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow())
			rollback := errors.New("rollback")

			err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				if _, err := repo.WithTx(tx).Create(ctx, client); err != nil {
					return err
				}
				if tt.rollback {
					return rollback
				}
				return nil
			})
			if tt.rollback {
				require.ErrorIs(t, err, rollback)
			} else {
				require.NoError(t, err)
			}

			stored, err := repo.GetByID(ctx, client.ID)
			require.NoError(t, err)
			if tt.wantStore {
				require.NotNil(t, stored)
				assert.Equal(t, client.ID, stored.ID)
				return
			}
			assert.Nil(t, stored)
		})
	}
}
