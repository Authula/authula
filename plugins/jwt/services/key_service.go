package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Authula/authula/models"
)

// keyService adapts the shared signing key service to the JWT plugin's rotation
// policy. All key material and persistence live in the JWT plugin.
type keyService struct {
	shared    SigningKeyService
	logger    models.Logger
	purpose   string
	algorithm string
}

func NewKeyService(shared SigningKeyService, logger models.Logger, purpose string, algorithm string) KeyService {
	return &keyService{
		shared:    shared,
		logger:    logger,
		purpose:   purpose,
		algorithm: algorithm,
	}
}

func (s *keyService) GenerateKeysIfMissing(ctx context.Context) error {
	if err := s.shared.GenerateKey(ctx, s.purpose, s.algorithm); err != nil {
		return fmt.Errorf("failed to generate keys: %w", err)
	}
	return nil
}

func (s *keyService) GetActiveKey(ctx context.Context) (*models.SigningKey, error) {
	key, err := s.shared.GetActiveKey(ctx, s.purpose, s.algorithm)
	if err != nil {
		if errors.Is(err, models.ErrSigningKeyNotFound) {
			return nil, fmt.Errorf("no active key found: %w", err)
		}
		return nil, fmt.Errorf("failed to get active key: %w", err)
	}
	return key, nil
}

func (s *keyService) GetPublicKeys(ctx context.Context) ([]models.SigningKey, error) {
	keys, err := s.shared.GetPublicKeys(ctx, s.purpose)
	if err != nil {
		return nil, fmt.Errorf("failed to get public keys: %w", err)
	}
	return keys, nil
}

func (s *keyService) IsKeyRotationDue(ctx context.Context, rotationInterval time.Duration) bool {
	key, err := s.GetActiveKey(ctx)
	if err != nil {
		return false
	}
	return time.Since(key.CreatedAt) > rotationInterval
}

func (s *keyService) RotateKeysIfNeeded(ctx context.Context, rotationInterval time.Duration, gracePeriod time.Duration, invalidateCacheFunc func(context.Context) error) (bool, error) {
	if !s.IsKeyRotationDue(ctx, rotationInterval) {
		return false, nil
	}

	if err := s.shared.RotateKeyWithGrace(ctx, s.purpose, s.algorithm, gracePeriod); err != nil {
		return false, fmt.Errorf("failed to rotate signing key: %w", err)
	}

	if invalidateCacheFunc != nil {
		if err := invalidateCacheFunc(ctx); err != nil {
			s.logger.Warn("failed to invalidate cache after key rotation", "error", err)
		}
	}

	return true, nil
}
