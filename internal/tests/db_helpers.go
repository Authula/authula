package tests

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

func NewSQLiteIntegrationDB(t *testing.T) *bun.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	sqlDB, err := openSQLiteForTests(dsn)
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func NewPostgresIntegrationDB(t *testing.T) *bun.DB {
	t.Helper()
	ctx := context.Background()

	container, err := runPostgresContainer(ctx)
	if err != nil {
		if isDockerUnavailableError(err) {
			t.Skipf("skipping postgres integration DB setup: %v", err)
			return nil
		}
		t.Fatalf("failed to start postgres testcontainer: %v", err)
	}
	t.Cleanup(func() {
		_ = testcontainers.TerminateContainer(container)
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get postgres connection string: %v", err)
	}

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open postgres db: %v", err)
	}
	if err := waitForDB(ctx, sqlDB, 30); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("failed waiting for postgres db readiness: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := bun.NewDB(sqlDB, pgdialect.New())
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func NewMySQLIntegrationDB(t *testing.T) *bun.DB {
	t.Helper()
	ctx := context.Background()

	container, err := runMySQLContainer(ctx)
	if err != nil {
		if isDockerUnavailableError(err) {
			t.Skipf("skipping mysql integration DB setup: %v", err)
			return nil
		}
		t.Fatalf("failed to start mysql testcontainer: %v", err)
	}
	t.Cleanup(func() {
		_ = testcontainers.TerminateContainer(container)
	})

	dsn, err := container.ConnectionString(ctx, "parseTime=true")
	if err != nil {
		t.Fatalf("failed to get mysql connection string: %v", err)
	}

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("failed to open mysql db: %v", err)
	}
	if err := waitForDB(ctx, sqlDB, 30); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("failed waiting for mysql db readiness: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := bun.NewDB(sqlDB, mysqldialect.New())
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func NewIntegrationTestDBFromEnv(t *testing.T) (*bun.DB, string) {
	t.Helper()

	provider := strings.ToLower(strings.TrimSpace(os.Getenv("GO_BETTER_AUTH_TEST_DB")))
	if provider == "" {
		provider = "sqlite"
	}

	switch provider {
	case "sqlite":
		return NewSQLiteIntegrationDB(t), "sqlite"
	case "postgres":
		return NewPostgresIntegrationDB(t), "postgres"
	case "mysql":
		return NewMySQLIntegrationDB(t), "mysql"
	default:
		t.Fatalf("unsupported GO_BETTER_AUTH_TEST_DB provider %q (expected sqlite|postgres|mysql)", provider)
		return nil, ""
	}
}

func openSQLiteForTests(dsn string) (*sql.DB, error) {
	if sqlite3DB, err := sql.Open("sqlite3", dsn); err == nil {
		if pingErr := sqlite3DB.PingContext(context.Background()); pingErr == nil {
			return sqlite3DB, nil
		}
		_ = sqlite3DB.Close()
	}

	sqliteDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := sqliteDB.PingContext(context.Background()); err != nil {
		_ = sqliteDB.Close()
		return nil, err
	}

	return sqliteDB, nil
}

func waitForDB(ctx context.Context, db *sql.DB, attempts int) error {
	var lastErr error
	for range attempts {
		if err := db.PingContext(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
	return lastErr
}

func runPostgresContainer(ctx context.Context) (container *tcpostgres.PostgresContainer, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic while starting postgres testcontainer: %v", recovered)
		}
	}()

	return tcpostgres.Run(
		ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("gobetterauth"),
		tcpostgres.WithUsername("gobetterauth"),
		tcpostgres.WithPassword("gobetterauth"),
	)
}

func runMySQLContainer(ctx context.Context) (container *tcmysql.MySQLContainer, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic while starting mysql testcontainer: %v", recovered)
		}
	}()

	return tcmysql.Run(
		ctx,
		"mysql:8.4",
		tcmysql.WithDatabase("gobetterauth"),
		tcmysql.WithUsername("gobetterauth"),
		tcmysql.WithPassword("gobetterauth"),
	)
}

func isDockerUnavailableError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "rootless docker not found") ||
		strings.Contains(msg, "cannot connect to the docker daemon") ||
		strings.Contains(msg, "docker host") ||
		strings.Contains(msg, "docker socket") ||
		strings.Contains(msg, "permission denied")
}
