package handlers

import (
	"context"
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

func TestVerifyEmailHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		path           string
		prepare        func(*plugintests.MockVerifyEmailUseCase)
		expectedStatus int
	}{
		{name: "missing_token", path: "/email-password/verify-email", expectedStatus: http.StatusUnprocessableEntity},
		{name: "usecase_error", path: "/email-password/verify-email?token=token-1", prepare: func(m *plugintests.MockVerifyEmailUseCase) {
			m.On("VerifyEmail", mock.Anything, "token-1").Return(models.VerificationType(""), errors.New("boom")).Once()
		}, expectedStatus: http.StatusUnprocessableEntity},
		{name: "success_without_callback", path: "/email-password/verify-email?token=token-1", prepare: func(m *plugintests.MockVerifyEmailUseCase) {
			m.On("VerifyEmail", mock.Anything, "token-1").Return(models.TypeEmailVerification, nil).Once()
		}, expectedStatus: http.StatusOK},
		{name: "password_reset_redirect", path: "/email-password/verify-email?token=token-1&callback_url=https://example.com/callback", prepare: func(m *plugintests.MockVerifyEmailUseCase) {
			m.On("VerifyEmail", mock.Anything, "token-1").Return(models.TypePasswordResetRequest, nil).Once()
		}, expectedStatus: http.StatusFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ve := &plugintests.MockVerifyEmailUseCase{}
			if tt.prepare != nil {
				tt.prepare(ve)
			}
			handler := &VerifyEmailHandler{VerifyEmailUseCase: ve}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, tt.path, nil, nil)
			reqCtx.Request = req
			handler.Handler().ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			ve.AssertExpectations(t)
		})
	}
}

func TestSendEmailVerificationHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		body           []byte
		prepare        func(*plugintests.MockSendEmailVerificationUseCase)
		expectedStatus int
	}{
		{name: "invalid_json", body: []byte("{"), expectedStatus: http.StatusBadRequest},
		{name: "usecase_error", body: internaltests.MarshalToJSON(t, types.SendEmailVerificationRequest{Email: "user@example.com"}), prepare: func(m *plugintests.MockSendEmailVerificationUseCase) {
			m.On("Send", mock.Anything, "user@example.com", (*string)(nil)).Return(errors.New("boom")).Once()
		}, expectedStatus: http.StatusInternalServerError},
		{name: "success", body: internaltests.MarshalToJSON(t, types.SendEmailVerificationRequest{Email: "user@example.com"}), prepare: func(m *plugintests.MockSendEmailVerificationUseCase) {
			m.On("Send", mock.Anything, "user@example.com", (*string)(nil)).Return(nil).Once()
		}, expectedStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := &plugintests.MockSendEmailVerificationUseCase{}
			if tt.prepare != nil {
				tt.prepare(uc)
			}
			handler := &SendEmailVerificationHandler{UseCase: uc}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/email-password/send-email-verification", tt.body, nil)
			handler.Handler().ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			uc.AssertExpectations(t)
		})
	}
}

func TestRequestPasswordResetHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		body           []byte
		prepare        func(*plugintests.MockRequestPasswordResetUseCase)
		expectedStatus int
	}{
		{name: "invalid_json", body: []byte("{"), expectedStatus: http.StatusUnprocessableEntity},
		{name: "usecase_error_is_ignored", body: internaltests.MarshalToJSON(t, types.RequestPasswordResetRequest{Email: "user@example.com"}), prepare: func(m *plugintests.MockRequestPasswordResetUseCase) {
			m.On("RequestReset", mock.Anything, "user@example.com", (*string)(nil)).Return(errors.New("boom")).Once()
		}, expectedStatus: http.StatusOK},
		{name: "success", body: internaltests.MarshalToJSON(t, types.RequestPasswordResetRequest{Email: "user@example.com"}), prepare: func(m *plugintests.MockRequestPasswordResetUseCase) {
			m.On("RequestReset", mock.Anything, "user@example.com", (*string)(nil)).Return(nil).Once()
		}, expectedStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := &plugintests.MockRequestPasswordResetUseCase{}
			if tt.prepare != nil {
				tt.prepare(uc)
			}
			handler := &RequestPasswordResetHandler{UseCase: uc}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/email-password/request-password-reset", tt.body, nil)
			handler.Handler().ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			uc.AssertExpectations(t)
		})
	}
}

func TestChangePasswordHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		withContext    bool
		body           []byte
		prepare        func(*plugintests.MockChangePasswordUseCase)
		expectedStatus int
	}{
		{name: "missing_request_context", withContext: false, body: internaltests.MarshalToJSON(t, types.ChangePasswordRequest{Token: "token", Password: "password123"}), expectedStatus: http.StatusInternalServerError},
		{name: "invalid_json", withContext: true, body: []byte("{"), expectedStatus: http.StatusUnprocessableEntity},
		{name: "usecase_error", withContext: true, body: internaltests.MarshalToJSON(t, types.ChangePasswordRequest{Token: "token", Password: "password123"}), prepare: func(m *plugintests.MockChangePasswordUseCase) {
			m.On("ChangePassword", mock.Anything, "token", "password123").Return(errors.New("boom")).Once()
		}, expectedStatus: http.StatusBadRequest},
		{name: "success", withContext: true, body: internaltests.MarshalToJSON(t, types.ChangePasswordRequest{Token: "token", Password: "password123"}), prepare: func(m *plugintests.MockChangePasswordUseCase) {
			m.On("ChangePassword", mock.Anything, "token", "password123").Return(nil).Once()
		}, expectedStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := &plugintests.MockChangePasswordUseCase{}
			if tt.prepare != nil {
				tt.prepare(uc)
			}
			handler := &ChangePasswordHandler{UseCase: uc}
			req, _, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/email-password/change-password", tt.body, nil)
			if !tt.withContext {
				req = req.WithContext(context.Background())
			}
			handler.Handler().ServeHTTP(reqCtx.ResponseWriter, req)
			if tt.withContext {
				require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			}
			uc.AssertExpectations(t)
		})
	}
}

func TestRequestEmailChangeHandler(t *testing.T) {
	t.Parallel()

	userID := "user-1"
	tests := []struct {
		name           string
		userID         *string
		body           []byte
		prepare        func(*plugintests.MockRequestEmailChangeUseCase)
		expectedStatus int
	}{
		{name: "unauthorized", body: internaltests.MarshalToJSON(t, types.RequestEmailChangeRequest{NewEmail: "new@example.com"}), expectedStatus: http.StatusUnauthorized},
		{name: "invalid_json", userID: &userID, body: []byte("{"), expectedStatus: http.StatusBadRequest},
		{name: "usecase_error", userID: &userID, body: internaltests.MarshalToJSON(t, types.RequestEmailChangeRequest{NewEmail: "new@example.com"}), prepare: func(m *plugintests.MockRequestEmailChangeUseCase) {
			m.On("RequestChange", mock.Anything, "user-1", "new@example.com", (*string)(nil)).Return(errors.New("boom")).Once()
		}, expectedStatus: http.StatusBadRequest},
		{name: "success", userID: &userID, body: internaltests.MarshalToJSON(t, types.RequestEmailChangeRequest{NewEmail: "new@example.com"}), prepare: func(m *plugintests.MockRequestEmailChangeUseCase) {
			m.On("RequestChange", mock.Anything, "user-1", "new@example.com", (*string)(nil)).Return(nil).Once()
		}, expectedStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := &plugintests.MockRequestEmailChangeUseCase{}
			if tt.prepare != nil {
				tt.prepare(uc)
			}
			handler := &RequestEmailChangeHandler{UseCase: uc}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/email-password/request-email-change", tt.body, tt.userID)
			handler.Handler().ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			uc.AssertExpectations(t)
		})
	}
}
