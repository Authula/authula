package services

import (
	"context"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/uptrace/bun"
)

// Mock SessionRepository for testing
type mockSessionRepository struct {
	sessions                      []*models.Session
	deleteExpiredCalled           bool
	deleteOldestByUserIDCalled    bool
	deleteOldestByUserIDUserID    string
	deleteOldestByUserIDKeepCount int
}

func (m *mockSessionRepository) GetByID(ctx context.Context, id string) (*models.Session, error) {
	for _, session := range m.sessions {
		if session.ID == id {
			return session, nil
		}
	}
	return nil, nil
}

func (m *mockSessionRepository) GetByToken(ctx context.Context, token string) (*models.Session, error) {
	for _, session := range m.sessions {
		if session.Token == token {
			return session, nil
		}
	}
	return nil, nil
}

func (m *mockSessionRepository) GetByUserID(ctx context.Context, userID string) (*models.Session, error) {
	for _, session := range m.sessions {
		if session.UserID == userID {
			return session, nil
		}
	}
	return nil, nil
}

func (m *mockSessionRepository) Create(ctx context.Context, session *models.Session) (*models.Session, error) {
	m.sessions = append(m.sessions, session)
	return session, nil
}

func (m *mockSessionRepository) Update(ctx context.Context, session *models.Session) (*models.Session, error) {
	for i, s := range m.sessions {
		if s.ID == session.ID {
			m.sessions[i] = session
			return session, nil
		}
	}
	return session, nil
}

