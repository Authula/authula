package tests

import (
	"testing"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/internal/testdb"
)

func NewIntegrationTestDB(t *testing.T, migrateFns ...func(bun.IDB)) *bun.DB {
	t.Helper()
	return testdb.NewIntegrationTestDB(t, migrateFns...)
}
