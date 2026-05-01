package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/rate-limit/types"
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
	now := time.Now().UTC()
	expiresAt := now.Add(window)
	record := &types.RateLimit{
		Key:       key,
		Count:     1,
		ExpiresAt: expiresAt,
	}

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// First, try to get the existing record
		existing := &types.RateLimit{}
		err := tx.NewSelect().
			Model(existing).
			Where("key = ?", key).
			Scan(ctx)

		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// If record exists, update it
		if err == nil {
			// If expired, reset counter; otherwise increment
			if existing.ExpiresAt.Before(now) || existing.ExpiresAt.Equal(now) {
				existing.Count = 1
				existing.ExpiresAt = expiresAt
			} else {
				existing.Count++
			}

			_, err = tx.NewUpdate().
				Model(existing).
				Where("key = ?", key).
				Exec(ctx)
			if err != nil {
				return err
			}
			*record = *existing
			return nil
		}

		// Record doesn't exist, insert it
		_, err = tx.NewInsert().
			Model(record).
			Exec(ctx)
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

func (r *rateLimitRepository) SetRule(ctx context.Context, key string, windowSeconds int, maxRequests int) error {
	record := &types.RateLimitRuleRecord{
		Key:           key,
		WindowSeconds: windowSeconds,
		MaxRequests:   maxRequests,
	}

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		existing := &types.RateLimitRuleRecord{}
		err := tx.NewSelect().
			Model(existing).
			Where("key = ?", key).
			Scan(ctx)

		if err != nil && err != sql.ErrNoRows {
			return err
		}

		if err == sql.ErrNoRows {
			_, err = tx.NewInsert().
				Model(record).
				Exec(ctx)
			return err
		}

		_, err = tx.NewUpdate().
			Model(record).
			Where("key = ?", key).
			Exec(ctx)
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to set rate limit rule: %w", err)
	}

	return nil
}

func (r *rateLimitRepository) GetRule(ctx context.Context, key string) (*types.RateLimitRuleRecord, error) {
	var record types.RateLimitRuleRecord
	err := r.db.NewSelect().
		Model(&record).
		Where("key = ?", key).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get rate limit rule: %w", err)
	}
	return &record, nil
}

func (r *rateLimitRepository) DeleteRule(ctx context.Context, key string) error {
	_, err := r.db.NewDelete().
		Model((*types.RateLimitRuleRecord)(nil)).
		Where("key = ?", key).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete rate limit rule: %w", err)
	}
	return nil
}
