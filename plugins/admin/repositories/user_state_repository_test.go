package repositories_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	"github.com/Authula/authula/plugins/admin/repositories"
	"github.com/Authula/authula/plugins/admin/types"
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

func TestBunUserStateRepository_GetByUserID(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		setup      func(ctx context.Context, repo *repositories.BunUserStateRepository)
		wantUserID string
	}{
		{
			name:   "not found",
			userID: "nope",
		},
		{
			name:   "found",
			userID: "u1",
			setup: func(ctx context.Context, repo *repositories.BunUserStateRepository) {
				_ = repo.Upsert(ctx, &types.AdminUserState{UserID: "u1", Banned: true})
			},
			wantUserID: "u1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, cleanup := setupRepo(t)
			defer cleanup()

			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, repo)
			}

			res, err := repo.GetByUserID(ctx, tt.userID)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.wantUserID == "" {
				if res != nil {
					t.Fatalf("expected nil result, got %v", res)
				}
			} else {
				if res == nil || res.UserID != tt.wantUserID {
					t.Fatalf("unexpected result: %v", res)
				}
			}
		})
	}
}

func TestBunUserStateRepository_Upsert(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(ctx context.Context, repo *repositories.BunUserStateRepository, state *types.AdminUserState)
		state      *types.AdminUserState
		wantBanned bool
	}{
		{
			name:       "creates record",
			state:      &types.AdminUserState{UserID: "u1", Banned: true},
			wantBanned: true,
		},
		{
			name: "updates existing record",
			setup: func(ctx context.Context, repo *repositories.BunUserStateRepository, state *types.AdminUserState) {
				_ = repo.Upsert(ctx, &types.AdminUserState{UserID: "u1", Banned: true})
			},
			state:      &types.AdminUserState{UserID: "u1", Banned: false},
			wantBanned: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, cleanup := setupRepo(t)
			defer cleanup()

			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, repo, tt.state)
			}
			if err := repo.Upsert(ctx, tt.state); err != nil {
				t.Fatalf("upsert failed: %v", err)
			}

			got, err := repo.GetByUserID(ctx, tt.state.UserID)
			if err != nil {
				t.Fatalf("fetch failed: %v", err)
			}
			if got == nil || got.UserID != tt.state.UserID || got.Banned != tt.wantBanned {
				t.Fatalf("unexpected state returned: %v", got)
			}
		})
	}
}

func TestBunUserStateRepository_Delete(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "deletes record"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, cleanup := setupRepo(t)
			defer cleanup()

			ctx := context.Background()
			if err := repo.Upsert(ctx, &types.AdminUserState{UserID: "u2"}); err != nil {
				t.Fatalf("upsert failed: %v", err)
			}
			if err := repo.Delete(ctx, "u2"); err != nil {
				t.Fatalf("delete failed: %v", err)
			}
			res, err := repo.GetByUserID(ctx, "u2")
			if err != nil {
				t.Fatalf("get failed: %v", err)
			}
			if res != nil {
				t.Fatalf("expected nil after delete, got %v", res)
			}
		})
	}
}

func TestBunUserStateRepository_GetBanned(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(ctx context.Context, repo *repositories.BunUserStateRepository)
		wantLen int
	}{
		{
			name:    "empty rows",
			wantLen: 0,
		},
		{
			name: "returns only banned records",
			setup: func(ctx context.Context, repo *repositories.BunUserStateRepository) {
				_ = repo.Upsert(ctx, &types.AdminUserState{UserID: "b1", Banned: true})
				_ = repo.Upsert(ctx, &types.AdminUserState{UserID: "nb", Banned: false})
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, cleanup := setupRepo(t)
			defer cleanup()

			ctx := context.Background()
			if tt.setup != nil {
				tt.setup(ctx, repo)
			}

			list, err := repo.GetBanned(ctx)
			if err != nil {
				t.Fatalf("get banned failed: %v", err)
			}
			if len(list) != tt.wantLen {
				t.Fatalf("expected %d rows, got %d: %v", tt.wantLen, len(list), list)
			}
			if tt.wantLen > 0 && list[0].UserID != "b1" {
				t.Fatalf("unexpected banned list: %v", list)
			}
		})
	}
}
