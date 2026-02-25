package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestSessionStateRepository_UpsertGetDeleteAndExists(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := repositories.NewAdminRepositories(db).SessionStateRepository()
	ctx := context.Background()

	seedRepositoryUser(t, db, "state-session-user-1", "state-session-user-1@example.com")
	seedRepositorySession(t, db, "state-session-1", "state-session-user-1")

	exists, err := repo.SessionExists(ctx, "state-session-1")
	if err != nil {
		t.Fatalf("failed to check session existence: %v", err)
	}
	if !exists {
		t.Fatalf("expected session to exist")
	}

	now := time.Now().UTC()
	reason := "manual revoke"
	actor := "state-session-user-1"

	if err := repo.Upsert(ctx, &admintypes.AdminSessionState{
		SessionID:       "state-session-1",
		RevokedAt:       &now,
		RevokedReason:   &reason,
		RevokedByUserID: &actor,
	}); err != nil {
		t.Fatalf("failed to upsert session state: %v", err)
	}

	state, err := repo.GetBySessionID(ctx, "state-session-1")
	if err != nil {
		t.Fatalf("failed to get session state: %v", err)
	}
	if state == nil || state.RevokedAt == nil {
		t.Fatalf("expected revoked session state")
	}

	revoked, err := repo.GetRevoked(ctx)
	if err != nil {
		t.Fatalf("failed to get revoked sessions: %v", err)
	}
	if len(revoked) == 0 {
		t.Fatalf("expected at least one revoked session state")
	}

	if err := repo.Delete(ctx, "state-session-1"); err != nil {
		t.Fatalf("failed to delete session state: %v", err)
	}

	deleted, err := repo.GetBySessionID(ctx, "state-session-1")
	if err != nil {
		t.Fatalf("failed to get deleted session state: %v", err)
	}
	if deleted != nil {
		t.Fatalf("expected nil state after delete")
	}
}
