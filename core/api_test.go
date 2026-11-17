package core_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/core"
	coreerrors "github.com/Authula/authula/core/errors"
	coretypes "github.com/Authula/authula/core/types"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

func TestBuildUseCases(t *testing.T) {
	t.Parallel()

	logger := &internaltests.MockLogger{}
	userService := &internaltests.MockUserService{}
	sessionService := &internaltests.MockSessionService{}

	useCases := core.BuildUseCases(logger, userService, sessionService)

	require.NotNil(t, useCases)
	require.NotNil(t, useCases.GetMeUseCase)
	require.NotNil(t, useCases.SignOutUseCase)
	assert.Same(t, logger, useCases.GetMeUseCase.Logger)
	assert.Same(t, userService, useCases.GetMeUseCase.UserService)
	assert.Same(t, sessionService, useCases.GetMeUseCase.SessionService)
	assert.Same(t, logger, useCases.SignOutUseCase.Logger)
	assert.Same(t, sessionService, useCases.SignOutUseCase.SessionService)
}

func TestNewCoreAPI_GetMe(t *testing.T) {
	t.Parallel()

	const userID = "user-1"

	user := &models.User{ID: userID, Name: "Test User", Email: "user@example.com"}
	session := &models.Session{ID: "session-1", UserID: userID, Token: "token-1"}
	foreignSession := &models.Session{ID: "session-2", UserID: "someone-else", Token: "token-2"}
	errUser := errors.New("user lookup failed")

	tests := []struct {
		name              string
		sessionID         *string
		configure         func(*internaltests.MockUserService, *internaltests.MockSessionService)
		wantUser          *models.User
		wantSession       *models.Session
		wantErr           error
		expectSessionCall bool
	}{
		{
			name: "returns the user for a stateless caller",
			configure: func(userService *internaltests.MockUserService, _ *internaltests.MockSessionService) {
				userService.On("GetByID", mock.Anything, userID).Return(user, nil)
			},
			wantUser: user,
		},
		{
			name:      "returns the session for a session caller",
			sessionID: new("session-1"),
			configure: func(userService *internaltests.MockUserService, sessionService *internaltests.MockSessionService) {
				userService.On("GetByID", mock.Anything, userID).Return(user, nil)
				sessionService.On("GetByID", mock.Anything, "session-1").Return(session, nil)
			},
			wantUser:          user,
			wantSession:       session,
			expectSessionCall: true,
		},
		{
			name: "propagates user lookup errors",
			configure: func(userService *internaltests.MockUserService, _ *internaltests.MockSessionService) {
				userService.On("GetByID", mock.Anything, userID).Return(nil, errUser)
			},
			wantErr: errUser,
		},
		{
			name:      "rejects a session owned by another user",
			sessionID: new("session-2"),
			configure: func(userService *internaltests.MockUserService, sessionService *internaltests.MockSessionService) {
				userService.On("GetByID", mock.Anything, userID).Return(user, nil)
				sessionService.On("GetByID", mock.Anything, "session-2").Return(foreignSession, nil)
			},
			wantErr:           coreerrors.ErrUnauthorized,
			expectSessionCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userService := &internaltests.MockUserService{}
			sessionService := &internaltests.MockSessionService{}
			tt.configure(userService, sessionService)
			api := core.NewCoreAPI(&internaltests.MockLogger{}, userService, sessionService)

			result, err := api.GetMe(context.Background(), userID, tt.sessionID)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.wantUser, result.User)
				assert.Equal(t, tt.wantSession, result.Session)
			}
			if !tt.expectSessionCall {
				sessionService.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
			}
			userService.AssertExpectations(t)
			sessionService.AssertExpectations(t)
		})
	}
}

func TestNewCoreAPI_SignOut(t *testing.T) {
	t.Parallel()

	const userID = "user-1"

	errDelete := errors.New("delete failed")

	tests := []struct {
		name               string
		currentSessionID   *string
		requestedSessionID *string
		signOutAll         *bool
		configure          func(*internaltests.MockSessionService)
		want               *coretypes.SignOutResult
		wantErr            error
	}{
		{
			name:       "signs out all sessions",
			signOutAll: new(true),
			configure: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("DeleteAllByUserID", mock.Anything, userID).Return(nil)
			},
			want: &coretypes.SignOutResult{Message: "signed out from all sessions", ClearedCurrentSession: true},
		},
		{
			name:             "deletes the current session",
			currentSessionID: new("session-current"),
			configure: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("Delete", mock.Anything, "session-current").Return(nil)
			},
			want: &coretypes.SignOutResult{Message: "signed out", ClearedCurrentSession: true},
		},
		{
			name:       "propagates delete errors",
			signOutAll: new(true),
			configure: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("DeleteAllByUserID", mock.Anything, userID).Return(errDelete)
			},
			wantErr: errDelete,
		},
		{
			name: "refuses to guess a session",
			configure: func(*internaltests.MockSessionService) {
			},
			wantErr: coreerrors.ErrNoSessionToSignOut,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sessionService := &internaltests.MockSessionService{}
			tt.configure(sessionService)
			api := core.NewCoreAPI(&internaltests.MockLogger{}, &internaltests.MockUserService{}, sessionService)

			result, err := api.SignOut(context.Background(), userID, tt.currentSessionID, tt.requestedSessionID, tt.signOutAll)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
			sessionService.AssertExpectations(t)
		})
	}
}

func TestCoreRoutes(t *testing.T) {
	t.Parallel()

	userService := &internaltests.MockUserService{}
	sessionService := &internaltests.MockSessionService{}

	routes := core.CoreRoutes(&internaltests.MockLogger{}, userService, sessionService)

	require.Len(t, routes, 2)

	assert.Equal(t, http.MethodGet, routes[0].Method)
	assert.Equal(t, "/me", routes[0].Path)
	assert.Len(t, routes[0].Middleware, 1)
	require.NotNil(t, routes[0].Handler)

	assert.Equal(t, http.MethodPost, routes[1].Method)
	assert.Equal(t, "/sign-out", routes[1].Path)
	assert.Len(t, routes[1].Middleware, 1)
	require.NotNil(t, routes[1].Handler)

	t.Run("get me route delegates to the user service", func(t *testing.T) {
		user := &models.User{ID: "user-1", Name: "Test User", Email: "user@example.com"}
		userService.On("GetByID", mock.Anything, "user-1").Return(user, nil)

		req, recorder, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/me", nil, new("user-1"))

		routes[0].Handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
		response := internaltests.DecodeResponseJSON[coretypes.GetMeResponse](t, reqCtx)
		assert.Equal(t, user, response.User)
		userService.AssertExpectations(t)
	})

	t.Run("sign out route delegates to the session service", func(t *testing.T) {
		sessionService.On("DeleteAllByUserID", mock.Anything, "user-1").Return(nil)

		req, recorder, reqCtx := internaltests.NewHandlerRequest(
			t,
			http.MethodPost,
			"/sign-out",
			[]byte(`{"sign_out_all":true}`),
			new("user-1"),
		)

		routes[1].Handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
		assert.Equal(t, true, reqCtx.Values[models.ContextAuthSignOut.String()])
		response := internaltests.DecodeResponseJSON[coretypes.SignOutResponse](t, reqCtx)
		assert.Equal(t, "signed out from all sessions", response.Message)
		sessionService.AssertExpectations(t)
	})
}
