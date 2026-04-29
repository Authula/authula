package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	plugintests "github.com/Authula/authula/plugins/email-password/tests"
)

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
