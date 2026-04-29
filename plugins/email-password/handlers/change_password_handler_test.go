package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	plugintests "github.com/Authula/authula/plugins/email-password/tests"
	"github.com/Authula/authula/plugins/email-password/types"
)

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
