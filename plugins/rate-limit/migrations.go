package ratelimit

import (
	"context"
	"embed"
	"fmt"
)

//go:embed migrations/sqlite/*.sql
var sqliteFS embed.FS

//go:embed migrations/postgres/*.sql
var postgresFS embed.FS

//go:embed migrations/mysql/*.sql
var mysqlFS embed.FS

// GetMigrations returns the migrations for the specified database provider.
func GetMigrations(ctx context.Context, provider string) (*embed.FS, error) {
	switch provider {
	case "sqlite":
		return &sqliteFS, nil
	case "postgres":
		return &postgresFS, nil
	case "mysql":
		return &mysqlFS, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", provider)
	}
}
