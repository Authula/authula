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

func TestNewDatabaseProvider(t *testing.T) {
	t.Parallel()

	if provider, err := NewDatabaseProvider(nil); err == nil || provider != nil {
		t.Fatalf("expected nil provider and error for nil db, got provider=%v err=%v", provider, err)
	}

	db := tests.NewSQLiteIntegrationDB(t)
	provider, err := NewDatabaseProvider(db)
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.Equal(t, "database", provider.GetName())
	require.NoError(t, provider.Close())
}

func TestDatabaseProvider_GetValue(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		seed    *types.RateLimit
		key     string
		closeDB bool
		wantNil bool
		wantErr bool
	}{
		{
			name: "returns_existing_value",
			seed: &types.RateLimit{Key: "key-1", Count: 4, ExpiresAt: time.Now().UTC().Add(time.Minute)},
			key:  "key-1",
		},
		{
			name:    "missing_value_returns_nil",
			key:     "missing",
			wantNil: true,
		},
		{
			name:    "closed_database_errors",
			key:     "key-1",
			closeDB: true,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := tests.NewSQLiteIntegrationDB(t)
			provider, err := NewDatabaseProviderWithConfig(db, types.DatabaseStorageConfig{CleanupInterval: time.Hour})
			require.NoError(t, err)
			provider.logger = &tests.MockLogger{}
			require.NoError(t, db.ResetModel(ctx, (*types.RateLimit)(nil)))

			if tc.seed != nil {
				_, err := db.NewInsert().Model(tc.seed).Exec(ctx)
				require.NoError(t, err)
			}
			if tc.closeDB {
				require.NoError(t, db.Close())
			}

			value, err := provider.GetValue(ctx, tc.key)

			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, value)
				return
			}
			require.NoError(t, err)
			if tc.wantNil {
				require.Nil(t, value)
				return
			}
			rateLimitValue, ok := value.(types.RateLimitValue)
			require.True(t, ok)
			require.Equal(t, tc.seed.Count, rateLimitValue.Count)
		})
	}
}

func TestDatabaseProvider_CheckAndIncrement(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		seed        *types.RateLimit
		max         int
		cancel      bool
		closeDB     bool
		wantErr     bool
		wantAllowed bool
		wantCount   int
	}{
		{
			name:        "new_record_is_allowed",
			max:         5,
			wantAllowed: true,
			wantCount:   1,
		},
		{
			name:        "existing_record_increments",
			seed:        &types.RateLimit{Key: "key-1", Count: 2, ExpiresAt: time.Now().UTC().Add(time.Minute)},
			max:         5,
			wantAllowed: true,
			wantCount:   3,
		},
		{
			name:        "over_limit_is_denied",
			seed:        &types.RateLimit{Key: "key-1", Count: 5, ExpiresAt: time.Now().UTC().Add(time.Minute)},
			max:         5,
			wantAllowed: false,
			wantCount:   6,
		},
		{
			name:    "canceled_context_errors",
			max:     5,
			cancel:  true,
			wantErr: true,
		},
		{
			name:    "closed_database_errors",
			max:     5,
			closeDB: true,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := tests.NewSQLiteIntegrationDB(t)
			provider, err := NewDatabaseProviderWithConfig(db, types.DatabaseStorageConfig{CleanupInterval: time.Hour})
			require.NoError(t, err)
			provider.logger = &tests.MockLogger{}
			require.NoError(t, db.ResetModel(ctx, (*types.RateLimit)(nil)))

			if tc.seed != nil {
				_, err := db.NewInsert().Model(tc.seed).Exec(ctx)
				require.NoError(t, err)
			}
			if tc.closeDB {
				require.NoError(t, db.Close())
			}
			if tc.cancel {
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
			}

			allowed, count, _, err := provider.CheckAndIncrement(ctx, "key-1", time.Minute, tc.max)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantAllowed, allowed)
			require.Equal(t, tc.wantCount, count)
		})
	}
}

func TestDatabaseProvider_RuleErrorsAndCleanup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := tests.NewSQLiteIntegrationDB(t)
	require.NoError(t, db.ResetModel(ctx, (*types.RateLimitRuleRecord)(nil)))
	provider, err := NewDatabaseProviderWithConfig(db, types.DatabaseStorageConfig{CleanupInterval: 5 * time.Millisecond})
	require.NoError(t, err)
	provider.logger = &tests.MockLogger{}

	require.NoError(t, provider.SetRule(ctx, "key-1", time.Minute, 5))
	window, maxRequests, found, err := provider.GetRule(ctx, "key-1")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, time.Minute, window)
	require.Equal(t, 5, maxRequests)

	require.NoError(t, db.ResetModel(ctx, (*types.RateLimit)(nil)))
	_, err = db.NewInsert().Model(&types.RateLimit{Key: "expired", Count: 1, ExpiresAt: time.Now().UTC().Add(-time.Minute)}).Exec(ctx)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		count, err := db.NewSelect().Model((*types.RateLimit)(nil)).Where("key = ?", "expired").Count(ctx)
		return err == nil && count == 0
	}, 2*time.Second, 10*time.Millisecond)

	require.NoError(t, db.Close())
	_, _, _, err = provider.GetRule(ctx, "key-1")
	require.Error(t, err)
	_, err = provider.GetValue(ctx, "key-1")
	require.Error(t, err)
	require.Error(t, provider.DeleteRule(ctx, "key-1"))
}
