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
