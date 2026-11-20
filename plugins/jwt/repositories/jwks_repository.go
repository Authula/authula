package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/jwt/types"
)

// JWKSRepository provides data access for JWKS (JSON Web Key Set) records
type JWKSRepository interface {
	// GetJWKSKeys retrieves all valid (non-expired) JWKS keys
	GetJWKSKeys(ctx context.Context) ([]*types.JWKS, error)

	// StoreJWKSKey saves a JWKS key
	StoreJWKSKey(ctx context.Context, key *types.JWKS) error

	// MarkKeyExpired marks a JWKS key as expired
	MarkKeyExpired(ctx context.Context, id string, expiresAt time.Time) error

	// PurgeExpiredKeys removes expired JWKS keys
	PurgeExpiredKeys(ctx context.Context) error
}

type bunJWKSRepository struct {
	db bun.IDB
}

// NewBunJWKSRepository creates a new Bun-based JWKS repository
func NewBunJWKSRepository(db bun.IDB) JWKSRepository {
	return &bunJWKSRepository{db: db}
}

func (r *bunJWKSRepository) GetJWKSKeys(ctx context.Context) ([]*types.JWKS, error) {
	var jwksKeys []*types.JWKS
	err := r.db.NewSelect().
		Model(&jwksKeys).
		Where("expires_at IS NULL OR expires_at > ?", time.Now()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get jwks keys: %w", err)
	}
	return jwksKeys, nil
}

func (r *bunJWKSRepository) StoreJWKSKey(ctx context.Context, key *types.JWKS) error {
	_, err := r.db.NewInsert().
		Model(key).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to store jwks key: %w", err)
	}
	return nil
}

func (r *bunJWKSRepository) MarkKeyExpired(ctx context.Context, id string, expiresAt time.Time) error {
	_, err := r.db.NewUpdate().
		Model((*types.JWKS)(nil)).
		Set("expires_at = ?", expiresAt).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to mark key as expired: %w", err)
	}
	return nil
}

const jwksPurgeRetention = 24 * time.Hour

func (r *bunJWKSRepository) PurgeExpiredKeys(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*types.JWKS)(nil)).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now().Add(-jwksPurgeRetention)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to purge expired keys: %w", err)
	}
	return nil
}
