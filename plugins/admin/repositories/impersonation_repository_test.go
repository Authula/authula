package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/repositories"
	"github.com/Authula/authula/plugins/admin/types"
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
	tests := []struct {
		name       string
		wantActive bool
		wantLatest bool
	}{
		{name: "creates and fetches active by id and actor", wantActive: true, wantLatest: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if tt.wantActive {
				got, err := repo.GetActiveImpersonationByID(ctx, "imp-1")
				if err != nil {
					t.Fatalf("get active failed: %v", err)
				}
				if got == nil || got.ID != "imp-1" {
					t.Fatalf("unexpected record: %v", got)
				}
			}

			if tt.wantLatest {
				got, err := repo.GetLatestActiveImpersonationByActor(ctx, "actor-1")
				if err != nil {
					t.Fatalf("latest active failed: %v", err)
				}
				if got == nil || got.ID != "imp-1" {
					t.Fatalf("unexpected latest record: %v", got)
				}
			}
		})
	}
}

func TestBunImpersonationRepository_GetAllImpersonations(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(ctx context.Context, repo *repositories.BunImpersonationRepository)
		wantLen int
	}{
		{
			name:    "empty rows",
			wantLen: 0,
		},
		{
			name: "returns all rows in order",
			setup: func(ctx context.Context, repo *repositories.BunImpersonationRepository) {
				now := time.Now().UTC()
				a := &types.Impersonation{ID: "a", ActorUserID: "x", TargetUserID: "y", StartedAt: now.Add(-1 * time.Hour), ExpiresAt: now.Add(1 * time.Hour)}
				b := &types.Impersonation{ID: "b", ActorUserID: "x", TargetUserID: "z", StartedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(1 * time.Hour)}
				_ = repo.CreateImpersonation(ctx, a)
				_ = repo.CreateImpersonation(ctx, b)
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, cleanup := setupImpersonationRepo(t)
			defer cleanup()

			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, repo)
			}

			list, err := repo.GetAllImpersonations(ctx)
			if err != nil {
				t.Fatalf("get all failed: %v", err)
			}
			if len(list) != tt.wantLen {
				t.Fatalf("expected %d rows, got %d", tt.wantLen, len(list))
			}
			if tt.wantLen == 2 && (list[0].ID != "a" || list[1].ID != "b") {
				t.Fatalf("ordering incorrect: %v", list)
			}
		})
	}
}

func TestBunImpersonationRepository_GetImpersonationByID(t *testing.T) {
	tests := []struct {
		name            string
		impersonationID string
		setup           func(ctx context.Context, repo *repositories.BunImpersonationRepository)
		wantID          string
	}{
		{
			name:            "not found",
			impersonationID: "nope",
		},
		{
			name:            "found",
			impersonationID: "imp-1",
			setup: func(ctx context.Context, repo *repositories.BunImpersonationRepository) {
				now := time.Now().UTC()
				_ = repo.CreateImpersonation(ctx, &types.Impersonation{ID: "imp-1", ActorUserID: "a", TargetUserID: "t", StartedAt: now, ExpiresAt: now.Add(time.Hour)})
			},
			wantID: "imp-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, cleanup := setupImpersonationRepo(t)
			defer cleanup()

			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, repo)
			}

			imp, err := repo.GetImpersonationByID(ctx, tt.impersonationID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantID == "" {
				if imp != nil {
					t.Fatalf("expected nil result, got %v", imp)
				}
			} else if imp == nil || imp.ID != tt.wantID {
				t.Fatalf("unexpected record: %v", imp)
			}
		})
	}
}

func TestBunImpersonationRepository_GetActiveImpersonationByID(t *testing.T) {
	tests := []struct {
		name            string
		impersonationID string
		setup           func(ctx context.Context, repo *repositories.BunImpersonationRepository)
		wantID          string
	}{
		{
			name:            "not found",
			impersonationID: "nope",
		},
		{
			name:            "ignores expired",
			impersonationID: "imp-exp",
			setup: func(ctx context.Context, repo *repositories.BunImpersonationRepository) {
				now := time.Now().UTC()
				_ = repo.CreateImpersonation(ctx, &types.Impersonation{
					ID: "imp-exp", ActorUserID: "actor-1", TargetUserID: "target-1", Reason: "expired",
					StartedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-1 * time.Hour),
				})
			},
		},
		{
			name:            "ignores ended",
			impersonationID: "imp-ended",
			setup: func(ctx context.Context, repo *repositories.BunImpersonationRepository) {
				now := time.Now().UTC()
				ended := now.Add(-30 * time.Minute)
				_ = repo.CreateImpersonation(ctx, &types.Impersonation{
					ID: "imp-ended", ActorUserID: "actor-1", TargetUserID: "target-1", Reason: "ended",
					StartedAt: now.Add(-3 * time.Hour), ExpiresAt: now.Add(1 * time.Hour), EndedAt: &ended,
				})
			},
		},
		{
			name:            "found active",
			impersonationID: "imp-active",
			setup: func(ctx context.Context, repo *repositories.BunImpersonationRepository) {
				now := time.Now().UTC()
				_ = repo.CreateImpersonation(ctx, &types.Impersonation{
					ID: "imp-active", ActorUserID: "actor-1", TargetUserID: "target-1",
					StartedAt: now, ExpiresAt: now.Add(1 * time.Hour),
				})
			},
			wantID: "imp-active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, cleanup := setupImpersonationRepo(t)
			defer cleanup()

			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, repo)
			}

			imp, err := repo.GetActiveImpersonationByID(ctx, tt.impersonationID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantID == "" {
				if imp != nil {
					t.Fatalf("expected nil result, got %v", imp)
				}
			} else if imp == nil || imp.ID != tt.wantID {
				t.Fatalf("unexpected record: %v", imp)
			}
		})
	}
}

