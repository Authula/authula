package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/rate-limit/types"
	"github.com/stretchr/testify/require"
)

func TestBunRateLimitRepositoryRuleLifecycle(t *testing.T) {
	t.Parallel()

	db := tests.NewSQLiteIntegrationDB(t)
	repo := NewRateLimitRepository(db)
	ctx := context.Background()

	require.NoError(t, db.ResetModel(ctx, (*types.RateLimitRuleRecord)(nil)))

	windowSeconds := 90
	maxRequests := 17
	require.NoError(t, repo.SetRule(ctx, "api-key-1", windowSeconds, maxRequests))

	record, err := repo.GetRule(ctx, "api-key-1")
	require.NoError(t, err)
	require.NotNil(t, record)
	require.Equal(t, "api-key-1", record.Key)
	require.Equal(t, windowSeconds, record.WindowSeconds)
	require.Equal(t, maxRequests, record.MaxRequests)

	require.NoError(t, repo.DeleteRule(ctx, "api-key-1"))
	record, err = repo.GetRule(ctx, "api-key-1")
	require.NoError(t, err)
	require.Nil(t, record)

	time.Sleep(0)
}
