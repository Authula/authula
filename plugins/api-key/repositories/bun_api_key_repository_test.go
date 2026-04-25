package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	"github.com/Authula/authula/plugins/api-key/types"
)

func newTestApiKeyDB(t *testing.T) *bun.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	if _, err := db.NewCreateTable().Model((*types.ApiKey)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("failed to create api_keys table: %v", err)
	}

	return db
}

func createTestApiKey(id string, ownerType string, referenceID string) *types.ApiKey {
	now := time.Now().UTC().Truncate(time.Second)
	prefix := "pref-"
	window := 60
	maxRequests := 100
	remaining := 50
	lastRequestedAt := now.Add(-time.Minute)
	expiresAt := now.Add(time.Hour)

	return &types.ApiKey{
		ID:                   id,
		KeyHash:              "hashed-" + id,
		Name:                 "api key " + id,
		OwnerType:            ownerType,
		ReferenceID:          referenceID,
		Start:                "pref-start",
		Prefix:               &prefix,
		Enabled:              true,
		RateLimitEnabled:     true,
		RateLimitTimeWindow:  &window,
		RateLimitMaxRequests: &maxRequests,
		RequestsRemaining:    &remaining,
		LastRequestedAt:      &lastRequestedAt,
		ExpiresAt:            &expiresAt,
		Permissions:          []string{"users:read", "users:write"},
		Metadata:             []byte(`{"key":"value"}`),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func createExpiredTestApiKey(id string, ownerType string, referenceID string) *types.ApiKey {
	key := createTestApiKey(id, ownerType, referenceID)
	expiredAt := time.Now().UTC().Add(-time.Hour)
	key.ExpiresAt = &expiredAt
	return key
}

func TestBunApiKeyRepository_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name      string
		apiKey    *types.ApiKey
		wantError bool
	}{
		{name: "success", apiKey: createTestApiKey("api-key-1", types.OwnerTypeUser, "user-1")},
		{name: "duplicate_id", apiKey: createTestApiKey("api-key-1", types.OwnerTypeUser, "user-1"), wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := newTestApiKeyDB(t)
			repo := NewBunApiKeyRepository(db)

			apiKey := createTestApiKey(tc.apiKey.ID, tc.apiKey.OwnerType, tc.apiKey.ReferenceID)
			if tc.name == "duplicate_id" {
				if _, err := repo.Create(ctx, createTestApiKey("api-key-1", types.OwnerTypeUser, "user-1")); err != nil {
					t.Fatalf("failed to seed api key: %v", err)
				}
				if _, err := repo.Create(ctx, apiKey); err == nil {
					t.Fatal("expected duplicate create to fail")
				}
				return
			}

			created, err := repo.Create(ctx, apiKey)
			if tc.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if created == nil {
				t.Fatal("expected api key, got nil")
			}
			if created.ID != apiKey.ID || created.Name != apiKey.Name || created.OwnerType != apiKey.OwnerType || created.ReferenceID != apiKey.ReferenceID {
				t.Fatalf("unexpected created api key: %#v", created)
			}

			stored, err := repo.GetByID(ctx, apiKey.ID)
			if err != nil {
				t.Fatalf("failed to fetch created api key: %v", err)
			}
			if stored == nil {
				t.Fatal("expected stored api key, got nil")
			}
		})
	}
}

func TestBunApiKeyRepository_GetByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTestApiKeyDB(t)
	repo := NewBunApiKeyRepository(db)

	seed := createTestApiKey("api-key-1", types.OwnerTypeUser, "user-1")
	if _, err := repo.Create(ctx, seed); err != nil {
		t.Fatalf("failed to seed api key: %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantNil bool
	}{
		{name: "success", id: "api-key-1"},
		{name: "missing", id: "missing", wantNil: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			apiKey, err := repo.GetByID(ctx, tc.id)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil {
				if apiKey != nil {
					t.Fatalf("expected nil api key, got %#v", apiKey)
				}
				return
			}
			if apiKey == nil || apiKey.ID != tc.id {
				t.Fatalf("unexpected api key: %#v", apiKey)
			}
		})
	}
}

