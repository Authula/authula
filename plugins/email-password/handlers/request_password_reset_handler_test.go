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
