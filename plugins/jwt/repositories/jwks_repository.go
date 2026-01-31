package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/types"
)

// JWKSRepository provides data access for JWKS (JSON Web Key Set) records
type JWKSRepository interface {
	// GetJWKSKeys retrieves all valid (non-expired) JWKS keys
	GetJWKSKeys(ctx context.Context) ([]*types.JWKS, error)

	// GetJWKSKeyByID retrieves a specific JWKS key by its ID
	GetJWKSKeyByID(ctx context.Context, id string) (*types.JWKS, error)

	// StoreJWKSKey saves a JWKS key
	StoreJWKSKey(ctx context.Context, key *types.JWKS) error

	// UpdateJWKSKey updates an existing JWKS key
	UpdateJWKSKey(ctx context.Context, key *types.JWKS) error

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

func (r *bunJWKSRepository) GetJWKSKeyByID(ctx context.Context, id string) (*types.JWKS, error) {
	key := &types.JWKS{}
	err := r.db.NewSelect().
		Model(key).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get jwks key by id: %w", err)
	}
	return key, nil
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

func (r *bunJWKSRepository) UpdateJWKSKey(ctx context.Context, key *types.JWKS) error {
	_, err := r.db.NewUpdate().
		Model(key).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update jwks key: %w", err)
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

func (r *bunJWKSRepository) PurgeExpiredKeys(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*types.JWKS)(nil)).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now().Add(-24*time.Hour)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to purge expired keys: %w", err)
	}
	return nil
}