func TestBunApiKeyRepository_GetByKeyHash(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTestApiKeyDB(t)
	repo := NewBunApiKeyRepository(db)

	seed := createTestApiKey("api-key-1", types.OwnerTypeUser, "user-1")
	if _, err := repo.Create(ctx, seed); err != nil {
		t.Fatalf("failed to seed api key: %v", err)
	}

	tests := []struct {
		name    string
		hash    string
		wantNil bool
	}{
		{name: "success", hash: seed.KeyHash},
		{name: "missing", hash: "missing", wantNil: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			apiKey, err := repo.GetByKeyHash(ctx, tc.hash)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil {
				if apiKey != nil {
					t.Fatalf("expected nil api key, got %#v", apiKey)
				}
				return
			}
			if apiKey == nil || apiKey.KeyHash != tc.hash {
				t.Fatalf("unexpected api key: %#v", apiKey)
			}
		})
	}
}

func TestBunApiKeyRepository_GetAll(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTestApiKeyDB(t)
	repo := NewBunApiKeyRepository(db)

	seedA := createTestApiKey("api-key-a", types.OwnerTypeUser, "user-1")
	seedA.CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
	seedA.UpdatedAt = seedA.CreatedAt
	seedB := createTestApiKey("api-key-b", types.OwnerTypeOrganization, "org-1")
	seedB.CreatedAt = time.Now().UTC().Add(-time.Hour)
	seedB.UpdatedAt = seedB.CreatedAt
	seedC := createTestApiKey("api-key-c", types.OwnerTypeUser, "user-2")
	seedC.CreatedAt = time.Now().UTC()
	seedC.UpdatedAt = seedC.CreatedAt

	for _, seed := range []*types.ApiKey{seedA, seedB, seedC} {
		if _, err := repo.Create(ctx, seed); err != nil {
			t.Fatalf("failed to seed api key: %v", err)
		}
	}

	ownerType := types.OwnerTypeUser
	referenceID := "user-1"

	tests := []struct {
		name        string
		ownerType   *string
		referenceID *string
		page        int
		limit       int
		wantIDs     []string
		wantTotal   int
	}{
		{name: "no filters page one", page: 1, limit: 2, wantIDs: []string{"api-key-c", "api-key-b"}, wantTotal: 3},
		{name: "owner filter", ownerType: &ownerType, page: 1, limit: 10, wantIDs: []string{"api-key-c", "api-key-a"}, wantTotal: 2},
		{name: "owner and reference filter", ownerType: &ownerType, referenceID: &referenceID, page: 1, limit: 10, wantIDs: []string{"api-key-a"}, wantTotal: 1},
		{name: "page two", page: 2, limit: 1, wantIDs: []string{"api-key-b"}, wantTotal: 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			items, total, err := repo.GetAll(ctx, tc.ownerType, tc.referenceID, tc.page, tc.limit)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if total != tc.wantTotal {
				t.Fatalf("expected total %d, got %d", tc.wantTotal, total)
			}
			if len(items) != len(tc.wantIDs) {
				t.Fatalf("expected %d items, got %d", len(tc.wantIDs), len(items))
			}
			for idx, wantID := range tc.wantIDs {
				if items[idx].ID != wantID {
					t.Fatalf("expected item %d to be %q, got %q", idx, wantID, items[idx].ID)
				}
			}
		})
	}
}

