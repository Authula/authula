package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// Helper function to create an in-memory SQLite database for testing
func newTestSessionDB(t *testing.T) bun.IDB {
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())

	// Create the Session table
	ctx := context.Background()
	if _, err := db.NewCreateTable().Model(&models.Session{}).Exec(ctx); err != nil {
		t.Fatalf("failed to create Session table: %v", err)
	}

	return db
}

// Helper function to create a test session
func createTestSession(userID string, expiresAt time.Time) *models.Session {
	return &models.Session{
		ID:        generateTestUUID(),
		UserID:    userID,
		Token:     "hashed_token_" + generateTestUUID(),
		ExpiresAt: expiresAt,
		IPAddress: nil,
		UserAgent: nil,
	}
}

// Helper function to generate test UUIDs
var testUUIDCounter = 0

func generateTestUUID() string {
	testUUIDCounter++
	return "test-uuid-" + string(rune(testUUIDCounter))
}

func TestBunSessionRepository_DeleteExpiredSessions(t *testing.T) {
	db := newTestSessionDB(t)
	repo := NewBunSessionRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()

	// Create some sessions with different expiration times
	expiredSession1 := createTestSession("user1", now.Add(-1*time.Hour))    // Expired
	expiredSession2 := createTestSession("user2", now.Add(-30*time.Minute)) // Expired
	activeSession1 := createTestSession("user1", now.Add(1*time.Hour))      // Not expired
	activeSession2 := createTestSession("user3", now.Add(2*time.Hour))      // Not expired

	// Insert all sessions
	for _, session := range []*models.Session{expiredSession1, expiredSession2, activeSession1, activeSession2} {
		_, err := repo.Create(ctx, session)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}
	}

	// Verify all 4 sessions exist
	allSessions := []*models.Session{}
	err := db.NewSelect().Model(&allSessions).Scan(ctx)
	if err != nil {
		t.Fatalf("failed to query sessions: %v", err)
	}
	if len(allSessions) != 4 {
		t.Fatalf("expected 4 sessions, got %d", len(allSessions))
	}

	// Delete expired sessions
	err = repo.DeleteExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("expected no error on DeleteExpiredSessions, got %v", err)
	}

	// Verify only 2 sessions remain (the active ones)
	remainingSessions := []*models.Session{}
	err = db.NewSelect().Model(&remainingSessions).Scan(ctx)
	if err != nil {
		t.Fatalf("failed to query remaining sessions: %v", err)
	}
	if len(remainingSessions) != 2 {
		t.Fatalf("expected 2 remaining sessions, got %d", len(remainingSessions))
	}

	// Verify the remaining sessions are the active ones
	for _, session := range remainingSessions {
		if session.ExpiresAt.Before(now) {
			t.Fatalf("expected remaining sessions to be active, but found expired session with ID %s", session.ID)
		}
	}
}

func TestBunSessionRepository_DeleteExpiredSessions_NoExpired(t *testing.T) {
	db := newTestSessionDB(t)
	repo := NewBunSessionRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()

	// Create only active sessions
	activeSession1 := createTestSession("user1", now.Add(1*time.Hour))
	activeSession2 := createTestSession("user2", now.Add(2*time.Hour))

	// Insert all sessions
	for _, session := range []*models.Session{activeSession1, activeSession2} {
		_, err := repo.Create(ctx, session)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}
	}

	// Delete expired sessions (should delete none)
	err := repo.DeleteExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("expected no error on DeleteExpiredSessions, got %v", err)
	}

	// Verify all 2 sessions still exist
	remainingSessions := []*models.Session{}
	err = db.NewSelect().Model(&remainingSessions).Scan(ctx)
	if err != nil {
		t.Fatalf("failed to query remaining sessions: %v", err)
	}
	if len(remainingSessions) != 2 {
		t.Fatalf("expected 2 remaining sessions, got %d", len(remainingSessions))
	}
}

