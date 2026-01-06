package configmanager

import (
	"context"
	"embed"
)

//go:embed migrations/postgres/*.sql
var postgresFS embed.FS

//go:embed migrations/mysql/*.sql
var mysqlFS embed.FS

//go:embed migrations/sqlite/*.sql
var sqliteFS embed.FS

// GetMigrations returns the migrations for the specified database provider.
func GetMigrations(ctx context.Context, provider string) (*embed.FS, error) {
	switch provider {
	case "postgres":
		return &postgresFS, nil
	case "mysql":
		return &mysqlFS, nil
	case "sqlite":
		return &sqliteFS, nil
	default:
		// For unsupported providers, return postgres as default
		return &postgresFS, nil
	}
}
