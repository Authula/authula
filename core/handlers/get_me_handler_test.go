package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/core/handlers"
	"github.com/Authula/authula/core/usecases"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

func TestGetMeHandler(t *testing.T) {
	t.Parallel()

	const userID = "user-1"

	user := &models.User{ID: userID, Name: "Test User", Email: "user@example.com"}
	session := &models.Session{ID: "session-1", UserID: userID}

	tests := []struct {
		name            string
		sessionID       *string
		user            *models.User
		userErr         error
		session         *models.Session
		sessionErr      error
		expectedStatus  int
		expectSession   bool
		expectedMessage string
	}{
		{
			name:           "stateless caller receives the user and a null session",
			user:           user,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "session caller receives the user and the session",
			sessionID:      new("session-1"),
			user:           user,
			session:        session,
			expectedStatus: http.StatusOK,
			expectSession:  true,
		},
		{
			name:            "missing session returns unauthorized",
			sessionID:       new("session-1"),
			user:            user,
			sessionErr:      coreerrors.ErrSessionNotFound,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "unauthorized",
		},
		{
			name:            "deleted user returns unauthorized",
			sessionID:       new("session-1"),
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "unauthorized",
		},
		{
			name:            "unexpected error returns internal server error",
			sessionID:       new("session-1"),
			user:            user,
			sessionErr:      errors.New("some error"),
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "some error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userService := &internaltests.MockUserService{}
			sessionService := &internaltests.MockSessionService{}

			userService.On("GetByID", mock.Anything, userID).Return(tt.user, tt.userErr).Once()
			if tt.sessionID != nil {
				sessionService.On("GetByID", mock.Anything, *tt.sessionID).Return(tt.session, tt.sessionErr).Once()
			}

			handler := &handlers.GetMeHandler{
				UseCase: &usecases.GetMeUseCase{
					Logger:         &internaltests.MockLogger{},
					UserService:    userService,
					SessionService: sessionService,
				},
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/me", nil, new(userID))
			if tt.sessionID != nil {
				reqCtx.Values[models.ContextSessionID.String()] = *tt.sessionID
			}

			handler.Handle()(w, req)

			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tt.expectedStatus, tt.expectedMessage)
			}
			if tt.expectedStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &body))
				require.Equal(t, userID, body["user"].(map[string]any)["id"])
				if tt.expectSession {
					require.Equal(t, "session-1", body["session"].(map[string]any)["id"])
				} else {
					require.Nil(t, body["session"])
				}
			}

			userService.AssertExpectations(t)
			sessionService.AssertExpectations(t)
		})
	}
}
