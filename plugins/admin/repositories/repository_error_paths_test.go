package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/admin/repositories"
	"github.com/Authula/authula/plugins/admin/types"
)

// An empty database has no admin tables, so every repository operation surfaces a wrapped
// database error rather than silently succeeding.
func TestBunUserStateRepository_ErrorPaths(t *testing.T) {
	t.Parallel()

	db := internaltests.NewSQLiteIntegrationDB(t)
	repo := repositories.NewBunUserStateRepository(db)
	ctx := context.Background()

	got, err := repo.GetByUserID(ctx, "u1")
	assert.Error(t, err)
	assert.Nil(t, got)

	assert.Error(t, repo.Upsert(ctx, &types.AdminUserState{UserID: "u1"}))
	assert.Error(t, repo.Create(ctx, &types.AdminUserState{UserID: "u1"}))
	assert.Error(t, repo.Update(ctx, &types.AdminUserState{UserID: "u1"}))
	assert.Error(t, repo.Delete(ctx, "u1"))

	rows, err := repo.GetBanned(ctx)
	assert.Error(t, err)
	assert.Nil(t, rows)
}

func TestBunSessionStateRepository_ErrorPaths(t *testing.T) {
	t.Parallel()

	db := internaltests.NewSQLiteIntegrationDB(t)
	repo := repositories.NewBunSessionStateRepository(db)
	ctx := context.Background()

	got, err := repo.GetBySessionID(ctx, "s1")
	assert.Error(t, err)
	assert.Nil(t, got)

	assert.Error(t, repo.Upsert(ctx, &types.AdminSessionState{SessionID: "s1"}))
	assert.Error(t, repo.Create(ctx, &types.AdminSessionState{SessionID: "s1"}))
	assert.Error(t, repo.Update(ctx, &types.AdminSessionState{SessionID: "s1"}))
	assert.Error(t, repo.Delete(ctx, "s1"))

	rows, err := repo.GetRevoked(ctx)
	assert.Error(t, err)
	assert.Nil(t, rows)

	exists, err := repo.SessionExists(ctx, "s1")
	assert.Error(t, err)
	assert.False(t, exists)

	userSessions, err := repo.GetByUserID(ctx, "u1")
	assert.Error(t, err)
	assert.Nil(t, userSessions)
}

func TestBunImpersonationRepository_ErrorPaths(t *testing.T) {
	t.Parallel()

	db := internaltests.NewSQLiteIntegrationDB(t)
	repo := repositories.NewBunImpersonationRepository(db)
	ctx := context.Background()

	assert.Error(t, repo.CreateImpersonation(ctx, &types.Impersonation{
		ID:           "imp-1",
		ActorUserID:  "actor",
		TargetUserID: "target",
		Reason:       "reason",
		StartedAt:    time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(time.Hour),
	}))

	all, err := repo.GetAllImpersonations(ctx)
	assert.Error(t, err)
	assert.Nil(t, all)

	byID, err := repo.GetImpersonationByID(ctx, "imp-1")
	assert.Error(t, err)
	assert.Nil(t, byID)

	active, err := repo.GetActiveImpersonationByID(ctx, "imp-1")
	assert.Error(t, err)
	assert.Nil(t, active)

	latest, err := repo.GetLatestActiveImpersonationByActor(ctx, "actor")
	assert.Error(t, err)
	assert.Nil(t, latest)

	assert.Error(t, repo.EndImpersonation(ctx, "imp-1", new("actor")))

	exists, err := repo.UserExists(ctx, "actor")
	assert.Error(t, err)
	assert.False(t, exists)
}
