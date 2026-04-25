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

func TestVerifyApiKeyHandler(t *testing.T) {
	t.Parallel()

	apiKey := &types.ApiKey{ID: "api-key-1", Name: "verified"}
	tests := []struct {
		name           string
		body           []byte
		prepare        func(*apiKeyTests.MockApiKeyService)
		expectedStatus int
		checkResponse  func(*testing.T, *models.RequestContext)
	}{
		{
			name:           "invalid_json",
			body:           []byte("{"),
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "invalid_request",
			body:           internaltests.MarshalToJSON(t, types.VerifyApiKeyRequest{}),
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "service_error",
			body: internaltests.MarshalToJSON(t, types.VerifyApiKeyRequest{Key: "key-1"}),
			prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Verify", mock.Anything, types.VerifyApiKeyRequest{Key: "key-1"}).Return((*types.VerifyApiKeyResult)(nil), internalerrors.ErrForbidden).Once()
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "invalid_key",
			body: internaltests.MarshalToJSON(t, types.VerifyApiKeyRequest{Key: "key-1"}),
			prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Verify", mock.Anything, types.VerifyApiKeyRequest{Key: "key-1"}).Return(&types.VerifyApiKeyResult{Valid: false}, nil).Once()
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "success",
			body: internaltests.MarshalToJSON(t, types.VerifyApiKeyRequest{Key: "key-1"}),
			prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Verify", mock.Anything, types.VerifyApiKeyRequest{Key: "key-1"}).Return(&types.VerifyApiKeyResult{Valid: true, ApiKey: apiKey}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.VerifyApiKeyResponse](t, reqCtx)
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

			handler := &VerifyApiKeyHandler{Service: service}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/api-keys/verify", tt.body, nil)
			handler.Handle().ServeHTTP(w, req)

			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.checkResponse != nil {
				tt.checkResponse(t, reqCtx)
			}
			service.AssertExpectations(t)
		})
	}
}
