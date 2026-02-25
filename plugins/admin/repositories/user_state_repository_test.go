package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestUserStateRepository_UpsertGetDelete(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := repositories.NewAdminRepositories(db).UserStateRepository()
	ctx := context.Background()

	seedRepositoryUser(t, db, "state-user-1", "state-user-1@example.com")
	until := time.Now().UTC().Add(2 * time.Hour)
	reason := "policy violation"
	actor := "state-user-1"

	if err := repo.Upsert(ctx, &admintypes.AdminUserState{
		UserID:         "state-user-1",
		IsBanned:       true,
		BannedAt:       &until,
		BannedUntil:    &until,
		BannedReason:   &reason,
		BannedByUserID: &actor,
	}); err != nil {
		t.Fatalf("failed to upsert user state: %v", err)
	}

	state, err := repo.GetByUserID(ctx, "state-user-1")
	if err != nil {
		t.Fatalf("failed to get user state: %v", err)
	}
	if state == nil || !state.IsBanned {
		t.Fatalf("expected banned user state")
	}

	banned, err := repo.GetBanned(ctx)
	if err != nil {
		t.Fatalf("failed to get banned users: %v", err)
	}
	if len(banned) == 0 {
		t.Fatalf("expected at least one banned user state")
	}

	if err := repo.Delete(ctx, "state-user-1"); err != nil {
		t.Fatalf("failed to delete user state: %v", err)
	}

	deleted, err := repo.GetByUserID(ctx, "state-user-1")
	if err != nil {
		t.Fatalf("failed to get deleted user state: %v", err)
	}
	if deleted != nil {
		t.Fatalf("expected nil state after delete")
	}
}