func TestBunSessionRepository_DeleteExpiredSessions_AllExpired(t *testing.T) {
	db := newTestSessionDB(t)
	repo := NewBunSessionRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()

	// Create only expired sessions
	expiredSession1 := createTestSession("user1", now.Add(-1*time.Hour))
	expiredSession2 := createTestSession("user2", now.Add(-30*time.Minute))
	expiredSession3 := createTestSession("user3", now.Add(-5*time.Minute))

	// Insert all sessions
	for _, session := range []*models.Session{expiredSession1, expiredSession2, expiredSession3} {
		_, err := repo.Create(ctx, session)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}
	}

	// Delete expired sessions (should delete all)
	err := repo.DeleteExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("expected no error on DeleteExpiredSessions, got %v", err)
	}

	// Verify no sessions remain
	remainingSessions := []*models.Session{}
	err = db.NewSelect().Model(&remainingSessions).Scan(ctx)
	if err != nil {
		t.Fatalf("failed to query remaining sessions: %v", err)
	}
	if len(remainingSessions) != 0 {
		t.Fatalf("expected 0 remaining sessions, got %d", len(remainingSessions))
	}
}

func TestBunSessionRepository_DeleteOldestSessionsByUserID(t *testing.T) {
	db := newTestSessionDB(t)
	repo := NewBunSessionRepository(db)
	ctx := context.Background()

	userID := "user1"

	// Create 5 sessions for the same user - insert one at a time with delays
	createdIDs := []string{}
	for i := 1; i <= 5; i++ {
		session := &models.Session{
			ID:        fmt.Sprintf("sess%d", i),
			UserID:    userID,
			Token:     fmt.Sprintf("token%d", i),
			ExpiresAt: time.Now().UTC().Add(time.Duration(i) * time.Hour),
		}
		_, err := repo.Create(ctx, session)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}
		createdIDs = append(createdIDs, session.ID)
		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Verify 5 sessions exist
	var before []*models.Session
	err := db.NewSelect().Model(&before).Where("user_id = ?", userID).Order("created_at ASC").Scan(ctx)
	if err != nil {
		t.Fatalf("failed to query sessions: %v", err)
	}
	if len(before) != 5 {
		t.Fatalf("expected 5 sessions before cleanup, got %d", len(before))
	}

	// The first two (oldest) created should be sess1 and sess2
	// The last two (newest) created should be sess4 and sess5
	newestTwoCreated := [2]string{createdIDs[3], createdIDs[4]}

	// Delete oldest 3, keep newest 2
	err = repo.DeleteOldestSessionsByUserID(ctx, userID, 2)
	if err != nil {
		t.Fatalf("expected no error on DeleteOldestSessionsByUserID, got %v", err)
	}

	// Verify only 2 sessions remain
	var after []*models.Session
	err = db.NewSelect().Model(&after).Where("user_id = ?", userID).Order("created_at ASC").Scan(ctx)
	if err != nil {
		t.Fatalf("failed to query remaining sessions: %v", err)
	}
	if len(after) != 2 {
		remainingIDs := make([]string, 0)
		for _, s := range after {
			remainingIDs = append(remainingIDs, s.ID)
		}
		t.Fatalf("expected 2 remaining sessions, got %d. Remaining: %v", len(after), remainingIDs)
	}

	// Verify the remaining sessions are the newest ones
	for _, session := range after {
		found := false
		for _, newestID := range newestTwoCreated {
			if session.ID == newestID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected remaining sessions to be %v, but found %s", newestTwoCreated, session.ID)
		}
	}
}

