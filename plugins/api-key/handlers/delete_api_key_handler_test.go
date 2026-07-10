package handlers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	apiKeyTests "github.com/Authula/authula/plugins/api-key/tests"
	"github.com/Authula/authula/plugins/api-key/types"
	"github.com/Authula/authula/plugins/api-key/usecases"
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
				m.On("GetByID", mock.Anything, mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: "user-1"}, nil).Once()
				m.On("Delete", mock.Anything, mock.Anything, "api-key-1").Return(coreerrors.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "success", path: "/api-keys/api-key-1", prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("GetByID", mock.Anything, mock.Anything, "api-key-1").Return(&types.ApiKey{ID: "api-key-1", OwnerType: types.OwnerTypeUser, OwnerID: "user-1"}, nil).Once()
				m.On("Delete", mock.Anything, mock.Anything, "api-key-1").Return(nil).Once()
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

			handler := &DeleteApiKeyHandler{UseCases: usecases.NewUseCases(service, &internaltests.NoopAuthorizer{})}
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
