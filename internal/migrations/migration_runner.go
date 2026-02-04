package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// MigrationOperation represents the type of migration operation
type MigrationOperation int

const (
	MigrateUpOperation MigrationOperation = iota
	MigrateDownOperation
)

// migrationRunner encapsulates the common migration execution logic
type migrationRunner struct {
	logger   models.Logger
	provider *goose.Provider
}

// newMigrationRunner creates a new migration runner with the given parameters
func newMigrationRunner(
	db bun.IDB,
	sqlFs embed.FS,
	migrationsDir string,
	provider string,
	verbose bool,
) (*migrationRunner, error) {
	subFs, err := fs.Sub(sqlFs, migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create sub filesystem: %w", err)
	}

	dialect, err := getDialect(provider)
	if err != nil {
		return nil, err
	}

	sqlDB := getSQLDB(db)
	if sqlDB == nil {
		return nil, fmt.Errorf("failed to get *sql.DB from bun.IDB")
	}

	providerInstance, err := goose.NewProvider(dialect, sqlDB, subFs, goose.WithVerbose(verbose))
	if err != nil {
		return nil, fmt.Errorf("failed to create goose provider: %w", err)
	}

	return &migrationRunner{
		provider: providerInstance,
	}, nil
}

// run executes the migration operation and logs results
func (r *migrationRunner) run(ctx context.Context, op MigrationOperation, logLevel string) error {
	var results []*goose.MigrationResult
	var err error

	switch op {
	case MigrateUpOperation:
		results, err = r.provider.Up(ctx)
		if err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		for _, result := range results {
			r.logMigration(result, logLevel, "Migrated")
		}
	case MigrateDownOperation:
		results, err = r.provider.DownTo(ctx, 0)
		if err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}
		for _, result := range results {
			r.logMigration(result, "info", "Rolled back")
		}
	}

	return nil
}

// logMigration logs a migration result with the appropriate level
func (r *migrationRunner) logMigration(result *goose.MigrationResult, level string, action string) {
	msg := fmt.Sprintf("%s: %s (%s)", action, result.Source.Path, result.Duration)
	if r.logger != nil {
		switch level {
		case "debug":
			r.logger.Debug(msg)
		case "info":
			r.logger.Info(msg)
		default:
			r.logger.Info(msg)
		}
	}
}

// RunCoreMigrations runs the core database migrations
func RunCoreMigrations(
	ctx context.Context,
	logger models.Logger,
	logLevel string,
	provider string,
	db bun.IDB,
) error {
	sqlFs, err := GetMigrations(ctx, provider)
	if err != nil {
		return err
	}

	runner, err := newMigrationRunner(db, *sqlFs, "migrations/"+provider, provider, logLevel == "debug")
	if err != nil {
		return err
	}
	runner.logger = logger

	return runner.run(ctx, MigrateUpOperation, logLevel)
}

// DropCoreMigrations rolls back all core database migrations
func DropCoreMigrations(
	ctx context.Context,
	logger models.Logger,
	logLevel string,
	provider string,
	db bun.IDB,
) error {
	sqlFs, err := GetMigrations(ctx, provider)
	if err != nil {
		return err
	}

	runner, err := newMigrationRunner(db, *sqlFs, "migrations/"+provider, provider, logLevel == "debug")
	if err != nil {
		return err
	}
	runner.logger = logger

	return runner.run(ctx, MigrateDownOperation, logLevel)
}

// RunMigrations runs migrations from an embedded filesystem for a specific plugin
func RunMigrations(
	ctx context.Context,
	logger models.Logger,
	provider string,
	db bun.IDB,
	sqlFs embed.FS,
	migrationsDir string,
) error {
	runner, err := newMigrationRunner(db, sqlFs, migrationsDir, provider, false)
	if err != nil {
		return err
	}
	runner.logger = logger

	return runner.run(ctx, MigrateUpOperation, "debug")
}

// DropMigrations rolls back migrations from an embedded filesystem for a specific plugin
func DropMigrations(
	ctx context.Context,
	logger models.Logger,
	provider string,
	db bun.IDB,
	sqlFs embed.FS,
	migrationsDir string,
) error {
	runner, err := newMigrationRunner(db, sqlFs, migrationsDir, provider, false)
	if err != nil {
		return err
	}
	runner.logger = logger

	return runner.run(ctx, MigrateDownOperation, "info")
}

// getDialect maps provider string to goose dialect
func getDialect(provider string) (database.Dialect, error) {
	switch provider {
	case "postgres":
		return goose.DialectPostgres, nil
	case "mysql":
		return goose.DialectMySQL, nil
	case "sqlite":
		return goose.DialectSQLite3, nil
	default:
		return "", fmt.Errorf("unsupported database provider: %s", provider)
	}
}

// getSQLDB extracts *sql.DB from bun.IDB
func getSQLDB(db bun.IDB) *sql.DB {
	switch d := db.(type) {
	case *bun.DB:
		return d.DB
	default:
		return nil
	}
}
