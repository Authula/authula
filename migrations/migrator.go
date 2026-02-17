package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

// Migration represents a single schema change with reversible logic.
type Migration struct {
	Version string
	Up      func(ctx context.Context, tx bun.Tx) error
	Down    func(ctx context.Context, tx bun.Tx) error
}

// Logger is a minimal logging interface used by the migrator.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// MigrationSet groups migrations under a plugin identifier with optional dependencies.
type MigrationSet struct {
	PluginID   string
	DependsOn  []string
	Migrations []Migration
}

// Migrator orchestrates plugin-aware migrations backed by Bun transactions.
type Migrator struct {
	db      *bun.DB
	logger  Logger
	table   string
	dialect string
}

// DefaultSchemaTable is the shared table storing migration metadata.
const (
	DefaultSchemaTable   = "auth_schema_migrations"
	schemaMigrationAlias = "schema_migration"
)

// MigratorOption configures optional Migrator behavior.
type MigratorOption func(*Migrator)

// WithTableName overrides the schema migrations table name.
func WithTableName(name string) MigratorOption {
	return func(m *Migrator) {
		if strings.TrimSpace(name) != "" {
			m.table = name
		}
	}
}

// NewMigrator constructs a Migrator backed by the provided Bun database.
func NewMigrator(db bun.IDB, logger Logger, opts ...MigratorOption) (*Migrator, error) {
	bunDB, ok := db.(*bun.DB)
	if !ok {
		return nil, errors.New("migrator requires *bun.DB instance")
	}

	m := &Migrator{
		db:      bunDB,
		logger:  logger,
		table:   DefaultSchemaTable,
		dialect: bunDB.Dialect().Name().String(),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

type schemaMigration struct {
	PluginID  string    `bun:"plugin_id,pk"`
	Version   string    `bun:",pk"`
	AppliedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

// AppliedMigration represents a persisted migration entry for introspection.
type AppliedMigration struct {
	PluginID  string
	Version   string
	AppliedAt time.Time
}

func (m *Migrator) ensureTable(ctx context.Context) error {
	if err := m.createSchemaTable(ctx); err != nil {
		return err
	}
	return m.ensureAppliedAtIndex(ctx)
}

func (m *Migrator) createSchemaTable(ctx context.Context) error {
	_, err := m.db.NewCreateTable().
		Model((*schemaMigration)(nil)).
		ModelTableExpr(m.table).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("create %s table: %w", m.table, err)
	}
	return nil
}

// Migrate applies all pending migrations respecting plugin dependencies.
func (m *Migrator) Migrate(ctx context.Context, sets []MigrationSet) error {
	if len(sets) == 0 {
		return nil
	}

	if err := m.ensureTable(ctx); err != nil {
		return err
	}

	ordered, err := sortMigrationSets(sets)
	if err != nil {
		return err
	}

	for _, set := range ordered {
		applied, err := m.appliedVersions(ctx, set.PluginID)
		if err != nil {
			return err
		}

		seen := make(map[string]struct{}, len(set.Migrations))
		for _, migration := range set.Migrations {
			if err := validateMigration(set.PluginID, migration); err != nil {
				return err
			}
			if _, exists := seen[migration.Version]; exists {
				return fmt.Errorf("duplicate migration version %s for plugin %s", migration.Version, set.PluginID)
			}
			seen[migration.Version] = struct{}{}

			if _, alreadyApplied := applied[migration.Version]; alreadyApplied {
				continue
			}

			if err := m.applyUp(ctx, set.PluginID, migration); err != nil {
				return err
			}
		}
	}

	return nil
}

// RollbackAll executes Down migrations for the provided sets in reverse dependency order.
func (m *Migrator) RollbackAll(ctx context.Context, sets []MigrationSet) error {
	if len(sets) == 0 {
		return nil
	}

	if err := m.ensureTable(ctx); err != nil {
		return err
	}

	ordered, err := sortMigrationSets(sets)
	if err != nil {
		return err
	}

	for i := len(ordered) - 1; i >= 0; i-- {
		set := ordered[i]
		applied, err := m.appliedVersions(ctx, set.PluginID)
		if err != nil {
			return err
		}

		if len(applied) == 0 {
			continue
		}

		for j := len(set.Migrations) - 1; j >= 0; j-- {
			migration := set.Migrations[j]
			if _, ok := applied[migration.Version]; !ok {
				continue
			}

			if err := validateMigration(set.PluginID, migration); err != nil {
				return err
			}

			if err := m.applyDown(ctx, set.PluginID, migration); err != nil {
				return err
			}
		}
	}

	return nil
}

// RollbackLast rolls back the most recent migration applied for a plugin in the provided set.
func (m *Migrator) RollbackLast(ctx context.Context, set MigrationSet) error {
	if err := m.ensureTable(ctx); err != nil {
		return err
	}

	if err := validateSet(set); err != nil {
		return err
	}

	latest, err := m.latestApplied(ctx, set.PluginID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	migration, ok := findMigration(set, latest.Version)
	if !ok {
		return fmt.Errorf("migration definition for %s:%s not found", set.PluginID, latest.Version)
	}

	return m.applyDown(ctx, set.PluginID, migration)
}

func (m *Migrator) applyUp(ctx context.Context, pluginID string, migration Migration) error {
	start := time.Now()
	err := m.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if migration.Up == nil {
			return fmt.Errorf("migration %s:%s missing Up function", pluginID, migration.Version)
		}

		if err := migration.Up(ctx, tx); err != nil {
			return fmt.Errorf("migration %s:%s up failed: %w", pluginID, migration.Version, err)
		}

		entry := &schemaMigration{
			PluginID:  pluginID,
			Version:   migration.Version,
			AppliedAt: time.Now().UTC(),
		}

		tableExpr, args := m.tableExpr(true)
		_, err := tx.NewInsert().
			Model(entry).
			ModelTableExpr(tableExpr, args...).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("record migration %s:%s: %w", pluginID, migration.Version, err)
		}

		return nil
	})

	if err == nil && m.logger != nil {
		m.logger.Info("migration applied", "plugin", pluginID, "version", migration.Version, "duration", time.Since(start))
	}

	return err
}

