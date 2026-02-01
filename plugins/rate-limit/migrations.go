package ratelimit

import (
	"embed"

	sharedmigrations "github.com/GoBetterAuth/go-better-auth/v2/plugins/shared/migrations"
)

//go:embed migrations/sqlite/*.sql
var sqliteMigrations embed.FS

//go:embed migrations/postgres/*.sql
var postgresMigrations embed.FS

//go:embed migrations/mysql/*.sql
var mysqlMigrations embed.FS

// MigrationFS holds all provider migrations for the rate-limit plugin.
// This is used by the plugin's Migrations() method when Provider is "database".
var MigrationFS = &sharedmigrations.ProviderFS{
	SQLite:   &sqliteMigrations,
	Postgres: &postgresMigrations,
	MySQL:    &mysqlMigrations,
}
