package systems_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
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

func TestInit_AutoCleanupDisabled_DoesNotCleanup(t *testing.T) {
	repo := &mockRepo{}
	s := systems.NewTrustedDevicesCleanupSystem(
		&internaltests.MockLogger{},
		configWithCleanup(false, 0),
		repo,
	)

	require.NoError(t, s.Init(context.Background()))

	// Give a moment to confirm no goroutine fires.
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(0), repo.callCount.Load())

	// Close must not block even when cleanup was never started.
	require.NoError(t, s.Close())
}

func TestInit_AutoCleanupEnabled_CallsDeleteExpired(t *testing.T) {
	repo := &mockRepo{}
	s := systems.NewTrustedDevicesCleanupSystem(
		&internaltests.MockLogger{},
		configWithCleanup(true, 20*time.Millisecond),
		repo,
	)

	require.NoError(t, s.Init(context.Background()))
	defer func() { require.NoError(t, s.Close()) }()

	// Wait long enough for at least two cleanup ticks.
	assert.Eventually(t, func() bool {
		return repo.callCount.Load() >= 2
	}, 500*time.Millisecond, 10*time.Millisecond)
}

func TestInit_ZeroInterval_FallsBackToOneHour(t *testing.T) {
	// We just need Init to not panic and not start a ridiculously fast ticker.
	// Since the default interval is 1 hour, we can verify by starting, then
	// immediately closing — no cleanup call should have fired.
	repo := &mockRepo{}
	s := systems.NewTrustedDevicesCleanupSystem(
		&internaltests.MockLogger{},
		configWithCleanup(true, 0), // zero → falls back to time.Hour
		repo,
	)

	require.NoError(t, s.Init(context.Background()))
	time.Sleep(20 * time.Millisecond)
	require.NoError(t, s.Close())

	assert.Equal(t, int32(0), repo.callCount.Load())
}

func TestClose_StopsCleanupLoop(t *testing.T) {
	repo := &mockRepo{}
	s := systems.NewTrustedDevicesCleanupSystem(
		&internaltests.MockLogger{},
		configWithCleanup(true, 10*time.Millisecond),
		repo,
	)

	require.NoError(t, s.Init(context.Background()))

	// Let at least one tick fire.
	assert.Eventually(t, func() bool {
		return repo.callCount.Load() >= 1
	}, 300*time.Millisecond, 5*time.Millisecond)

	require.NoError(t, s.Close())

	// Record count at close.
	countAfterClose := repo.callCount.Load()

	// After a short wait, count must not increase.
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, countAfterClose, repo.callCount.Load())
}

func TestCleanup_RepoError_DoesNotPanic(t *testing.T) {
	repo := &mockRepo{err: errors.New("db unavailable")}
	s := systems.NewTrustedDevicesCleanupSystem(
		&internaltests.MockLogger{},
		configWithCleanup(true, 10*time.Millisecond),
		repo,
	)

	require.NoError(t, s.Init(context.Background()))

	// Error shouldn't stop the loop; it should keep ticking.
	assert.Eventually(t, func() bool {
		return repo.callCount.Load() >= 2
	}, 300*time.Millisecond, 5*time.Millisecond)

	require.NoError(t, s.Close())
}
