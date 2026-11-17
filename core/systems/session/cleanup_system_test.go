package session

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

func TestSessionCleanupSystem_Name(t *testing.T) {
	t.Parallel()

	system := NewSessionCleanupSystem(&internaltests.MockLogger{}, models.SessionConfig{}, &internaltests.MockSessionService{})

	assert.Equal(t, "SessionCleanupSystem", system.Name())
}

func TestSessionCleanupSystem_InitDisabled(t *testing.T) {
	t.Parallel()

	sessionService := &internaltests.MockSessionService{}
	system := NewSessionCleanupSystem(&internaltests.MockLogger{}, models.SessionConfig{AutoCleanup: false}, sessionService)

	require.NoError(t, system.Init(context.Background()))
	require.NoError(t, system.Close())

	sessionService.AssertNotCalled(t, "DeleteAllExpired", mock.Anything)
	sessionService.AssertNotCalled(t, "GetDistinctUserIDs", mock.Anything)
}

func TestSessionCleanupSystem_RunsAtInterval(t *testing.T) {
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
				sessionService := &internaltests.MockSessionService{}
				sessionService.On("DeleteAllExpired", mock.Anything).Return(nil)

				system := NewSessionCleanupSystem(&internaltests.MockLogger{}, models.SessionConfig{
					AutoCleanup:     true,
					CleanupInterval: tt.interval,
				}, sessionService)

				require.NoError(t, system.Init(context.Background()))

				time.Sleep(tt.advance)
				synctest.Wait()
				require.NoError(t, system.Close())

				sessionService.AssertNumberOfCalls(t, "DeleteAllExpired", tt.wantCleanup)
				sessionService.AssertNotCalled(t, "GetDistinctUserIDs", mock.Anything)
				sessionService.AssertExpectations(t)
			})
		})
	}
}

func TestSessionCleanupSystem_CloseStopsCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sessionService := &internaltests.MockSessionService{}
		sessionService.On("DeleteAllExpired", mock.Anything).Return(nil)

		system := NewSessionCleanupSystem(&internaltests.MockLogger{}, models.SessionConfig{
			AutoCleanup:     true,
			CleanupInterval: 10 * time.Millisecond,
		}, sessionService)

		require.NoError(t, system.Init(context.Background()))
		time.Sleep(15 * time.Millisecond)
		synctest.Wait()

		require.NoError(t, system.Close())
		callsAtClose := len(sessionService.Calls)

		time.Sleep(time.Hour)
		synctest.Wait()

		assert.Equal(t, callsAtClose, len(sessionService.Calls))
		sessionService.AssertExpectations(t)
	})
}

func TestSessionCleanupSystem_EnforcesMaxSessions(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sessionService := &internaltests.MockSessionService{}
		sessionService.On("DeleteAllExpired", mock.Anything).Return(nil)
		sessionService.On("GetDistinctUserIDs", mock.Anything).Return([]string{"user-1", "user-2"}, nil)
		sessionService.On("DeleteOldestByUserID", mock.Anything, "user-1", 2).Return(nil)
		sessionService.On("DeleteOldestByUserID", mock.Anything, "user-2", 2).Return(nil)

		system := NewSessionCleanupSystem(&internaltests.MockLogger{}, models.SessionConfig{
			AutoCleanup:        true,
			CleanupInterval:    10 * time.Millisecond,
			MaxSessionsPerUser: 2,
		}, sessionService)

		require.NoError(t, system.Init(context.Background()))
		time.Sleep(15 * time.Millisecond)
		synctest.Wait()
		require.NoError(t, system.Close())

		sessionService.AssertExpectations(t)
	})
}

func TestSessionCleanupSystem_CleanupErrorHandling(t *testing.T) {
	t.Parallel()

	errDelete := errors.New("delete failed")
	errDistinct := errors.New("distinct failed")
	errUser := errors.New("user delete failed")

	tests := []struct {
		name         string
		config       models.SessionConfig
		configure    func(*internaltests.MockSessionService)
		wantMessages []string
	}{
		{
			name:   "logs expired session deletion errors",
			config: models.SessionConfig{},
			configure: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("DeleteAllExpired", mock.Anything).Return(errDelete)
			},
			wantMessages: []string{"session expired cleanup failed"},
		},
		{
			name:   "logs distinct user ID errors when max sessions is enabled",
			config: models.SessionConfig{MaxSessionsPerUser: 2},
			configure: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("DeleteAllExpired", mock.Anything).Return(nil)
				sessionService.On("GetDistinctUserIDs", mock.Anything).Return(nil, errDistinct)
			},
			wantMessages: []string{"session max sessions enforcement failed"},
		},
		{
			name:   "logs per-user errors and continues with the next user",
			config: models.SessionConfig{MaxSessionsPerUser: 3},
			configure: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("DeleteAllExpired", mock.Anything).Return(nil)
				sessionService.On("GetDistinctUserIDs", mock.Anything).Return([]string{"user-1", "user-2"}, nil)
				sessionService.On("DeleteOldestByUserID", mock.Anything, "user-1", 3).Return(errUser)
				sessionService.On("DeleteOldestByUserID", mock.Anything, "user-2", 3).Return(nil)
			},
			wantMessages: []string{"failed to enforce max sessions for user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := &internaltests.RecordingLogger{}
			sessionService := &internaltests.MockSessionService{}
			tt.configure(sessionService)
			system := NewSessionCleanupSystem(logger, tt.config, sessionService)

			system.cleanup(context.Background())

			assert.Equal(t, tt.wantMessages, logger.ErrorCalls)
			sessionService.AssertExpectations(t)
		})
	}
}

func TestSessionCleanupSystem_CleanupSkipsMaxSessionsWhenDisabled(t *testing.T) {
	t.Parallel()

	sessionService := &internaltests.MockSessionService{}
	sessionService.On("DeleteAllExpired", mock.Anything).Return(nil)
	system := NewSessionCleanupSystem(&internaltests.MockLogger{}, models.SessionConfig{}, sessionService)

	system.cleanup(context.Background())

	sessionService.AssertNotCalled(t, "GetDistinctUserIDs", mock.Anything)
	sessionService.AssertNotCalled(t, "DeleteOldestByUserID", mock.Anything, mock.Anything, mock.Anything)
	sessionService.AssertExpectations(t)
}

func TestSessionCleanupSystem_CleanupSuccessLogsNothing(t *testing.T) {
	t.Parallel()

	logger := &internaltests.RecordingLogger{}
	sessionService := &internaltests.MockSessionService{}
	sessionService.On("DeleteAllExpired", mock.Anything).Return(nil)
	sessionService.On("GetDistinctUserIDs", mock.Anything).Return([]string{}, nil)
	system := NewSessionCleanupSystem(logger, models.SessionConfig{MaxSessionsPerUser: 1}, sessionService)

	system.cleanup(context.Background())

	assert.Empty(t, logger.ErrorCalls)
	sessionService.AssertExpectations(t)
}

func TestSessionCleanupSystem_EnforceMaxSessionsSkipsWhenDisabled(t *testing.T) {
	t.Parallel()

	sessionService := &internaltests.MockSessionService{}
	system := NewSessionCleanupSystem(&internaltests.MockLogger{}, models.SessionConfig{MaxSessionsPerUser: 0}, sessionService)

	require.NoError(t, system.enforceMaxSessionsPerUser(context.Background()))

	sessionService.AssertNotCalled(t, "GetDistinctUserIDs", mock.Anything)
}
