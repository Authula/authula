package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/rate-limit/types"
	"github.com/stretchr/testify/require"
)

func TestBunRateLimitRepository_GetByKey(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name       string
		seed       *types.RateLimit
		key        string
		wantNil    bool
		closeDB    bool
		wantErrMsg string
	}

	cases := []testCase{
		{
			name: "returns existing record",
			seed: &types.RateLimit{Key: "api-key-1", Count: 3, ExpiresAt: time.Now().UTC().Add(10 * time.Minute)},
			key:  "api-key-1",
		},
		{
			name:    "returns nil for missing record",
			key:     "missing-key",
			wantNil: true,
		},
		{
			name:       "returns wrapped error when db is closed",
			key:        "broken-key",
			closeDB:    true,
			wantErrMsg: "failed to get rate limit record",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := tests.NewSQLiteIntegrationDB(t)
			repo := NewRateLimitRepository(db)
			require.NoError(t, db.ResetModel(ctx, (*types.RateLimit)(nil)))
			defer func() { require.NoError(t, db.Close()) }()

			if tc.seed != nil {
				_, err := db.NewInsert().Model(tc.seed).Exec(ctx)
				require.NoError(t, err)
			}

			if tc.closeDB {
				require.NoError(t, db.Close())
			}

			record, err := repo.GetByKey(ctx, tc.key)
			if tc.wantErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}

			require.NoError(t, err)
			if tc.wantNil {
				require.Nil(t, record)
				return
			}

			require.NotNil(t, record)
			require.Equal(t, tc.seed.Key, record.Key)
			require.Equal(t, tc.seed.Count, record.Count)
			require.WithinDuration(t, tc.seed.ExpiresAt, record.ExpiresAt, time.Second)
		})
	}
}

func TestBunRateLimitRepository_UpdateOrCreate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name       string
		seed       *types.RateLimit
		key        string
		window     time.Duration
		wantCount  int
		wantReset  bool
		closeDB    bool
		wantErrMsg string
	}

	cases := []testCase{
		{name: "insert new record", key: "insert-key", window: 2 * time.Minute, wantCount: 1},
		{name: "increment existing unexpired record", seed: &types.RateLimit{Key: "increment-key", Count: 4, ExpiresAt: time.Now().UTC().Truncate(time.Second).Add(10 * time.Minute)}, key: "increment-key", window: 5 * time.Minute, wantCount: 5},
		{name: "reset existing expired record", seed: &types.RateLimit{Key: "expired-key", Count: 9, ExpiresAt: time.Now().UTC().Truncate(time.Second).Add(-time.Minute)}, key: "expired-key", window: 7 * time.Minute, wantCount: 1, wantReset: true},
		{name: "database failure", key: "broken-key", window: time.Minute, closeDB: true, wantErrMsg: "ratelimit upsert failed"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := tests.NewSQLiteIntegrationDB(t)
			repo := NewRateLimitRepository(db)
			require.NoError(t, db.ResetModel(ctx, (*types.RateLimit)(nil)))
			defer func() { require.NoError(t, db.Close()) }()

			if tc.seed != nil {
				_, err := db.NewInsert().Model(tc.seed).Exec(ctx)
				require.NoError(t, err)
			}

			if tc.closeDB {
				require.NoError(t, db.Close())
			}

			record, err := repo.UpdateOrCreate(ctx, tc.key, tc.window)
			if tc.wantErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.key, record.Key)
			require.Equal(t, tc.wantCount, record.Count)
			if tc.wantReset {
				require.WithinDuration(t, time.Now().UTC().Add(tc.window), record.ExpiresAt, 2*time.Second)
			} else if tc.seed != nil {
				require.WithinDuration(t, tc.seed.ExpiresAt, record.ExpiresAt, 2*time.Second)
			} else {
				require.WithinDuration(t, time.Now().UTC().Add(tc.window), record.ExpiresAt, 2*time.Second)
			}
		})
	}
}

func TestBunRateLimitRepository_CleanupExpired(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		seed     []types.RateLimit
		wantKeys []string
	}

	now := time.Now().UTC().Truncate(time.Second)
	cases := []testCase{
		{
			name: "removes only expired records",
			seed: []types.RateLimit{
				{Key: "expired", Count: 1, ExpiresAt: now.Add(-time.Minute)},
				{Key: "boundary", Count: 1, ExpiresAt: now},
				{Key: "active", Count: 1, ExpiresAt: now.Add(time.Minute)},
			},
			wantKeys: []string{"active", "boundary"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := tests.NewSQLiteIntegrationDB(t)
			repo := NewRateLimitRepository(db)
			require.NoError(t, db.ResetModel(ctx, (*types.RateLimit)(nil)))
			defer func() { require.NoError(t, db.Close()) }()

			for _, row := range tc.seed {
				_, err := db.NewInsert().Model(&row).Exec(ctx)
				require.NoError(t, err)
			}

			require.NoError(t, repo.CleanupExpired(ctx, now))

			remaining := []*types.RateLimit{}
			require.NoError(t, db.NewSelect().Model(&remaining).Order("key ASC").Scan(ctx))
			require.Len(t, remaining, len(tc.wantKeys))
			for i, key := range tc.wantKeys {
				require.Equal(t, key, remaining[i].Key)
			}
		})
	}
}

