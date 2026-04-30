package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Authula/authula/plugins/rate-limit/types"
	"github.com/uptrace/bun"
)

type rateLimitRepository struct {
	db bun.IDB
}

func NewRateLimitRepository(db bun.IDB) RateLimitRepository {
	return &rateLimitRepository{db: db}
}

func (r *rateLimitRepository) GetByKey(ctx context.Context, key string) (*types.RateLimit, error) {
	var record types.RateLimit

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

func (r *rateLimitRepository) UpdateOrCreate(ctx context.Context, key string, window time.Duration) (*types.RateLimit, error) {
	now := time.Now()
	expiresAt := now.Add(window)
	record := &types.RateLimit{
		Key:       key,
		Count:     1,
		ExpiresAt: expiresAt,
	}

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		result, err := tx.NewUpdate().
			Model(record).
			TableExpr("rate_limits AS rl").
			Set("count = CASE WHEN rl.expires_at <= ? THEN 1 ELSE rl.count + 1 END", now).
			Set("expires_at = CASE WHEN rl.expires_at <= ? THEN ? ELSE rl.expires_at END", now, expiresAt).
			Where("rl.key = ?", key).
			Exec(ctx)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			_, err = tx.NewInsert().
				Model(record).
				Exec(ctx)
			if err != nil {
				return err
			}
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

func (r *rateLimitRepository) CleanupExpired(ctx context.Context, now time.Time) error {
	_, err := r.db.NewDelete().
		Model((*types.RateLimit)(nil)).
		Where("expires_at < ?", now).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to cleanup expired records: %w", err)
	}

	return nil
}
