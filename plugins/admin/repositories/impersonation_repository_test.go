package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestImpersonationRepository_CreateGetAndEnd(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := repositories.NewAdminRepositories(db).ImpersonationRepository()
	ctx := context.Background()

	seedRepositoryUser(t, db, "imp-actor", "imp-actor@example.com")
	seedRepositoryUser(t, db, "imp-target", "imp-target@example.com")

	exists, err := repo.UserExists(ctx, "imp-actor")
	if err != nil {
		t.Fatalf("failed to check user existence: %v", err)
	}
	if !exists {
		t.Fatalf("expected actor user to exist")
	}

	impersonation := &admintypes.Impersonation{
		ID:           "imp-1",
		ActorUserID:  "imp-actor",
		TargetUserID: "imp-target",
		Reason:       "support",
		StartedAt:    time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(5 * time.Minute),
	}

	if err := repo.CreateImpersonation(ctx, impersonation); err != nil {
		t.Fatalf("failed to create impersonation: %v", err)
	}

	active, err := repo.GetActiveImpersonationByID(ctx, impersonation.ID)
	if err != nil {
		t.Fatalf("failed to get active impersonation by id: %v", err)
	}
	if active == nil {
		t.Fatalf("expected active impersonation")
	}

	latest, err := repo.GetLatestActiveImpersonationByActor(ctx, "imp-actor")
	if err != nil {
		t.Fatalf("failed to get latest active impersonation: %v", err)
	}
	if latest == nil || latest.ID != impersonation.ID {
		t.Fatalf("expected latest impersonation id %s", impersonation.ID)
	}

	endedBy := "imp-actor"
	if err := repo.EndImpersonation(ctx, impersonation.ID, &endedBy); err != nil {
		t.Fatalf("failed to end impersonation: %v", err)
	}

	activeAfterEnd, err := repo.GetActiveImpersonationByID(ctx, impersonation.ID)
	if err != nil {
		t.Fatalf("failed to check ended impersonation: %v", err)
	}
	if activeAfterEnd != nil {
		t.Fatalf("expected impersonation to be inactive after end")
	}
}
