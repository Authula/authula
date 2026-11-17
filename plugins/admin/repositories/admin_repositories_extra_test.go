package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/admin/repositories"
	"github.com/Authula/authula/plugins/admin/types"
)

func TestNewAdminRepositories_WiresRepositories(t *testing.T) {
	t.Parallel()

	db := internaltests.NewSQLiteIntegrationDB(t)
	repos := repositories.NewAdminRepositories(db)
	require.NotNil(t, repos)

	assert.NotNil(t, repos.UserStateRepository())
	assert.NotNil(t, repos.SessionStateRepository())
	assert.NotNil(t, repos.ImpersonationRepository())

	_, ok := repos.UserStateRepository().(*repositories.BunUserStateRepository)
	assert.True(t, ok)
	_, ok = repos.SessionStateRepository().(*repositories.BunSessionStateRepository)
	assert.True(t, ok)
	_, ok = repos.ImpersonationRepository().(*repositories.BunImpersonationRepository)
	assert.True(t, ok)
}

func TestBunUserStateRepository_CreateAndUpdate(t *testing.T) {
	t.Parallel()

	db := internaltests.NewSQLiteIntegrationDB(t)
	ctx := context.Background()
	_, err := db.NewCreateTable().Model((*types.AdminUserState)(nil)).IfNotExists().Exec(ctx)
	require.NoError(t, err)

	repo := repositories.NewBunUserStateRepository(db)

	state := &types.AdminUserState{UserID: "u1", Banned: false}
	require.NoError(t, repo.Create(ctx, state))

	got, err := repo.GetByUserID(ctx, "u1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.Banned)

	// Creating the same primary key again must surface the conflict as an error.
	require.Error(t, repo.Create(ctx, &types.AdminUserState{UserID: "u1"}))

	bannedAt := time.Now().UTC()
	bannedUntil := bannedAt.Add(time.Hour)
	require.NoError(t, repo.Update(ctx, &types.AdminUserState{
		UserID:       "u1",
		Banned:       true,
		BannedAt:     &bannedAt,
		BannedUntil:  &bannedUntil,
		BannedReason: new("reason"),
	}))

	got, err = repo.GetByUserID(ctx, "u1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.Banned)
	require.NotNil(t, got.BannedReason)
	assert.Equal(t, "reason", *got.BannedReason)
	require.NotNil(t, got.BannedUntil)
	assert.WithinDuration(t, bannedUntil, *got.BannedUntil, time.Second)
}

func TestBunSessionStateRepository_CreateAndUpdate(t *testing.T) {
	t.Parallel()

	db := internaltests.NewSQLiteIntegrationDB(t)
	ctx := context.Background()
	_, err := db.NewCreateTable().Model((*types.AdminSessionState)(nil)).IfNotExists().Exec(ctx)
	require.NoError(t, err)

	repo := repositories.NewBunSessionStateRepository(db)

	state := &types.AdminSessionState{SessionID: "s1"}
	require.NoError(t, repo.Create(ctx, state))

	got, err := repo.GetBySessionID(ctx, "s1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Nil(t, got.RevokedAt)

	require.Error(t, repo.Create(ctx, &types.AdminSessionState{SessionID: "s1"}))

	now := time.Now().UTC()
	require.NoError(t, repo.Update(ctx, &types.AdminSessionState{
		SessionID:              "s1",
		RevokedAt:              &now,
		RevokedReason:          new("reason"),
		RevokedByUserID:        new("actor"),
		ImpersonatorUserID:     new("actor"),
		ImpersonationReason:    new("impersonation"),
		ImpersonationExpiresAt: &now,
	}))

	got, err = repo.GetBySessionID(ctx, "s1")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.RevokedAt)
	require.NotNil(t, got.RevokedReason)
	assert.Equal(t, "reason", *got.RevokedReason)
	require.NotNil(t, got.RevokedByUserID)
	assert.Equal(t, "actor", *got.RevokedByUserID)
}

func TestBunImpersonationRepository_GetLatestActive_NotFound(t *testing.T) {
	t.Parallel()

	db := internaltests.NewSQLiteIntegrationDB(t)
	ctx := context.Background()
	_, err := db.NewCreateTable().Model((*types.Impersonation)(nil)).IfNotExists().Exec(ctx)
	require.NoError(t, err)

	repo := repositories.NewBunImpersonationRepository(db)

	got, err := repo.GetLatestActiveImpersonationByActor(ctx, "missing")
	require.NoError(t, err)
	assert.Nil(t, got)
}
