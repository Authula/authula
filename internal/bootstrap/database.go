package bootstrap

import (
	"database/sql"
	"fmt"
	"os"
	"runtime"
	"time"

	_ "github.com/lib/pq"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"

	"github.com/Authula/authula/env"
	"github.com/Authula/authula/models"
)

type DatabaseOptions struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func InitDatabase(opts DatabaseOptions, logger models.Logger, logLevel string) (bun.IDB, error) {
	databaseURL := os.Getenv(env.EnvDatabaseURL)
	if databaseURL == "" {
		if opts.URL == "" {
			return nil, fmt.Errorf("database connection string must be specified via %s or config", env.EnvDatabaseURL)
		}
		databaseURL = opts.URL
	}

	sqlDB, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	db := bun.NewDB(sqlDB, pgdialect.New())

	configurePool(sqlDB, opts)
	enableDebugging(db, logLevel)

	return db, nil
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
