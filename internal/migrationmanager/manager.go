package migrationmanager

import (
	"context"
	"fmt"
	"strings"

	migrationsmodule "github.com/GoBetterAuth/go-better-auth/v2/migrations"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// PluginSelector determines whether a plugin should participate in a migration operation.
type PluginSelector func(plugin models.Plugin) bool

// Manager centralizes migration orchestration for core and plugin schemas.
type Manager struct {
	migrator *migrationsmodule.Migrator
}

// NewManager constructs a Manager backed by the provided migrator.
func NewManager(migrator *migrationsmodule.Migrator) *Manager {
	return &Manager{migrator: migrator}
}

// Migrator exposes the underlying migrator for advanced use cases.
func (m *Manager) Migrator() *migrationsmodule.Migrator {
	if m == nil {
		return nil
	}
	return m.migrator
}

// RunCore executes the core migrations for the configured provider.
func (m *Manager) RunCore(ctx context.Context, provider string) error {
	if err := m.requireMigrator(); err != nil {
		return err
	}

	set, err := migrationsmodule.CoreMigrationSet(provider)
	if err != nil {
		return err
	}

	return m.migrator.Migrate(ctx, []migrationsmodule.MigrationSet{set})
}

// DropCore rolls back all core migrations.
func (m *Manager) DropCore(ctx context.Context, provider string) error {
	if err := m.requireMigrator(); err != nil {
		return err
	}

	set, err := migrationsmodule.CoreMigrationSet(provider)
	if err != nil {
		return err
	}

	return m.migrator.RollbackAll(ctx, []migrationsmodule.MigrationSet{set})
}

// RunPlugins executes migrations for all selected plugins.
func (m *Manager) RunPlugins(ctx context.Context, provider string, plugins []models.Plugin, selector PluginSelector) error {
	if err := m.requireMigrator(); err != nil {
		return err
	}

	sets := m.PlanPlugins(provider, plugins, selector)
	if len(sets) == 0 {
		return nil
	}

	return m.migrator.Migrate(ctx, sets)
}

// DropPlugins rolls back migrations for all selected plugins.
func (m *Manager) DropPlugins(ctx context.Context, provider string, plugins []models.Plugin, selector PluginSelector) error {
	if err := m.requireMigrator(); err != nil {
		return err
	}

	sets := m.PlanPlugins(provider, plugins, selector)
	if len(sets) == 0 {
		return nil
	}

	return m.migrator.RollbackAll(ctx, sets)
}

// PlanPlugins builds migration sets for plugins matching the selector.
func (m *Manager) PlanPlugins(provider string, plugins []models.Plugin, selector PluginSelector) []migrationsmodule.MigrationSet {
	if len(plugins) == 0 {
		return nil
	}

	sets := make([]migrationsmodule.MigrationSet, 0, len(plugins))
	for _, plugin := range plugins {
		if selector != nil && !selector(plugin) {
			continue
		}

		migratable, ok := plugin.(models.PluginWithMigrations)
		if !ok {
			continue
		}

		migrations := migratable.Migrations(provider)
		if len(migrations) == 0 {
			continue
		}

		sets = append(sets, migrationsmodule.MigrationSet{
			PluginID:   plugin.Metadata().ID,
			DependsOn:  migratable.DependsOn(),
			Migrations: migrations,
		})
	}

	return sets
}

// OnlyPluginIDs returns a selector that matches specific plugin IDs.
func OnlyPluginIDs(ids ...string) PluginSelector {
	normalized := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		normalized[id] = struct{}{}
	}

	if len(normalized) == 0 {
		return nil
	}

	return func(plugin models.Plugin) bool {
		_, ok := normalized[plugin.Metadata().ID]
		return ok
	}
}

// ExceptPluginIDs returns a selector that excludes the provided plugin IDs.
func ExceptPluginIDs(ids ...string) PluginSelector {
	normalized := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		normalized[id] = struct{}{}
	}

	if len(normalized) == 0 {
		return nil
	}

	return func(plugin models.Plugin) bool {
		_, blocked := normalized[plugin.Metadata().ID]
		return !blocked
	}
}

// ComposeSelectors combines multiple selectors into a single predicate.
func ComposeSelectors(selectors ...PluginSelector) PluginSelector {
	cleaned := make([]PluginSelector, 0, len(selectors))
	for _, selector := range selectors {
		if selector == nil {
			continue
		}
		cleaned = append(cleaned, selector)
	}

	if len(cleaned) == 0 {
		return nil
	}

	return func(plugin models.Plugin) bool {
		for _, selector := range cleaned {
			if !selector(plugin) {
				return false
			}
		}
		return true
	}
}

func (m *Manager) requireMigrator() error {
	if m == nil || m.migrator == nil {
		return fmt.Errorf("migrator not initialized")
	}
	return nil
}
