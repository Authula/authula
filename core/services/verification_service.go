package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Authula/authula/core/repositories"
	"github.com/Authula/authula/core/security"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type verificationService struct {
	repo         repositories.VerificationRepository
	signer       security.TokenSigner
	serviceHooks *models.CoreServiceHooksConfig
}

func NewVerificationService(
	repo repositories.VerificationRepository,
	signer security.TokenSigner,
	serviceHooks *models.CoreServiceHooksConfig,
) services.VerificationService {
	return &verificationService{
		repo:         repo,
		signer:       signer,
		serviceHooks: serviceHooks,
	}
}

func runVerificationHooks(hooks []models.VerificationHook, verification *models.Verification) error {
	for _, hook := range hooks {
		if err := hook(verification); err != nil {
			return err
		}
	}
	return nil
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

	if s.serviceHooks != nil && s.serviceHooks.Verifications != nil {
		if err := runVerificationHooks(s.serviceHooks.Verifications.BeforeCreateHooks(), verification); err != nil {
			return nil, err
		}
	}

	created, err := s.repo.Create(ctx, verification)
	if err != nil {
		return nil, err
	}

	if s.serviceHooks != nil && s.serviceHooks.Verifications != nil {
		if err := runVerificationHooks(s.serviceHooks.Verifications.AfterCreateHooks(), created); err != nil {
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

func (s *verificationService) DeleteExpired(ctx context.Context) error {
	return s.repo.DeleteExpired(ctx)
}
