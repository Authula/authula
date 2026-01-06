package ratelimit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

type rateLimitRepositoryImpl struct {
	db bun.IDB
}

// NewRateLimitRepository creates a new RateLimitRepository with Bun backend
func NewRateLimitRepository(db bun.IDB) RateLimitRepository {
	return &rateLimitRepositoryImpl{db: db}
}

// GetByKey retrieves a rate limit record by its key
func (r *rateLimitRepositoryImpl) GetByKey(ctx context.Context, key string) (*RateLimit, error) {
	var record RateLimit

	err := r.db.NewSelect().
		Model(&record).
		Where("key = ?", key).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get rate limit record: %w", err)
	}

	return &record, nil
}

// UpdateOrCreate updates an existing rate limit record or creates a new one
func (r *rateLimitRepositoryImpl) UpdateOrCreate(ctx context.Context, key string, window time.Duration) (*RateLimit, error) {
	now := time.Now()
	expiresAt := now.Add(window)
	record := &RateLimit{
		Key:       key,
		Count:     1,
		ExpiresAt: expiresAt,
	}

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(record).
			On("CONFLICT (key) DO UPDATE").
			Set("count = CASE WHEN rate_limit.expires_at <= ? THEN 1 ELSE rate_limit.count + 1 END", now).
			Set("expires_at = CASE WHEN rate_limit.expires_at <= ? THEN EXCLUDED.expires_at ELSE rate_limit.expires_at END", now).
			Exec(ctx)
		if err != nil {
			return err
		}

		err = tx.NewSelect().
			Model(record).
			Where("key = ?", key).
			Scan(ctx)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("ratelimit upsert failed: %w", err)
	}

	return record, nil
}

// CleanupExpired removes expired rate limit records
func (r *rateLimitRepositoryImpl) CleanupExpired(ctx context.Context, now time.Time) error {
	_, err := r.db.NewDelete().
		Model((*RateLimit)(nil)).
		Where("expires_at < ?", now).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to cleanup expired records: %w", err)
	}

	return nil
}
