package testdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

const containerName = "authula-test-pg"

var (
	mu         sync.Mutex
	connString string
)

func randomHex() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func waitForDB(sqlDB *sql.DB) error {
	var lastErr error
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for range 60 {
		if err := sqlDB.PingContext(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return lastErr
}

func ensureContainer() string {
	// Check if already running
	cmd := exec.Command("docker", "ps",
		"--filter", fmt.Sprintf("name=%s", containerName),
		"--filter", "status=running",
		"--format", "{{.ID}}")
	out, _ := cmd.Output()
	if strings.TrimSpace(string(out)) != "" {
		// Container exists — get its IP
		cmd2 := exec.Command("docker", "inspect",
			"-f", `{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}`,
			containerName)
		ipOut, _ := cmd2.Output()
		ip := strings.TrimSpace(string(ipOut))
		if ip != "" {
			return fmt.Sprintf("postgres://authula:authula@%s:5432/authula?sslmode=disable", ip)
		}
	}

	// Remove any leftover container
	_ = exec.Command("docker", "rm", "-f", containerName).Run()

	// Start new container
	cmd = exec.Command("docker", "run", "-d",
		"--name", containerName,
		"--label", "authula-test-pg=true",
		"-e", "POSTGRES_DB=authula",
		"-e", "POSTGRES_USER=authula",
		"-e", "POSTGRES_PASSWORD=authula",
		"postgres:18-alpine",
		"-c", "max_connections=200")
	out, err := cmd.Output()
	if err != nil {
		panic(fmt.Errorf("failed to start postgres container: %w\n%s", err, string(out)))
	}

	cid := strings.TrimSpace(string(out))

	// Wait for container to be running
	for range 30 {
		cmd = exec.Command("docker", "inspect", "-f", "{{.State.Status}}", cid)
		status, _ := cmd.Output()
		if strings.TrimSpace(string(status)) == "running" {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Get container IP on the Docker bridge network
	cmd = exec.Command("docker", "inspect",
		"-f", `{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}`, cid)
	out, err = cmd.Output()
	if err != nil {
		panic(fmt.Errorf("failed to get container ip: %w", err))
	}

	ip := strings.TrimSpace(string(out))
	return fmt.Sprintf("postgres://authula:authula@%s:5432/authula?sslmode=disable", ip)
}

func getConnectionString() string {
	mu.Lock()
	defer mu.Unlock()

	if connString != "" {
		// Verify cached DSN is still valid
		sqlDB, err := sql.Open("postgres", connString)
		if err == nil {
			if waitForDB(sqlDB) == nil {
				sqlDB.Close()
				return connString
			}
			sqlDB.Close()
		}
		connString = ""
	}

	dsn := ensureContainer()
	connString = dsn
	return dsn
}

func NewIntegrationTestDB(t *testing.T, migrateFns ...func(bun.IDB)) *bun.DB {
	t.Helper()

	dsn := getConnectionString()

	schema := "test_" + randomHex()

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open postgres db: %v", err)
	}

	if _, err := sqlDB.Exec("CREATE SCHEMA " + schema); err != nil {
		sqlDB.Close()
		t.Fatalf("failed to create schema %s: %v", schema, err)
	}

	if _, err := sqlDB.Exec("SET search_path TO " + schema + ", public"); err != nil {
		sqlDB.Close()
		t.Fatalf("failed to set search_path: %v", err)
	}

	db := bun.NewDB(sqlDB, pgdialect.New())
	db.SetMaxOpenConns(1)

	for _, fn := range migrateFns {
		fn(db)
	}

	t.Cleanup(func() {
		_, _ = sqlDB.Exec("DROP SCHEMA IF EXISTS " + schema + " CASCADE")
		_ = sqlDB.Close()
	})

	return db
}
