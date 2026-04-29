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
	userID := "user-1"
	tests := []struct {
		name           string
		body           []byte
		userID         *string
		values         map[string]any
		pluginConfig   types.EmailPasswordPluginConfig
		prepare        func(*plugintests.MockSignInUseCase, *plugintests.MockSendEmailVerificationUseCase)
		expectedStatus int
	}{
		{name: "invalid_json", body: []byte("{"), expectedStatus: http.StatusUnprocessableEntity},
		{name: "existing_session_reuse", body: internaltests.MarshalToJSON(t, types.SignInRequest{Email: "user@example.com", Password: "password123"}), userID: &userID, values: map[string]any{models.ContextSessionID.String(): "session-1"}, prepare: func(m *plugintests.MockSignInUseCase, _ *plugintests.MockSendEmailVerificationUseCase) {
			m.On("GetSessionByID", mock.Anything, "session-1").Return(session, nil).Once()
			m.On("GetUserByID", mock.Anything, "user-1").Return(user, nil).Once()
		}, expectedStatus: http.StatusOK},
		{name: "usecase_error", body: internaltests.MarshalToJSON(t, types.SignInRequest{Email: "user@example.com", Password: "password123"}), prepare: func(m *plugintests.MockSignInUseCase, _ *plugintests.MockSendEmailVerificationUseCase) {
			m.On("SignIn", mock.Anything, "user@example.com", "password123", (*string)(nil), mock.Anything, mock.Anything).Return((*types.SignInResult)(nil), errors.New("boom")).Once()
		}, expectedStatus: http.StatusUnauthorized},
		{name: "success", body: internaltests.MarshalToJSON(t, types.SignInRequest{Email: "user@example.com", Password: "password123"}), pluginConfig: types.EmailPasswordPluginConfig{RequireEmailVerification: true, SendEmailOnSignIn: false}, prepare: func(m *plugintests.MockSignInUseCase, _ *plugintests.MockSendEmailVerificationUseCase) {
			m.On("SignIn", mock.Anything, "user@example.com", "password123", (*string)(nil), mock.Anything, mock.Anything).Return(&types.SignInResult{User: user, Session: session, SessionToken: "session-token"}, nil).Once()
		}, expectedStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			si := &plugintests.MockSignInUseCase{}
			send := &plugintests.MockSendEmailVerificationUseCase{}
			if tt.prepare != nil {
				tt.prepare(si, send)
			}
			handler := &SignInHandler{Logger: &internaltests.MockLogger{}, Config: &models.Config{Session: models.SessionConfig{ExpiresIn: time.Hour}}, PluginConfig: tt.pluginConfig, SignInUseCase: si, SendEmailVerificationUseCase: send}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/email-password/sign-in", tt.body, tt.userID)
			if tt.values != nil {
				maps.Copy(reqCtx.Values, tt.values)
			}
			handler.Handler().ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			si.AssertExpectations(t)
			send.AssertExpectations(t)
		})
	}
}
