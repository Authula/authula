package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	coreservices "github.com/Authula/authula/core/services"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	rootservices "github.com/Authula/authula/services"
)

func newSessionServiceForTest() (rootservices.SessionService, *internaltests.MockSessionRepository) {
	repo := new(internaltests.MockSessionRepository)
	svc := coreservices.NewSessionService(repo, nil, nil, new(internaltests.MockLogger))
	return svc, repo
}

func TestSessionService_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name          string
		hashedToken   string
		ipAddress     *string
		userAgent     *string
		maxAge        time.Duration
		beforeHooks   []models.ServiceHook[models.Session]
		afterHooks    []models.ServiceHook[models.Session]
		repoErr       error
		expectCreate  bool
		wantErr       string
		wantLogs      int
		wantUserAgent *string
		wantIPAddress *string
	}{
		{
			name:          "creates a session with generated id and expiry",
			hashedToken:   "hashed-token",
			ipAddress:     new("127.0.0.1"),
			userAgent:     new("test-agent"),
			maxAge:        time.Hour,
			expectCreate:  true,
			wantUserAgent: new("test-agent"),
			wantIPAddress: new("127.0.0.1"),
		},
		{
			name:        "rejects an empty hashed token",
			hashedToken: "",
			wantErr:     "hashedToken cannot be empty",
		},
		{
			name:         "returns the repository error",
			hashedToken:  "hashed-token",
			maxAge:       time.Hour,
			repoErr:      errors.New("db down"),
			expectCreate: true,
			wantErr:      "db down",
		},
		{
			name:        "before create hook error blocks creation",
			hashedToken: "hashed-token",
			maxAge:      time.Hour,
			beforeHooks: []models.ServiceHook[models.Session]{
				func(*models.Session) error { return errors.New("blocked") },
			},
			wantErr: "blocked",
		},
		{
			name:        "before create hook mutation reaches the repository",
			hashedToken: "hashed-token",
			maxAge:      time.Hour,
			beforeHooks: []models.ServiceHook[models.Session]{
				func(session *models.Session) error {
					session.UserAgent = new("hooked-agent")
					return nil
				},
			},
			expectCreate:  true,
			wantUserAgent: new("hooked-agent"),
		},
		{
			name:        "after create hook error is logged but does not fail",
			hashedToken: "hashed-token",
			maxAge:      time.Hour,
			afterHooks: []models.ServiceHook[models.Session]{
				func(*models.Session) error { return errors.New("notify failed") },
			},
			expectCreate: true,
			wantLogs:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := new(internaltests.MockSessionRepository)
			logger := new(internaltests.RecordingLogger)
			var passedSession *models.Session

			if tt.expectCreate {
				repo.On("Create", mock.Anything, mock.MatchedBy(func(session *models.Session) bool {
					passedSession = session
					return true
				})).Return(&models.Session{ID: "session-1", UserID: "user-1"}, tt.repoErr).Once()
			}

			hooks := newSessionHooks()
			for _, hook := range tt.beforeHooks {
				hooks.Sessions.RegisterBeforeCreate(hook)
			}
			for _, hook := range tt.afterHooks {
				hooks.Sessions.RegisterAfterCreate(hook)
			}

			svc := coreservices.NewSessionService(repo, nil, hooks, logger)
			created, err := svc.Create(ctx, "user-1", tt.hashedToken, tt.ipAddress, tt.userAgent, tt.maxAge)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, created)
				if !tt.expectCreate {
					repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
				}
				assert.Len(t, logger.ErrorCalls, tt.wantLogs)
				repo.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, created)
			require.NotNil(t, passedSession)

			assert.Equal(t, "session-1", created.ID)
			assert.NotEmpty(t, passedSession.ID)
			assert.Equal(t, "user-1", passedSession.UserID)
			assert.Equal(t, tt.hashedToken, passedSession.Token)
			assert.Equal(t, tt.wantIPAddress, passedSession.IPAddress)
			assert.Equal(t, tt.wantUserAgent, passedSession.UserAgent)
			assert.WithinDuration(t, time.Now().UTC().Add(tt.maxAge), passedSession.ExpiresAt, 5*time.Second)

			assert.Len(t, logger.ErrorCalls, tt.wantLogs)
			repo.AssertExpectations(t)
		})
	}
}

