package repositories_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func setupRepo(t *testing.T) (*repositories.BunUserStateRepository, func()) {
	t.Helper()
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())

	ctx := context.Background()
	if _, err := db.NewCreateTable().Model((*types.AdminUserState)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	repo := repositories.NewBunUserStateRepository(db)

	cleanup := func() {
		db.Close()
		sqldb.Close()
	}
	return repo, cleanup
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
	state := &types.AdminUserState{
		UserID:         "u1",
		IsBanned:       true,
		BannedAt:       tests.PtrTime(t, 0),
		BannedReason:   tests.PtrString(t, "reason"),
		BannedByUserID: tests.PtrString(t, "actor"),
	}

	if err := repo.Upsert(ctx, state); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	got, err := repo.GetByUserID(ctx, "u1")
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if got == nil || got.UserID != "u1" || !got.IsBanned {
		t.Fatalf("unexpected state returned: %v", got)
	}

	// update
	state.IsBanned = false
	if err := repo.Upsert(ctx, state); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	got2, _ := repo.GetByUserID(ctx, "u1")
	if got2 == nil || got2.IsBanned {
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
	b1 := &types.AdminUserState{UserID: "b1", IsBanned: true}
	nb := &types.AdminUserState{UserID: "nb", IsBanned: false}
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