func TestBunSessionRepository_DeleteOldestSessionsByUserID_KeepAllWhenBelowLimit(t *testing.T) {
	db := newTestSessionDB(t)
	repo := NewBunSessionRepository(db)
	ctx := context.Background()

	userID := "user1"
	now := time.Now().UTC()

	// Create only 2 sessions for the user
	sessions := []*models.Session{
		createTestSession(userID, now.Add(1*time.Hour)),
		createTestSession(userID, now.Add(2*time.Hour)),
	}

	for _, session := range sessions {
		_, err := repo.Create(ctx, session)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}
	}

	// Try to delete oldest, but keep 5 (more than exists)
	err := repo.DeleteOldestSessionsByUserID(ctx, userID, 5)
	if err != nil {
		t.Fatalf("expected no error on DeleteOldestSessionsByUserID, got %v", err)
	}

	// Verify all 2 sessions still exist
	remainingSessions := []*models.Session{}
	err = db.NewSelect().Model(&remainingSessions).Where("user_id = ?", userID).Scan(ctx)
	if err != nil {
		t.Fatalf("failed to query remaining sessions: %v", err)
	}
	if len(remainingSessions) != 2 {
		t.Fatalf("expected 2 remaining sessions, got %d", len(remainingSessions))
	}
}

func TestBunSessionRepository_DeleteOldestSessionsByUserID_DeleteAllWhenKeepIsZero(t *testing.T) {
	db := newTestSessionDB(t)
	repo := NewBunSessionRepository(db)
	ctx := context.Background()

	userID := "user1"
	now := time.Now().UTC()

	// Create 3 sessions for the user
	sessions := []*models.Session{
		createTestSession(userID, now.Add(1*time.Hour)),
		createTestSession(userID, now.Add(2*time.Hour)),
		createTestSession(userID, now.Add(3*time.Hour)),
	}

	for _, session := range sessions {
		_, err := repo.Create(ctx, session)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}
	}

	// Delete all sessions (keep 0)
	err := repo.DeleteOldestSessionsByUserID(ctx, userID, 0)
	if err != nil {
		t.Fatalf("expected no error on DeleteOldestSessionsByUserID, got %v", err)
	}

	// Verify no sessions remain for this user
	remainingSessions := []*models.Session{}
	err = db.NewSelect().Model(&remainingSessions).Where("user_id = ?", userID).Scan(ctx)
	if err != nil {
		t.Fatalf("failed to query remaining sessions: %v", err)
	}
	if len(remainingSessions) != 0 {
		t.Fatalf("expected 0 remaining sessions for user, got %d", len(remainingSessions))
	}
}

func TestBunSessionRepository_DeleteOldestSessionsByUserID_MultipleUsers(t *testing.T) {
	db := newTestSessionDB(t)
	repo := NewBunSessionRepository(db)
	ctx := context.Background()

	user1ID := "user1"
	user2ID := "user2"
	now := time.Now().UTC()

	// Create 3 sessions for user1 with good spacing
	for i := 0; i < 3; i++ {
		session := createTestSession(user1ID, now.Add(time.Duration(i+1)*time.Hour))
		_, err := repo.Create(ctx, session)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Small gap between users
	time.Sleep(50 * time.Millisecond)

	// Create 3 sessions for user2 - use later timestamps to distinguish
	for i := range 3 {
		session := createTestSession(user2ID, now.Add(time.Duration(i+10)*time.Hour))
		_, err := repo.Create(ctx, session)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Delete oldest for user1 only, keep 1
	err := repo.DeleteOldestSessionsByUserID(ctx, user1ID, 1)
	if err != nil {
		t.Fatalf("expected no error on DeleteOldestSessionsByUserID, got %v", err)
	}

	// Verify user1 has only 1 session
	user1Sessions := []*models.Session{}
	err = db.NewSelect().Model(&user1Sessions).Where("user_id = ?", user1ID).Scan(ctx)
	if err != nil {
		t.Fatalf("failed to query user1 sessions: %v", err)
	}
	if len(user1Sessions) != 1 {
		t.Fatalf("expected 1 session for user1, got %d (IDs: ", len(user1Sessions))
	}

	// Verify user2 still has 3 sessions (unchanged)
	user2Sessions := []*models.Session{}
	err = db.NewSelect().Model(&user2Sessions).Where("user_id = ?", user2ID).Scan(ctx)
	if err != nil {
		t.Fatalf("failed to query user2 sessions: %v", err)
	}
	if len(user2Sessions) != 3 {
		t.Fatalf("expected 3 sessions for user2, got %d", len(user2Sessions))
	}
}
