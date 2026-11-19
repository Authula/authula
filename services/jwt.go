package services

import (
	"context"
	"time"

	"github.com/Authula/authula/models"
)

// JWTService is the public contract for the JWT plugin. It validates tokens and
// exposes the plugin-owned signing keys so other plugins can sign OIDC tokens
// without importing the plugin package.
type JWTService interface {
	ValidateToken(ctx context.Context, token string) (*models.Actor, error)
	GetActiveKey(ctx context.Context, purpose, algorithm string) (*models.SigningKey, error)
	GetKeyByID(ctx context.Context, kid string) (*models.SigningKey, error)
	GetPublicKeys(ctx context.Context, purpose string) ([]models.SigningKey, error)
	GenerateKey(ctx context.Context, purpose, algorithm string) error
	RotateKey(ctx context.Context, purpose, algorithm string) error
	RotateKeyWithGrace(ctx context.Context, purpose, algorithm string, gracePeriod time.Duration) error
}
