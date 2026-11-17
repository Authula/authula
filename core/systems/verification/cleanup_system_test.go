package verification

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

func TestVerificationCleanupSystem_Name(t *testing.T) {
	t.Parallel()

	system := NewVerificationCleanupSystem(&internaltests.MockLogger{}, models.VerificationConfig{}, &internaltests.MockVerificationService{})

	assert.Equal(t, "VerificationCleanupSystem", system.Name())
}

func TestVerificationCleanupSystem_InitDisabled(t *testing.T) {
	t.Parallel()

	verificationService := &internaltests.MockVerificationService{}
	system := NewVerificationCleanupSystem(&internaltests.MockLogger{}, models.VerificationConfig{AutoCleanup: false}, verificationService)

	require.NoError(t, system.Init(context.Background()))
	require.NoError(t, system.Close())

	verificationService.AssertNotCalled(t, "DeleteExpired", mock.Anything)
}

func TestVerificationCleanupSystem_RunsAtInterval(t *testing.T) {
	tests := []struct {
		name        string
		interval    time.Duration
		advance     time.Duration
		wantCleanup int
	}{
		{
			name:        "custom interval",
			interval:    10 * time.Millisecond,
			advance:     25 * time.Millisecond,
			wantCleanup: 2,
		},
		{
			name:        "zero interval falls back to one minute",
			interval:    0,
			advance:     90 * time.Second,
			wantCleanup: 1,
		},
		{
			name:        "negative interval falls back to one minute",
			interval:    -time.Second,
			advance:     2 * time.Minute,
			wantCleanup: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				verificationService := &internaltests.MockVerificationService{}
				verificationService.On("DeleteExpired", mock.Anything).Return(nil)

				system := NewVerificationCleanupSystem(&internaltests.MockLogger{}, models.VerificationConfig{
					AutoCleanup:     true,
					CleanupInterval: tt.interval,
				}, verificationService)

				require.NoError(t, system.Init(context.Background()))

				time.Sleep(tt.advance)
				synctest.Wait()
				require.NoError(t, system.Close())

				verificationService.AssertNumberOfCalls(t, "DeleteExpired", tt.wantCleanup)
				verificationService.AssertExpectations(t)
			})
		})
	}
}

func TestVerificationCleanupSystem_CloseStopsCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		verificationService := &internaltests.MockVerificationService{}
		verificationService.On("DeleteExpired", mock.Anything).Return(nil)

		system := NewVerificationCleanupSystem(&internaltests.MockLogger{}, models.VerificationConfig{
			AutoCleanup:     true,
			CleanupInterval: 10 * time.Millisecond,
		}, verificationService)

		require.NoError(t, system.Init(context.Background()))
		time.Sleep(15 * time.Millisecond)
		synctest.Wait()

		require.NoError(t, system.Close())
		callsAtClose := len(verificationService.Calls)

		time.Sleep(time.Hour)
		synctest.Wait()

		assert.Equal(t, callsAtClose, len(verificationService.Calls))
		verificationService.AssertExpectations(t)
	})
}

func TestVerificationCleanupSystem_LogsCleanupErrors(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		logger := &internaltests.RecordingLogger{}
		verificationService := &internaltests.MockVerificationService{}
		verificationService.On("DeleteExpired", mock.Anything).Return(errors.New("db unavailable"))

		system := NewVerificationCleanupSystem(logger, models.VerificationConfig{
			AutoCleanup:     true,
			CleanupInterval: 10 * time.Millisecond,
		}, verificationService)

		require.NoError(t, system.Init(context.Background()))
		time.Sleep(15 * time.Millisecond)
		synctest.Wait()
		require.NoError(t, system.Close())

		verificationService.AssertExpectations(t)
		assert.Equal(t, []string{"Verification expired cleanup failed"}, logger.ErrorCalls)
	})
}
