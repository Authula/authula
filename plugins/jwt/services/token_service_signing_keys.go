package services

import (
	"context"
	"time"

	"github.com/Authula/authula/models"
)

// The JWT service exposes the plugin-owned signing keys to other plugins
// through the public services.JWTService contract. These methods delegate
// directly to the signing key service without changing its behaviour.

func (s *tokenService) GetActiveKey(ctx context.Context, purpose, algorithm string) (*models.SigningKey, error) {
	return s.signingKeyService.GetActiveKey(ctx, purpose, algorithm)
}

func (s *tokenService) GetKeyByID(ctx context.Context, kid string) (*models.SigningKey, error) {
	return s.signingKeyService.GetKeyByID(ctx, kid)
}

func (s *tokenService) GetPublicKeys(ctx context.Context, purpose string) ([]models.SigningKey, error) {
	return s.signingKeyService.GetPublicKeys(ctx, purpose)
}

func (s *tokenService) GenerateKey(ctx context.Context, purpose, algorithm string) error {
	return s.signingKeyService.GenerateKey(ctx, purpose, algorithm)
}

func (s *tokenService) RotateKey(ctx context.Context, purpose, algorithm string) error {
	return s.signingKeyService.RotateKey(ctx, purpose, algorithm)
}

func (s *tokenService) RotateKeyWithGrace(ctx context.Context, purpose, algorithm string, gracePeriod time.Duration) error {
	return s.signingKeyService.RotateKeyWithGrace(ctx, purpose, algorithm, gracePeriod)
}
