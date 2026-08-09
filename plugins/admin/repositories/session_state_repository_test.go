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

func TestBunSessionStateRepository_GetBySessionID(t *testing.T) {
	tests := []struct {
		name          string
		sessionID     string
		setup         func(ctx context.Context, repo *repositories.BunSessionStateRepository)
		wantSessionID string
	}{
		{
			name:      "not found",
			sessionID: "no-sess",
		},
		{
			name:      "found",
			sessionID: "s1",
			setup: func(ctx context.Context, repo *repositories.BunSessionStateRepository) {
				now := time.Now().UTC()
				_ = repo.Upsert(ctx, &types.AdminSessionState{SessionID: "s1", RevokedAt: &now})
			},
			wantSessionID: "s1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, cleanup := setupSessionRepo(t)
			defer cleanup()

			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, repo)
			}

			s, err := repo.GetBySessionID(ctx, tt.sessionID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantSessionID == "" {
				if s != nil {
					t.Fatalf("expected nil, got %v", s)
				}
			} else {
				if s == nil || s.SessionID != tt.wantSessionID {
					t.Fatalf("unexpected record: %v", s)
				}
			}
		})
	}
}

func TestBunSessionStateRepository_Upsert(t *testing.T) {
	tests := []struct {
		name             string
		setup            func(ctx context.Context, repo *repositories.BunSessionStateRepository, state *types.AdminSessionState)
		state            *types.AdminSessionState
		wantRevoke       bool
		wantRevokeReason string
	}{
		{
			name:       "creates record",
			state:      &types.AdminSessionState{SessionID: "s1"},
			wantRevoke: false,
		},
		{
			name: "updates existing record",
			setup: func(ctx context.Context, repo *repositories.BunSessionStateRepository, state *types.AdminSessionState) {
				_ = repo.Upsert(ctx, &types.AdminSessionState{SessionID: "s1"})
			},
			state:            &types.AdminSessionState{SessionID: "s1", RevokedReason: new("reason")},
			wantRevokeReason: "reason",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, cleanup := setupSessionRepo(t)
			defer cleanup()

			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, repo, tt.state)
			}
			if err := repo.Upsert(ctx, tt.state); err != nil {
				t.Fatalf("upsert failed: %v", err)
			}

			got, err := repo.GetBySessionID(ctx, tt.state.SessionID)
			if err != nil {
				t.Fatalf("get failed: %v", err)
			}
			if got == nil || got.SessionID != tt.state.SessionID {
				t.Fatalf("unexpected record: %v", got)
			}
			if tt.wantRevokeReason != "" && (got.RevokedReason == nil || *got.RevokedReason != tt.wantRevokeReason) {
				t.Fatalf("update not applied: %v", got)
			}
		})
	}
}

func TestBunSessionStateRepository_Delete(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "deletes record"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, cleanup := setupSessionRepo(t)
			defer cleanup()

			ctx := context.Background()
			_ = repo.Upsert(ctx, &types.AdminSessionState{SessionID: "s2"})
			if err := repo.Delete(ctx, "s2"); err != nil {
				t.Fatalf("delete error: %v", err)
			}
			if s, err := repo.GetBySessionID(ctx, "s2"); err != nil || s != nil {
				t.Fatalf("expected nil after delete, got %v (err %v)", s, err)
			}
		})
	}
}

func TestBunSessionStateRepository_GetRevoked(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(ctx context.Context, repo *repositories.BunSessionStateRepository)
		wantLen int
	}{
		{
			name:    "empty rows",
			wantLen: 0,
		},
		{
			name: "returns only revoked records",
			setup: func(ctx context.Context, repo *repositories.BunSessionStateRepository) {
				now := time.Now().UTC()
				_ = repo.Upsert(ctx, &types.AdminSessionState{SessionID: "r1", RevokedAt: &now})
				_ = repo.Upsert(ctx, &types.AdminSessionState{SessionID: "nr", RevokedAt: nil})
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, cleanup := setupSessionRepo(t)
			defer cleanup()

			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, repo)
			}

			list, err := repo.GetRevoked(ctx)
			if err != nil {
				t.Fatalf("get revoked failed: %v", err)
			}
			if len(list) != tt.wantLen {
				t.Fatalf("expected %d rows, got %d: %v", tt.wantLen, len(list), list)
			}
			if tt.wantLen > 0 && list[0].SessionID != "r1" {
				t.Fatalf("unexpected revoked list: %v", list)
			}
		})
	}
}

func TestBunSessionStateRepository_SessionExists(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		setup     func(ctx context.Context, db *bun.DB, repo *repositories.BunSessionStateRepository)
		wantExist bool
	}{
		{
			name:      "returns false for missing session",
			sessionID: "nope",
		},
		{
			name:      "returns true for existing session",
			sessionID: "sess-1",
			setup: func(ctx context.Context, db *bun.DB, _ *repositories.BunSessionStateRepository) {
				sess := &models.Session{ID: "sess-1", UserID: "u1", Token: "t", ExpiresAt: time.Now().UTC()}
				if _, err := db.NewInsert().Model(sess).Exec(ctx); err != nil {
					t.Fatalf("failed to insert session: %v", err)
				}
			},
			wantExist: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, db, cleanup := setupSessionRepo(t)
			defer cleanup()

			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, db, repo)
			}

			exists, err := repo.SessionExists(ctx, tt.sessionID)
			if err != nil {
				t.Fatalf("exists error: %v", err)
			}
			if exists != tt.wantExist {
				t.Fatalf("expected exists=%v, got %v", tt.wantExist, exists)
			}
		})
	}
}

func TestBunSessionStateRepository_GetByUserID(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		setup   func(ctx context.Context, db *bun.DB, repo *repositories.BunSessionStateRepository)
		wantLen int
	}{
		{
			name:   "returns sessions with their state",
			userID: "u-1",
			setup: func(ctx context.Context, db *bun.DB, repo *repositories.BunSessionStateRepository) {
				now := time.Now().UTC()
				s1 := &models.Session{ID: "s1", UserID: "u-1", Token: "t", ExpiresAt: now}
				s2 := &models.Session{ID: "s2", UserID: "u-1", Token: "t", ExpiresAt: now}
				if _, err := db.NewInsert().Model(s1).Exec(ctx); err != nil {
					t.Fatalf("failed to insert session: %v", err)
				}
				if _, err := db.NewInsert().Model(s2).Exec(ctx); err != nil {
					t.Fatalf("failed to insert session: %v", err)
				}
				revokedAt := now.Add(-time.Minute)
				_ = repo.Upsert(ctx, &types.AdminSessionState{SessionID: "s1", RevokedAt: &revokedAt})
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, db, cleanup := setupSessionRepo(t)
			defer cleanup()

			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, db, repo)
			}

			rows, err := repo.GetByUserID(ctx, tt.userID)
			if err != nil {
				t.Fatalf("getbyuserid failed: %v", err)
			}
			if len(rows) != tt.wantLen {
				t.Fatalf("expected %d sessions, got %d", tt.wantLen, len(rows))
			}

			var foundState bool
			for _, r := range rows {
				if r.Session.ID == "s1" {
					foundState = true
					if r.State == nil || r.State.SessionID != "s1" {
						t.Fatalf("state mismatch for s1: %v", r.State)
					}
				}
			}
			if !foundState {
				t.Fatal("missing s1 in result")
			}
		})
	}
}
