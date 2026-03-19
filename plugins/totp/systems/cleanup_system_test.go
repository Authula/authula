package systems_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/systems"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type mockRepo struct {
	callCount atomic.Int32
	err       error
}

func (m *mockRepo) DeleteExpiredTrustedDevices(_ context.Context) error {
	m.callCount.Add(1)
	return m.err
}

func configWithCleanup(enabled bool, interval time.Duration) *types.TOTPPluginConfig {
	return &types.TOTPPluginConfig{
		TrustedDevicesAutoCleanup:     enabled,
		TrustedDevicesCleanupInterval: interval,
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestName(t *testing.T) {
	s := systems.NewTrustedDevicesCleanupSystem(
		&internaltests.MockLogger{},
		configWithCleanup(false, 0),
		&mockRepo{},
	)
	assert.Equal(t, "TrustedDevicesCleanupSystem", s.Name())
}

func TestTrustedDevicesCleanupSystem(t *testing.T) {
	type testCase struct {
		name           string
		enabled        bool
		interval       time.Duration
		repoErr        error
		advanceBefore  time.Duration
		wantMinCalls   int32
		checkStopAfter bool
		advanceAfter   time.Duration
		wantNoIncrease bool
	}

	for _, tc := range []testCase{
		{
			name:          "auto cleanup disabled does not cleanup",
			enabled:       false,
			advanceBefore: 0,
			wantMinCalls:  0,
		},
		{
			name:          "auto cleanup enabled calls delete expired",
			enabled:       true,
			interval:      20 * time.Millisecond,
			advanceBefore: 45 * time.Millisecond,
			wantMinCalls:  2,
		},
		{
			name:          "zero interval falls back to one hour",
			enabled:       true,
			interval:      0,
			advanceBefore: 20 * time.Millisecond,
			wantMinCalls:  0,
		},
		{
			name:           "close stops cleanup loop",
			enabled:        true,
			interval:       10 * time.Millisecond,
			advanceBefore:  15 * time.Millisecond,
			wantMinCalls:   1,
			checkStopAfter: true,
			advanceAfter:   30 * time.Millisecond,
			wantNoIncrease: true,
		},
		{
			name:          "repo error does not panic",
			enabled:       true,
			interval:      10 * time.Millisecond,
			repoErr:       errors.New("db unavailable"),
			advanceBefore: 25 * time.Millisecond,
			wantMinCalls:  2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				repo := &mockRepo{err: tc.repoErr}
				s := systems.NewTrustedDevicesCleanupSystem(
					&internaltests.MockLogger{},
					configWithCleanup(tc.enabled, tc.interval),
					repo,
				)

				require.NoError(t, s.Init(context.Background()))
				if tc.advanceBefore > 0 {
					time.Sleep(tc.advanceBefore)
				}
				synctest.Wait()

				assert.GreaterOrEqual(t, repo.callCount.Load(), tc.wantMinCalls)

				if tc.checkStopAfter {
					require.NoError(t, s.Close())
					countAfterClose := repo.callCount.Load()
					time.Sleep(tc.advanceAfter)
					synctest.Wait()
					if tc.wantNoIncrease {
						assert.Equal(t, countAfterClose, repo.callCount.Load())
					}
					return
				}

				require.NoError(t, s.Close())
			})
		})
	}
}
