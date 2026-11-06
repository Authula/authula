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

func TestBunOAuthClientRepository_CreateAndGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunOAuthClientRepository(newTestOAuth2DB(t))
	now := testNow()

	created, err := repo.Create(ctx, createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", now))
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, []string{types.ScopeOpenID, types.ScopeProfile}, created.AllowedScopes)

	byID, err := repo.GetByID(ctx, "id-1")
	require.NoError(t, err)
	require.NotNil(t, byID)
	assert.Equal(t, "client-1", byID.ClientID)
	assert.Equal(t, []string{"https://app.example.com/callback"}, byID.RedirectURIs)

	byClientID, err := repo.GetByClientID(ctx, "client-1")
	require.NoError(t, err)
	require.NotNil(t, byClientID)
	assert.Equal(t, "id-1", byClientID.ID)

	missing, err := repo.GetByID(ctx, "missing")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestBunOAuthClientRepository_ListFiltersAndPaginates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunOAuthClientRepository(newTestOAuth2DB(t))
	base := testNow()
	orgID := "org-1"

	for i, client := range []*types.OAuthClient{
		createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", base.Add(-3*time.Minute)),
		createTestClient("id-2", "client-2", types.OAuthClientOwnerTypeUser, "user-1", base.Add(-2*time.Minute)),
		createTestClient("id-3", "client-3", types.OAuthClientOwnerTypeOrganization, "org-1", base.Add(-time.Minute)),
	} {
		if i == 2 {
			client.OrganizationID = &orgID
		}
		_, err := repo.Create(ctx, client)
		require.NoError(t, err)
	}

	clients, err := repo.List(ctx, OAuthClientListFilter{
		OwnerType: types.OAuthClientOwnerTypeUser,
		OwnerID:   "user-1",
		Limit:     1,
	})
	require.NoError(t, err)
	require.Len(t, clients, 1)
	assert.Equal(t, "id-2", clients[0].ID)

	next, err := repo.List(ctx, OAuthClientListFilter{
		OwnerType: types.OAuthClientOwnerTypeUser,
		OwnerID:   "user-1",
		Limit:     1,
		Cursor:    &OAuthClientCursor{CreatedAt: clients[0].CreatedAt, ID: clients[0].ID},
	})
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.Equal(t, "id-1", next[0].ID)

	orgClients, err := repo.List(ctx, OAuthClientListFilter{OrganizationID: &orgID})
	require.NoError(t, err)
	require.Len(t, orgClients, 1)
	assert.Equal(t, "id-3", orgClients[0].ID)
}

func TestBunOAuthClientRepository_UpdateRejectsStaleWrites(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunOAuthClientRepository(newTestOAuth2DB(t))
	_, err := repo.Create(ctx, createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()))
	require.NoError(t, err)

	stored, err := repo.GetByID(ctx, "id-1")
	require.NoError(t, err)
	require.NotNil(t, stored)

	expectedUpdatedAt := stored.UpdatedAt
	stored.Name = "renamed"
	stored.UpdatedAt = expectedUpdatedAt.Add(time.Second)
	updated, err := repo.Update(ctx, stored, &expectedUpdatedAt)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "renamed", updated.Name)

	stored.Name = "stale rename"
	_, err = repo.Update(ctx, stored, &expectedUpdatedAt)
	require.ErrorIs(t, err, ErrOAuthClientConflict)
}

func TestBunOAuthClientRepository_SoftDeleteHidesClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewBunOAuthClientRepository(newTestOAuth2DB(t))
	_, err := repo.Create(ctx, createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()))
	require.NoError(t, err)

	deleted, err := repo.SoftDelete(ctx, "id-1", testNow())
	require.NoError(t, err)
	assert.True(t, deleted)

	client, err := repo.GetByID(ctx, "id-1")
	require.NoError(t, err)
	assert.Nil(t, client)

	client, err = repo.GetByClientID(ctx, "client-1")
	require.NoError(t, err)
	assert.Nil(t, client)

	client, err = repo.GetByIDIncludingDeleted(ctx, "id-1")
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.DeletedAt)
	assert.False(t, client.IsActive)

	deleted, err = repo.SoftDelete(ctx, "id-1", testNow())
	require.NoError(t, err)
	assert.False(t, deleted)

	clients, err := repo.List(ctx, OAuthClientListFilter{OwnerID: "user-1"})
	require.NoError(t, err)
	assert.Empty(t, clients)

	clients, err = repo.List(ctx, OAuthClientListFilter{OwnerID: "user-1", IncludeDeleted: true})
	require.NoError(t, err)
	assert.Len(t, clients, 1)
}

func TestBunOAuthClientRepository_WithTxRollsBack(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTestOAuth2DB(t)
	repo := NewBunOAuthClientRepository(db)

	rollback := errors.New("rollback")
	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		txRepo := repo.WithTx(tx)
		_, err := txRepo.Create(ctx, createTestClient("id-1", "client-1", types.OAuthClientOwnerTypeUser, "user-1", testNow()))
		require.NoError(t, err)
		return rollback
	})
	require.ErrorIs(t, err, rollback)

	client, err := repo.GetByID(ctx, "id-1")
	require.NoError(t, err)
	assert.Nil(t, client)
}
