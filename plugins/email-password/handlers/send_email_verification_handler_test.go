package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	plugintests "github.com/Authula/authula/plugins/email-password/tests"
	"github.com/Authula/authula/plugins/email-password/types"
)

func TestSendEmailVerificationHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		userID         *string
		body           []byte
		prepare        func(*plugintests.MockSendEmailVerificationUseCase)
		expectedStatus int
	}{
		{name: "unauthorized", body: internaltests.MarshalToJSON(t, types.SendEmailVerificationRequest{}), expectedStatus: http.StatusUnauthorized},
		{name: "invalid_json", userID: func() *string { v := "user-1"; return &v }(), body: []byte("{"), expectedStatus: http.StatusUnprocessableEntity},
		{name: "usecase_error", userID: func() *string { v := "user-1"; return &v }(), body: internaltests.MarshalToJSON(t, map[string]any{"email": "attacker@example.com"}), prepare: func(m *plugintests.MockSendEmailVerificationUseCase) {
			m.On("Send", mock.Anything, "user-1", (*string)(nil)).Return(errors.New("boom")).Once()
		}, expectedStatus: http.StatusInternalServerError},
		{name: "success", userID: func() *string { v := "user-1"; return &v }(), body: internaltests.MarshalToJSON(t, map[string]any{"email": "attacker@example.com"}), prepare: func(m *plugintests.MockSendEmailVerificationUseCase) {
			m.On("Send", mock.Anything, "user-1", (*string)(nil)).Return(nil).Once()
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
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/email-password/send-email-verification", tt.body, tt.userID)
			handler.Handler().ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			uc.AssertExpectations(t)
		})
	}
}
