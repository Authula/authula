package services

import (
	"context"
	"fmt"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/internal/security"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/services"
)

type verificationService struct {
	repo   repositories.VerificationRepository
	signer security.TokenSigner
	hooks  *models.CoreDatabaseHooks
}

func NewVerificationService(
	repo repositories.VerificationRepository,
	signer security.TokenSigner,
	hooks *models.CoreDatabaseHooks,
) services.VerificationService {
	return &verificationService{
		repo:   repo,
		signer: signer,
		hooks:  hooks,
	}
}

func (s *verificationService) Create(
	ctx context.Context,
	userID string,
	hashedToken string,
	vType models.VerificationType,
	value string,
	expiry time.Duration,
) (*models.Verification, error) {
	if hashedToken == "" {
		return nil, fmt.Errorf("hashedToken cannot be empty")
	}

	verification := &models.Verification{
		ID:         util.GenerateUUID(),
		UserID:     &userID,
		Identifier: value,
		Token:      hashedToken,
		Type:       vType,
		ExpiresAt:  time.Now().UTC().Add(expiry),
	}

	if s.hooks != nil && s.hooks.Verifications != nil && s.hooks.Verifications.BeforeCreate != nil {
		if err := s.hooks.Verifications.BeforeCreate(verification); err != nil {
			return nil, err
		}
	}

	created, err := s.repo.Create(ctx, verification)
	if err != nil {
		return nil, err
	}

	if s.hooks != nil && s.hooks.Verifications != nil && s.hooks.Verifications.AfterCreate != nil {
		if err := s.hooks.Verifications.AfterCreate(*created); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *verificationService) GetByToken(ctx context.Context, hashedToken string) (*models.Verification, error) {
	return s.repo.GetByToken(ctx, hashedToken)
}

func (s *verificationService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *verificationService) DeleteByUserIDAndType(ctx context.Context, userID string, vType models.VerificationType) error {
	return s.repo.DeleteByUserIDAndType(ctx, userID, vType)
}

func (s *verificationService) IsExpired(v *models.Verification) bool {
	return time.Now().UTC().After(v.ExpiresAt)
}