func (m *Migrator) applyDown(ctx context.Context, pluginID string, migration Migration) error {
	start := time.Now()
	err := m.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if migration.Down == nil {
			return fmt.Errorf("migration %s:%s missing Down function", pluginID, migration.Version)
		}

		if err := migration.Down(ctx, tx); err != nil {
			return fmt.Errorf("migration %s:%s down failed: %w", pluginID, migration.Version, err)
		}

		tableExpr, args := m.tableExpr(true)
		_, err := tx.NewDelete().
			Model((*schemaMigration)(nil)).
			ModelTableExpr(tableExpr, args...).
			Where("plugin_id = ?", pluginID).
			Where("version = ?", migration.Version).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("remove migration record %s:%s: %w", pluginID, migration.Version, err)
		}

		return nil
	})

	if err == nil && m.logger != nil {
		m.logger.Info("migration rolled back", "plugin", pluginID, "version", migration.Version, "duration", time.Since(start))
	}

	return err
}

func (m *Migrator) appliedVersions(ctx context.Context, pluginID string) (map[string]schemaMigration, error) {
	if pluginID == "" {
		return nil, errors.New("plugin ID cannot be empty")
	}

	var records []schemaMigration
	tableExpr, args := m.tableExpr(true)
	err := m.db.NewSelect().
		Model(&records).
		ModelTableExpr(tableExpr, args...).
		Where("plugin_id = ?", pluginID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select migrations for %s: %w", pluginID, err)
	}

	result := make(map[string]schemaMigration, len(records))
	for _, record := range records {
		result[record.Version] = record
	}

	return result, nil
}

