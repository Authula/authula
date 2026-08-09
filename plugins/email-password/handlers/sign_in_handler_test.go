package handlers

import (
	"errors"
	"maps"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	plugintests "github.com/Authula/authula/plugins/email-password/tests"
	"github.com/Authula/authula/plugins/email-password/types"
)

func TestSignInHandler(t *testing.T) {
	t.Parallel()

	user := &models.User{ID: "user-1", Email: "user@example.com", EmailVerified: false}
	session := &models.Session{ID: "session-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}
	actor := &models.Actor{ID: "user-1", Type: models.ActorUser}
	tests := []struct {
		name           string
		body           []byte
		actor          *models.Actor
		values         map[string]any
		pluginConfig   types.EmailPasswordPluginConfig
		prepare        func(*plugintests.MockSignInUseCase, *plugintests.MockSendEmailVerificationUseCase)
		expectedStatus int
	}{
		{name: "invalid_json", body: []byte("{"), expectedStatus: http.StatusUnprocessableEntity},
		{name: "existing_session_reuse", body: internaltests.MarshalToJSON(t, types.SignInRequest{Email: "user@example.com", Password: "password123"}), actor: actor, values: map[string]any{models.ContextSessionID.String(): "session-1"}, prepare: func(m *plugintests.MockSignInUseCase, _ *plugintests.MockSendEmailVerificationUseCase) {
			m.On("GetSessionByID", mock.Anything, "session-1").Return(session, nil).Once()
			m.On("GetUserByID", mock.Anything, "user-1").Return(user, nil).Once()
		}, expectedStatus: http.StatusOK},
		{name: "usecase_error", body: internaltests.MarshalToJSON(t, types.SignInRequest{Email: "user@example.com", Password: "password123"}), prepare: func(m *plugintests.MockSignInUseCase, _ *plugintests.MockSendEmailVerificationUseCase) {
			m.On("SignIn", mock.Anything, "user@example.com", "password123", (*string)(nil), mock.Anything, mock.Anything).Return((*types.SignInResult)(nil), errors.New("some error")).Once()
		}, expectedStatus: http.StatusUnauthorized},
		{name: "success", body: internaltests.MarshalToJSON(t, types.SignInRequest{Email: "user@example.com", Password: "password123"}), pluginConfig: types.EmailPasswordPluginConfig{RequireEmailVerification: true, SendEmailOnSignIn: false}, prepare: func(m *plugintests.MockSignInUseCase, _ *plugintests.MockSendEmailVerificationUseCase) {
			m.On("SignIn", mock.Anything, "user@example.com", "password123", (*string)(nil), mock.Anything, mock.Anything).Return(&types.SignInResult{User: user, Session: session, SessionToken: "session-token"}, nil).Once()
		}, expectedStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSignInUseCase := &plugintests.MockSignInUseCase{}
			mockSendEmailVerificationUseCase := &plugintests.MockSendEmailVerificationUseCase{}
			if tt.prepare != nil {
				tt.prepare(mockSignInUseCase, mockSendEmailVerificationUseCase)
			}
			handler := &SignInHandler{Logger: &internaltests.MockLogger{}, Config: &models.Config{Session: models.SessionConfig{ExpiresIn: time.Hour}}, PluginConfig: tt.pluginConfig, SignInUseCase: mockSignInUseCase, SendEmailVerificationUseCase: mockSendEmailVerificationUseCase}
			req, w, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodPost, "/email-password/sign-in", tt.body, tt.actor)
			if tt.values != nil {
				maps.Copy(reqCtx.Values, tt.values)
			}
			handler.Handler().ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			mockSignInUseCase.AssertExpectations(t)
			mockSendEmailVerificationUseCase.AssertExpectations(t)
		})
	}
}
