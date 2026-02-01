package migrations

import (
	"context"
	"embed"
	"fmt"
)

// ProviderFS holds embedded migrations for each database provider.
// Plugins use this to organize their migrations by database type.
type ProviderFS struct {
	SQLite   *embed.FS
	Postgres *embed.FS
	MySQL    *embed.FS
}

// GetMigrations returns the appropriate migration FS for the provider.
// Returns nil if no migrations are defined for the provider.
func (p *ProviderFS) GetMigrations(ctx context.Context, dbProvider string) (*embed.FS, error) {
	switch dbProvider {
	case "sqlite":
		if p.SQLite == nil {
			return nil, nil
		}
		return p.SQLite, nil
	case "postgres":
		if p.Postgres == nil {
			return nil, nil
		}
		return p.Postgres, nil
	case "mysql":
		if p.MySQL == nil {
			return nil, nil
		}
		return p.MySQL, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", dbProvider)
	}
}

// BaseMigrationProvider is a helper struct that plugins can embed to automatically
// implement the PluginWithMigrations interface. Plugins that always need database
// tables (like JWT plugin) can embed this and set FS to their MigrationFS variable.
//
// Example usage in a plugin:
//
//	type MyPlugin struct {
//		pluginmigrations.BaseMigrationProvider
//		// ... other fields
//	}
//
//	func New(config Config) *MyPlugin {
//		return &MyPlugin{
//			BaseMigrationProvider: pluginmigrations.BaseMigrationProvider{
//				FS: MigrationFS,
//			},
//			// ... other init
//		}
//	}
type BaseMigrationProvider struct {
	FS *ProviderFS
}

// Migrations implements the PluginWithMigrations interface.
// Returns the appropriate migration FS for the provider.
func (b *BaseMigrationProvider) Migrations(ctx context.Context, dbProvider string) (*embed.FS, error) {
	if b.FS == nil {
		return nil, nil
	}
	return b.FS.GetMigrations(ctx, dbProvider)
}