func TestBunRateLimitRepository_SetRule(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name       string
		key        string
		window     int
		max        int
		wantWindow int
		wantMax    int
		closeDB    bool
		wantErrMsg string
	}

	cases := []testCase{
		{name: "creates a new rule", key: "api-key-1", window: 90, max: 17, wantWindow: 90, wantMax: 17},
		{name: "updates an existing rule", key: "api-key-1", window: 120, max: 25, wantWindow: 120, wantMax: 25},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := tests.NewSQLiteIntegrationDB(t)
			repo := NewRateLimitRepository(db)
			require.NoError(t, db.ResetModel(ctx, (*types.RateLimitRuleRecord)(nil)))
			defer func() { require.NoError(t, db.Close()) }()

			require.NoError(t, repo.SetRule(ctx, tc.key, tc.window, tc.max))

			record, err := repo.GetRule(ctx, tc.key)
			require.NoError(t, err)
			require.NotNil(t, record)
			require.Equal(t, tc.key, record.Key)
			require.Equal(t, tc.wantWindow, record.WindowSeconds)
			require.Equal(t, tc.wantMax, record.MaxRequests)
		})
	}
}

func TestBunRateLimitRepository_GetRule(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name       string
		seed       *types.RateLimitRuleRecord
		key        string
		wantNil    bool
		closeDB    bool
		wantErrMsg string
	}

	cases := []testCase{
		{name: "returns existing rule", seed: &types.RateLimitRuleRecord{Key: "api-key-1", WindowSeconds: 90, MaxRequests: 17}, key: "api-key-1"},
		{name: "returns nil for missing rule", key: "missing-key", wantNil: true},
		{name: "returns wrapped error when db is closed", key: "broken-key", closeDB: true, wantErrMsg: "failed to get rate limit rule"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := tests.NewSQLiteIntegrationDB(t)
			repo := NewRateLimitRepository(db)
			require.NoError(t, db.ResetModel(ctx, (*types.RateLimitRuleRecord)(nil)))
			defer func() { require.NoError(t, db.Close()) }()

			if tc.seed != nil {
				_, err := db.NewInsert().Model(tc.seed).Exec(ctx)
				require.NoError(t, err)
			}

			if tc.closeDB {
				require.NoError(t, db.Close())
			}

			record, err := repo.GetRule(ctx, tc.key)
			if tc.wantErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}

			require.NoError(t, err)
			if tc.wantNil {
				require.Nil(t, record)
				return
			}

			require.NotNil(t, record)
			require.Equal(t, tc.seed.Key, record.Key)
			require.Equal(t, tc.seed.WindowSeconds, record.WindowSeconds)
			require.Equal(t, tc.seed.MaxRequests, record.MaxRequests)
		})
	}
}

func TestBunRateLimitRepository_DeleteRule(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name       string
		seed       *types.RateLimitRuleRecord
		key        string
		closeDB    bool
		wantErrMsg string
	}

	cases := []testCase{
		{name: "deletes existing rule", seed: &types.RateLimitRuleRecord{Key: "api-key-1", WindowSeconds: 90, MaxRequests: 17}, key: "api-key-1"},
		{name: "returns wrapped error when db is closed", key: "broken-key", closeDB: true, wantErrMsg: "failed to delete rate limit rule"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := tests.NewSQLiteIntegrationDB(t)
			repo := NewRateLimitRepository(db)
			require.NoError(t, db.ResetModel(ctx, (*types.RateLimitRuleRecord)(nil)))
			defer func() { require.NoError(t, db.Close()) }()

			if tc.seed != nil {
				_, err := db.NewInsert().Model(tc.seed).Exec(ctx)
				require.NoError(t, err)
			}

			if tc.closeDB {
				require.NoError(t, db.Close())
			}

			err := repo.DeleteRule(ctx, tc.key)
			if tc.wantErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}

			require.NoError(t, err)
			record, err := repo.GetRule(ctx, tc.key)
			require.NoError(t, err)
			require.Nil(t, record)
		})
	}
}
