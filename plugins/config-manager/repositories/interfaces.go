package repositories

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/plugins/config-manager/types"
)

// AuthSettingsRepository defines the interface for auth settings persistence
type AuthSettingsRepository interface {
	// GetByKey retrieves an auth setting by its key
	GetByKey(ctx context.Context, key string) (*types.AuthSettings, error)

	// Save saves or updates an auth setting
	Save(ctx context.Context, setting *types.AuthSettings) error

	// Upsert performs an "upsert" operation - insert or update on conflict
	Upsert(ctx context.Context, setting *types.AuthSettings) error

	// Delete removes an auth setting by key
	Delete(ctx context.Context, key string) error
}
