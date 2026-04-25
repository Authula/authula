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

func TestUpdateApiKeyHandler(t *testing.T) {
	t.Parallel()

	name := "updated"
	apiKey := &types.ApiKey{ID: "api-key-1", Name: "updated"}
	tests := []struct {
		name           string
		path           string
		body           []byte
		prepare        func(*apiKeyTests.MockApiKeyService)
		expectedStatus int
		checkResponse  func(*testing.T, *models.RequestContext)
	}{
		{
			name:           "missing_id",
			path:           "/api-keys",
			body:           internaltests.MarshalToJSON(t, types.UpdateApiKeyRequest{Name: &name}),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid_json",
			body:           []byte("{"),
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "invalid_request",
			body:           internaltests.MarshalToJSON(t, types.UpdateApiKeyRequest{Name: new(string)}),
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "not_found",
			path: "/api-keys/api-key-1",
			body: internaltests.MarshalToJSON(t, types.UpdateApiKeyRequest{Name: &name}),
			prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Update", mock.Anything, "api-key-1", types.UpdateApiKeyRequest{Name: &name}).Return((*types.ApiKey)(nil), internalerrors.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "success",
			path: "/api-keys/api-key-1",
			body: internaltests.MarshalToJSON(t, types.UpdateApiKeyRequest{Name: &name}),
			prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Update", mock.Anything, "api-key-1", types.UpdateApiKeyRequest{Name: &name}).Return(apiKey, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.UpdateApiKeyResponse](t, reqCtx)
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

			handler := &UpdateApiKeyHandler{Service: service}
			path := tt.path
			if path == "" {
				path = "/api-keys/api-key-1"
			}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPut, path, tt.body, nil)
			if path != "/api-keys" {
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
