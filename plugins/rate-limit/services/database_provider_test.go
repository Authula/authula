package services

import (
	"context"
	"testing"
	"time"

	"github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/rate-limit/types"
	"github.com/stretchr/testify/require"
)

func TestDatabaseProviderRuleLifecycle(t *testing.T) {
	t.Parallel()

	db := tests.NewSQLiteIntegrationDB(t)
	provider, err := NewDatabaseProviderWithConfig(db, types.DatabaseStorageConfig{})
	require.NoError(t, err)

	ctx := context.Background()
	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS rate_limit_rules (
		key TEXT PRIMARY KEY,
		window_seconds INTEGER NOT NULL,
		max_requests INTEGER NOT NULL
	);`)
	require.NoError(t, err)

	window := 3 * time.Minute
	maxRequests := 13

	require.NoError(t, provider.SetRule(ctx, "api-key-1", window, maxRequests))

	gotWindow, gotMaxRequests, found, err := provider.GetRule(ctx, "api-key-1")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, window, gotWindow)
	require.Equal(t, maxRequests, gotMaxRequests)

	require.NoError(t, provider.DeleteRule(ctx, "api-key-1"))
	_, _, found, err = provider.GetRule(ctx, "api-key-1")
	require.NoError(t, err)
	require.False(t, found)
}