func TestSessionService_GetByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	expected := &models.Session{ID: "session-1"}
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		session *models.Session
		repoErr error
		wantErr error
	}{
		{
			name:    "returns the session",
			session: expected,
		},
		{
			name:    "propagates the not found sentinel",
			repoErr: coreerrors.ErrSessionNotFound,
			wantErr: coreerrors.ErrSessionNotFound,
		},
		{
			name:    "propagates other errors",
			repoErr: dbErr,
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newSessionServiceForTest()
			repo.On("GetByID", mock.Anything, "session-1").Return(tt.session, tt.repoErr).Once()

			got, err := svc.GetByID(ctx, "session-1")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.session, got)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestSessionService_GetByToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	expected := &models.Session{ID: "session-1", Token: "hashed-token"}
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		session *models.Session
		repoErr error
		wantErr error
	}{
		{
			name:    "returns the session",
			session: expected,
		},
		{
			name:    "propagates the not found sentinel",
			repoErr: coreerrors.ErrSessionNotFound,
			wantErr: coreerrors.ErrSessionNotFound,
		},
		{
			name:    "propagates other errors",
			repoErr: dbErr,
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newSessionServiceForTest()
			repo.On("GetByToken", mock.Anything, "hashed-token").Return(tt.session, tt.repoErr).Once()

			got, err := svc.GetByToken(ctx, "hashed-token")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.session, got)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestSessionService_GetByUserID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	expected := &models.Session{ID: "session-1", UserID: "user-1"}
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		session *models.Session
		repoErr error
		wantErr error
	}{
		{
			name:    "returns the session",
			session: expected,
		},
		{
			name:    "propagates the not found sentinel",
			repoErr: coreerrors.ErrSessionNotFound,
			wantErr: coreerrors.ErrSessionNotFound,
		},
		{
			name:    "propagates other errors",
			repoErr: dbErr,
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newSessionServiceForTest()
			repo.On("GetByUserID", mock.Anything, "user-1").Return(tt.session, tt.repoErr).Once()

			got, err := svc.GetByUserID(ctx, "user-1")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.session, got)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestSessionService_Update(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name          string
		beforeHooks   []models.ServiceHook[models.Session]
		afterHooks    []models.ServiceHook[models.Session]
		repoErr       error
		expectUpdate  bool
		wantErr       string
		wantLogs      int
		wantUserAgent *string
	}{
		{
			name:         "updates the session",
			expectUpdate: true,
		},
		{
			name:         "returns the repository error",
			repoErr:      errors.New("db down"),
			expectUpdate: true,
			wantErr:      "db down",
		},
		{
			name: "before update hook error blocks the update",
			beforeHooks: []models.ServiceHook[models.Session]{
				func(*models.Session) error { return errors.New("blocked") },
			},
			wantErr: "blocked",
		},
		{
			name: "before update hook mutation reaches the repository",
			beforeHooks: []models.ServiceHook[models.Session]{
				func(session *models.Session) error {
					session.UserAgent = new("hooked-agent")
					return nil
				},
			},
			expectUpdate:  true,
			wantUserAgent: new("hooked-agent"),
		},
		{
			name: "after update hook error is logged but does not fail",
			afterHooks: []models.ServiceHook[models.Session]{
				func(*models.Session) error { return errors.New("sync failed") },
			},
			expectUpdate: true,
			wantLogs:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := new(internaltests.MockSessionRepository)
			logger := new(internaltests.RecordingLogger)
			session := &models.Session{ID: "session-1", UserID: "user-1"}
			var passedSession *models.Session

			if tt.expectUpdate {
				repo.On("Update", mock.Anything, mock.MatchedBy(func(session *models.Session) bool {
					passedSession = session
					return true
				})).Return(session, tt.repoErr).Once()
			}

			hooks := newSessionHooks()
			for _, hook := range tt.beforeHooks {
				hooks.Sessions.RegisterBeforeUpdate(hook)
			}
			for _, hook := range tt.afterHooks {
				hooks.Sessions.RegisterAfterUpdate(hook)
			}

			svc := coreservices.NewSessionService(repo, nil, hooks, logger)
			updated, err := svc.Update(ctx, session)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, updated)
				if !tt.expectUpdate {
					repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
				}
				assert.Len(t, logger.ErrorCalls, tt.wantLogs)
				repo.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, session, updated)
			require.NotNil(t, passedSession)
			if tt.wantUserAgent != nil {
				assert.Equal(t, tt.wantUserAgent, passedSession.UserAgent)
			}

			assert.Len(t, logger.ErrorCalls, tt.wantLogs)
			repo.AssertExpectations(t)
		})
	}
}

func TestSessionService_Delete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{
			name: "deletes the session",
		},
		{
			name:    "propagates repository errors",
			repoErr: dbErr,
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newSessionServiceForTest()
			repo.On("Delete", mock.Anything, "session-1").Return(tt.repoErr).Once()

			err := svc.Delete(ctx, "session-1")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestSessionService_DeleteAllByUserID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{
			name: "deletes every session for the user",
		},
		{
			name:    "propagates repository errors",
			repoErr: dbErr,
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newSessionServiceForTest()
			repo.On("DeleteByUserID", mock.Anything, "user-1").Return(tt.repoErr).Once()

			err := svc.DeleteAllByUserID(ctx, "user-1")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestSessionService_DeleteAllExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{
			name: "deletes every expired session",
		},
		{
			name:    "propagates repository errors",
			repoErr: dbErr,
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newSessionServiceForTest()
			repo.On("DeleteExpired", mock.Anything).Return(tt.repoErr).Once()

			err := svc.DeleteAllExpired(ctx)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestSessionService_GetDistinctUserIDs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	expected := []string{"user-1", "user-2"}
	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		userIDs []string
		repoErr error
		wantErr error
	}{
		{
			name:    "returns the distinct user ids",
			userIDs: expected,
		},
		{
			name:    "propagates repository errors",
			repoErr: dbErr,
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newSessionServiceForTest()
			repo.On("GetDistinctUserIDs", mock.Anything).Return(tt.userIDs, tt.repoErr).Once()

			got, err := svc.GetDistinctUserIDs(ctx)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.userIDs, got)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestSessionService_DeleteOldestByUserID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbErr := errors.New("db down")

	tests := []struct {
		name     string
		maxCount int
		repoErr  error
		wantErr  error
	}{
		{
			name:     "deletes the oldest sessions for the user",
			maxCount: 3,
		},
		{
			name:     "propagates repository errors",
			maxCount: 3,
			repoErr:  dbErr,
			wantErr:  dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newSessionServiceForTest()
			repo.On("DeleteOldestByUserID", mock.Anything, "user-1", tt.maxCount).Return(tt.repoErr).Once()

			err := svc.DeleteOldestByUserID(ctx, "user-1", tt.maxCount)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}
