package bootstrap

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/extra/bundebug"

	"github.com/GoBetterAuth/go-better-auth/v2/env"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// DatabaseOptions configures database initialization
type DatabaseOptions struct {
	Provider        string
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// InitDatabase creates a Bun database connection
func InitDatabase(opts DatabaseOptions, logger models.Logger, logLevel string) (bun.IDB, error) {
	if opts.Provider == "" {
		return nil, fmt.Errorf("database provider must be specified")
	}

	databaseURL := os.Getenv(env.EnvDatabaseURL)
	if databaseURL == "" {
		if opts.URL == "" {
			return nil, fmt.Errorf("database connection string must be specified via %s or config", env.EnvDatabaseURL)
		}
		databaseURL = opts.URL
	}

	var (
		sqlDB *sql.DB
		err   error
	)

	switch opts.Provider {
	case "sqlite":
		if !filepath.IsAbs(databaseURL) {
			cwd, _ := os.Getwd()
			databaseURL = filepath.Join(cwd, databaseURL)
		}

		dbDir := filepath.Dir(databaseURL)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}

		sqlDB, err = sql.Open("sqlite3", databaseURL)
		if err != nil {
			return nil, err
		}

		db := bun.NewDB(sqlDB, sqlitedialect.New())
		configurePool(sqlDB, opts)
		enableDebugging(db, logLevel)
		return db, nil

	case "postgres":
		sqlDB, err = sql.Open("postgres", databaseURL)
		if err != nil {
			return nil, err
		}

		db := bun.NewDB(sqlDB, pgdialect.New())
		configurePool(sqlDB, opts)
		enableDebugging(db, logLevel)
		return db, nil

	case "mysql":
		sqlDB, err = sql.Open("mysql", databaseURL)
		if err != nil {
			return nil, err
		}

		db := bun.NewDB(sqlDB, mysqldialect.New())
		configurePool(sqlDB, opts)
		enableDebugging(db, logLevel)
		return db, nil

	default:
		return nil, fmt.Errorf("unsupported database provider: %s", opts.Provider)
	}
}

func configurePool(sqlDB *sql.DB, opts DatabaseOptions) {
	numCPU := runtime.NumCPU()

	maxOpenConns := opts.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = numCPU * 4
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)

	maxIdleConns := opts.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = numCPU * 2
	}
	sqlDB.SetMaxIdleConns(maxIdleConns)

	connMaxLifetime := opts.ConnMaxLifetime
	if connMaxLifetime == 0 {
		connMaxLifetime = 10 * time.Minute
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
}

func enableDebugging(db *bun.DB, logLevel string) {
	if logLevel == "debug" {
		db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
		))
	}
}
