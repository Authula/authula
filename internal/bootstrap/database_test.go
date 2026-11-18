package bootstrap

import (
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"

	"github.com/Authula/authula/env"
	"github.com/Authula/authula/internal/sqlitedsn"
)

// openDatabase calls InitDatabase with an isolated AUTHULA_DATABASE_URL and
// registers a cleanup that closes the returned connection.
func openDatabase(t *testing.T, envURL string, opts DatabaseOptions, logLevel string) bun.IDB {
	t.Helper()

	t.Setenv(env.EnvDatabaseURL, envURL)

	db, err := InitDatabase(opts, nil, logLevel)
	require.NoError(t, err)
	require.NotNil(t, db)

	t.Cleanup(func() {
		closeDatabase(t, db)
	})

	return db
}

func closeDatabase(t *testing.T, db bun.IDB) {
	t.Helper()

	bunDB, ok := db.(*bun.DB)
	require.Truef(t, ok, "InitDatabase returned %T, want *bun.DB", db)
	assert.NoError(t, bunDB.Close())
}

func requireBunDatabase(t *testing.T, db bun.IDB) *bun.DB {
	t.Helper()

	bunDB, ok := db.(*bun.DB)
	require.Truef(t, ok, "InitDatabase returned %T, want *bun.DB", db)

	return bunDB
}

func requireUsableSQLite(t *testing.T, db bun.IDB) {
	t.Helper()

	var one int
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT 1").Scan(&one))
	assert.Equal(t, 1, one)
}

func TestInitDatabaseValidation(t *testing.T) {
	tests := []struct {
		name    string
		opts    DatabaseOptions
		wantErr string
	}{
		{
			name:    "missing provider",
			opts:    DatabaseOptions{URL: ":memory:"},
			wantErr: "database provider must be specified",
		},
		{
			name:    "missing url and environment",
			opts:    DatabaseOptions{Provider: "sqlite"},
			wantErr: "database connection string must be specified",
		},
		{
			name:    "unsupported provider",
			opts:    DatabaseOptions{Provider: "oracle", URL: ":memory:"},
			wantErr: "unsupported database provider: oracle",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(env.EnvDatabaseURL, "")

			db, err := InitDatabase(tc.opts, nil, "info")

			require.Error(t, err)
			assert.Nil(t, db)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestInitDatabaseSQLiteMemory(t *testing.T) {
	db := openDatabase(t, "", DatabaseOptions{
		Provider:     "sqlite",
		URL:          ":memory:",
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}, "info")

	bunDB := requireBunDatabase(t, db)
	assert.Equal(t, dialect.SQLite, bunDB.Dialect().Name())
	requireUsableSQLite(t, db)

	// Pool options flow through InitDatabase into configurePool. Only the max
	// open limit is exposed by database/sql stats.
	stats := bunDB.Stats()
	assert.Equal(t, 5, stats.MaxOpenConnections)
}

func TestInitDatabaseSQLiteFile(t *testing.T) {
	t.Run("absolute path creates parent directory and database", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "nested", "authula.db")

		db := openDatabase(t, "", DatabaseOptions{Provider: "sqlite", URL: dbPath}, "info")

		requireUsableSQLite(t, db)
		_, err := os.Stat(dbPath)
		require.NoError(t, err)
	})

	t.Run("relative path resolves against working directory", func(t *testing.T) {
		workingDir := t.TempDir()
		t.Chdir(workingDir)

		db := openDatabase(t, "", DatabaseOptions{Provider: "sqlite", URL: filepath.Join("data", "authula.db")}, "info")

		requireUsableSQLite(t, db)
		_, err := os.Stat(filepath.Join(workingDir, "data", "authula.db"))
		require.NoError(t, err)
	})

	t.Run("directory creation failure is returned", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "blocker")
		require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), 0o600))

		t.Setenv(env.EnvDatabaseURL, "")

		db, err := InitDatabase(DatabaseOptions{
			Provider: "sqlite",
			URL:      filepath.Join(blocker, "nested", "authula.db"),
		}, nil, "info")

		require.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "failed to create database directory")
	})
}

func TestInitDatabaseUsesEnvironmentURLOverOptions(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), "from-env", "authula.db")
	optionPath := filepath.Join(t.TempDir(), "from-options", "authula.db")

	db := openDatabase(t, envPath, DatabaseOptions{Provider: "sqlite", URL: optionPath}, "info")

	requireUsableSQLite(t, db)
	_, err := os.Stat(envPath)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Dir(optionPath))
	assert.True(t, os.IsNotExist(err), "options URL must be ignored when AUTHULA_DATABASE_URL is set, stat error: %v", err)
}

