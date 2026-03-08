package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func setupSessionRepo(t *testing.T) (*repositories.BunSessionStateRepository, *bun.DB, func()) {
	t.Helper()
	db, _ := tests.NewIntegrationTestDBFromEnv(t)

	ctx := context.Background()
	if _, err := db.NewCreateTable().Model((*types.AdminSessionState)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create admin session state table: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*models.Session)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create sessions table: %v", err)
	}

	repo := repositories.NewBunSessionStateRepository(db)
	cleanup := func() { _ = db.Close() }
	return repo, db, cleanup
}

func TestBunSessionStateRepository_GetBySessionID_NotFound(t *testing.T) {
	repo, _, cleanup := setupSessionRepo(t)
	defer cleanup()

	ctx := context.Background()
	s, err := repo.GetBySessionID(ctx, "no-sess")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != nil {
		t.Fatalf("expected nil, got %v", s)
	}
}

func TestBunSessionStateRepository_UpsertAndGet(t *testing.T) {
	repo, _, cleanup := setupSessionRepo(t)
	defer cleanup()
	ctx := context.Background()

	st := &types.AdminSessionState{SessionID: "s1", RevokedAt: ptrTime(t, 0)}
	if err := repo.Upsert(ctx, st); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	got, err := repo.GetBySessionID(ctx, "s1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got == nil || got.SessionID != "s1" {
		t.Fatalf("unexpected record: %v", got)
	}
	// update
	st.RevokedReason = ptrString(t, "reason")
	if err := repo.Upsert(ctx, st); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	got2, _ := repo.GetBySessionID(ctx, "s1")
	if got2 == nil || got2.RevokedReason == nil || *got2.RevokedReason != "reason" {
		t.Fatalf("update not applied: %v", got2)
	}
}

func TestBunSessionStateRepository_Delete(t *testing.T) {
	repo, _, cleanup := setupSessionRepo(t)
	defer cleanup()
	ctx := context.Background()

	_ = repo.Upsert(ctx, &types.AdminSessionState{SessionID: "s2"})
	if err := repo.Delete(ctx, "s2"); err != nil {
		t.Fatalf("delete error: %v", err)
	}
	if s, _ := repo.GetBySessionID(ctx, "s2"); s != nil {
		t.Fatalf("expected nil after delete, got %v", s)
	}
}

func TestBunSessionStateRepository_GetRevoked(t *testing.T) {
	repo, _, cleanup := setupSessionRepo(t)
	defer cleanup()
	ctx := context.Background()

	_ = repo.Upsert(ctx, &types.AdminSessionState{SessionID: "r1", RevokedAt: ptrTime(t, 0)})
	_ = repo.Upsert(ctx, &types.AdminSessionState{SessionID: "nr", RevokedAt: nil})

	list, err := repo.GetRevoked(ctx)
	if err != nil {
		t.Fatalf("get revoked failed: %v", err)
	}
	if len(list) != 1 || list[0].SessionID != "r1" {
		t.Fatalf("unexpected revoked list: %v", list)
	}
}

func TestBunSessionStateRepository_SessionExists(t *testing.T) {
	repo, db, cleanup := setupSessionRepo(t)
	defer cleanup()
	ctx := context.Background()

	// insert session row manually
	sess := &models.Session{ID: "sess-1", UserID: "u1", Token: "t", ExpiresAt: time.Now().UTC()}
	if _, err := db.NewInsert().Model(sess).Exec(ctx); err != nil {
		t.Fatalf("failed to insert session: %v", err)
	}
	exists, err := repo.SessionExists(ctx, "sess-1")
	if err != nil {
		t.Fatalf("exists error: %v", err)
	}
	if !exists {
		t.Fatalf("expected session to exist")
	}
	noExists, _ := repo.SessionExists(ctx, "nope")
	if noExists {
		t.Fatalf("expected nonexistent to be false")
	}
}

func TestBunSessionStateRepository_GetByUserID(t *testing.T) {
	repo, db, cleanup := setupSessionRepo(t)
	defer cleanup()
	ctx := context.Background()

	// create session rows
	now := time.Now().UTC()
	s1 := &models.Session{ID: "s1", UserID: "u-1", Token: "t", ExpiresAt: now}
	s2 := &models.Session{ID: "s2", UserID: "u-1", Token: "t", ExpiresAt: now}
	_, _ = db.NewInsert().Model(s1).Exec(ctx)
	_, _ = db.NewInsert().Model(s2).Exec(ctx)

	// only s1 has state
	_ = repo.Upsert(ctx, &types.AdminSessionState{SessionID: "s1", RevokedAt: ptrTime(t, 0)})

	rows, err := repo.GetByUserID(ctx, "u-1")
	if err != nil {
		t.Fatalf("getbyuserid failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(rows))
	}
	// ensure first is s1 or s2 depending on ordering but state pointer correct
	var found1 bool
	for _, r := range rows {
		if r.Session.ID == "s1" {
			found1 = true
			if r.State == nil || r.State.SessionID != "s1" {
				t.Fatalf("state mismatch for s1: %v", r.State)
			}
		}
	}
	if !found1 {
		t.Fatal("missing s1 in result")
	}
}

// helper
func ptrString(t *testing.T, s string) *string {
	t.Helper()
	return &s
}

func ptrTime(t *testing.T, offset int) *time.Time {
	t.Helper()
	tm := time.Now().Add(time.Duration(offset) * time.Second).UTC()
	return &tm
}
