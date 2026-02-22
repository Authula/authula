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
	sessions                     []*models.Session
	deleteExpiredCalled          bool
	deleteOldestByUserIDCalled   bool
	deleteOldestByUserIDUserID   string
	deleteOldestByUserIDMaxCount int
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

func (m *mockSessionRepository) DeleteExpired(ctx context.Context) error {
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

func (m *mockSessionRepository) DeleteOldestByUserID(ctx context.Context, userID string, maxCount int) error {
	m.deleteOldestByUserIDCalled = true
	m.deleteOldestByUserIDUserID = userID
	m.deleteOldestByUserIDMaxCount = maxCount

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

func TestSessionService_DeleteAllExpired(t *testing.T) {
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

	err := service.DeleteAllExpired(ctx)
	if err != nil {
		t.Fatalf("expected no error on DeleteAllExpired, got %v", err)
	}

	if !mockRepo.deleteExpiredCalled {
		t.Fatal("expected DeleteExpired to be called on repository")
	}

	if len(mockRepo.sessions) != 1 {
		t.Fatalf("expected 1 remaining session, got %d", len(mockRepo.sessions))
	}
	if mockRepo.sessions[0].ID != "active-1" {
		t.Fatalf("expected active session to remain, got %s", mockRepo.sessions[0].ID)
	}
}