func (m *mockSessionRepository) Delete(ctx context.Context, id string) error {
	for i, session := range m.sessions {
		if session.ID == id {
			m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	var remaining []*models.Session
	for _, session := range m.sessions {
		if session.UserID != userID {
			remaining = append(remaining, session)
		}
	}
	m.sessions = remaining
	return nil
}

func (m *mockSessionRepository) DeleteExpiredSessions(ctx context.Context) error {
	m.deleteExpiredCalled = true
	now := time.Now().UTC()
	var remaining []*models.Session
	for _, session := range m.sessions {
		if session.ExpiresAt.After(now) {
			remaining = append(remaining, session)
		}
	}
	m.sessions = remaining
	return nil
}

func (m *mockSessionRepository) DeleteOldestSessionsByUserID(ctx context.Context, userID string, maxCount int) error {
	m.deleteOldestByUserIDCalled = true
	m.deleteOldestByUserIDUserID = userID
	m.deleteOldestByUserIDKeepCount = maxCount

	var userSessions []*models.Session
	var otherSessions []*models.Session
	for _, session := range m.sessions {
		if session.UserID == userID {
			userSessions = append(userSessions, session)
		} else {
			otherSessions = append(otherSessions, session)
		}
	}

	for i := 0; i < len(userSessions)-1; i++ {
		for j := i + 1; j < len(userSessions); j++ {
			if userSessions[j].CreatedAt.Before(userSessions[i].CreatedAt) {
				userSessions[i], userSessions[j] = userSessions[j], userSessions[i]
			}
		}
	}

	if maxCount < len(userSessions) {
		userSessions = userSessions[len(userSessions)-maxCount:]
	}

	m.sessions = append(otherSessions, userSessions...)
	return nil
}

func (m *mockSessionRepository) WithTx(tx bun.IDB) repositories.SessionRepository {
	return m
}

func (m *mockSessionRepository) GetDistinctUserIDs(ctx context.Context) ([]string, error) {
	userMap := make(map[string]bool)
	for _, session := range m.sessions {
		userMap[session.UserID] = true
	}
	var userIDs []string
	for userID := range userMap {
		userIDs = append(userIDs, userID)
	}
	return userIDs, nil
}

func TestSessionService_CleanupExpiredSessions(t *testing.T) {
	mockRepo := &mockSessionRepository{}
	service := NewSessionService(mockRepo, nil, nil)
	ctx := context.Background()

	now := time.Now().UTC()

	expiredSession := &models.Session{
		ID:        "expired-1",
		UserID:    "user1",
		Token:     "token1",
		ExpiresAt: now.Add(-1 * time.Hour),
		CreatedAt: now.Add(-2 * time.Hour),
	}
	activeSession := &models.Session{
		ID:        "active-1",
		UserID:    "user1",
		Token:     "token2",
		ExpiresAt: now.Add(1 * time.Hour),
		CreatedAt: now.Add(-1 * time.Hour),
	}

	mockRepo.sessions = []*models.Session{expiredSession, activeSession}

	err := service.CleanupExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("expected no error on CleanupExpiredSessions, got %v", err)
	}

	if !mockRepo.deleteExpiredCalled {
		t.Fatal("expected DeleteExpiredSessions to be called on repository")
	}

	if len(mockRepo.sessions) != 1 {
		t.Fatalf("expected 1 remaining session, got %d", len(mockRepo.sessions))
	}
	if mockRepo.sessions[0].ID != "active-1" {
		t.Fatalf("expected active session to remain, got %s", mockRepo.sessions[0].ID)
	}
}

func TestSessionService_EnforceMaxSessionsPerUser(t *testing.T) {
	mockRepo := &mockSessionRepository{}
	service := NewSessionService(mockRepo, nil, nil)
	ctx := context.Background()

	now := time.Now().UTC()

	// User1 has 5 sessions, should enforce max of 3
	user1Sessions := []*models.Session{
		{
			ID:        "user1-session-1",
			UserID:    "user1",
			Token:     "token1",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now.Add(-4 * time.Hour),
		},
		{
			ID:        "user1-session-2",
			UserID:    "user1",
			Token:     "token2",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now.Add(-3 * time.Hour),
		},
		{
			ID:        "user1-session-3",
			UserID:    "user1",
			Token:     "token3",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:        "user1-session-4",
			UserID:    "user1",
			Token:     "token4",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now.Add(-1 * time.Hour),
		},
		{
			ID:        "user1-session-5",
			UserID:    "user1",
			Token:     "token5",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now,
		},
	}

	// User2 has 2 sessions, should not be affected by max of 3
	user2Sessions := []*models.Session{
		{
			ID:        "user2-session-1",
			UserID:    "user2",
			Token:     "token6",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now.Add(-1 * time.Hour),
		},
		{
			ID:        "user2-session-2",
			UserID:    "user2",
			Token:     "token7",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now,
		},
	}

	mockRepo.sessions = append(user1Sessions, user2Sessions...)

	// Enforce max of 3 sessions per user
	err := service.EnforceMaxSessionsPerUser(ctx, 3)
	if err != nil {
		t.Fatalf("expected no error on EnforceMaxSessionsPerUser, got %v", err)
	}

	// Verify user1 session count
	var user1RemainingCount int
	for _, session := range mockRepo.sessions {
		if session.UserID == "user1" {
			user1RemainingCount++
		}
	}
	if user1RemainingCount != 3 {
		t.Fatalf("expected 3 sessions for user1 after cleanup, got %d", user1RemainingCount)
	}

	// Verify user2 session count (should remain unchanged)
	var user2RemainingCount int
	for _, session := range mockRepo.sessions {
		if session.UserID == "user2" {
			user2RemainingCount++
		}
	}
	if user2RemainingCount != 2 {
		t.Fatalf("expected 2 sessions for user2 after cleanup, got %d", user2RemainingCount)
	}

	// Verify the newest sessions for user1 were kept
	expectedUserIDs := map[string]bool{
		"user1-session-3": true,
		"user1-session-4": true,
		"user1-session-5": true,
	}
	for _, session := range mockRepo.sessions {
		if session.UserID == "user1" {
			if !expectedUserIDs[session.ID] {
				t.Fatalf("expected session %s to be deleted", session.ID)
			}
		}
	}
}

func TestSessionService_RunCleanup(t *testing.T) {
	mockRepo := &mockSessionRepository{}
	service := NewSessionService(mockRepo, nil, nil)
	ctx := context.Background()

	now := time.Now().UTC()

	// Create sessions: 2 expired, 3 for user1 (total 5), 2 for user2
	mockRepo.sessions = []*models.Session{
		// Expired sessions
		{
			ID:        "expired-1",
			UserID:    "user3",
			Token:     "token1",
			ExpiresAt: now.Add(-1 * time.Hour),
			CreatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:        "expired-2",
			UserID:    "user4",
			Token:     "token2",
			ExpiresAt: now.Add(-30 * time.Minute),
			CreatedAt: now.Add(-1 * time.Hour),
		},
		// User1 sessions (5 total, should be reduced to 3 by cleanup)
		{
			ID:        "user1-session-1",
			UserID:    "user1",
			Token:     "token3",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now.Add(-4 * time.Hour),
		},
		{
			ID:        "user1-session-2",
			UserID:    "user1",
			Token:     "token4",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now.Add(-3 * time.Hour),
		},
		{
			ID:        "user1-session-3",
			UserID:    "user1",
			Token:     "token5",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:        "user1-session-4",
			UserID:    "user1",
			Token:     "token6",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now.Add(-1 * time.Hour),
		},
		{
			ID:        "user1-session-5",
			UserID:    "user1",
			Token:     "token7",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now,
		},
		// User2 sessions (2 total, OK with max of 3)
		{
			ID:        "user2-session-1",
			UserID:    "user2",
			Token:     "token8",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now.Add(-1 * time.Hour),
		},
		{
			ID:        "user2-session-2",
			UserID:    "user2",
			Token:     "token9",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now,
		},
	}

	// Run cleanup (should delete expired + enforce max of 3 per user)
	err := service.RunCleanup(ctx, 3)
	if err != nil {
		t.Fatalf("expected no error on RunCleanup, got %v", err)
	}

	// Verify expired sessions are deleted
	for _, session := range mockRepo.sessions {
		if session.ExpiresAt.Before(now) {
			t.Fatalf("expected expired session %s to be deleted", session.ID)
		}
	}

	// Verify user1 has only 3 sessions
	var user1Count int
	for _, session := range mockRepo.sessions {
		if session.UserID == "user1" {
			user1Count++
		}
	}
	if user1Count != 3 {
		t.Fatalf("expected 3 sessions for user1 after cleanup, got %d", user1Count)
	}

	// Verify user2 still has 2 sessions
	var user2Count int
	for _, session := range mockRepo.sessions {
		if session.UserID == "user2" {
			user2Count++
		}
	}
	if user2Count != 2 {
		t.Fatalf("expected 2 sessions for user2 after cleanup, got %d", user2Count)
	}
}

func TestSessionService_RunCleanup_WithMaxZero(t *testing.T) {
	mockRepo := &mockSessionRepository{}
	service := NewSessionService(mockRepo, nil, nil)
	ctx := context.Background()

	now := time.Now().UTC()

	// Create one active session for user1
	mockRepo.sessions = []*models.Session{
		{
			ID:        "user1-session-1",
			UserID:    "user1",
			Token:     "token1",
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now,
		},
	}

	// Run cleanup with max = 0
	// When max <= 0, EnforceMaxSessionsPerUser should not delete (it's skipped)
	// So the session should still exist after cleanup
	err := service.RunCleanup(ctx, 0)
	if err != nil {
		t.Fatalf("expected no error on RunCleanup with max=0, got %v", err)
	}

	// With max=0, EnforceMaxSessionsPerUser is skipped, so session remains unchanged
	// This is by design - max=0 means "don't enforce max sessions"
	var user1Count int
	for _, session := range mockRepo.sessions {
		if session.UserID == "user1" {
			user1Count++
		}
	}
	if user1Count != 1 {
		t.Fatalf("expected 1 session for user1 (max enforcement skipped for max<=0), got %d", user1Count)
	}
}
