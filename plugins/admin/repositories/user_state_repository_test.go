package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/admin/repositories"
	"github.com/Authula/authula/plugins/admin/types"
)

func setupRepo(t *testing.T) (*repositories.BunUserStateRepository, func()) {
	t.Helper()
	db := tests.NewIntegrationTestDB(t)

	ctx := context.Background()
	if _, err := db.NewCreateTable().Model((*types.AdminUserState)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	repo := repositories.NewBunUserStateRepository(db)

	return repo, func() {}
}

func TestBunUserStateRepository_GetByUserID_NotFound(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	res, err := repo.GetByUserID(ctx, "nope")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil result for missing user, got %v", res)
	}
}

func TestBunUserStateRepository_UpsertAndRetrieve(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()
	state := &types.AdminUserState{
		UserID:         "u1",
		Banned:         true,
		BannedAt:       &now,
		BannedReason:   tests.PtrString("reason"),
		BannedByUserID: tests.PtrString("actor"),
	}

	if err := repo.Upsert(ctx, state); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	got, err := repo.GetByUserID(ctx, "u1")
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if got == nil || got.UserID != "u1" || !got.Banned {
		t.Fatalf("unexpected state returned: %v", got)
	}

	// update
	state.Banned = false
	if err := repo.Upsert(ctx, state); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	got2, _ := repo.GetByUserID(ctx, "u1")
	if got2 == nil || got2.Banned {
		t.Fatalf("update did not persist: %v", got2)
	}
}

func TestBunUserStateRepository_Delete(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	state := &types.AdminUserState{UserID: "u2"}
	if err := repo.Upsert(ctx, state); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if err := repo.Delete(ctx, "u2"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	res, _ := repo.GetByUserID(ctx, "u2")
	if res != nil {
		t.Fatalf("expected nil after delete, got %v", res)
	}
}

func TestBunUserStateRepository_GetBanned_EmptyRows(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()

	list, err := repo.GetBanned(ctx)
	if err != nil {
		t.Fatalf("get banned failed: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list: %v", list)
	}
}

func TestBunUserStateRepository_GetBanned(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	ctx := context.Background()
	b1 := &types.AdminUserState{UserID: "b1", Banned: true}
	nb := &types.AdminUserState{UserID: "nb", Banned: false}
	_ = repo.Upsert(ctx, b1)
	_ = repo.Upsert(ctx, nb)

	list, err := repo.GetBanned(ctx)
	if err != nil {
		t.Fatalf("get banned failed: %v", err)
	}
	if len(list) != 1 || list[0].UserID != "b1" {
		t.Fatalf("unexpected banned list: %v", list)
	}
}
