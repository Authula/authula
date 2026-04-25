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

func TestGetApiKeyHandler(t *testing.T) {
	t.Parallel()

	apiKey := &types.ApiKey{ID: "api-key-1", Name: "Key 1", OwnerType: types.OwnerTypeUser, ReferenceID: "user-1"}
	tests := []struct {
		name           string
		path           string
		prepare        func(*apiKeyTests.MockApiKeyService)
		expectedStatus int
		checkResponse  func(*testing.T, *models.RequestContext)
	}{
		{
			name:           "missing_id",
			path:           "/api-keys",
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "not_found",
			path: "/api-keys/api-key-1",
			prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("GetByID", mock.Anything, "api-key-1").Return((*types.ApiKey)(nil), internalerrors.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "success",
			path: "/api-keys/api-key-1",
			prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("GetByID", mock.Anything, "api-key-1").Return(apiKey, nil).Once()
			},
			expectedStatus: http.StatusOK, checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.GetApiKeyResponse](t, reqCtx)
				assert.Equal(t, "api-key-1", payload.ApiKey.ID)
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

			handler := &GetApiKeyHandler{Service: service}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, tt.path, nil, nil)
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
