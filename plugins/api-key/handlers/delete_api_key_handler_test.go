package handlers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalerrors "github.com/Authula/authula/internal/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	apiKeyTests "github.com/Authula/authula/plugins/api-key/tests"
	"github.com/Authula/authula/plugins/api-key/types"
)

func TestDeleteApiKeyHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		path           string
		prepare        func(*apiKeyTests.MockApiKeyService)
		expectedStatus int
		checkResponse  func(*testing.T, *models.RequestContext)
	}{
		{
			name: "missing_id", path: "/api-keys",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service_error", path: "/api-keys/api-key-1",
			prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Delete", mock.Anything, "api-key-1").Return(internalerrors.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "success", path: "/api-keys/api-key-1", prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Delete", mock.Anything, "api-key-1").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.DeleteApiKeyResponse](t, reqCtx)
				assert.Equal(t, "API key deleted successfully", payload.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &apiKeyTests.MockApiKeyService{}
			if tt.prepare != nil {
				tt.prepare(service)
			}

			handler := &DeleteApiKeyHandler{Service: service}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodDelete, tt.path, nil, nil)
			if tt.path != "/api-keys" {
				req.SetPathValue("id", "api-key-1")
			}
			handler.Handle().ServeHTTP(w, req)

			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.checkResponse != nil {
				tt.checkResponse(t, reqCtx)
			}
			service.AssertExpectations(t)
		})
	}
}
