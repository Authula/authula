package services

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type VerificationServiceImpl struct {
	config *models.Config
	db     *gorm.DB
}

func NewVerificationServiceImpl(config *models.Config, db *gorm.DB) *VerificationServiceImpl {
	return &VerificationServiceImpl{config: config, db: db}
}

// Creates a new verification record
func (s *VerificationServiceImpl) CreateVerification(v *models.Verification) error {
	v.ID = uuid.NewString()

	now := time.Now().UTC()
	v.CreatedAt = now
	v.UpdatedAt = now
	v.ExpiresAt = now.Add(time.Hour)

	if s.config.DatabaseHooks.Verifications != nil && s.config.DatabaseHooks.Verifications.BeforeCreate != nil {
		if err := s.config.DatabaseHooks.Verifications.BeforeCreate(v); err != nil {
			return err
		}
	}

	if err := s.db.Create(v).Error; err != nil {
		return err
	}

	if s.config.DatabaseHooks.Verifications != nil && s.config.DatabaseHooks.Verifications.AfterCreate != nil {
		go func() {
			if err := s.config.DatabaseHooks.Verifications.AfterCreate(*v); err != nil {
				slog.Error("verification after create hook failed", "error", err.Error())
			}
		}()
	}

	return nil
}

// Retrieves a verification record by token
func (s *VerificationServiceImpl) GetVerificationByToken(token string) (*models.Verification, error) {
	var v models.Verification
	if err := s.db.Where("token = ?", token).First(&v).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

// Deletes a verification record by ID
func (s *VerificationServiceImpl) DeleteVerification(id string) error {
	return s.db.Delete(&models.Verification{}, "id = ?", id).Error
}

// Checks if the verification token is expired
func (s *VerificationServiceImpl) IsExpired(verification *models.Verification) bool {
	return time.Now().UTC().After(verification.ExpiresAt)
}