func (m *Migrator) latestApplied(ctx context.Context, pluginID string) (*schemaMigration, error) {
	var record schemaMigration
	tableExpr, args := m.tableExpr(true)
	err := m.db.NewSelect().
		Model(&record).
		ModelTableExpr(tableExpr, args...).
		Where("plugin_id = ?", pluginID).
		OrderExpr("applied_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListApplied returns applied migrations for all plugins or a specific plugin when pluginID is provided.
func (m *Migrator) ListApplied(ctx context.Context, pluginID string) ([]schemaMigration, error) {
	if err := m.ensureTable(ctx); err != nil {
		return nil, err
	}

	var records []schemaMigration
	tableExpr, args := m.tableExpr(true)
	query := m.db.NewSelect().
		Model(&records).
		ModelTableExpr(tableExpr, args...).
		OrderExpr("plugin_id ASC, applied_at ASC")

	if strings.TrimSpace(pluginID) != "" {
		query = query.Where("plugin_id = ?", pluginID)
	}

	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}

	result := make([]schemaMigration, len(records))
	for i, record := range records {
		result[i] = schemaMigration{
			PluginID:  record.PluginID,
			Version:   record.Version,
			AppliedAt: record.AppliedAt,
		}
	}

	return result, nil
}

func sortMigrationSets(sets []MigrationSet) ([]MigrationSet, error) {
	if len(sets) == 0 {
		return nil, nil
	}

	indices := make(map[string]int, len(sets))
	indegree := make(map[string]int, len(sets))
	graph := make(map[string][]string, len(sets))

	for i := range sets {
		if err := validateSet(sets[i]); err != nil {
			return nil, err
		}
		id := sets[i].PluginID
		if _, exists := indices[id]; exists {
			return nil, fmt.Errorf("duplicate migration set for plugin %s", id)
		}
		indices[id] = i
		indegree[id] = 0
	}

	for _, set := range sets {
		deps := dedupeStrings(set.DependsOn)
		for _, dep := range deps {
			if dep == set.PluginID {
				return nil, fmt.Errorf("plugin %s cannot depend on itself", set.PluginID)
			}
			idx, ok := indices[dep]
			if !ok {
				return nil, fmt.Errorf("plugin %s depends on %s which has no registered migrations", set.PluginID, dep)
			}
			_ = idx // dependency existence already validated
			indegree[set.PluginID]++
			graph[dep] = append(graph[dep], set.PluginID)
		}
	}

	queue := make([]string, 0, len(sets))
	for id, degree := range indegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	orderedIDs := make([]string, 0, len(sets))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		orderedIDs = append(orderedIDs, id)

		for _, neighbor := range graph[id] {
			indegree[neighbor]--
			if indegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
		sort.Strings(queue)
	}

	if len(orderedIDs) != len(sets) {
		return nil, errors.New("migration dependency cycle detected")
	}

	ordered := make([]MigrationSet, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		ordered = append(ordered, sets[indices[id]])
	}

	return ordered, nil
}

func validateSet(set MigrationSet) error {
	if strings.TrimSpace(set.PluginID) == "" {
		return errors.New("plugin ID cannot be empty")
	}
	return nil
}

func validateMigration(pluginID string, migration Migration) error {
	if strings.TrimSpace(migration.Version) == "" {
		return fmt.Errorf("migration version cannot be empty for plugin %s", pluginID)
	}
	if migration.Up == nil {
		return fmt.Errorf("migration %s:%s missing Up function", pluginID, migration.Version)
	}
	return nil
}

func findMigration(set MigrationSet, version string) (Migration, bool) {
	for _, migration := range set.Migrations {
		if migration.Version == version {
			return migration, true
		}
	}
	return Migration{}, false
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (m *Migrator) ensureAppliedAtIndex(ctx context.Context) error {
	indexName := fmt.Sprintf("%s_plugin_applied_idx", sanitizeIdentifier(m.table))
	stmt := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (plugin_id, applied_at)", indexName, m.table)

	if m.dialect == "mysql" {
		stmt = fmt.Sprintf("CREATE INDEX %s ON %s (plugin_id, applied_at)", indexName, m.table)
	}

	if _, err := m.db.ExecContext(ctx, stmt); err != nil {
		if m.dialect == "mysql" && strings.Contains(strings.ToLower(err.Error()), "duplicate key name") {
			return nil
		}
		return fmt.Errorf("ensure index on %s: %w", m.table, err)
	}

	return nil
}

func sanitizeIdentifier(value string) string {
	replacer := strings.NewReplacer("`", "", "\"", "", ".", "_", " ", "_")
	return replacer.Replace(value)
}

// ExecStatements executes statements sequentially, skipping blanks.
func ExecStatements(ctx context.Context, tx bun.Tx, statements ...string) error {
	for _, stmt := range statements {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec statement failed: %w", err)
		}
	}
	return nil
}

func (m *Migrator) tableExpr(withAlias bool) (string, []any) {
	if withAlias {
		return fmt.Sprintf("? AS %s", schemaMigrationAlias), []any{bun.Ident(m.table)}
	}
	return "?", []any{bun.Ident(m.table)}
}