func TestInitDatabaseNetworkProvidersOpenLazily(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		url           string
		wantDialect   dialect.Name
		wantOpenError bool
	}{
		{
			name:        "postgres valid dsn opens without dialing",
			provider:    "postgres",
			url:         "postgres://authula:secret@127.0.0.1:1/authula?sslmode=disable&connect_timeout=1",
			wantDialect: dialect.PG,
		},
		{
			name:        "postgres malformed dsn is parsed lazily",
			provider:    "postgres",
			url:         "postgres://authula:secret@127.0.0.1:not-a-port/authula",
			wantDialect: dialect.PG,
		},
		{
			name:          "mysql malformed dsn fails at open time",
			provider:      "mysql",
			url:           "authula:secret@tcp(127.0.0.1:1",
			wantOpenError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(env.EnvDatabaseURL, "")

			db, err := InitDatabase(DatabaseOptions{Provider: tc.provider, URL: tc.url}, nil, "info")

			if tc.wantOpenError {
				require.Error(t, err)
				assert.Nil(t, db)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, db)
			t.Cleanup(func() {
				closeDatabase(t, db)
			})

			// Only open-time behaviour is asserted; no query is executed, so
			// nothing dials the unreachable address above.
			bunDB := requireBunDatabase(t, db)
			assert.Equal(t, tc.wantDialect, bunDB.Dialect().Name())
		})
	}
}

func TestConfigurePool(t *testing.T) {
	tests := []struct {
		name        string
		opts        DatabaseOptions
		wantMaxOpen int
	}{
		{
			name:        "defaults derive from cpu count",
			opts:        DatabaseOptions{},
			wantMaxOpen: runtime.NumCPU() * 4,
		},
		{
			name:        "zero values use defaults",
			opts:        DatabaseOptions{MaxOpenConns: 0, MaxIdleConns: 0},
			wantMaxOpen: runtime.NumCPU() * 4,
		},
		{
			name:        "negative values use defaults",
			opts:        DatabaseOptions{MaxOpenConns: -1, MaxIdleConns: -1},
			wantMaxOpen: runtime.NumCPU() * 4,
		},
		{
			name:        "custom values are applied",
			opts:        DatabaseOptions{MaxOpenConns: 7, MaxIdleConns: 3, ConnMaxLifetime: 30 * time.Second},
			wantMaxOpen: 7,
		},
		{
			name:        "custom lifetime does not change the max open limit",
			opts:        DatabaseOptions{ConnMaxLifetime: time.Second},
			wantMaxOpen: runtime.NumCPU() * 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB, err := sql.Open("sqlite", sqlitedsn.Build(":memory:"))
			require.NoError(t, err)
			t.Cleanup(func() {
				assert.NoError(t, sqlDB.Close())
			})

			configurePool(sqlDB, tc.opts)

			assert.Equal(t, tc.wantMaxOpen, sqlDB.Stats().MaxOpenConnections)
		})
	}

	// database/sql does not expose the idle limit, so assert it by behaviour:
	// three borrowed connections must collapse to a single idle one when the
	// configured idle limit is one.
	t.Run("max idle connections is honoured", func(t *testing.T) {
		db := openDatabase(t, "", DatabaseOptions{
			Provider:     "sqlite",
			URL:          filepath.Join(t.TempDir(), "authula.db"),
			MaxOpenConns: 3,
			MaxIdleConns: 1,
		}, "info")
		bunDB := requireBunDatabase(t, db)

		ctx := context.Background()
		conns := make([]*sql.Conn, 0, 3)
		for range 3 {
			conn, err := bunDB.DB.Conn(ctx)
			require.NoError(t, err)
			conns = append(conns, conn)
		}

		require.Eventually(t,
			func() bool { return bunDB.Stats().OpenConnections == 3 },
			5*time.Second, 10*time.Millisecond,
			"expected three open connections before releasing them")

		for _, conn := range conns {
			require.NoError(t, conn.Close())
		}

		require.Eventually(t,
			func() bool { return bunDB.Stats().Idle == 1 },
			5*time.Second, 10*time.Millisecond,
			"expected idle connections to be capped at one")
	})
}

// TestEnableDebuggingQueryHook captures stderr because bundebug writes there.
// The test is deliberately sequential: it swaps a process-global stream.
func TestEnableDebuggingQueryHook(t *testing.T) {
	tests := []struct {
		name       string
		logLevel   string
		wantLogged bool
	}{
		{name: "debug level logs queries", logLevel: "debug", wantLogged: true},
		{name: "info level does not log queries", logLevel: "info", wantLogged: false},
		{name: "warn level does not log queries", logLevel: "warn", wantLogged: false},
		{name: "empty level does not log queries", logLevel: "", wantLogged: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader, writer, err := os.Pipe()
			require.NoError(t, err)

			originalStderr := os.Stderr
			os.Stderr = writer
			t.Cleanup(func() {
				os.Stderr = originalStderr
				_ = writer.Close()
				_ = reader.Close()
			})

			t.Setenv(env.EnvDatabaseURL, "")

			db, err := InitDatabase(DatabaseOptions{Provider: "sqlite", URL: ":memory:"}, nil, tc.logLevel)
			os.Stderr = originalStderr

			require.NoError(t, err)
			require.NotNil(t, db)
			t.Cleanup(func() {
				closeDatabase(t, db)
			})

			requireUsableSQLite(t, db)

			require.NoError(t, writer.Close())

			logged, err := io.ReadAll(reader)
			require.NoError(t, err)

			if tc.wantLogged {
				assert.Contains(t, string(logged), "[bun]")
				assert.Contains(t, string(logged), "SELECT")
				return
			}

			assert.Empty(t, string(logged))
		})
	}
}
