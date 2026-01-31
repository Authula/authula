package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/config-manager/types"
)

type bunConfigManagerRepository struct {
	db bun.IDB
}

func NewBunConfigManagerRepository(db bun.IDB) AuthSettingsRepository {
	return &bunConfigManagerRepository{db: db}
}

// GetByKey retrieves a config setting by its key
func (r *bunConfigManagerRepository) GetByKey(ctx context.Context, key string) (*types.AuthSettings, error) {
	var setting types.AuthSettings
	err := r.db.NewSelect().
		Model(&setting).
		Where("key = ?", key).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get config setting: %w", err)
	}
	return &setting, nil
}

// Save saves or updates a config setting
func (r *bunConfigManagerRepository) Save(ctx context.Context, setting *types.AuthSettings) error {
	_, err := r.db.NewInsert().
		Model(setting).
		On("CONFLICT (key) DO UPDATE").
		Set("value = EXCLUDED.value").
		Set("config_version = EXCLUDED.config_version + 1").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to save config setting: %w", err)
	}
	return nil
}

// Upsert performs an "upsert" operation - insert or update on conflict
// This uses SQL's ON CONFLICT clause for efficient upsert semantics
func (r *bunConfigManagerRepository) Upsert(ctx context.Context, setting *types.AuthSettings) error {
	_, err := r.db.NewInsert().
		Model(setting).
		On("CONFLICT (key) DO UPDATE").
		Set("value = EXCLUDED.value").
		Set("config_version = EXCLUDED.config_version + 1").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to upsert config setting: %w", err)
	}
	return nil
}

// Delete removes a config setting by key
func (r *bunConfigManagerRepository) Delete(ctx context.Context, key string) error {
	_, err := r.db.NewDelete().
		Model((*types.AuthSettings)(nil)).
		Where("key = ?", key).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete config setting: %w", err)
	}
	return nil
}