func TestBunImpersonationRepository_GetLatestActiveImpersonationByActor(t *testing.T) {
	tests := []struct {
		name   string
		actor  string
		setup  func(ctx context.Context, repo *repositories.BunImpersonationRepository)
		wantID string
	}{
		{
			name:  "returns latest active",
			actor: "actor-1",
			setup: func(ctx context.Context, repo *repositories.BunImpersonationRepository) {
				now := time.Now().UTC()
				_ = repo.CreateImpersonation(ctx, &types.Impersonation{
					ID: "imp1", ActorUserID: "actor-1", TargetUserID: "target-A",
					StartedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(1 * time.Hour),
				})
				_ = repo.CreateImpersonation(ctx, &types.Impersonation{
					ID: "imp2", ActorUserID: "actor-1", TargetUserID: "target-B",
					StartedAt: now.Add(-1 * time.Hour), ExpiresAt: now.Add(1 * time.Hour),
				})
			},
			wantID: "imp2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, cleanup := setupImpersonationRepo(t)
			defer cleanup()

			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, repo)
			}

			got, err := repo.GetLatestActiveImpersonationByActor(ctx, tt.actor)
			if err != nil {
				t.Fatalf("latest active failed: %v", err)
			}
			if got == nil || got.ID != tt.wantID {
				t.Fatalf("incorrect latest active: %v", got)
			}
		})
	}
}

func TestBunImpersonationRepository_EndImpersonation(t *testing.T) {
	tests := []struct {
		name    string
		endedBy string
	}{
		{name: "marks record ended and no longer active", endedBy: "admin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, cleanup := setupImpersonationRepo(t)
			defer cleanup()

			ctx := context.Background()
			now := time.Now().UTC()
			imp := &types.Impersonation{
				ID: "imp-end", ActorUserID: "actor-1", TargetUserID: "target-1",
				StartedAt: now, ExpiresAt: now.Add(1 * time.Hour),
			}
			_ = repo.CreateImpersonation(ctx, imp)

			if err := repo.EndImpersonation(ctx, "imp-end", new(tt.endedBy)); err != nil {
				t.Fatalf("end impersonation failed: %v", err)
			}

			got, err := repo.GetImpersonationByID(ctx, "imp-end")
			if err != nil {
				t.Fatalf("fetch failed: %v", err)
			}
			if got == nil || got.EndedAt == nil || got.EndedByUserID == nil || *got.EndedByUserID != tt.endedBy {
				t.Fatalf("ended data not set: %v", got)
			}
			if imp2, err := repo.GetActiveImpersonationByID(ctx, "imp-end"); err != nil || imp2 != nil {
				t.Fatalf("ended record still active: %v", imp2)
			}
		})
	}
}

func TestBunImpersonationRepository_UserExists(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setup     func(ctx context.Context, db *bun.DB)
		wantExist bool
	}{
		{
			name:      "returns false when user missing",
			userID:    "u1",
			wantExist: false,
		},
		{
			name:   "returns true when user exists",
			userID: "u1",
			setup: func(ctx context.Context, db *bun.DB) {
				user := &models.User{
					ID: "u1", Name: "n", Email: "e@example.com", EmailVerified: true,
					CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
				}
				if _, err := db.NewInsert().Model(user).Exec(ctx); err != nil {
					t.Fatalf("failed to insert user: %v", err)
				}
			},
			wantExist: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, db, cleanup := setupImpersonationRepo(t)
			defer cleanup()

			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, db)
			}

			exists, err := repo.UserExists(ctx, tt.userID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if exists != tt.wantExist {
				t.Fatalf("expected exists=%v, got %v", tt.wantExist, exists)
			}
		})
	}
}
