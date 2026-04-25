package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	apiKeyTests "github.com/Authula/authula/plugins/api-key/tests"
	"github.com/Authula/authula/plugins/api-key/types"
)

func TestGetAllApiKeysHandler(t *testing.T) {
	t.Parallel()

	ownerType := types.OwnerTypeUser
	referenceID := "user-1"
	resp := &types.GetAllApiKeysResponse{Items: []*types.ApiKey{{ID: "api-key-1"}}, Total: 1, Page: 0, Limit: 10}

	tests := []struct {
		name           string
		path           string
		prepare        func(*apiKeyTests.MockApiKeyService)
		expectedStatus int
		checkResponse  func(*testing.T, *models.RequestContext)
	}{
		{
			name: "success_with_filters",
			path: "/api-keys?page=2&limit=50&owner_type=user&reference_id=user-1",
			prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("GetAll", mock.Anything, types.GetApiKeysRequest{Page: 2, Limit: 50, OwnerType: &ownerType, ReferenceID: &referenceID}).Return(resp, nil).Once()
			},
			expectedStatus: http.StatusOK, checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				payload := internaltests.DecodeResponseJSON[types.GetAllApiKeysResponse](t, reqCtx)
				assert.Len(t, payload.Items, 1)
				assert.Equal(t, 1, payload.Total)
				assert.Equal(t, 0, payload.Page)
				assert.Equal(t, 10, payload.Limit)
			},
		},
		{
			name: "service_error",
			path: "/api-keys",
			prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("GetAll", mock.Anything, types.GetApiKeysRequest{Page: 0, Limit: 0}).Return((*types.GetAllApiKeysResponse)(nil), errors.New("boom")).Once()
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "success_defaults",
			path: "/api-keys",
			prepare: func(m *apiKeyTests.MockApiKeyService) {
				m.On("GetAll", mock.Anything, types.GetApiKeysRequest{Page: 0, Limit: 0}).Return(resp, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &apiKeyTests.MockApiKeyService{}
			if tt.prepare != nil {
				tt.prepare(service)
			}

			handler := &GetAllApiKeysHandler{Service: service}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, tt.path, nil, nil)
			handler.Handle().ServeHTTP(w, req)

			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.checkResponse != nil {
				tt.checkResponse(t, reqCtx)
			}
			service.AssertExpectations(t)
		})
	}
}
