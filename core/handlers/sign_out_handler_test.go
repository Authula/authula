package handlers_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/Authula/authula/core/handlers"
	"github.com/Authula/authula/core/usecases"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

func TestSignOutHandler(t *testing.T) {
	t.Parallel()

	const (
		userID             = "user-1"
		currentSessionID   = "session-current"
		requestedSessionID = "session-requested"
	)

	tests := []struct {
		name             string
		body             []byte
		currentSessionID *string
		setup            func(sessionService *internaltests.MockSessionService)
		expectedStatus   int
		expectedMessage  string
		expectSignOut    bool
	}{
		{
			name:             "no body signs out the current session and clears the cookie",
			currentSessionID: new(currentSessionID),
			setup: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("Delete", mock.Anything, currentSessionID).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectSignOut:  true,
		},
		{
			name:             "signing out another own session by id does not clear the cookie",
			body:             []byte(`{"session_id":"` + requestedSessionID + `"}`),
			currentSessionID: new(currentSessionID),
			setup: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("GetByID", mock.Anything, requestedSessionID).Return(&models.Session{ID: requestedSessionID, UserID: userID}, nil).Once()
				sessionService.On("Delete", mock.Anything, requestedSessionID).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectSignOut:  false,
		},
		{
			name:             "signing out the current session by id clears the cookie",
			body:             []byte(`{"session_id":"` + currentSessionID + `"}`),
			currentSessionID: new(currentSessionID),
			setup: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("GetByID", mock.Anything, currentSessionID).Return(&models.Session{ID: currentSessionID, UserID: userID}, nil).Once()
				sessionService.On("Delete", mock.Anything, currentSessionID).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectSignOut:  true,
		},
		{
			name:             "sign out all clears the cookie",
			body:             []byte(`{"sign_out_all":true}`),
			currentSessionID: new(currentSessionID),
			setup: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("DeleteAllByUserID", mock.Anything, userID).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectSignOut:  true,
		},
		{
			name:             "session owned by another user returns forbidden",
			body:             []byte(`{"session_id":"` + requestedSessionID + `"}`),
			currentSessionID: new(currentSessionID),
			setup: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("GetByID", mock.Anything, requestedSessionID).Return(&models.Session{ID: requestedSessionID, UserID: "user-2"}, nil).Once()
			},
			expectedStatus:  http.StatusForbidden,
			expectedMessage: "forbidden",
		},
		{
			name:             "unknown session id returns not found",
			body:             []byte(`{"session_id":"` + requestedSessionID + `"}`),
			currentSessionID: new(currentSessionID),
			setup: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("GetByID", mock.Anything, requestedSessionID).Return((*models.Session)(nil), nil).Once()
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found",
		},
		{
			name:             "unexpected error returns internal server error",
			currentSessionID: new(currentSessionID),
			setup: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("Delete", mock.Anything, currentSessionID).Return(errors.New("db down")).Once()
			},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "failed to sign out",
		},
		{
			name:            "no current session and no body is rejected",
			setup:           func(sessionService *internaltests.MockSessionService) {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "no session to sign out: provide session_id or sign_out_all",
		},
		{
			name: "sign out all works without a current session",
			body: []byte(`{"sign_out_all":true}`),
			setup: func(sessionService *internaltests.MockSessionService) {
				sessionService.On("DeleteAllByUserID", mock.Anything, userID).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectSignOut:  true,
		},
		{
			name:            "empty session id is rejected",
			body:            []byte(`{"session_id":"  "}`),
			setup:           func(sessionService *internaltests.MockSessionService) {},
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedMessage: "session_id cannot be empty if provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sessionService := &internaltests.MockSessionService{}
			tt.setup(sessionService)

			handler := &handlers.SignOutHandler{
				UseCase: &usecases.SignOutUseCase{
					Logger:         &internaltests.MockLogger{},
					SessionService: sessionService,
				},
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/sign-out", tt.body, new(userID))
			if tt.currentSessionID != nil {
				reqCtx.Values[models.ContextSessionID.String()] = *tt.currentSessionID
			}

			handler.Handle()(w, req)

			if reqCtx.ResponseStatus != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, reqCtx.ResponseStatus)
			}
			if tt.expectedMessage != "" {
				internaltests.AssertErrorMessage(t, reqCtx, tt.expectedStatus, tt.expectedMessage)
			}

			signedOut, _ := reqCtx.Values[models.ContextAuthSignOut.String()].(bool)
			if signedOut != tt.expectSignOut {
				t.Fatalf("expected auth.sign_out flag %v, got %v", tt.expectSignOut, signedOut)
			}

			sessionService.AssertExpectations(t)
		})
	}
}
