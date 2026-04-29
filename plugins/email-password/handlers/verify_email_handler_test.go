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
		name             string
		path             string
		trustedOrigins   []string
		prepare          func(*plugintests.MockVerifyEmailUseCase)
		expectedStatus   int
		expectedRedirect string
		assertBody       func(*testing.T, *models.RequestContext)
	}{
		{name: "missing_token", path: "/email-password/verify-email", expectedStatus: http.StatusUnprocessableEntity},
		{name: "usecase_error", path: "/email-password/verify-email?token=token-1", prepare: func(m *plugintests.MockVerifyEmailUseCase) {
			m.On("VerifyEmail", mock.Anything, "token-1").Return(models.VerificationType(""), errors.New("boom")).Once()
		}, expectedStatus: http.StatusUnprocessableEntity},
		{name: "success_without_callback", path: "/email-password/verify-email?token=token-1", prepare: func(m *plugintests.MockVerifyEmailUseCase) {
			m.On("VerifyEmail", mock.Anything, "token-1").Return(models.TypeEmailVerification, nil).Once()
		}, expectedStatus: http.StatusOK},
		{name: "trusted_absolute_redirect", path: "/email-password/verify-email?token=token-1&callback_url=https://example.com/callback", trustedOrigins: []string{"https://example.com"}, prepare: func(m *plugintests.MockVerifyEmailUseCase) {
			m.On("VerifyEmail", mock.Anything, "token-1").Return(models.TypePasswordResetRequest, nil).Once()
		}, expectedStatus: http.StatusFound, expectedRedirect: "https://example.com/callback?token=token-1"},
		{name: "trusted_absolute_redirect_preserves_query", path: "/email-password/verify-email?token=token-1&callback_url=https://example.com/callback?from=email", trustedOrigins: []string{"https://example.com"}, prepare: func(m *plugintests.MockVerifyEmailUseCase) {
			m.On("VerifyEmail", mock.Anything, "token-1").Return(models.TypeEmailVerification, nil).Once()
		}, expectedStatus: http.StatusFound, expectedRedirect: "https://example.com/callback?from=email"},
		{name: "relative_redirect", path: "/email-password/verify-email?token=token-1&callback_url=/welcome", prepare: func(m *plugintests.MockVerifyEmailUseCase) {
			m.On("VerifyEmail", mock.Anything, "token-1").Return(models.TypeEmailVerification, nil).Once()
		}, expectedStatus: http.StatusFound, expectedRedirect: "/welcome"},
		{name: "untrusted_absolute_redirect", path: "/email-password/verify-email?token=token-1&callback_url=https://evil.example/callback", trustedOrigins: []string{"https://example.com"}, expectedStatus: http.StatusBadRequest, assertBody: func(t *testing.T, reqCtx *models.RequestContext) {
			internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "callback_url is not a trusted origin")
		}},
		{name: "unsafe_scheme_redirect", path: "/email-password/verify-email?token=token-1&callback_url=javascript:alert(1)", expectedStatus: http.StatusBadRequest, assertBody: func(t *testing.T, reqCtx *models.RequestContext) {
			internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "callback_url uses an unsafe scheme")
		}},
		{name: "protocol_relative_redirect", path: "/email-password/verify-email?token=token-1&callback_url=//evil.example/callback", expectedStatus: http.StatusBadRequest, assertBody: func(t *testing.T, reqCtx *models.RequestContext) {
			internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "callback_url must not be protocol-relative")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ve := &plugintests.MockVerifyEmailUseCase{}
			if tt.prepare != nil {
				tt.prepare(ve)
			}
			handler := &VerifyEmailHandler{VerifyEmailUseCase: ve, TrustedOrigins: tt.trustedOrigins}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, tt.path, nil, nil)
			reqCtx.Request = req
			handler.Handler().ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.expectedRedirect != "" {
				require.Equal(t, tt.expectedRedirect, reqCtx.RedirectURL)
			}
			if tt.assertBody != nil {
				tt.assertBody(t, reqCtx)
			}
			ve.AssertExpectations(t)
		})
	}
}
