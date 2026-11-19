package repositories

import (
	"context"
	"time"

	"github.com/Authula/authula/models"
)

// SigningKeyRepository stores signing keys. It only performs data access; key
// generation and rotation policy live in the signing key service.
type SigningKeyRepository interface {
	GetActiveKey(ctx context.Context, purpose, algorithm string) (*models.SigningKey, error)
	GetKeyByID(ctx context.Context, keyID string) (*models.SigningKey, error)
	GetPublicKeys(ctx context.Context, purpose string, now time.Time) ([]*models.SigningKey, error)
	Create(ctx context.Context, key *models.SigningKey) error
	Rotate(ctx context.Context, purpose string, key *models.SigningKey, expiresAt time.Time) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}
