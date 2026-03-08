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

func setupImpersonationRepo(t *testing.T) (*repositories.BunImpersonationRepository, *bun.DB, func()) {
	t.Helper()
	db, _ := tests.NewIntegrationTestDBFromEnv(t)

	ctx := context.Background()
	if _, err := db.NewCreateTable().Model((*types.Impersonation)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create admin impersonations table: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*models.User)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	repo := repositories.NewBunImpersonationRepository(db)
	cleanup := func() { _ = db.Close() }
	return repo, db, cleanup
}

func TestBunImpersonationRepository_CreateAndGetActive(t *testing.T) {
	repo, _, cleanup := setupImpersonationRepo(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()
	imp := &types.Impersonation{
		ID:           "imp-1",
		ActorUserID:  "actor-1",
		TargetUserID: "target-1",
		Reason:       "reason",
		StartedAt:    now,
		ExpiresAt:    now.Add(1 * time.Hour),
	}

	if err := repo.CreateImpersonation(ctx, imp); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	got, err := repo.GetActiveImpersonationByID(ctx, "imp-1")
	if err != nil {
		t.Fatalf("get active failed: %v", err)
	}
	if got == nil || got.ID != "imp-1" {
		t.Fatalf("unexpected record: %v", got)
	}

	got2, err := repo.GetLatestActiveImpersonationByActor(ctx, "actor-1")
	if err != nil {
		t.Fatalf("latest active failed: %v", err)
	}
	if got2 == nil || got2.ID != "imp-1" {
		t.Fatalf("unexpected latest record: %v", got2)
	}
}

func TestBunImpersonationRepository_GetAllImpersonations_EmptyRows(t *testing.T) {
	repo, _, cleanup := setupImpersonationRepo(t)
	defer cleanup()

	ctx := context.Background()

	list, err := repo.GetAllImpersonations(ctx)
	if err != nil {
		t.Fatalf("get all failed: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(list))
	}
}

func TestBunImpersonationRepository_GetAllImpersonations(t *testing.T) {
	repo, _, cleanup := setupImpersonationRepo(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()
	a := &types.Impersonation{ID: "a", ActorUserID: "x", TargetUserID: "y", StartedAt: now.Add(-1 * time.Hour), ExpiresAt: now.Add(1 * time.Hour)}
	b := &types.Impersonation{ID: "b", ActorUserID: "x", TargetUserID: "z", StartedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(1 * time.Hour)}
	_ = repo.CreateImpersonation(ctx, a)
	_ = repo.CreateImpersonation(ctx, b)

	list, err := repo.GetAllImpersonations(ctx)
	if err != nil {
		t.Fatalf("get all failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(list))
	}
	if list[0].ID != "a" || list[1].ID != "b" {
		t.Fatalf("ordering incorrect: %v", list)
	}
}

func TestBunImpersonationRepository_GetImpersonationByID_NotFound(t *testing.T) {
	repo, _, cleanup := setupImpersonationRepo(t)
	defer cleanup()

	ctx := context.Background()
	imp, err := repo.GetImpersonationByID(ctx, "nope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if imp != nil {
		t.Fatalf("expected nil result, got %v", imp)
	}
}

func TestBunImpersonationRepository_GetActiveImpersonationByID_NotFound(t *testing.T) {
	repo, _, cleanup := setupImpersonationRepo(t)
	defer cleanup()

	ctx := context.Background()
	imp, err := repo.GetActiveImpersonationByID(ctx, "nope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if imp != nil {
		t.Fatalf("expected nil result, got %v", imp)
	}
}

func TestBunImpersonationRepository_GetActiveImpersonationByID_NotActive(t *testing.T) {
	repo, _, cleanup := setupImpersonationRepo(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()
	// expired
	imp1 := &types.Impersonation{
		ID:           "imp-exp",
		ActorUserID:  "actor-1",
		TargetUserID: "target-1",
		Reason:       "expired",
		StartedAt:    now.Add(-2 * time.Hour),
		ExpiresAt:    now.Add(-1 * time.Hour),
	}
	// ended
	end := now.Add(-30 * time.Minute)
	imp2 := &types.Impersonation{
		ID:           "imp-ended",
		ActorUserID:  "actor-1",
		TargetUserID: "target-1",
		Reason:       "ended",
		StartedAt:    now.Add(-3 * time.Hour),
		ExpiresAt:    now.Add(1 * time.Hour),
		EndedAt:      &end,
	}
	_ = repo.CreateImpersonation(ctx, imp1)
	_ = repo.CreateImpersonation(ctx, imp2)

	if imp, _ := repo.GetActiveImpersonationByID(ctx, "imp-exp"); imp != nil {
		t.Fatalf("expected expired to be ignored")
	}
	if imp, _ := repo.GetActiveImpersonationByID(ctx, "imp-ended"); imp != nil {
		t.Fatalf("expected ended to be ignored")
	}
}

func TestBunImpersonationRepository_GetLatestActiveImpersonationByActor(t *testing.T) {
	repo, _, cleanup := setupImpersonationRepo(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()
	first := &types.Impersonation{
		ID:           "imp1",
		ActorUserID:  "actor-1",
		TargetUserID: "target-A",
		StartedAt:    now.Add(-2 * time.Hour),
		ExpiresAt:    now.Add(1 * time.Hour),
	}
	second := &types.Impersonation{
		ID:           "imp2",
		ActorUserID:  "actor-1",
		TargetUserID: "target-B",
		StartedAt:    now.Add(-1 * time.Hour),
		ExpiresAt:    now.Add(1 * time.Hour),
	}
	_ = repo.CreateImpersonation(ctx, first)
	_ = repo.CreateImpersonation(ctx, second)

	got, err := repo.GetLatestActiveImpersonationByActor(ctx, "actor-1")
	if err != nil {
		t.Fatalf("latest active failed: %v", err)
	}
	if got == nil || got.ID != "imp2" {
		t.Fatalf("incorrect latest active: %v", got)
	}
}

func TestBunImpersonationRepository_EndImpersonation(t *testing.T) {
	repo, _, cleanup := setupImpersonationRepo(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()
	imp := &types.Impersonation{
		ID:           "imp-end",
		ActorUserID:  "actor-1",
		TargetUserID: "target-1",
		StartedAt:    now,
		ExpiresAt:    now.Add(1 * time.Hour),
	}
	_ = repo.CreateImpersonation(ctx, imp)

	if err := repo.EndImpersonation(ctx, "imp-end", tests.PtrString("admin")); err != nil {
		t.Fatalf("end impersonation failed: %v", err)
	}

	got, err := repo.GetImpersonationByID(ctx, "imp-end")
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if got == nil || got.EndedAt == nil || got.EndedByUserID == nil || *got.EndedByUserID != "admin" {
		t.Fatalf("ended data not set: %v", got)
	}

	// should no longer appear active
	if imp2, _ := repo.GetActiveImpersonationByID(ctx, "imp-end"); imp2 != nil {
		t.Fatalf("ended record still active: %v", imp2)
	}
}

func TestBunImpersonationRepository_UserExists(t *testing.T) {
	repo, db, cleanup := setupImpersonationRepo(t)
	defer cleanup()

	ctx := context.Background()
	exists, err := repo.UserExists(ctx, "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatalf("expected false when user missing")
	}

	user := &models.User{
		ID:            "u1",
		Name:          "n",
		Email:         "e@example.com",
		EmailVerified: true,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if _, err := db.NewInsert().Model(user).Exec(ctx); err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	exists2, err := repo.UserExists(ctx, "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists2 {
		t.Fatalf("expected user to exist")
	}
}
