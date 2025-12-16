package services

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type SessionServiceImpl struct {
	config *models.Config
	db     *gorm.DB
}

func NewSessionServiceImpl(config *models.Config, db *gorm.DB) *SessionServiceImpl {
	return &SessionServiceImpl{config: config, db: db}
}

// CreateSession creates a new session for a user
func (s *SessionServiceImpl) CreateSession(userID string, token string) (*models.Session, error) {
	session := &models.Session{
		ID:        uuid.NewString(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if s.config.DatabaseHooks.Sessions != nil && s.config.DatabaseHooks.Sessions.BeforeCreate != nil {
		if err := s.config.DatabaseHooks.Sessions.BeforeCreate(session); err != nil {
			return nil, err
		}
	}

	if err := s.db.Create(session).Error; err != nil {
		return nil, err
	}

	if s.config.DatabaseHooks.Sessions != nil && s.config.DatabaseHooks.Sessions.AfterCreate != nil {
		go func() {
			if err := s.config.DatabaseHooks.Sessions.AfterCreate(*session); err != nil {
				slog.Error("session after create hook failed", "error", err.Error())
			}
		}()
	}

	return session, nil
}

// GetSessionByUserID retrieves a session by the associated userID.
func (s *SessionServiceImpl) GetSessionByUserID(userID string) (*models.Session, error) {
	var sess models.Session
	if err := s.db.Where("user_id = ?", userID).First(&sess).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &sess, nil
}

// GetSessionByToken retrieves a session by its token.
func (s *SessionServiceImpl) GetSessionByToken(token string) (*models.Session, error) {
	var sess models.Session
	if err := s.db.Where("token = ?", token).First(&sess).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &sess, nil
}

// DeleteSessionByID deletes a session by its ID.
func (s *SessionServiceImpl) DeleteSessionByID(ID string) error {
	return s.db.Where("id = ?", ID).Delete(&models.Session{}).Error
}