func TestBunApiKeyRepository_Update(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTestApiKeyDB(t)
	repo := NewBunApiKeyRepository(db)

	seed := createTestApiKey("api-key-1", types.OwnerTypeUser, "user-1")
	if _, err := repo.Create(ctx, seed); err != nil {
		t.Fatalf("failed to seed api key: %v", err)
	}

	updatedName := "updated"
	updatedRemaining := 7
	updatedMetadata := []byte(`{"scope":"all"}`)
	updated := createTestApiKey("api-key-1", types.OwnerTypeUser, "user-1")
	updated.Name = updatedName
	updated.Enabled = false
	updated.RequestsRemaining = &updatedRemaining
	updated.Metadata = updatedMetadata

	result, err := repo.Update(ctx, updated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.Name != updatedName || result.Enabled {
		t.Fatalf("unexpected updated api key: %#v", result)
	}

	stored, err := repo.GetByID(ctx, "api-key-1")
	if err != nil {
		t.Fatalf("failed to fetch updated api key: %v", err)
	}
	if stored == nil || stored.Name != updatedName || stored.Enabled || stored.RequestsRemaining == nil || *stored.RequestsRemaining != updatedRemaining {
		t.Fatalf("unexpected stored api key after update: %#v", stored)
	}
}

func TestBunApiKeyRepository_Delete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTestApiKeyDB(t)
	repo := NewBunApiKeyRepository(db)

	seed := createTestApiKey("api-key-1", types.OwnerTypeUser, "user-1")
	if _, err := repo.Create(ctx, seed); err != nil {
		t.Fatalf("failed to seed api key: %v", err)
	}

	if err := repo.Delete(ctx, "api-key-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	deleted, err := repo.GetByID(ctx, "api-key-1")
	if err != nil {
		t.Fatalf("unexpected error fetching deleted api key: %v", err)
	}
	if deleted != nil {
		t.Fatalf("expected deleted api key to be nil, got %#v", deleted)
	}
}

func TestBunApiKeyRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTestApiKeyDB(t)
	repo := NewBunApiKeyRepository(db)

	if _, err := repo.Create(ctx, createExpiredTestApiKey("expired", types.OwnerTypeUser, "user-1")); err != nil {
		t.Fatalf("failed to seed expired api key: %v", err)
	}
	if _, err := repo.Create(ctx, createTestApiKey("active", types.OwnerTypeUser, "user-1")); err != nil {
		t.Fatalf("failed to seed active api key: %v", err)
	}
	noExpiry := createTestApiKey("no-expiry", types.OwnerTypeUser, "user-1")
	noExpiry.ExpiresAt = nil
	if _, err := repo.Create(ctx, noExpiry); err != nil {
		t.Fatalf("failed to seed no-expiry api key: %v", err)
	}

	if err := repo.DeleteExpired(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expired, err := repo.GetByID(ctx, "expired")
	if err != nil {
		t.Fatalf("unexpected error checking expired key: %v", err)
	}
	if expired != nil {
		t.Fatalf("expected expired api key to be deleted, got %#v", expired)
	}

	active, err := repo.GetByID(ctx, "active")
	if err != nil {
		t.Fatalf("unexpected error checking active key: %v", err)
	}
	if active == nil {
		t.Fatal("expected active api key to remain")
	}

	kept, err := repo.GetByID(ctx, "no-expiry")
	if err != nil {
		t.Fatalf("unexpected error checking no-expiry key: %v", err)
	}
	if kept == nil {
		t.Fatal("expected no-expiry api key to remain")
	}
}

func TestBunApiKeyRepository_DeleteAllByOwner(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTestApiKeyDB(t)
	repo := NewBunApiKeyRepository(db)

	seeds := []*types.ApiKey{
		createTestApiKey("user-1-a", types.OwnerTypeUser, "user-1"),
		createTestApiKey("user-1-b", types.OwnerTypeUser, "user-1"),
		createTestApiKey("user-2", types.OwnerTypeUser, "user-2"),
		createTestApiKey("org-1", types.OwnerTypeOrganization, "org-1"),
	}
	for _, seed := range seeds {
		if _, err := repo.Create(ctx, seed); err != nil {
			t.Fatalf("failed to seed api key: %v", err)
		}
	}

	if err := repo.DeleteAllByOwner(ctx, types.OwnerTypeUser, "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, id := range []string{"user-1-a", "user-1-b"} {
		deleted, err := repo.GetByID(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error checking deleted key %s: %v", id, err)
		}
		if deleted != nil {
			t.Fatalf("expected %s to be deleted, got %#v", id, deleted)
		}
	}

	for _, id := range []string{"user-2", "org-1"} {
		kept, err := repo.GetByID(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error checking kept key %s: %v", id, err)
		}
		if kept == nil {
			t.Fatalf("expected %s to remain", id)
		}
	}
}

func TestBunApiKeyRepository_WithTx(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTestApiKeyDB(t)
	repo := NewBunApiKeyRepository(db)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	txRepo := repo.WithTx(tx)
	if txRepo == nil {
		t.Fatal("expected tx repo, got nil")
	}

	key := createTestApiKey("tx-key", types.OwnerTypeUser, "user-1")
	if _, err := txRepo.Create(ctx, key); err != nil {
		t.Fatalf("unexpected error creating key in tx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit tx: %v", err)
	}

	stored, err := repo.GetByID(ctx, "tx-key")
	if err != nil {
		t.Fatalf("unexpected error fetching tx key: %v", err)
	}
	if stored == nil {
		t.Fatal("expected tx key to persist after commit")
	}
}
