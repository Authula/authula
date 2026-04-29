package handlers

import (
	"errors"
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

func TestSignUpHandler(t *testing.T) {
	t.Parallel()

	user := &models.User{ID: "user-1", Email: "user@example.com"}
	session := &models.Session{ID: "session-1", UserID: "user-1"}
	tests := []struct {
		name           string
		body           []byte
		pluginConfig   types.EmailPasswordPluginConfig
		prepare        func(*plugintests.MockSignUpUseCase, *plugintests.MockSendEmailVerificationUseCase)
		expectedStatus int
	}{
		{name: "invalid_json", body: []byte("{"), expectedStatus: http.StatusBadRequest},
		{name: "usecase_error", body: internaltests.MarshalToJSON(t, types.SignUpRequest{Name: "Jane", Email: "jane@example.com", Password: "password123"}), prepare: func(m *plugintests.MockSignUpUseCase, _ *plugintests.MockSendEmailVerificationUseCase) {
			m.On("SignUp", mock.Anything, "Jane", "jane@example.com", "password123", (*string)(nil), mock.Anything, (*string)(nil), mock.Anything, mock.Anything).Return((*types.SignUpResult)(nil), errors.New("boom")).Once()
		}, expectedStatus: http.StatusForbidden},
		{name: "success_auto_sign_in_disabled", body: internaltests.MarshalToJSON(t, types.SignUpRequest{Name: "Jane", Email: "jane@example.com", Password: "password123"}), pluginConfig: types.EmailPasswordPluginConfig{AutoSignIn: false}, prepare: func(m *plugintests.MockSignUpUseCase, _ *plugintests.MockSendEmailVerificationUseCase) {
			m.On("SignUp", mock.Anything, "Jane", "jane@example.com", "password123", (*string)(nil), mock.Anything, (*string)(nil), mock.Anything, mock.Anything).Return(&types.SignUpResult{User: user}, nil).Once()
		}, expectedStatus: http.StatusCreated},
		{name: "success_auto_sign_in_enabled", body: internaltests.MarshalToJSON(t, types.SignUpRequest{Name: "Jane", Email: "jane@example.com", Password: "password123"}), pluginConfig: types.EmailPasswordPluginConfig{AutoSignIn: true}, prepare: func(m *plugintests.MockSignUpUseCase, _ *plugintests.MockSendEmailVerificationUseCase) {
			m.On("SignUp", mock.Anything, "Jane", "jane@example.com", "password123", (*string)(nil), mock.Anything, (*string)(nil), mock.Anything, mock.Anything).Return(&types.SignUpResult{User: user, Session: session, SessionToken: "session-token"}, nil).Once()
		}, expectedStatus: http.StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			su := &plugintests.MockSignUpUseCase{}
			send := &plugintests.MockSendEmailVerificationUseCase{}
			if tt.prepare != nil {
				tt.prepare(su, send)
			}
			handler := &SignUpHandler{Logger: &internaltests.MockLogger{}, Config: &models.Config{Session: models.SessionConfig{ExpiresIn: time.Hour}}, PluginConfig: tt.pluginConfig, SignUpUseCase: su, SendEmailVerificationUseCase: send}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/email-password/sign-up", tt.body, nil)
			handler.Handler().ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			su.AssertExpectations(t)
			send.AssertExpectations(t)
		})
	}
}
