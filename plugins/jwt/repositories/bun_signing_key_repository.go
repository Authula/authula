package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/models"
)

type bunSigningKeyRepository struct {
	db bun.IDB
}

// NewBunSigningKeyRepository creates a Bun-backed signing key repository.
func NewBunSigningKeyRepository(db bun.IDB) SigningKeyRepository {
	return &bunSigningKeyRepository{db: db}
}

func (r *bunSigningKeyRepository) GetActiveKey(ctx context.Context, purpose, algorithm string) (*models.SigningKey, error) {
	key := new(models.SigningKey)
	err := r.db.NewSelect().
		Model(key).
		Where("key_purpose = ? AND algorithm = ? AND is_active = ?", purpose, algorithm, true).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now().UTC()).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrSigningKeyNotFound
		}
		return nil, fmt.Errorf("failed to get active signing key: %w", err)
	}
	return key, nil
}

func (r *bunSigningKeyRepository) GetKeyByID(ctx context.Context, keyID string) (*models.SigningKey, error) {
	key := new(models.SigningKey)
	err := r.db.NewSelect().Model(key).Where("key_id = ?", keyID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrSigningKeyNotFound
		}
		return nil, fmt.Errorf("failed to get signing key by key id: %w", err)
	}
	return key, nil
}

func (r *bunSigningKeyRepository) GetPublicKeys(ctx context.Context, purpose string, now time.Time) ([]*models.SigningKey, error) {
	keys := make([]*models.SigningKey, 0)
	err := r.db.NewSelect().
		Model(&keys).
		Where("key_purpose = ?", purpose).
		Where("(expires_at IS NULL OR expires_at > ?)", now.UTC()).
		OrderExpr("created_at ASC, id ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list signing keys: %w", err)
	}
	return keys, nil
}

func (r *bunSigningKeyRepository) Create(ctx context.Context, key *models.SigningKey) error {
	if _, err := r.db.NewInsert().Model(key).Exec(ctx); err != nil {
		return fmt.Errorf("failed to store signing key: %w", err)
	}
	return nil
}

// Rotate deactivates every currently active key for the purpose and inserts the
// new active key in a single transaction. The partial unique index on
// (key_purpose, algorithm) for active rows makes concurrent generation safe.
func (r *bunSigningKeyRepository) Rotate(ctx context.Context, purpose string, key *models.SigningKey, expiresAt time.Time) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().
			Model((*models.SigningKey)(nil)).
			Set("is_active = ?", false).
			Set("expires_at = ?", expiresAt.UTC()).
			Where("key_purpose = ? AND is_active = ?", purpose, true).
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to deactivate old signing keys: %w", err)
		}

		if _, err := tx.NewInsert().Model(key).Exec(ctx); err != nil {
			return fmt.Errorf("failed to store signing key: %w", err)
		}
		return nil
	})
}

func (r *bunSigningKeyRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*models.SigningKey)(nil)).
		Where("expires_at IS NOT NULL AND expires_at < ?", before.UTC()).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired signing keys: %w", err)
	}
	return result.RowsAffected()
}
