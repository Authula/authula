package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"

	gobetterauth "github.com/GoBetterAuth/go-better-auth/v2"
	"github.com/GoBetterAuth/go-better-auth/v2/cmd/shared/configloader"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/bootstrap"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/migrationmanager"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/migrations"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

var (
	rootCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Database migration commands for go-better-auth",
		Long:  "Manage core and plugin migrations for the go-better-auth system",
	}
	configPath string
	timeout    int
)

func init() {
	_ = godotenv.Load(".env")

	rootCmd.PersistentFlags().StringVar(&configPath, "config", getEnv("GBA_CONFIG_PATH", "config.toml"), "Path to config file")
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 30, "Timeout in seconds for migration operations (default: 30s)")

	rootCmd.AddCommand(
		newCoreCommand(),
		newPluginsCommand(),
		newStatusCommand(),
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("migrate command failed", "error", err)
		os.Exit(1)
	}
}

func newCoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "core (up|down)",
		Short:     "Manage core database migrations",
		Long:      "Run or rollback core database migrations",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"up", "down"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCore(args[0])
		},
	}
	return cmd
}

func runCore(action string) error {
	runtime, err := bootstrapRuntime(configPath)
	if err != nil {
		return err
	}
	defer runtime.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	switch action {
	case "up":
		runtime.logger.Info("running core migrations")
		return runtime.manager.RunCore(ctx, runtime.config.Database.Provider)
	case "down":
		applied, err := runtime.manager.Migrator().ListApplied(ctx, "")
		if err != nil {
			return fmt.Errorf("check applied migrations: %w", err)
		}
		for _, rec := range applied {
			if rec.PluginID != migrations.CorePluginID {
				return fmt.Errorf("cannot drop core schema while plugin migrations are applied (plugin=%s, version=%s); drop plugin migrations first", rec.PluginID, rec.Version)
			}
		}

		runtime.logger.Info("rolling back core migrations")
		return runtime.manager.DropCore(ctx, runtime.config.Database.Provider)
	default:
		return fmt.Errorf("unknown core action: %s", action)
	}
}

func newPluginsCommand() *cobra.Command {
	var (
		only            string
		except          string
		includeDisabled bool
	)

	cmd := &cobra.Command{
		Use:       "plugins (up|down)",
		Short:     "Manage plugin database migrations",
		Long:      "Run or rollback plugin database migrations with optional filtering",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"up", "down"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlugins(args[0], only, except, includeDisabled)
		},
	}

	cmd.Flags().StringVar(&only, "only", "", "Comma-separated plugin IDs to include")
	cmd.Flags().StringVar(&except, "except", "", "Comma-separated plugin IDs to exclude")
	cmd.Flags().BoolVar(&includeDisabled, "all", false, "Include disabled plugins")

	return cmd
}

func runPlugins(action, only, except string, includeDisabled bool) error {
	runtime, err := bootstrapRuntime(configPath)
	if err != nil {
		return err
	}
	defer runtime.Close()

	selector := buildPluginSelector(runtime.config, only, except, includeDisabled, action == "down")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	switch action {
	case "up":
		appliedAll, err := runtime.manager.Migrator().ListApplied(ctx, "")
		if err != nil {
			return fmt.Errorf("check applied migrations: %w", err)
		}
		coreApplied := false
		for _, rec := range appliedAll {
			if rec.PluginID == migrations.CorePluginID {
				coreApplied = true
				break
			}
		}
		if !coreApplied {
			return fmt.Errorf("core migrations must be applied before running plugin migrations; run 'make migrate-core-up' first")
		}

		runtime.logger.Debug("running plugin migrations")
		return runtime.manager.RunPlugins(ctx, runtime.config.Database.Provider, runtime.plugins, selector)
	case "down":
		runtime.logger.Debug("rolling back plugin migrations")
		return runtime.manager.DropPlugins(ctx, runtime.config.Database.Provider, runtime.plugins, selector)
	default:
		return fmt.Errorf("unknown plugins action: %s", action)
	}
}

func newStatusCommand() *cobra.Command {
	var pluginID string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		Long:  "Display the status of applied migrations, optionally filtered by plugin",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(pluginID)
		},
	}

	cmd.Flags().StringVar(&pluginID, "plugin", "", "Optional plugin ID to filter")

	return cmd
}

func runStatus(pluginID string) error {
	runtime, err := bootstrapRuntime(configPath)
	if err != nil {
		return err
	}
	defer runtime.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	applied, err := runtime.manager.Migrator().ListApplied(ctx, pluginID)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		fmt.Println("No migrations have been applied yet.")
		return nil
	}

	fmt.Printf("% -20s % -32s %s\n", "PLUGIN", "VERSION", "APPLIED AT")
	for _, record := range applied {
		fmt.Printf("% -20s % -32s %s\n", record.PluginID, record.Version, record.AppliedAt.UTC().Format(time.RFC3339))
	}

	return nil
}

type runtimeEnv struct {
	config   *models.Config
	logger   models.Logger
	db       bun.IDB
	migrator *migrations.Migrator
	manager  *migrationmanager.Manager
	plugins  []models.Plugin
}

func (env *runtimeEnv) Close() {
	if env.db != nil {
		if closer, ok := env.db.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
}

func bootstrapRuntime(configPath string) (*runtimeEnv, error) {
	config, exists, err := configloader.Load(configPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		slog.Debug("No config file found, using defaults", "path", configPath)
	}

	logger := gobetterauth.InitLogger(config)
	db, err := gobetterauth.InitDatabase(config, logger, config.Logger.Level)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	migrator, err := migrations.NewMigrator(db, logger)
	if err != nil {
		return nil, fmt.Errorf("init migrator: %w", err)
	}

	plugins := bootstrap.BuildPluginsFromConfig(config)

	return &runtimeEnv{
		config:   config,
		logger:   logger,
		db:       db,
		migrator: migrator,
		manager:  migrationmanager.NewManager(migrator),
		plugins:  plugins,
	}, nil
}

func buildPluginSelector(config *models.Config, only, except string, includeDisabled bool, isDown bool) migrationmanager.PluginSelector {
	selectors := make([]migrationmanager.PluginSelector, 0, 3)

	if !includeDisabled && !isDown {
		selectors = append(selectors, func(plugin models.Plugin) bool {
			enabled := util.IsPluginEnabled(config, plugin.Metadata().ID)
			if !enabled {
				slog.Debug("plugin disabled, skipping", "plugin", plugin.Metadata().ID)
			}
			return enabled
		})
	}

	if trimmed := strings.TrimSpace(only); trimmed != "" {
		selectors = append(selectors, migrationmanager.OnlyPluginIDs(splitCSV(trimmed)...))
	}

	if trimmed := strings.TrimSpace(except); trimmed != "" {
		selectors = append(selectors, migrationmanager.ExceptPluginIDs(splitCSV(trimmed)...))
	}

	return migrationmanager.ComposeSelectors(selectors...)
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
