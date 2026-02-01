package internal

import (
	"context"
	"embed"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// RunMigrationsFromFS runs migrations from an embedded filesystem.
func RunMigrationsFromFS(ctx context.Context, db bun.IDB, logger models.Logger, name string, fs *embed.FS) error {
	if fs == nil {
		logger.Debug("no migrations to run", "component", name)
		return nil
	}

	migrations := migrate.NewMigrations()
	if err := migrations.Discover(*fs); err != nil {
		logger.Debug("no migrations found", "component", name)
		return nil
	}

	bunDB, ok := db.(*bun.DB)
	if !ok {
		logger.Debug("database is not *bun.DB, skipping migrations", "component", name)
		return nil
	}

	m := migrate.NewMigrator(bunDB, migrations)

	if err := m.Init(ctx); err != nil {
		return fmt.Errorf("failed to init migrations table for %s: %w", name, err)
	}

	if _, err := m.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to run migrations for %s: %w", name, err)
	}

	logger.Debug("migrations completed", "component", name)
	return nil
}

// DropMigrationsFromFS rolls back migrations from an embedded filesystem.
func DropMigrationsFromFS(ctx context.Context, db bun.IDB, logger models.Logger, name string, fs *embed.FS) error {
	if fs == nil {
		logger.Debug("no migrations to drop", "component", name)
		return nil
	}

	migrations := migrate.NewMigrations()
	if err := migrations.Discover(*fs); err != nil {
		logger.Debug("no migrations to drop", "component", name)
		return nil
	}

	bunDB, ok := db.(*bun.DB)
	if !ok {
		logger.Debug("database is not *bun.DB, skipping drop migrations", "component", name)
		return nil
	}

	m := migrate.NewMigrator(bunDB, migrations)

	if err := m.Init(ctx); err != nil {
		return fmt.Errorf("failed to init migrations table for %s: %w", name, err)
	}

	for {
		group, err := m.Rollback(ctx)
		if err != nil {
			return fmt.Errorf("failed to rollback migrations for %s: %w", name, err)
		}
		if group == nil || len(group.Migrations) == 0 {
			break
		}
	}

	logger.Debug("migrations dropped", "component", name)
	return nil
}

// DetectProvider detects the database provider from a bun.IDB instance.
func DetectProvider(db bun.IDB) string {
	if bunDB, ok := db.(*bun.DB); ok {
		dialectName := bunDB.Dialect().Name().String()
		switch dialectName {
		case "pg":
			return "postgres"
		case "mysql":
			return "mysql"
		case "sqlite":
			return "sqlite"
		}
	}

	// Fallback: return empty string, caller should handle
	return ""
}

// RunPluginMigrations runs migrations for all plugins that implement PluginWithMigrations.
func RunPluginMigrations(ctx context.Context, db bun.IDB, logger models.Logger, plugins []models.Plugin) error {
	provider := DetectProvider(db)
	if provider == "" {
		logger.Warn("could not detect database provider, skipping plugin migrations")
		return nil
	}

	for _, plugin := range plugins {
		pluginID := plugin.Metadata().ID

		migrator, ok := plugin.(models.PluginWithMigrations)
		if !ok {
			continue
		}

		fs, err := migrator.Migrations(ctx, provider)
		if err != nil {
			logger.Error("failed to get migrations for plugin",
				"plugin", pluginID,
				"error", err)
			continue
		}

		if err := RunMigrationsFromFS(ctx, db, logger, pluginID, fs); err != nil {
			logger.Error("failed to run migrations for plugin",
				"plugin", pluginID,
				"error", err)
			continue
		}

		if fs != nil {
			logger.Info("plugin migrations completed", "plugin", pluginID)
		}
	}

	return nil
}

// DropPluginMigrations rolls back migrations for all plugins that implement PluginWithMigrations.
func DropPluginMigrations(ctx context.Context, db bun.IDB, logger models.Logger, plugins []models.Plugin) error {
	provider := DetectProvider(db)
	if provider == "" {
		logger.Warn("could not detect database provider, skipping drop plugin migrations")
		return nil
	}

	for _, plugin := range plugins {
		pluginID := plugin.Metadata().ID

		migrator, ok := plugin.(models.PluginWithMigrations)
		if !ok {
			continue
		}

		fs, err := migrator.Migrations(ctx, provider)
		if err != nil {
			logger.Error("failed to get migrations for plugin",
				"plugin", pluginID,
				"error", err)
			continue
		}

		if err := DropMigrationsFromFS(ctx, db, logger, pluginID, fs); err != nil {
			logger.Error("failed to drop migrations for plugin",
				"plugin", pluginID,
				"error", err)
			continue
		}

		if fs != nil {
			logger.Info("plugin migrations dropped", "plugin", pluginID)
		}
	}

	return nil
}
