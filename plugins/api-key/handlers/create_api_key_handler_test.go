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

func TestCreateApiKeyHandler(t *testing.T) {
	t.Parallel()

	validReq := types.CreateApiKeyRequest{Name: "Key 1", OwnerType: types.OwnerTypeUser, OwnerID: "user-1"}
	apiKey := &types.ApiKey{ID: "api-key-1", Name: "Key 1", OwnerType: types.OwnerTypeUser, OwnerID: "user-1", Start: "1234", Last: "5678", Enabled: true}

	tests := []struct {
		name           string
		body           []byte
		prepare        func(*apiKeyTests.MockApiKeyService)
		expectedStatus int
		checkResponse  func(*testing.T, *models.RequestContext)
	}{
		{
			name: "invalid_json", body: []byte("{"),
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "invalid_request",
			body:           internaltests.MarshalToJSON(t, types.CreateApiKeyRequest{Name: "", OwnerType: types.OwnerTypeUser, OwnerID: "user-1"}),
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "service_error",
			body: internaltests.MarshalToJSON(t, validReq),
			prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Create", mock.Anything, mock.Anything, validReq).Return((*types.CreateApiKeyResponse)(nil), coreerrors.ErrForbidden).Once()
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "success",
			body: internaltests.MarshalToJSON(t, validReq),
			prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("Create", mock.Anything, mock.Anything, validReq).Return(&types.CreateApiKeyResponse{RawApiKey: "raw-key", ApiKey: apiKey}, nil).Once()
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.CreateApiKeyResponse](t, reqCtx)
				assert.Equal(t, "raw-key", payload.RawApiKey)
				assert.Equal(t, "api-key-1", payload.ApiKey.ID)
				assert.Equal(t, "1234", payload.ApiKey.Start)
				assert.Equal(t, "5678", payload.ApiKey.Last)
				assert.Nil(t, payload.ApiKey.Prefix)
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

			handler := &CreateApiKeyHandler{UseCases: usecases.NewUseCases(service, &internaltests.NoopAuthorizer{})}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/api-keys", tt.body, nil)
			handler.Handle().ServeHTTP(w, req)

			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.checkResponse != nil {
				tt.checkResponse(t, reqCtx)
			}
			service.AssertExpectations(t)
		})
	}
}
